# Versioning File System #

Versioning File System (uninterestingly abbreviated VFS) is a simple user-space file system implemented on top of
[FUSE](https://en.wikipedia.org/?title=Filesystem_in_Userspace) with the aid of [Bazil](https://bazil.org/fuse/),
written in Go. Development on this project was started as means to learn more about the way Linux handles file systems,
while at the same time answering a personal need of an easy to understand, transparent versioning file system for data
backup. Although it should not yet be considered to be production ready, VFS is already usable in its current state.

## Motivation ##

My goal was to build file system which could handle changes to file data and directory structure between mount cycles,
in a simple, transparent way. I wanted to avoid storing version information in binary blobs, which are completely
incomprehensible to the user. I strongly believed that the user should not be locked into a versioning scheme that
prevented trivial migration or export of data. As such, versioned file data is stored in timestamped directories on a
host file system (with minimal metadata stored in a human-readable format). This approach is fundamentally different
from that of other tools.

Each version consists of a root node and child nodes that represent modified files or directories for that version;
unmodified data is not duplicated between versions. Other information (such as records about file and directory
deletions) are stored in a JSON file next to the version root. Although VFS provides a mechanism for enumerating and
mounting specific snapshots, the user is capable of browsing the version data directly if they choose to do so.

Go was selected as the language of choice for this project as it combines the performance and safety of a compiled
language without sacrificing the readability and maintainability of a high level scripting language such as Python.
Furthermore, the fact that Go programs are statically linked ensures that binaries will be compatible between various
Linux distribution; recompilation is not required.

## Installation ##

If you already have the Go environment and toolchain set up, you can get the latest version by running:
```
$ go get github.com/FooSoft/vfs
```
Users of Debian-based distributions can use [godeb](https://github.com/niemeyer/godeb) to quickly install the Go
compiler on their machines.

## Usage ##

Usage information can be seen by running VFS without command line arguments:

```
Usage: ./vfs [options] database [mountpoint]

Parameters:
  -readonly=false: mount filesystem as readonly
  -version=0: version index (0 for head)
```
In the output above, the `database` parameter refers to a directory containing VFS versions; an empty directory is a
valid database. The `mountpoint` parameter refers to the path on your system where the file system will be accessible
(mounted).

### Listing Volume Versions ###

Specifying a database path without the mount point will output a timestamped listing of available versions. This listing
includes identifiers which can be used to specify a specific version to mount with the `-version` parameter. Note that
only read-only mounting is possible of prior versions.

```
version: 1  time: 2015-06-19 11:14:13 +0900 JST
version: 2  time: 2015-06-20 13:08:04 +0900 JST
version: 3  time: 2015-06-22 15:12:09 +0900 JST
version: 4  time: 2015-06-24 12:41:43 +0900 JST
```

### Mounting a Volume ###

1.  Add yourself to the `fuse` user group if you are not added already (a requirement of FUSE). You can optionally skip
    this step by mounting the volume as the `root` user, but this is not recommended.
2.  Execute `./vfs database_dir mountpoint_dir`, using actual paths on your system to mount a volume. If you wish to
    provide an identifier for a specific version to mount, you may specify it with the `-version` parameter. Passing a
    non-zero value (zero indicates most recent version) will make the mount read-only. Explicit read-only mounting is
    also possible by setting the `-readonly` switch.
3.  When you are finished using the volume, unmount it via the `fusermount -u mountpoint_dir` command.

## Walkthrough ##

When you execute VFS for the first time, you will probably neither have a version database nor a mount point.  Since an
empty directory is a valid empty version database and mount points should by always be empty, let's create two new
directories (`db` for the database and `mp` for the mount point).

```
$ mkdir db mp
$ ls
db/  mp/
```

Now let's mount the empty database directory `db` onto our mount point `mp` (note that you have to be in the `fuse`
group or `root` in order for this to work):

```
$ vfs db mp
```

Using a new terminal window, let's create some files and directories:

```
$ echo hello > mp/greeting.txt
$ mkdir mp/pizza
$ touch mp/pizza/pepperoni mp/pizza/cheese
```

Now that we are finished working with this version, let's unmount it:

```
$ fusermount -u mp
```

Let's take a look at the version structure that VFS has created in the database directory:

```
$ ls -R db
db:
ver_00000000559a17e4/

db/ver_00000000559a17e4:
meta.json  root/

db/ver_00000000559a17e4/root:
greeting.txt  pizza/

db/ver_00000000559a17e4/root/pizza:
cheese  pepperoni
```

Some points of interest about this structure:

*   Database directory contains the new version; the value `00000000559a17e4` is the creation time stamp in hexadecimal.
*   The `meta.json` file contains information about deletions; it is empty for now.
*   The `root` directory contains the files that were created or modified in this version.

Let's continue our walkthrough by mounting the now non-empty database once more:

```
$ vfs db mp
```

Now let's make a couple of changes to the files and directory structure:

```
$ echo howdy > mp/greeting.txt
$ rm mp/pizza/pepperoni
$ touch mp/pizza/bacon
```

...and verify that everything is as it should be; looks good so far!

```
$ cat mp/greeting.txt
howdy
$ ls -R mp
mp:
greeting.txt  pizza/

mp/pizza:
bacon  cheese
```

Now let's unmount and examine at the contents of the database directory:

```
$ fusermount -u mp
$ ls -l db
total 8
drwxr-xr-x 3 alex alex 4096 Jul  6 14:58 ver_00000000559a17e4/
drwxr-xr-x 3 alex alex 4096 Jul  6 15:08 ver_00000000559a19b4/
```

Cool, we have a new version! Let's take a closer look at its structure:

```
$ ls -R db/ver_00000000559a19b4/
db/ver_00000000559a19b4/:
meta.json  root/

db/ver_00000000559a19b4/root:
greeting.txt  pizza/

db/ver_00000000559a19b4/root/pizza:
bacon
```

We can see that `greeting.txt` and `bacon` show up; this reflects the fact that these files were modified and created,
respectively. Notice that `cheese` is not listed; we didn't make any changes to this file so the data from the previous
version is used. If we look at the contents of `meta.json`, we will see that the `pepperoni` file was deleted:

```
$ cat db/ver_00000000559a19b4/meta.json
{"deleted":["/pizza/pepperoni"]}
```

Now let's examine the database from VFS tool directly:

```
$ vfs db
version: 1      time: 2015-07-06 14:53:40 +0900 JST
version: 2      time: 2015-07-06 15:01:24 +0900 JST
```

Finally, let's mount the version that we created in the beginning of this walkthrough by specifying its index with the
`version` parameter...

```
$ vfs -version=1 db mp
```

...and in a different terminal verify its contents; they are identical to the first version!

```
$ cat mp/greeting.txt
hello
$ ls -R mp
mp:
greeting.txt  pizza/

mp/pizza:
cheese  pepperoni
```

Hopefully this brief walkthrough of the system helped illustrate the basic concepts behind how VFS works. At its core,
it is a fundamentally simple system, the workings of which can be examined with any file browser and text editor. I
encourage those interested in this utility to follow this guide and to ask me any questions they may have.

## Limitations ##

There are a few lingering limitations of the system in its current state. While the architecture supports their
inclusion into a future version, I have not yet gotten around to taking care of this "laundry list" of items:

*   Add support for file linking
*   Handle the `..` directory for root
*   Further reduce data duplication when:
    +   File is renamed
    +   File attributes are modified
*   Compress shadowed files
*   Support for Mac OS X

## License ##

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
