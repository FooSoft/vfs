/*
 * Copyright (c) 2015 Alex Yatskov <alex@foosoft.net>
 * Author: Alex Yatskov <alex@foosoft.net>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package main

import (
	"os"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

//
//	verFile
//

type handleMap map[fuse.HandleID]*verFileHandle

type verFile struct {
	node    *verNode
	inode   uint64
	parent  *verDir
	handles handleMap
	mutex   sync.Mutex
}

func newVerFile(node *verNode, parent *verDir) *verFile {
	return &verFile{node, allocInode(), parent, make(handleMap), sync.Mutex{}}
}

func (vf *verFile) version() error {
	vf.mutex.Lock()
	defer vf.mutex.Unlock()

	if vf.node.flags&NodeFlagNew == NodeFlagNew {
		return nil
	}

	node := newVerNode(vf.node.path, vf.node.ver.db.lastVer(), vf.node, NodeFlagNew)
	if _, err := copyFile(vf.node.rebasedPath(), node.rebasedPath()); err != nil {
		return err
	}

	vf.node = node
	node.ver.meta.modifyNode(node.path)

	return nil
}

func (vf *verFile) open(flags fuse.OpenFlags, mode os.FileMode) (*verFileHandle, fuse.HandleID, error) {
	if !flags.IsReadOnly() {
		if err := vf.version(); err != nil {
			return nil, 0, err
		}
	}

	path := vf.node.rebasedPath()

	handle, err := os.OpenFile(path, int(flags), mode)
	if err != nil {
		return nil, 0, err
	}

	id := allocHandleId()
	verHandle := &verFileHandle{vf, path, handle}

	vf.mutex.Lock()
	vf.handles[id] = verHandle
	vf.mutex.Unlock()

	return verHandle, id, nil
}

func (vf *verFile) release(handle fuse.HandleID) {
	vf.mutex.Lock()
	delete(vf.handles, handle)
	vf.mutex.Unlock()
}

// Node
func (vf *verFile) Attr(ctx context.Context, attr *fuse.Attr) error {
	vf.node.attr(attr)
	attr.Inode = vf.inode
	return nil
}

// NodeGetattrer
func (vf *verFile) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	return vf.Attr(ctx, &resp.Attr)
}

// NodeSetattrer
func (vf *verFile) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	return vf.node.setAttr(req, resp)
}

// NodeOpener
func (vf *verFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	handle, id, err := vf.open(req.Flags, 0644)
	if err != nil {
		return nil, err
	}

	resp.Handle = id
	return handle, nil
}

// NodeFsyncer
func (vf *verFile) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	vf.mutex.Lock()
	defer vf.mutex.Unlock()
	return vf.handles[req.Handle].handle.Sync()
}

//
// verFileHandle
//

type verFileHandle struct {
	node   *verFile
	path   string
	handle *os.File
}

// HandleReader
func (vfh *verFileHandle) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	resp.Data = make([]byte, req.Size)
	if _, err := vfh.handle.ReadAt(resp.Data, req.Offset); err != nil {
		return err
	}

	return nil
}

// HandleReadAller
func (vfh *verFileHandle) ReadAll(ctx context.Context) ([]byte, error) {
	info, err := os.Stat(vfh.path)
	if err != nil {
		return nil, err
	}

	data := make([]byte, info.Size())
	if _, err := vfh.handle.Read(data); err != nil {
		return nil, err
	}

	return data, nil
}

// HandleWriter
func (vfh *verFileHandle) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	size, err := vfh.handle.WriteAt(req.Data, req.Offset)
	if err != nil {
		return err
	}

	resp.Size = size
	return nil
}

// HandleReleaser
func (vfh *verFileHandle) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	vfh.handle.Close()
	vfh.handle = nil

	vfh.node.release(req.Handle)
	return nil
}
