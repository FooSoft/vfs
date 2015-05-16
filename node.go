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
	path string
	info os.FileInfo
	ver  *version
}

type versionedNodeMap map[string]*versionedNode

func (this *versionedNode) rebasedPath() string {
	return this.ver.rebasePath(this.path)
}

func (this *versionedNode) attr(attr *fuse.Attr) {
	stat := this.info.Sys().(*syscall.Stat_t)

	attr.Size = uint64(this.info.Size())
	attr.Blocks = uint64(stat.Blocks)
	attr.Atime = time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
	attr.Mtime = this.info.ModTime()
	attr.Ctime = time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))
	attr.Mode = this.info.Mode()
	attr.Nlink = uint32(stat.Nlink)
	attr.Uid = stat.Uid
	attr.Gid = stat.Gid
	attr.Rdev = uint32(stat.Rdev)
}

func (this *versionedNode) String() string {
	return fmt.Sprintf("%s (%s)", this.path, this.rebasedPath())
}
