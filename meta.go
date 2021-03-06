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
	"path"
	"strings"
	"sync"
)

//
//	verMeta
//

type verFmt struct {
	Deleted []string `json:"deleted"`
}

type verMeta struct {
	path     string
	deleted  map[string]bool
	modified bool
	mutex    sync.Mutex
}

func newVerMeta(path string) (*verMeta, error) {
	meta := &verMeta{path, make(map[string]bool), false, sync.Mutex{}}
	if err := meta.load(); err != nil {
		return nil, err
	}

	return meta, nil
}

func (m *verMeta) filter(nodes verNodeMap) {
	for path, deleted := range m.deleted {
		if !deleted {
			continue
		}

		for name, node := range nodes {
			if strings.HasPrefix(node.path, path) {
				delete(nodes, name)
			}
		}
	}
}

func (m *verMeta) removeNode(path string) {
	m.mutex.Lock()
	m.deleted[path] = true
	m.mutex.Unlock()

	m.modified = true
}

func (m *verMeta) createNode(path string) {
	m.mutex.Lock()
	m.deleted[path] = false
	m.mutex.Unlock()

	m.modified = true
}

func (m *verMeta) modifyNode(path string) {
	m.modified = true
}

func (m *verMeta) load() error {
	if _, err := os.Stat(m.path); os.IsNotExist(err) {
		return nil
	}

	bytes, err := ioutil.ReadFile(m.path)
	if err != nil {
		return err
	}

	var vd verFmt
	if err := json.Unmarshal(bytes, &vd); err != nil {
		return err
	}

	m.deleted = make(map[string]bool)
	for _, path := range vd.Deleted {
		m.deleted[path] = true
	}

	m.modified = false
	return nil
}

func (m *verMeta) checkRedundantDel(child string) bool {
	parent := path.Dir(child)
	if parent == "/" {
		return false
	}

	if deleted, _ := m.deleted[parent]; deleted {
		return true
	}

	return m.checkRedundantDel(parent)
}

func (m *verMeta) save() error {
	if !m.modified {
		return nil
	}

	var vd verFmt
	for path, deleted := range m.deleted {
		if !deleted {
			continue
		}

		if m.checkRedundantDel(path) {
			continue
		}

		vd.Deleted = append(vd.Deleted, path)
	}

	js, err := json.Marshal(vd)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(m.path, js, 0644)
}
