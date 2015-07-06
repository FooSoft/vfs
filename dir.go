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
//	verDir
//

type verDir struct {
	dirs   map[string]*verDir
	files  map[string]*verFile
	node   *verNode
	inode  uint64
	parent *verDir
}

func newVerDir(node *verNode, parent *verDir) *verDir {
	dirs := make(map[string]*verDir)
	files := make(map[string]*verFile)

	return &verDir{dirs, files, node, allocInode(), parent}
}

func (vd *verDir) version() error {
	if vd.node.flags&NodeFlagNew == NodeFlagNew {
		return nil
	}

	node := newVerNode(vd.node.path, vd.node.ver.db.lastVer(), vd.node, NodeFlagDir|NodeFlagNew)
	if err := os.MkdirAll(node.rebasedPath(), 0755); err != nil {
		return err
	}

	vd.node = node
	node.ver.meta.modifyNode(node.path)

	return nil
}

func (vd *verDir) createDir(name string) (*verDir, error) {
	if err := vd.version(); err != nil {
		return nil, err
	}

	childPath := path.Join(vd.node.path, name)
	if err := os.Mkdir(vd.node.ver.rebasePath(childPath), 0755); err != nil {
		return nil, err
	}

	node := newVerNode(childPath, vd.node.ver, nil, NodeFlagDir|NodeFlagNew)
	dir := newVerDir(node, vd)

	vd.dirs[name] = dir

	node.ver.meta.createNode(node.path)
	return dir, nil
}

func (vd *verDir) createFile(name string, flags fuse.OpenFlags, mode os.FileMode) (*verFile, *verFileHandle, fuse.HandleID, error) {
	if err := vd.version(); err != nil {
		return nil, nil, 0, err
	}

	childPath := path.Join(vd.node.path, name)
	node := newVerNode(childPath, vd.node.ver, nil, NodeFlagNew)
	file := newVerFile(node, vd)

	handle, id, err := file.open(flags, mode)
	if err != nil {
		return nil, nil, 0, err
	}

	vd.files[name] = file

	node.ver.meta.createNode(node.path)
	return file, handle, id, nil
}

func (vd *verDir) removeDir(name string) error {
	if err := vd.version(); err != nil {
		return err
	}

	node := vd.dirs[name].node
	if node.flags&NodeFlagNew == NodeFlagNew {
		if err := os.Remove(node.rebasedPath()); err != nil {
			return err
		}
	} else {
		vd.node.ver.meta.removeNode(node.path)
	}

	delete(vd.dirs, name)
	return nil
}

func (vd *verDir) removeFile(name string) error {
	if err := vd.version(); err != nil {
		return err
	}

	node := vd.files[name].node
	if node.flags&NodeFlagNew == NodeFlagNew {
		if err := os.Remove(node.rebasedPath()); err != nil {
			return err
		}
	} else {
		vd.node.ver.meta.removeNode(node.path)
	}

	delete(vd.files, name)
	return nil
}

// Node
func (vd *verDir) Attr(ctx context.Context, attr *fuse.Attr) error {
	if err := vd.node.attr(attr); err != nil {
		return err
	}

	attr.Inode = vd.inode
	return nil
}

// NodeGetattrer
func (vd *verDir) Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error {
	return vd.Attr(ctx, &resp.Attr)
}

// NodeSetattrer
func (vd *verDir) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	vd.version()
	return vd.node.setAttr(req, resp)
}

// NodeCreater
func (vd *verDir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (node fs.Node, handle fs.Handle, err error) {
	if req.Mode.IsDir() {
		node, err = vd.createDir(req.Name)
		handle = node
	} else if req.Mode.IsRegular() {
		node, handle, resp.Handle, err = vd.createFile(req.Name, req.Flags, req.Mode)
	} else {
		err = errors.New("unsupported filetype")
	}

	return
}

// NodeMkdirer
func (vd *verDir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	return vd.createDir(req.Name)
}

// NodeRemover
func (vd *verDir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	if req.Dir {
		return vd.removeDir(req.Name)
	} else {
		return vd.removeFile(req.Name)
	}
}

// NodeRequestLookuper
func (vd *verDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if dir, ok := vd.dirs[name]; ok {
		return dir, nil
	}

	if file, ok := vd.files[name]; ok {
		return file, nil
	}

	return nil, fuse.ENOENT
}

// HandleReadDirAller
func (vd *verDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
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
