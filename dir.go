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
	"os"
	"path"
)

//
//	versionedDir
//

type versionedDir struct {
	dirs   map[string]*versionedDir
	files  map[string]*versionedFile
	node   *versionedNode
	inode  uint64
	parent *versionedDir
}

func newVersionedDir(node *versionedNode, parent *versionedDir) *versionedDir {
	dirs := make(map[string]*versionedDir)
	files := make(map[string]*versionedFile)

	return &versionedDir{dirs, files, node, allocInode(), parent}
}

func (this *versionedDir) version() error {
	if this.node.flags&NodeFlagVer == NodeFlagVer {
		return nil
	}

	node := newVersionedNode(this.node.path, this.node.ver.db.lastVersion(), this.node, NodeFlagDir|NodeFlagVer)

	if err := os.MkdirAll(node.rebasedPath(), 0755); err != nil {
		return err
	}

	this.node = node
	return nil
}

func (this *versionedDir) createDir(name string) (*versionedDir, error) {
	if err := this.version(); err != nil {
		return nil, err
	}

	childPath := path.Join(this.node.path, name)
	if err := os.Mkdir(this.node.ver.rebasePath(childPath), 0755); err != nil {
		return nil, err
	}

	node := newVersionedNode(childPath, this.node.ver, nil, NodeFlagDir)
	dir := newVersionedDir(node, this)
	this.dirs[name] = dir

	return dir, nil
}

func (this *versionedDir) createFile(name string, flags int) (*versionedFile, error) {
	if err := this.version(); err != nil {
		return nil, err
	}

	childPath := path.Join(this.node.path, name)
	handle, err := os.OpenFile(this.node.ver.rebasePath(childPath), flags, 0644)
	if err != nil {
		return nil, err
	}

	node := newVersionedNode(childPath, this.node.ver, nil, 0)
	file := newVersionedFile(node, this)
	file.handle = handle
	this.files[name] = file

	return file, nil
}

func (this *versionedDir) Attr(ctx context.Context, attr *fuse.Attr) error {
	if err := this.node.attr(attr); err != nil {
		return err
	}

	attr.Inode = this.inode
	return nil
}

func (this *versionedDir) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	return this.Attr(ctx, &resp.Attr)
}

func (this *versionedDir) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	this.version()
	return this.node.setAttr(req, resp)
}

func (this *versionedDir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (node fs.Node, handle fs.Handle, err error) {
	if req.Mode.IsDir() {
		var dir *versionedDir
		if dir, err = this.createDir(req.Name); err == nil {
			node = dir
			handle = dir
		}
	} else if req.Mode.IsRegular() {
		var file *versionedFile
		if file, err = this.createFile(req.Name, int(req.Flags)); err == nil {
			node = file
			handle = file
		}
	} else {
		err = errors.New("unsupported filetype")
	}

	return
}

func (this *versionedDir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	return this.createDir(req.Name)
}

func (this *versionedDir) Remove(ctx context.Context, req *fuse.RemoveRequest) (err error) {
	if req.Dir {
		if err = this.dirs[req.Name].node.remove(); err == nil {
			delete(this.dirs, req.Name)
		}
	} else {
		if err = this.files[req.Name].node.remove(); err == nil {
			delete(this.files, req.Name)
		}
	}

	return
}

func (this *versionedDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries := []fuse.Dirent{{Inode: this.inode, Name: ".", Type: fuse.DT_Dir}}
	if this.parent != nil {
		entry := fuse.Dirent{Inode: this.parent.inode, Name: "..", Type: fuse.DT_Dir}
		entries = append(entries, entry)
	}

	for name, dir := range this.dirs {
		entry := fuse.Dirent{Inode: dir.inode, Name: name, Type: fuse.DT_Dir}
		entries = append(entries, entry)
	}

	for name, file := range this.files {
		entry := fuse.Dirent{Inode: file.inode, Name: name, Type: fuse.DT_File}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (this *versionedDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if dir, ok := this.dirs[name]; ok {
		return dir, nil
	}

	if file, ok := this.files[name]; ok {
		return file, nil
	}

	return nil, fuse.ENOENT
}
