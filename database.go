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
	"io/ioutil"
	"path"
	"path/filepath"
	"sort"
)

//
//	database
//

type database struct {
	base string
	vers versionList
}

func newDatabase(dir string) (*database, error) {
	db := &database{base: dir}
	if err := db.load(dir); err != nil {
		return nil, err
	}

	return db, nil
}

func (this *database) load(dir string) error {
	var err error

	this.base, err = filepath.Abs(dir)
	if err != nil {
		return err
	}

	if err := buildNewVersion(this.base); err != nil {
		return err
	}

	this.vers, err = this.buildVersions(this.base)
	if err != nil {
		return err
	}

	if lastVer := this.lastVersion(); lastVer != nil {
		return lastVer.resolve()
	}

	return nil
}

func (this *database) save() error {
	for _, ver := range this.vers {
		if err := ver.finalize(); err != nil {
			return err
		}
	}

	return nil
}

func (this *database) buildVersions(base string) (versionList, error) {
	nodes, err := ioutil.ReadDir(base)
	if err != nil {
		return nil, err
	}

	var vers versionList
	for _, node := range nodes {
		if !node.IsDir() {
			continue
		}

		timestamp, err := parseVerName(node.Name())
		if err != nil {
			return nil, err
		}

		ver, err := newVersion(path.Join(base, node.Name()), timestamp, this)
		if err != nil {
			return nil, err
		}

		vers = append(vers, ver)
	}

	sort.Sort(vers)

	var parVer *version
	for _, ver := range vers {
		ver.parent = parVer
		parVer = ver
	}

	return vers, nil
}

func (this *database) lastVersion() *version {
	count := len(this.vers)
	if count == 0 {
		return nil
	}

	return this.vers[count-1]
}

func (this *database) Root() (fs.Node, error) {
	return this.lastVersion().root, nil
}
