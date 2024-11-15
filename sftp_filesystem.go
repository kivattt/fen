package main

import (
	"errors"
	"io/fs"
	"strconv"

	"github.com/pkg/sftp"
)

type SFTPFileSystem struct {
	NoWrite bool
	NoWriteFS

	client *sftp.Client

	// Implements these interfaces:
	fs.FS
	fs.StatFS
	fs.ReadDirFS
	fs.ReadFileFS
	ReadlinkFS
	RenameFS
	RemoveFS
	CreateFS
	MkdirFS
	FileUserAndGroupNameFS
	FreeDiskSpaceBytesFS
}

func NewSFTPFileSystem(sftpClient *sftp.Client) *SFTPFileSystem {
	theFSType = SFTP
	theFSPathSeparator = '/'
	return &SFTPFileSystem{client: sftpClient}
}

func (s *SFTPFileSystem) SetNoWrite(newValue bool) {
	s.NoWrite = newValue
}

func (s *SFTPFileSystem) GetNoWrite() bool {
	return s.NoWrite
}

func (s *SFTPFileSystem) Open(name string) (fs.File, error) {
	return s.client.Open(name)
}

func (s *SFTPFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	fileInfos, err := s.client.ReadDir(name)
	if err != nil {
		return []fs.DirEntry{}, err
	}

	ret := make([]fs.DirEntry, len(fileInfos))
	for i, e := range fileInfos {
		ret[i] = fs.FileInfoToDirEntry(e)
	}

	return ret, nil
}

func (s *SFTPFileSystem) Stat(name string) (fs.FileInfo, error) {
	return s.client.Stat(name)
}

func (s *SFTPFileSystem) Lstat(name string) (fs.FileInfo, error) {
	return s.client.Lstat(name)
}

func (s *SFTPFileSystem) Readlink(name string) (string, error) {
	return s.client.ReadLink(name)
}

func (s *SFTPFileSystem) Rename(oldpath, newpath string) error {
	if s.NoWrite {
		return errors.New("no-write enabled")
	}
	return s.client.Rename(oldpath, newpath)
}

func (s *SFTPFileSystem) Remove(name string) error {
	if s.NoWrite {
		return errors.New("no-write enabled")
	}
	return s.client.Remove(name)
}

func (s *SFTPFileSystem) RemoveAll(path string) error {
	if s.NoWrite {
		return errors.New("no-write enabled")
	}
	return s.client.RemoveAll(path)
}

// This function is only used in GenerateLuaConfigFromOldJSONConfig(), where theFS should only be HostFileSystem anyway
func (s *SFTPFileSystem) ReadFile(name string) ([]byte, error) {
	return []byte{}, errors.New("unimplemented in SFTPFileSystem")
}

func (s *SFTPFileSystem) Symlink(oldname, newname string) error {
	if s.NoWrite {
		return errors.New("no-write enabled")
	}
	return s.client.Symlink(oldname, newname)
}

func (s *SFTPFileSystem) Create(name string) (AWritableFile, error) {
	if s.NoWrite {
		return nil, errors.New("no-write enabled")
	}

	return s.client.Create(name)
}

func (s *SFTPFileSystem) Mkdir(name string, mode fs.FileMode) error {
	if s.NoWrite {
		return errors.New("no-write enabled")
	}
	return s.client.Mkdir(name)
}

func (s *SFTPFileSystem) FileUserAndGroupName(stat fs.FileInfo) (string, string, error) {
	uidGid, ok := stat.Sys().(*sftp.FileStat)
	if !ok {
		return "", "", errors.New("unable to get UID/GID")
	}

	uid := (*uidGid).UID
	gid := (*uidGid).GID

	var uidStr string
	var gidStr string

	// Hacky, the coloring of these in the bottombar etc. relies on checking the host system UIDs for these username strings
	// So "1005" would be interpreted as a username, and colored according to its UID
	// "root" will only be red if the host system also has a user called "root" which has a UID of 0
	if uid == 0 {
		uidStr = "root"
	} else {
		uidStr = strconv.FormatInt(int64(uid), 10)
	}

	if gid == 0 {
		gidStr = "root"
	} else {
		gidStr = strconv.FormatInt(int64(gid), 10)
	}

	return uidStr, gidStr, nil
}

func (s *SFTPFileSystem) FreeDiskSpaceBytes(path string) (uint64, error) {
	vfs, err := s.client.StatVFS(path)
	if err != nil {
		return 0, err
	}

	return vfs.FreeSpace(), nil
}
