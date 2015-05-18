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
	"golang.org/x/net/context"
	"io/ioutil"
	"log"
)

type versionedFile struct {
	node   *versionedNode
	inode  uint64
	parent *versionedDir
}

func newVersionedFile(node *versionedNode, inode uint64, parent *versionedDir) *versionedFile {
	return &versionedFile{
		node:   node,
		inode:  inode,
		parent: parent}
}

func (this versionedFile) Attr(attr *fuse.Attr) {
	log.Printf("versionedFile::Attr: %s", this.node)

	this.node.attr(attr)
	attr.Inode = this.inode
}

func (this versionedFile) ReadAll(ctx context.Context) ([]byte, error) {
	log.Printf("versionedFile::ReadAll: %s", this.node)

	bytes, err := ioutil.ReadFile(this.node.rebasedPath())
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
