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
	"bazil.org/fuse/fs"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type version struct {
	base      string
	parent    *version
	terminus  *version
	timestamp time.Time
	meta      *versionMetadata
	root      *versionedDir
	inodeAloc InodeAllocator
}

type InodeAllocator interface {
	AllocInode() uint64
}

func newVersion(base string, timestamp time.Time, allocator InodeAllocator, parent, terminus *version) (*version, error) {
	meta, err := newVersionMetadata(filepath.Join(base, "meta.json"))
	if err != nil {
		return nil, err
	}

	ver := &version{
		base:      base,
		parent:    parent,
		terminus:  terminus,
		timestamp: timestamp,
		meta:      meta,
		inodeAloc: allocator}

	return ver, nil
}

func (this *version) scanDir(path string) (versionedNodeMap, error) {
	var baseNodes versionedNodeMap
	if this.parent != nil {
		var err error

		baseNodes, err = this.parent.scanDir(path)
		if err != nil {
			return nil, err
		}

		this.meta.filter(baseNodes)
	}

	ownNodes := make(versionedNodeMap)
	{
		infos, err := ioutil.ReadDir(this.rebasePath(path))
		if !os.IsNotExist(err) {
			if err != nil {
				return nil, err
			}

			for _, info := range infos {
				childName := info.Name()
				childPath := filepath.Join(path, childName)
				ownNodes[childName] = newVersionedNodeStat(childPath, this, info)
			}
		}

		this.meta.filter(ownNodes)
	}

	if baseNodes == nil {
		return ownNodes, nil
	}

	for ownName, ownNode := range ownNodes {
		ownNode.shadow = baseNodes[ownName]
		baseNodes[ownName] = ownNode
	}

	return baseNodes, nil
}

func (this *version) buildVerDir(dir *versionedDir) error {
	nodes, err := this.scanDir(dir.node.path)
	if err != nil {
		return err
	}

	for name, node := range nodes {
		if node.info.IsDir() {
			subDir := newVersionedDir(node, dir)
			if err := this.buildVerDir(subDir); err != nil {
				return err
			}

			dir.dirs[name] = subDir
		} else {
			dir.files[name] = newVersionedFile(node, dir)
		}
	}

	return nil
}

func (this *version) resolve() error {
	node, err := newVersionedNode("/", this)
	if err != nil {
		return err
	}

	root := newVersionedDir(node, nil)
	if err = this.buildVerDir(root); err != nil {
		return err
	}

	this.root = root
	return nil
}

func (this *version) rebasePath(paths ...string) string {
	combined := append([]string{this.base, "root"}, paths...)
	return filepath.Join(combined...)
}

func (this *version) Root() (fs.Node, error) {
	return this.root, nil
}

func (this *version) dump(root *versionedDir, depth int) {
	indent := strings.Repeat("\t", depth)
	for name, dir := range root.dirs {
		fmt.Printf("%s+ %s [%s@%x]\n", indent, name, dir.node.path, this.timestamp.Unix())
		this.dump(dir, depth+1)
	}
	for name, file := range root.files {
		fmt.Printf("%s- %s [%s@%x]\n", indent, name, file.node.path, this.timestamp.Unix())
	}
}

func (this *version) dumpRoot() {
	this.dump(this.root, 0)
}
