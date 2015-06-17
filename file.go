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
	"errors"
	"io"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

//
//	versionedFile
//

type versionedFile struct {
	node   *versionedNode
	inode  uint64
	parent *versionedDir
	handle *os.File
}

func newVersionedFile(node *versionedNode, parent *versionedDir) *versionedFile {
	return &versionedFile{node, allocInode(), parent, nil}
}

func (this *versionedFile) version() error {
	if this.node.flags&NodeFlagVer == NodeFlagVer {
		return nil
	}

	node := newVersionedNode(this.node.path, this.node.ver.db.lastVersion(), this.node, NodeFlagVer)

	if _, err := fileCopy(this.node.rebasedPath(), node.rebasedPath()); err != nil {
		return err
	}

	node.ver.meta.modifyNode(node.path)
	this.node = node

	return nil
}

func (this *versionedFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	if this.handle != nil {
		return nil, errors.New("attempted to open already opened file")
	}

	if !req.Flags.IsReadOnly() {
		if err := this.version(); err != nil {
			return nil, err
		}
	}

	handle, err := os.OpenFile(this.node.rebasedPath(), int(req.Flags), 0644)
	if err != nil {
		return nil, err
	}

	this.handle = handle
	return this, nil
}

func (this *versionedFile) Attr(ctx context.Context, attr *fuse.Attr) error {
	this.node.attr(attr)
	attr.Inode = this.inode
	return nil
}

func (this *versionedFile) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	return this.Attr(ctx, &resp.Attr)
}

func (this *versionedFile) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	return this.node.setAttr(req, resp)
}

func (this *versionedFile) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	if this.handle == nil {
		return errors.New("attempted to read from unopened file")
	}

	resp.Data = make([]byte, req.Size)
	if _, err := this.handle.ReadAt(resp.Data, req.Offset); err != nil {
		return err
	}

	return nil
}

func (this *versionedFile) ReadAll(ctx context.Context) ([]byte, error) {
	if this.handle == nil {
		return nil, errors.New("attempted to read from unopened file")
	}

	info, err := os.Stat(this.node.rebasedPath())
	if err != nil {
		return nil, err
	}

	data := make([]byte, info.Size())
	if _, err := this.handle.Read(data); err != nil {
		return nil, err
	}

	return data, nil
}

func (this *versionedFile) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	if this.handle == nil {
		return errors.New("attempted to write to unopened file")
	}

	size, err := this.handle.WriteAt(req.Data, req.Offset)
	if err != nil {
		return err
	}

	resp.Size = size
	return nil
}

func (this *versionedFile) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	if this.handle == nil {
		return errors.New("attempted to release unopened file")
	}

	this.handle.Close()
	this.handle = nil

	return nil
}

func (this *versionedFile) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	if this.handle == nil {
		return errors.New("attempted to sync unopened file")
	}

	return this.handle.Sync()
}

//
// file helpers
//

func fileCopy(src, dst string) (int64, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	return io.Copy(srcFile, dstFile)
}
