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
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"errors"
	"golang.org/x/net/context"
	"log"
	"os"
)

type versionedFile struct {
	node   *versionedNode
	inode  uint64
	parent *versionedDir
	handle *os.File
}

func newVersionedFile(node *versionedNode, parent *versionedDir) *versionedFile {
	return &versionedFile{
		node:   node,
		inode:  node.ver.inodeAloc.AllocInode(),
		parent: parent}
}

func (this *versionedFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	if this.handle != nil {
		return nil, errors.New("attempted to open already opened file")
	}

	handle, err := os.OpenFile(this.node.rebasedPath(), int(req.Flags), 0644)
	if err != nil {
		return nil, err
	}

	this.handle = handle
	return this, nil
}

func (this *versionedFile) Attr(attr *fuse.Attr) {
	this.node.attr(attr)
	attr.Inode = this.inode
}

func (this *versionedFile) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	if this.handle == nil {
		return errors.New("attempted to write unopened file")
	}

	size, err := this.handle.WriteAt(req.Data, req.Offset)
	if err != nil {
		return err
	}

	resp.Size = size
	return nil
}

func (this *versionedFile) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	info, err := os.Stat(this.node.rebasedPath())
	if err != nil {
		return err
	}

	log.Printf("Setattr: %s => %v", this.node.path, req)

	this.node.info = info
	this.Attr(&resp.Attr)
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
	return nil
}

func (this *versionedFile) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	if this.handle == nil {
		return errors.New("attempted to read unopened file")
	}

	resp.Data = make([]byte, req.Size)
	if _, err := this.handle.ReadAt(resp.Data, req.Offset); err != nil {
		return err
	}

	return nil
}

func (this *versionedFile) ReadAll(ctx context.Context) ([]byte, error) {
	if this.handle == nil {
		return nil, errors.New("attempted to read unopened file")
	}

	data := make([]byte, this.node.info.Size())
	if _, err := this.handle.Read(data); err != nil {
		return nil, err
	}

	return data, nil
}
