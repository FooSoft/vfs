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
	"io/ioutil"
	"os"
	"strings"
)

//
//	versionMetadata
//

type versionMetadata struct {
	Deleted []string `json:"deleted"`
	path    string
	dirty   bool
}

func newVersionMetadata(path string) (*versionMetadata, error) {
	meta := &versionMetadata{path: path}
	if err := meta.load(); err != nil {
		return nil, err
	}

	return meta, nil
}

func (this *versionMetadata) filter(nodes versionedNodeMap) {
	for _, delPath := range this.Deleted {
		for name, node := range nodes {
			if strings.HasPrefix(node.path, delPath) {
				delete(nodes, name)
			}
		}
	}
}

func (this *versionMetadata) destroyPath(path string) {
	this.Deleted = append(this.Deleted, path)
	this.dirty = true
}

func (this *versionMetadata) createPath(path string) {
	this.dirty = true
}

func (this *versionMetadata) load() error {
	this.dirty = false

	if _, err := os.Stat(this.path); os.IsNotExist(err) {
		return nil
	}

	bytes, err := ioutil.ReadFile(this.path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes, &this); err != nil {
		return err
	}

	return nil
}

func (this *versionMetadata) save() error {
	if !this.dirty {
		return nil
	}

	js, err := json.Marshal(this)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(this.path, js, 0644)
}
