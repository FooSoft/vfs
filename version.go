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
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

type versionMetadata struct {
	Deleted []string
}

type versionedFile struct {
	info  os.FileInfo
	inode uint64
}

type versionedDir struct {
	dirs  map[string]versionedDir
	files map[string]versionedFile
	info  os.FileInfo
	inode uint64
}

func (this versionedDir) Attr(attr *fuse.Attr) {
	attr.Mode = this.info.Mode()
	attr.Inode = this.inode
}

func (this versionedDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var entries []fuse.Dirent

	for name, dir := range this.dirs {
		entry := fuse.Dirent{Inode: dir.inode, Name: name, Type: fuse.DT_File}
		entries = append(entries, entry)
	}

	for name, file := range this.files {
		entry := fuse.Dirent{Inode: file.inode, Name: name, Type: fuse.DT_Dir}
		entries = append(entries, entry)
	}

	return nil, nil
}

type version struct {
	base      string
	parent    *version
	timestamp time.Time
	meta      versionMetadata
	root      versionedDir
	inodeCnt  uint64
}

func newVersion(base string, parent *version) (*version, error) {
	re, err := regexp.Compile(`/vfs_([0-9a-f])$`)
	if err != nil {
		return nil, err
	}

	matches := re.FindStringSubmatch(base)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid version identifier for %s", base)
	}

	timeval, err := strconv.ParseInt(matches[1], 16, 64)
	if err != nil {
		return nil, err
	}

	ver := &version{
		base:      base,
		parent:    parent,
		timestamp: time.Unix(timeval, 0)}

	if err := ver.loadMetadata(); err != nil {
		return nil, err
	}

	if err := ver.scanFs(); err != nil {
		return nil, err
	}

	return ver, nil
}

func (this *version) scanDir(path string) (map[string]os.FileInfo, error) {
	ownNodes := make(map[string]os.FileInfo)
	{
		fullPath := filepath.Join(this.rootPath(), path)
		nodes, err := ioutil.ReadDir(fullPath)
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			ownNodes[node.Name()] = node
		}
	}

	if this.parent == nil {
		return ownNodes, nil
	}

	baseNodes, err := this.parent.scanDir(path)
	if err != nil {
		return nil, err
	}

	for ownName, ownInfo := range ownNodes {
		baseNodes[ownName] = ownInfo
	}

	return baseNodes, nil
}

func (this *version) buildDir(path string, dir *versionedDir) error {
	nodes, err := this.scanDir(path)
	if err != nil {
		return err
	}

	for name, info := range nodes {
		if info.IsDir() {
			subDir := versionedDir{info: info, inode: this.allocInode()}
			subDirPath := filepath.Join(path, name)
			if err := this.buildDir(subDirPath, &subDir); err != nil {
				return err
			}
			dir.dirs[name] = subDir
		} else {
			dir.files[name] = versionedFile{info: info, inode: this.allocInode()}
		}
	}

	return nil
}

func (this *version) scanFs() error {
	rootNode, err := os.Stat(this.rootPath())
	if err != nil {
		return err
	}

	this.root = versionedDir{info: rootNode, inode: this.allocInode()}
	return this.buildDir(this.rootPath(), &this.root)
}

func (this *version) loadMetadata() error {
	path := this.metadataPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	bytes, err := ioutil.ReadFile(this.metadataPath())
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes, &this.meta); err != nil {
		return err
	}

	return nil
}

func (this *version) saveMetadata() error {
	js, err := json.Marshal(this.meta)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(this.metadataPath(), js, 0644)
}

func (this *version) metadataPath() string {
	return filepath.Join(this.base, "meta.json")
}

func (this *version) rootPath() string {
	return filepath.Join(this.base, "root")
}

func (this *version) allocInode() uint64 {
	this.inodeCnt++
	return this.inodeCnt
}

func (this *version) Root() (fs.Node, error) {
	return this.root, nil
}

// func (this *version) Attr(a *fuse.Attr) {
// 	// type Attr struct {
// 	// Inode  uint64      // inode number
// 	// Size   uint64      // size in bytes
// 	// Blocks uint64      // size in blocks
// 	// Atime  time.Time   // time of last access
// 	// Mtime  time.Time   // time of last modification
// 	// Ctime  time.Time   // time of last inode change
// 	// Crtime time.Time   // time of creation (OS X only)
// 	// Mode   os.FileMode // file mode
// 	// Nlink  uint32      // number of links
// 	// Uid    uint32      // owner uid
// 	// Gid    uint32      // group gid
// 	// Rdev   uint32      // device numbers
// 	// Flags  uint32      // chflags(2) flags (OS X only)
// 	a.Inode = 1
// 	a.Mode = os.ModeDir | 0755
// }

// func (this *version) Lookup(ctx context.Context, name string) (fs.Node, error) {
// 	// if name == "hello" {
// 	// 	return File{}, nil
// 	// }
// 	return nil, fuse.ENOENT
// }

// func (this *version) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
// 	var dirDirs = []fuse.Dirent{
// 		{Inode: 2, Name: "hello", Type: fuse.DT_File},
// 	}

// 	return dirDirs, nil
// }
