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
	"path/filepath"
)

type Database struct {
	base     string
	versions []*Version
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

	dirs, err := this.scan(base)
	if err != nil {
		return err
	}

	versions, err := this.version(dirs)
	if err != nil {
		return err
	}

	this.base = base
	this.versions = versions

	return nil
}

func (this *Database) version(dirs []string) ([]*Version, error) {
	var versions []*Version
	var parent *Version

	for _, dir := range dirs {
		version, err := NewVersion(dir, parent)
		if err != nil {
			return nil, err
		}

		parent = version
		versions = append(versions, version)
	}

	return versions, nil
}

func (this *Database) scan(dir string) ([]string, error) {
	nodes, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, node := range nodes {
		if node.IsDir() {
			dirs = append(dirs, node.Name())
		}
	}

	return dirs, nil
}
