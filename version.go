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
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"bazil.org/fuse/fs"
)

//
//	version
//

type version struct {
	base      string
	parent    *version
	timestamp time.Time
	meta      *verMeta
	root      *verDir
	db        *database
}

func newVersion(base string, timestamp time.Time, db *database) (*version, error) {
	meta, err := newVerMeta(filepath.Join(base, "meta.json"))
	if err != nil {
		return nil, err
	}

	return &version{base, nil, timestamp, meta, nil, db}, nil
}

func (v *version) scanDir(path string) (verNodeMap, error) {
	var baseNodes verNodeMap
	if v.parent != nil {
		var err error

		baseNodes, err = v.parent.scanDir(path)
		if err != nil {
			return nil, err
		}

		v.meta.filter(baseNodes)
	}

	ownNodes := make(verNodeMap)
	{
		infos, err := ioutil.ReadDir(v.rebasePath(path))
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

				ownNodes[childName] = newVerNode(childPath, v, nil, childFlags)
			}
		}

		v.meta.filter(ownNodes)
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

func (v *version) buildDir(dir *verDir) error {
	nodes, err := v.scanDir(dir.node.path)
	if err != nil {
		return err
	}

	for name, node := range nodes {
		if node.flags&NodeFlagDir == NodeFlagDir {
			subDir := newVerDir(node, dir)
			if err := v.buildDir(subDir); err != nil {
				return err
			}

			dir.dirs[name] = subDir
		} else {
			dir.files[name] = newVerFile(node, dir)
		}
	}

	return nil
}

func (v *version) resolve() error {
	node := newVerNode("/", v, nil, NodeFlagDir)
	root := newVerDir(node, nil)

	if err := v.buildDir(root); err != nil {
		return err
	}

	v.root = root
	return nil
}

func (v *version) rebasePath(paths ...string) string {
	combined := append([]string{v.base, "root"}, paths...)
	return filepath.Join(combined...)
}

func (v *version) finalize(last bool) error {
	if v.meta.dirty {
		return v.meta.save()
	} else if last {
		if err := os.RemoveAll(v.base); err != nil {
			return err
		}
	}

	return nil
}

func (v *version) Root() (fs.Node, error) {
	return v.root, nil
}

//
// verList
//

type verList []*version

func (v verList) Len() int {
	return len(v)
}

func (v verList) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v verList) Less(i, j int) bool {
	return v[i].timestamp.Unix() < v[j].timestamp.Unix()
}
