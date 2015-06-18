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
	"os"
	"path"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
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

func (vd *versionedDir) version() error {
	if vd.node.flags&NodeFlagVer == NodeFlagVer {
		return nil
	}

	node := newVersionedNode(vd.node.path, vd.node.ver.db.lastVersion(), vd.node, NodeFlagDir|NodeFlagVer)

	if err := os.MkdirAll(node.rebasedPath(), 0755); err != nil {
		return err
	}

	node.ver.meta.modifyNode(node.path)
	vd.node = node

	return nil
}

func (vd *versionedDir) createDir(name string) (*versionedDir, error) {
	if err := vd.version(); err != nil {
		return nil, err
	}

	childPath := path.Join(vd.node.path, name)
	if err := os.Mkdir(vd.node.ver.rebasePath(childPath), 0755); err != nil {
		return nil, err
	}

	node := newVersionedNode(childPath, vd.node.ver, nil, NodeFlagDir|NodeFlagVer)
	dir := newVersionedDir(node, vd)
	vd.dirs[name] = dir

	node.ver.meta.createNode(node.path)
	return dir, nil
}

func (vd *versionedDir) createFile(name string, flags int) (*versionedFile, error) {
	if err := vd.version(); err != nil {
		return nil, err
	}

	childPath := path.Join(vd.node.path, name)
	handle, err := os.OpenFile(vd.node.ver.rebasePath(childPath), flags, 0644)
	if err != nil {
		return nil, err
	}

	node := newVersionedNode(childPath, vd.node.ver, nil, NodeFlagVer)
	file := newVersionedFile(node, vd)
	file.handle = handle
	vd.files[name] = file

	node.ver.meta.createNode(node.path)
	return file, nil
}

// Node
func (vd *versionedDir) Attr(ctx context.Context, attr *fuse.Attr) error {
	if err := vd.node.attr(attr); err != nil {
		return err
	}

	attr.Inode = vd.inode
	return nil
}

// NodeGetattrer
func (vd *versionedDir) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	return vd.Attr(ctx, &resp.Attr)
}

// NodeSetattrer
func (vd *versionedDir) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	vd.version()
	return vd.node.setAttr(req, resp)
}

// NodeCreater
func (vd *versionedDir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (node fs.Node, handle fs.Handle, err error) {
	if req.Mode.IsDir() {
		var dir *versionedDir
		if dir, err = vd.createDir(req.Name); err == nil {
			node = dir
			handle = dir
		}
	} else if req.Mode.IsRegular() {
		var file *versionedFile
		if file, err = vd.createFile(req.Name, int(req.Flags)); err == nil {
			node = file
			handle = file
		}
	} else {
		err = errors.New("unsupported filetype")
	}

	return
}

// NodeMkdirer
func (vd *versionedDir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	return vd.createDir(req.Name)
}

// NodeRemover
func (vd *versionedDir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	if req.Dir {
		node := vd.dirs[req.Name].node
		ver := node.ver

		if node.flags&NodeFlagVer == NodeFlagVer {
			if err := os.Remove(node.rebasedPath()); err != nil {
				return err
			}

			ver = ver.parent
		}

		ver.meta.removeNode(node.path)
		delete(vd.dirs, req.Name)
	} else {
		node := vd.files[req.Name].node
		ver := node.ver

		if node.flags&NodeFlagVer == NodeFlagVer {
			if err := os.Remove(node.rebasedPath()); err != nil {
				return err
			}

			ver = ver.parent
		}

		ver.meta.removeNode(node.path)
		delete(vd.files, req.Name)
	}

	return nil
}

// NodeRequestLookuper
func (vd *versionedDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if dir, ok := vd.dirs[name]; ok {
		return dir, nil
	}

	if file, ok := vd.files[name]; ok {
		return file, nil
	}

	return nil, fuse.ENOENT
}

// HandleReadDirAller
func (vd *versionedDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries := []fuse.Dirent{{Inode: vd.inode, Name: ".", Type: fuse.DT_Dir}}
	if vd.parent != nil {
		entry := fuse.Dirent{Inode: vd.parent.inode, Name: "..", Type: fuse.DT_Dir}
		entries = append(entries, entry)
	}

	for name, dir := range vd.dirs {
		entry := fuse.Dirent{Inode: dir.inode, Name: name, Type: fuse.DT_Dir}
		entries = append(entries, entry)
	}

	for name, file := range vd.files {
		entry := fuse.Dirent{Inode: file.inode, Name: name, Type: fuse.DT_File}
		entries = append(entries, entry)
	}

	return entries, nil
}
