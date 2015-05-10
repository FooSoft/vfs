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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

type Metadata struct {
	Deleted []string
}

type Version struct {
	base      string
	parent    *Version
	timestamp time.Time
	metadata  Metadata
}

func NewVersion(base string, parent *Version) (*Version, error) {
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

	version := &Version{
		base:      base,
		parent:    parent,
		timestamp: time.Unix(timeval, 0)}

	return version, nil
}

func (this *Version) loadMetadata() error {
	path := this.metadataPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	bytes, err := ioutil.ReadFile(this.metadataPath())
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes, &this.metadata); err != nil {
		return err
	}

	return nil
}

func (this *Version) saveMetadata() error {
	js, err := json.Marshal(this.metadata)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(this.metadataPath(), js, 0644)
}

func (this *Version) metadataPath() string {
	return filepath.Join(this.base, "meta.json")
}
