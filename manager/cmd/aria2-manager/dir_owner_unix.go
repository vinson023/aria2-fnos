//go:build unix

package main

import (
	"os"
	"syscall"
)

// directoryOwner ????????? root ?????????????????
func directoryOwner(info os.FileInfo) (int, int, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return -1, -1, false
	}
	return int(stat.Uid), int(stat.Gid), true
}

func chownDirectory(path string, uid, gid int) error {
	return os.Chown(path, uid, gid)
}
