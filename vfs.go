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
	"flag"
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	// _ "bazil.org/fuse/fs/fstestutil"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] database [mountpoint]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Parameters:\n")
	flag.PrintDefaults()
}

func main() {
	version := flag.Uint("version", 0, "version index (0 for head)")
	readonly := flag.Bool("readonly", false, "mount filesystem as readonly")
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		os.Exit(2)
	}

	mountable := flag.NArg() > 1
	mutable := mountable && !*readonly && *version == 0

	db, err := newDatabase(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	if mutable {
		if err := db.createNewVer(); err != nil {
			log.Fatal(err)
		}
	}

	if err := db.load(*version); err != nil {
		log.Fatal(err)
	}

	if mountable {
		var options []fuse.MountOption
		if !mutable {
			options = append(options, fuse.ReadOnly())
		}

		conn, err := fuse.Mount(flag.Arg(1), options...)
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		if err := fs.Serve(conn, db); err != nil {
			log.Fatal(err)
		}

		<-conn.Ready
		if err := conn.MountError; err != nil {
			log.Fatal(err)
		}

		if mutable {
			if err := db.save(); err != nil {
				log.Fatal(err)
			}
		}
	} else {
		db.dump()
	}
}
