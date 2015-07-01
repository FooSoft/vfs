# Versioning File System #

Versioning File System (uninterestingly abbreviated VFS) is a simple user-space file system implemented on top of
[FUSE](https://en.wikipedia.org/?title=Filesystem_in_Userspace) with the aid of [Bazil ](https://bazil.org/fuse/),
written in [Golang](https://golang.org/). Development on this project was started as means to learn more about the way
Linux handles file systems, while at the same time answering a personal need of an easy to understand, transparent
versioning file system for data backup. Although it should not yet be considered to be production ready, VFS is already
usable in its current state.

## Design ##

My goal was to build file system which could handle changes to file data and directory structure between mount cycles,
in a simple, transparent way. I wanted to avoid storing version information in binary blobs, which are completely
incomprehensible to the user. I strongly believed that the user should not be locked into a versioning scheme that
prevented trivial migration or export of data. As such, versioned file data is stored in timestamped directories on a
host file system (with minimal metadata stored in a human-readable format).

Each version consists of a root node and child nodes that represent modified files or directories for that version;
unmodified data is not duplicated between versions. Other information (such as records about file and directory
deletions) are stored in a JSON file next to the version root. Although VFS provides a mechanism for enumerating and
mounting specific snapshots, the user is capable of browsing the version data directly if they choose to do so.

## Installation ##

If you already have the Go environment and toolchain set up, you can get the latest version by running:
```
go get github.com/FooSoft/vfs
```
Users of Debian-based distributions can use [godeb](https://github.com/niemeyer/godeb) to quickly install the Go
compiler on their machines.

## Usage ##

Usage information can be seen by running VFS without command-line arguments:

```
Usage: ./vfs [options] database [mountpoint]

Parameters:
  -readonly=false: mount filesystem as readonly
  -version=0: version index (0 for head)

```
In the output above, the `database` parameter refers to a directory containing VFS versions; an empty directory is a
valid database.  The `mountpoint` parameter refers to the path on your system where the file system will be accessible
(mounted).

In order to mount a new or existing volume, follow the steps below:

1.  Add yourself to the `fuse` user group if you are not added already (a requirement of FUSE).
2.  Execute `./vfs database_dir mountpoint_dir`, using actual paths on your system to mount a volume.
3.  When you are finished using the volume, unmount it via the `fusermount -u mountpoint_dir` command.
