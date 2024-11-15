package main

import (
	"errors"
	"io/fs"
	"os"
)

type NoWriteFS interface {
	GetNoWrite() bool
	SetNoWrite(newValue bool)
}

type AWritableFile interface {
	fs.File
	Close() error
	Chmod(mode fs.FileMode) error

	Write(p []byte) (n int, err error)
}

type HostWriteableFile struct {
	os.File
}

// TODO: Replace with fs.ReadlinkFS when this feature is added to Go
// https://github.com/golang/go/issues/49580#issuecomment-2448243225
type ReadlinkFS interface {
	Readlink(name string) (string, error)
	Lstat(name string) (fs.FileInfo, error)
}

type RenameFS interface {
	Rename(oldpath, newpath string) error
}

type RemoveFS interface {
	Remove(name string) error
	RemoveAll(path string) error
}

type SymlinkFS interface {
	Symlink(oldname, newname string) error
}

type CreateFS interface {
	Create(name string) (AWritableFile, error)
}

type MkdirFS interface {
	Mkdir(name string, perm fs.FileMode) error
}

type FileUserAndGroupNameFS interface {
	FileUserAndGroupName(stat fs.FileInfo) (string, string, error)
}

type FreeDiskSpaceBytesFS interface {
	FreeDiskSpaceBytes(path string) (uint64, error)
}

type FSType int

const (
	Host FSType = iota
	SFTP
)

// This is a wrapper for the os package filesystem stuff
type HostFileSystem struct {
	NoWrite bool
	NoWriteFS

	// Implements these interfaces:
	fs.FS
	fs.StatFS
	fs.ReadDirFS
	fs.ReadFileFS
	ReadlinkFS
	RenameFS
	RemoveFS
	CreateFS
	FileUserAndGroupNameFS
	FreeDiskSpaceBytesFS
}

func NewHostFileSystem() *HostFileSystem {
	theFSType = Host
	theFSPathSeparator = os.PathSeparator
	return &HostFileSystem{}
}

func (h *HostFileSystem) SetNoWrite(newValue bool) {
	h.NoWrite = newValue
}

func (h *HostFileSystem) GetNoWrite() bool {
	return h.NoWrite
}

func (h *HostFileSystem) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (h *HostFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (h *HostFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (h *HostFileSystem) Lstat(name string) (fs.FileInfo, error) {
	return os.Lstat(name)
}

func (h *HostFileSystem) Readlink(name string) (string, error) {
	return os.Readlink(name)
}

func (h *HostFileSystem) Rename(oldpath, newpath string) error {
	if h.NoWrite {
		return errors.New("no-write enabled")
	}
	return os.Rename(oldpath, newpath)
}

func (h *HostFileSystem) Remove(name string) error {
	if h.NoWrite {
		return errors.New("no-write enabled")
	}
	return os.Remove(name)
}

func (h *HostFileSystem) RemoveAll(path string) error {
	if h.NoWrite {
		return errors.New("no-write enabled")
	}
	return os.RemoveAll(path)
}

func (h *HostFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (h *HostFileSystem) Symlink(oldname, newname string) error {
	if h.NoWrite {
		return errors.New("no-write enabled")
	}
	return os.Symlink(oldname, newname)
}

func (h *HostFileSystem) Create(name string) (AWritableFile, error) {
	if h.NoWrite {
		return nil, errors.New("no-write enabled")
	}
	return os.Create(name)
}

func (h *HostFileSystem) Mkdir(name string, perm fs.FileMode) error {
	if h.NoWrite {
		return errors.New("no-write enabled")
	}
	return os.Mkdir(name, perm)
}

func (h *HostFileSystem) FileUserAndGroupName(stat fs.FileInfo) (string, string, error) {
	return FileUserAndGroupName(stat)
}

func (h *HostFileSystem) FreeDiskSpaceBytes(path string) (uint64, error) {
	return FreeDiskSpaceBytes(path)
}
