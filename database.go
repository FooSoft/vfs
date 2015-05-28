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
	"sync/atomic"
	"time"
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

	if err := this.buildNewVersion(this.base); err != nil {
		return nil
	}

	names, err := this.scanVersions(this.base)
	if err != nil {
		return err
	}

	this.vers, err = this.buildVersions(this.base, names)
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

func (this *database) buildVersions(base string, names []string) ([]*version, error) {
	var vers []*version

	var parent *version
	for _, name := range names {
		timestamp, err := this.parseVerName(name)
		if err != nil {
			return nil, err
		}

		ver, err := newVersion(path.Join(base, name), timestamp, this, parent)
		if err != nil {
			return nil, err
		}

		vers = append(vers, ver)
		parent = ver
	}

	for _, ver := range vers {
		ver.terminus = vers[len(vers)-1]
	}

	return vers, nil
}

func (this *database) buildNewVersion(base string) error {
	name := this.buildVerName(time.Now())

	if err := os.Mkdir(path.Join(base, name), 0755); err != nil {
		return err
	}
	if err := os.Mkdir(path.Join(base, name, "root"), 0755); err != nil {
		return err
	}

	return nil
}

func (this *database) lastVersion() *version {
	count := len(this.vers)
	if count == 0 {
		return nil
	}

	return this.vers[count-1]
}

func (this *database) scanVersions(dir string) ([]string, error) {
	nodes, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, node := range nodes {
		if node.IsDir() {
			names = append(names, node.Name())
		}
	}

	return names, nil
}

func (this *database) Root() (fs.Node, error) {
	return this.lastVersion().root, nil
}

func (this *database) AllocInode() uint64 {
	return atomic.AddUint64(&this.inodeCnt, 1)
}

func (this *database) buildVerName(timestamp time.Time) string {
	return fmt.Sprintf("ver_%.16x", timestamp.Unix())
}

func (this *database) parseVerName(name string) (time.Time, error) {
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
