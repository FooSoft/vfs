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

func newVersion(base string, timestamp time.Time, db *database) (*version, error) {
	meta, err := newVersionMetadata(filepath.Join(base, "meta.json"))
	if err != nil {
		return nil, err
	}

	return &version{base, nil, timestamp, meta, nil, db}, nil
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
				childFlags := 0
				if info.IsDir() {
					childFlags |= NodeFlagDir
				}

				childName := info.Name()
				childPath := filepath.Join(path, childName)

				ownNodes[childName] = newVersionedNode(childPath, this, nil, childFlags)
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

func (this *version) buildDir(dir *versionedDir) error {
	nodes, err := this.scanDir(dir.node.path)
	if err != nil {
		return err
	}

	for name, node := range nodes {
		if node.flags&NodeFlagDir == NodeFlagDir {
			subDir := newVersionedDir(node, dir)
			if err := this.buildDir(subDir); err != nil {
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
	node := newVersionedNode("/", this, nil, NodeFlagDir)
	root := newVersionedDir(node, nil)

	if err := this.buildDir(root); err != nil {
		return err
	}

	this.root = root
	return nil
}

func (this *version) rebasePath(paths ...string) string {
	combined := append([]string{this.base, "root"}, paths...)
	return filepath.Join(combined...)
}

func (this *version) finalize() error {
	return this.meta.save()
}

func (this *version) Root() (fs.Node, error) {
	return this.root, nil
}

//
// versionList
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

//
//	version helpers
//

func buildNewVersion(base string) error {
	name := buildVerName(time.Now())
	if err := os.MkdirAll(path.Join(base, name, "root"), 0755); err != nil {
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
