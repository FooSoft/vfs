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
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"sort"

	"bazil.org/fuse/fs"
)

//
//	database
//

type database struct {
	base string
	vers verList
}

func newDatabase(dir string, index int, writable bool) (*database, error) {
	db := &database{base: dir}
	if err := db.load(dir, index, writable); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *database) load(dir string, index int, writable bool) error {
	var err error

	db.base, err = filepath.Abs(dir)
	if err != nil {
		return err
	}

	if writable {
		if err := buildNewVersion(db.base); err != nil {
			return err
		}
	}

	db.vers, err = db.buildVersions(db.base)
	if err != nil {
		return err
	}

	if index >= 0 {
		db.vers = db.vers[index:]
	}

	if lastVer := db.lastVersion(); lastVer != nil {
		return lastVer.resolve()
	}

	return nil
}

func (db *database) save() error {
	lastVer := db.lastVersion()

	for _, ver := range db.vers {
		if err := ver.finalize(ver == lastVer); err != nil {
			return err
		}
	}

	return nil
}

func (db *database) buildVersions(base string) (verList, error) {
	nodes, err := ioutil.ReadDir(base)
	if err != nil {
		return nil, err
	}

	var vers verList
	for _, node := range nodes {
		if !node.IsDir() {
			continue
		}

		timestamp, err := parseVerName(node.Name())
		if err != nil {
			return nil, err
		}

		ver, err := newVersion(path.Join(base, node.Name()), timestamp, db)
		if err != nil {
			return nil, err
		}

		vers = append(vers, ver)
	}

	sort.Sort(vers)

	var pv *version
	for _, ver := range vers {
		ver.parent = pv
		pv = ver
	}

	return vers, nil
}

func (db *database) lastVersion() *version {
	count := len(db.vers)
	if count == 0 {
		return nil
	}

	return db.vers[count-1]
}

func (db *database) dump() {
	for index, ver := range db.vers {
		fmt.Printf("version: %d\ttime: %s\n", index, ver.timestamp.String())
	}
}

// FS
func (db *database) Root() (fs.Node, error) {
	return db.lastVersion().root, nil
}
