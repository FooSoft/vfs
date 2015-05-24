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

type versionedDir struct {
	dirs   map[string]*versionedDir
	files  map[string]*versionedFile
	node   *versionedNode
	inode  uint64
	parent *versionedDir
}

func newVersionedDir(node *versionedNode, inode uint64, parent *versionedDir) *versionedDir {
	return &versionedDir{
		dirs:   make(map[string]*versionedDir),
		files:  make(map[string]*versionedFile),
		node:   node,
		inode:  inode,
		parent: parent}
}

func (this *versionedDir) createDir(name string) (*versionedDir, error) {
	childPath := path.Join(this.node.path, name)
	childPathFull := this.node.ver.rebasePath(childPath)

	if err := os.Mkdir(childPathFull, 0755); err != nil {
		return nil, err
	}

	info, err := os.Stat(childPathFull)
	if err != nil {
		return nil, err
	}

	node := &versionedNode{childPath, info, this.node.ver}
	return newVersionedDir(node, this.node.ver.inodeAloc.AllocInode(), this), nil
}

func (this *versionedDir) createFile(name string, flags int) (*versionedFile, error) {
	childPath := path.Join(this.node.path, name)
	childPathFull := this.node.ver.rebasePath(childPath)

	file, err := os.OpenFile(childPathFull, flags, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := os.Stat(childPathFull)
	if err != nil {
		return nil, err
	}

	node := &versionedNode{childPath, info, this.node.ver}
	return newVersionedFile(node, this.node.ver.inodeAloc.AllocInode(), this), nil
}

func (this *versionedDir) Attr(attr *fuse.Attr) {
	this.node.attr(attr)
	attr.Inode = this.inode
}

func (this *versionedDir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
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
	return this.createDir(req.Name)
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
