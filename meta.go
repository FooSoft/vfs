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
//	verMeta
//

type verMeta struct {
	Deleted []string `json:"deleted"`
	path    string
	dirty   bool
}

func newVerMeta(path string) (*verMeta, error) {
	meta := &verMeta{path: path}
	if err := meta.load(); err != nil {
		return nil, err
	}

	return meta, nil
}

func (m *verMeta) filter(nodes verNodeMap) {
	for _, delPath := range m.Deleted {
		for name, node := range nodes {
			if strings.HasPrefix(node.path, delPath) {
				delete(nodes, name)
			}
		}
	}
}

func (m *verMeta) removeNode(path string) {
	m.Deleted = append(m.Deleted, path)
	m.dirty = true
}

func (m *verMeta) createNode(path string) {
	m.dirty = true
}

func (m *verMeta) modifyNode(path string) {
	m.dirty = true
}

func (m *verMeta) load() error {
	m.dirty = false

	if _, err := os.Stat(m.path); os.IsNotExist(err) {
		return nil
	}

	bytes, err := ioutil.ReadFile(m.path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes, &m); err != nil {
		return err
	}

	return nil
}

func (m *verMeta) save() error {
	if !m.dirty {
		return nil
	}

	js, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(m.path, js, 0644)
}
