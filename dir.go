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
	"log"
)

type versionedDir struct {
	dirs  map[string]*versionedDir
	files map[string]*versionedFile
	node  *versionedNode
	inode uint64
}

func newVersionedDir(node *versionedNode, inode uint64) *versionedDir {
	return &versionedDir{
		dirs:  make(map[string]*versionedDir),
		files: make(map[string]*versionedFile),
		node:  node,
		inode: inode}
}

func (this versionedDir) Attr(attr *fuse.Attr) {
	log.Printf("versionedDir::Attr: %s", this.node)

	this.node.attr(attr)
	attr.Inode = this.inode
}

func (this versionedDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	log.Printf("versionedDir::ReadDirAll: %s", this.node)

	var entries []fuse.Dirent
	for name, dir := range this.dirs {
		entry := fuse.Dirent{Inode: dir.inode, Name: name, Type: fuse.DT_File}
		entries = append(entries, entry)
	}
	for name, file := range this.files {
		entry := fuse.Dirent{Inode: file.inode, Name: name, Type: fuse.DT_Dir}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (this versionedDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	log.Printf("versionedDir::Lookup: %s", this.node)

	if dir, ok := this.dirs[name]; ok {
		return dir, nil
	}

	if file, ok := this.files[name]; ok {
		return file, nil
	}

	return nil, fuse.ENOENT
}
