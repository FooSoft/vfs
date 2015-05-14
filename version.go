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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type versionMetadata struct {
	Deleted []string
}

type versionedNode struct {
	path string
	info os.FileInfo
}

type version struct {
	base      string
	parent    *version
	timestamp time.Time
	meta      versionMetadata
	root      *versionedDir
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

	return ver, nil
}

func (this *version) scanDir(path string) (map[string]versionedNode, error) {
	var baseNodes map[string]versionedNode
	if this.parent != nil {
		var err error
		baseNodes, err = this.parent.scanDir(path)
		if err != nil {
			return nil, err
		}
	}

	ownNodes := make(map[string]versionedNode)
	{
		nodes, err := ioutil.ReadDir(this.rebasePath(path))
		if !os.IsNotExist(err) {
			if err != nil {
				return nil, err
			}

			for _, node := range nodes {
				name := node.Name()
				ownNodes[name] = versionedNode{this.rebasePath(path, name), node}
			}
		}
	}

	if baseNodes == nil {
		return ownNodes, nil
	}

	for ownName, ownNode := range ownNodes {
		baseNodes[ownName] = ownNode
	}

	return baseNodes, nil
}

func (this *version) buildDir(path string, dir *versionedDir) error {
	nodes, err := this.scanDir(path)
	if err != nil {
		return err
	}

	for name, node := range nodes {
		if node.info.IsDir() {
			subDir := newVersionedDir(node, this.allocInode())
			if err := this.buildDir(filepath.Join(path, name), subDir); err != nil {
				return err
			}

			dir.dirs[name] = subDir
		} else {
			dir.files[name] = newVersionedFile(node, this.allocInode())
		}
	}

	return nil
}

func (this *version) resolve() error {
	node, err := os.Stat(this.rebasePath("/"))
	if err != nil {
		return err
	}

	this.root = newVersionedDir(
		versionedNode{"/", node},
		this.allocInode())

	return this.buildDir("/", this.root)
}

func (this *version) loadMetadata() error {
	if _, err := os.Stat(this.metadataPath()); os.IsNotExist(err) {
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

func (this *version) rebasePath(paths ...string) string {
	combined := append([]string{this.base, "root"}, paths...)
	return filepath.Join(combined...)
}

func (this *version) allocInode() uint64 {
	this.inodeCnt++
	return this.inodeCnt
}

func (this *version) Root() (fs.Node, error) {
	return this.root, nil
}

func (this *version) dump(root *versionedDir, depth int) {
	indent := strings.Repeat("\t", depth)
	for name, dir := range root.dirs {
		fmt.Printf("%s+ %s (%s)\n", indent, name, dir.node.path)
		this.dump(dir, depth+1)
	}
	for name, file := range root.files {
		fmt.Printf("%s- %s (%s)\n", indent, name, file.node.path)
	}
}

func (this *version) dumpRoot() {
	this.dump(this.root, 0)
}
