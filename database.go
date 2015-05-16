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

type database struct {
	base     string
	vers     []*version
	inodeCnt uint64
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

	dirs, err := this.scan(this.base)
	if err != nil {
		return err
	}

	this.vers, err = this.versions(dirs)
	if err != nil {
		return err
	}

	if lastVer := this.lastVersion(); lastVer != nil {
		return lastVer.resolve()
	}

	return nil
}

func (this *database) save() error {
	return nil
}

func (this *database) versions(dirs []string) ([]*version, error) {
	var vers []*version

	var parent *version
	for _, dir := range dirs {
		ver, err := newVersion(dir, this, parent)
		if err != nil {
			return nil, err
		}

		vers = append(vers, ver)
		parent = ver
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

func (this *database) scan(dir string) ([]string, error) {
	nodes, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, node := range nodes {
		if node.IsDir() {
			dirs = append(dirs, filepath.Join(dir, node.Name()))
		}
	}

	return dirs, nil
}

func (this *database) AllocInode() uint64 {
	this.inodeCnt++
	return this.inodeCnt
}
