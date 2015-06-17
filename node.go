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
	"os"
	"syscall"
	"time"

	"bazil.org/fuse"
)

//
//	versionedNode
//

const (
	NodeFlagDir = 1 << iota
	NodeFlagVer
)

type versionedNode struct {
	path   string
	ver    *version
	parent *versionedNode
	flags  int
}

func newVersionedNode(path string, ver *version, parent *versionedNode, flags int) *versionedNode {
	return &versionedNode{path, ver, parent, flags}
}

func (n *versionedNode) setAttr(req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	if err := n.attr(&resp.Attr); err != nil {
		return err
	}

	if req.Valid&fuse.SetattrMode != 0 {
		if err := os.Chmod(n.rebasedPath(), req.Mode); err != nil {
			return err
		}

		resp.Attr.Mode = req.Mode
	}

	if setGid, setUid := req.Valid&fuse.SetattrGid != 0, req.Valid&fuse.SetattrUid != 0; setGid || setUid {
		if setGid {
			resp.Attr.Gid = req.Gid
		}
		if setUid {
			resp.Attr.Uid = req.Uid
		}

		if err := os.Chown(n.rebasedPath(), int(resp.Attr.Uid), int(resp.Attr.Gid)); err != nil {
			return err
		}
	}

	if setAtime, setMtime := req.Valid&fuse.SetattrAtime != 0, req.Valid&fuse.SetattrMtime != 0; setAtime || setMtime {
		if setAtime {
			resp.Attr.Atime = req.Atime
		}
		if setMtime {
			resp.Attr.Mtime = req.Mtime
		}

		if err := os.Chtimes(n.rebasedPath(), resp.Attr.Atime, resp.Attr.Mtime); err != nil {
			return err
		}
	}

	return nil
}

func (n *versionedNode) rebasedPath() string {
	return n.ver.rebasePath(n.path)
}

func (n *versionedNode) owner(stat syscall.Stat_t) (gid, uid uint32) {
	gid = stat.Gid
	uid = stat.Uid
	return
}

func (n *versionedNode) times(stat syscall.Stat_t) (atime, mtime, ctime time.Time) {
	atime = time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
	mtime = time.Unix(int64(stat.Mtim.Sec), int64(stat.Mtim.Nsec))
	ctime = time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))
	return
}

func (n *versionedNode) attr(attr *fuse.Attr) error {
	info, err := os.Stat(n.rebasedPath())
	if err != nil {
		return err
	}

	stat := info.Sys().(*syscall.Stat_t)

	attr.Size = uint64(stat.Size)
	attr.Blocks = uint64(stat.Blocks)
	attr.Atime, attr.Mtime, attr.Ctime = n.times(*stat)
	attr.Mode = info.Mode()
	attr.Nlink = uint32(stat.Nlink)
	attr.Gid, attr.Uid = n.owner(*stat)
	attr.Rdev = uint32(stat.Rdev)

	return nil
}

//
// versionedNodeMap
//

type versionedNodeMap map[string]*versionedNode
