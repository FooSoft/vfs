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
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"flag"
	"fmt"
	"log"
	"os"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] database mountpoint\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Parameters:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 2 {
		usage()
		os.Exit(2)
	}

	database := flag.Arg(0)
	mountpoint := flag.Arg(1)

	db, err := newDatabase(database)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := fuse.Mount(mountpoint)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	err = fs.Serve(conn, db)
	if err != nil {
		log.Fatal(err)
	}

	<-conn.Ready
	if err := conn.MountError; err != nil {
		log.Fatal(err)
	}

	if err := db.save(); err != nil {
		log.Fatal(err)
	}
}
