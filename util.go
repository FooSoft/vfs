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
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"sync/atomic"
	"time"

	"bazil.org/fuse"
)

var inodeCnt, handleCnt uint64

func allocInode() uint64 {
	return atomic.AddUint64(&inodeCnt, 1)
}

func allocHandleId() fuse.HandleID {
	return fuse.HandleID(atomic.AddUint64(&handleCnt, 1))
}

func copyFile(src, dst string) (int64, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	return io.Copy(srcFile, dstFile)
}

func buildVerName(timestamp time.Time) string {
	return fmt.Sprintf("ver_%.16x", timestamp.Unix())
}

func parseVerName(name string) (time.Time, error) {
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
