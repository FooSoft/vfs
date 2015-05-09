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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Database struct {
	base     string
	versions []Version
}

func NewDatabase(dir string) (*Database, error) {
	db := &Database{base: dir}
	if err := db.load(dir); err != nil {
		return nil, err
	}

	return db, nil
}

func (this *Database) load(dir string) error {
	base, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	this.base = base

	dirs, err := this.scan(dir)
	if err != nil {
		return err
	}

	this.version(dirs)
	return nil
}

func (this *Database) version(dirs []string) []Version {
	versions := make([]Version, 0, len(dirs))

	// var parent *Version
	// for _, dir := range dirs {
	// 	base := filepath.Join(this.base, dir)
	// 	// timestamp := this.timestamp(dir)
	// 	// version := NewVersion(base, timestamp, parent)
	// 	// parent = version
	// }

	return versions
}

func (this *Database) scan(dir string) ([]VersionMeta, error) {
	nodes, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(`^vfs_([0-9a-f])$`)

	var meta []VersionMeta
	for _, node := range nodes {
		if !node.IsDir() {
			continue
		}

		matches := re.FindStringSubmatch(node.Name())
		if len(matches) < 2 {
			continue
		}

		timestamp, err := strconv.ParseInt(matches[1], 16, 64)
		if err != nil {
			continue
		}

		item := VersionMeta{
			path:      filepath.Join(dir, matches[0]),
			timestamp: time.Unix(timestamp, 0)}

		meta = append(meta, item)
	}

	return meta, nil
}

func (this *Database) scanProps(path string) error {

}
