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
	dirty  bool
}

func newVersionedDir(node *versionedNode, parent *versionedDir) *versionedDir {
	return &versionedDir{
		dirs:   make(map[string]*versionedDir),
		files:  make(map[string]*versionedFile),
		node:   node,
		inode:  allocInode(),
		parent: parent,
		dirty:  false}
}

func (this *versionedDir) version() error {
	if this.dirty {
		return nil
	}

	version := this.node.ver.db.lastVersion()
	rebasedPath := version.rebasePath(this.node.path)

	if err := os.MkdirAll(rebasedPath, 0755); err != nil {
		return err
	}

	node, err := newVersionedNode(this.node.path, version, this.node)
	if err != nil {
		return err
	}

	this.node = node
	this.dirty = true

	return nil
}

func (this *versionedDir) createDir(name string) (*versionedDir, error) {
	childPath := path.Join(this.node.path, name)

	if err := os.Mkdir(this.node.ver.rebasePath(childPath), 0755); err != nil {
		return nil, err
	}

	node, err := newVersionedNode(childPath, this.node.ver, nil)
	if err != nil {
		return nil, err
	}

	dir := newVersionedDir(node, this)

	this.dirs[name] = dir
	return dir, nil
}

func (this *versionedDir) createFile(name string, flags int) (*versionedFile, error) {
	childPath := path.Join(this.node.path, name)

	handle, err := os.OpenFile(this.node.ver.rebasePath(childPath), flags, 0644)
	if err != nil {
		return nil, err
	}

	node, err := newVersionedNode(childPath, this.node.ver, nil)
	if err != nil {
		return nil, err
	}

	file := newVersionedFile(node, this)
	file.handle = handle

	this.files[name] = file
	return file, nil
}

func (this *versionedDir) Attr(ctx context.Context, attr *fuse.Attr) error {
	this.node.attr(attr)
	attr.Inode = this.inode
	return nil
}

func (this *versionedDir) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	if err := this.node.sync(); err != nil {
		return err
	}

	this.Attr(ctx, &resp.Attr)
	return nil
}

func (this *versionedDir) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	return this.node.setAttr(req, resp)
}

func (this *versionedDir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	if err := this.version(); err != nil {
		return nil, nil, err
	}

	if req.Mode.IsDir() {
		dir, err := this.createDir(req.Name)
		if err != nil {
			return nil, nil, err
		}

		return dir, dir, nil
	} else {
		file, err := this.createFile(req.Name, int(req.Flags))
		if err != nil {
			return nil, nil, err
		}

		return file, file, nil
	}
}

func (this *versionedDir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	if err := this.version(); err != nil {
		return nil, err
	}

	return this.createDir(req.Name)
}

func (this *versionedDir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	var fullPath string
	if req.Dir {
		fullPath = this.dirs[req.Name].node.rebasedPath()
		delete(this.dirs, req.Name)
	} else {
		fullPath = this.files[req.Name].node.rebasedPath()
		delete(this.files, req.Name)
	}

	return os.Remove(fullPath)
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
