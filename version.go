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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//
//	version
//

type version struct {
	base      string
	parent    *version
	timestamp time.Time
	meta      *versionMetadata
	root      *versionedDir
	db        *database
}

func newVersion(base string, timestamp time.Time, db *database, parent *version) (*version, error) {
	meta, err := newVersionMetadata(filepath.Join(base, "meta.json"))
	if err != nil {
		return nil, err
	}

	return &version{base, parent, timestamp, meta, nil, db}, nil
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
				ownNodes[childName] = newVersionedNodeStat(childPath, this, nil, info)
			}
		}

		this.meta.filter(ownNodes)
	}

	if baseNodes == nil {
		return ownNodes, nil
	}

	for ownName, ownNode := range ownNodes {
		ownNode.parent, _ = baseNodes[ownName]
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
	node, err := newVersionedNode("/", this, nil)
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

func (this *version) removePath(path string) {
	this.meta.Deleted = append(this.meta.Deleted, path)
}

func (this *version) finalize() error {
	return this.meta.save()
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

func (this *version) Root() (fs.Node, error) {
	return this.root, nil
}

//
//	version helpers
//

type versionList []*version

func (this versionList) Len() int {
	return len(this)
}

func (this versionList) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

func (this versionList) Less(i, j int) bool {
	return this[i].timestamp.Unix() < this[j].timestamp.Unix()
}

func buildNewVersion(base string) error {
	name := buildVerName(time.Now())

	if err := os.Mkdir(path.Join(base, name), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(path.Join(base, name, "root"), 0755); err != nil {
		return err
	}

	return nil
}

func buildVerName(timestamp time.Time) string {
	return fmt.Sprintf("ver_%.16x", timestamp.Unix())
}

func parseVerName(name string) (time.Time, error) {
	re, err := regexp.Compile(`ver_([0-9a-f]+)$`)
	if err != nil {
		return time.Unix(0, 0), err
	}

	matches := re.FindStringSubmatch(name)
	if len(matches) < 2 {
		return time.Unix(0, 0), errors.New("invalid version identifier")
	}

	timestamp, err := strconv.ParseInt(matches[1], 16, 64)
	if err != nil {
		return time.Unix(0, 0), err
	}

	return time.Unix(timestamp, 0), nil
}
