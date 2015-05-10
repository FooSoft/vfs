// Hellofs implements a simple "hello world" file system.
package main

import (
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	// "flag"
	// "fmt"
	"golang.org/x/net/context"
	"os"
)

// func main() {
// 	flag.Usage = Usage
// 	flag.Parse()

// 	if flag.NArg() != 1 {
// 		Usage()
// 		os.Exit(2)
// 	}
// 	mountpoint := flag.Arg(0)

// 	c, err := fuse.Mount(
// 		mountpoint,
// 		fuse.FSName("helloworld"),
// 		fuse.Subtype("hellofs"),
// 		fuse.LocalVolume(),
// 		fuse.VolumeName("Hello world!"),
// 	)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer c.Close()

// 	err = fs.Serve(c, FS{})
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// check if the mount process has an error to report
// 	<-c.Ready
// 	if err := c.MountError; err != nil {
// 		log.Fatal(err)
// 	}
// }

// FS implements the hello world file system.
type FS struct{}

func (FS) Root() (fs.Node, error) {
	return Dir{}, nil
}

// Dir implements both Node and Handle for the root directory.
type Dir struct{}

func (Dir) Attr(a *fuse.Attr) {
	a.Inode = 1
	a.Mode = os.ModeDir | 0777
}

func (Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {

	if name == "hello" {
		return File{}, nil
	}
	return nil, fuse.ENOENT
}

var dirDirs = []fuse.Dirent{
	{Inode: 2, Name: "hello", Type: fuse.DT_File},
}

func (Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return dirDirs, nil
}

// File implements both Node and Handle for the hello file.
type File struct{}

const greeting = "hello, world\n"

func (File) Attr(a *fuse.Attr) {
	a.Inode = 2
	a.Mode = 0777
	a.Size = uint64(len(greeting))
}

func (File) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(greeting), nil
}
