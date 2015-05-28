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
	"fmt"
	"os"
	"syscall"
	"time"
)

type versionedNode struct {
	path   string
	info   os.FileInfo
	parent *versionedNode
	shadow *versionedNode
	ver    *version
}

type versionedNodeMap map[string]*versionedNode

func newVersionedNode(path string, parent *versionedNode, ver *version) (*versionedNode, error) {
	info, err := os.Stat(ver.rebasePath(path))
	if err != nil {
		return nil, err
	}

	return newVersionedNodeStat(path, parent, ver, info), nil
}

func newVersionedNodeStat(path string, parent *versionedNode, ver *version, info os.FileInfo) *versionedNode {
	return &versionedNode{path, info, parent, nil, ver}
}

func (this *versionedNode) setAttr(req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	if req.Valid&fuse.SetattrMode != 0 {
		if err := os.Chmod(this.rebasedPath(), req.Mode); err != nil {
			return err
		}
	}

	if setGid, setUid := req.Valid&fuse.SetattrGid != 0, req.Valid&fuse.SetattrUid != 0; setGid || setUid {
		gid, uid := this.owner()
		if setGid {
			gid = req.Gid
		}
		if setUid {
			uid = req.Uid
		}

		if err := os.Chown(this.rebasedPath(), int(uid), int(gid)); err != nil {
			return err
		}
	}

	if setAtime, setMtime := req.Valid&fuse.SetattrAtime != 0, req.Valid&fuse.SetattrMtime != 0; setAtime || setMtime {
		atime, mtime, _ := this.times()
		if setAtime {
			atime = req.Atime
		}
		if setMtime {
			mtime = req.Mtime
		}

		if err := os.Chtimes(this.rebasedPath(), atime, mtime); err != nil {
			return err
		}
	}

	if err := this.updateInfo(); err != nil {
		return err
	}

	this.attr(&resp.Attr)
	return nil
}

func (this *versionedNode) updateInfo() error {
	info, err := os.Stat(this.rebasedPath())
	if err != nil {
		return err
	}

	this.info = info
	return nil
}

func (this *versionedNode) rebasedPath() string {
	return this.ver.rebasePath(this.path)
}

func (this *versionedNode) owner() (gid, uid uint32) {
	stat := this.info.Sys().(*syscall.Stat_t)

	gid = stat.Gid
	uid = stat.Uid
	return
}

func (this *versionedNode) times() (atime, mtime, ctime time.Time) {
	stat := this.info.Sys().(*syscall.Stat_t)

	atime = time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
	mtime = this.info.ModTime()
	ctime = time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))
	return
}

func (this *versionedNode) attr(attr *fuse.Attr) {
	stat := this.info.Sys().(*syscall.Stat_t)

	attr.Size = uint64(this.info.Size())
	attr.Blocks = uint64(stat.Blocks)
	attr.Atime, attr.Mtime, attr.Ctime = this.times()
	attr.Mode = this.info.Mode()
	attr.Nlink = uint32(stat.Nlink)
	attr.Gid, attr.Uid = this.owner()
	attr.Rdev = uint32(stat.Rdev)
}

func (this *versionedNode) String() string {
	return fmt.Sprintf("%s (%s)", this.path, this.rebasedPath())
}
