//go:build unix

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// directoryOwner 从文件信息里取出属主，非 root 运行时用于让新建目录跟随上级目录。
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

// writableAsUser 以专属应用用户的身份在目录内创建并删除一个临时文件。
// 优先用 setpriv --init-groups：飞牛的访问权常常来自附加组（trimacl），只切换主组
// 会得到假阴性。setpriv 不可用时退回直接设置 Credential。
func writableAsUser(dir string, uid, gid int) error {
	name := filepath.Join(dir, fmt.Sprintf(".aria2-write-test-%d", time.Now().UnixNano()))
	command := exec.Command("setpriv",
		fmt.Sprintf("--reuid=%d", uid), fmt.Sprintf("--regid=%d", gid), "--init-groups",
		"touch", "--", name)
	if _, err := exec.LookPath("setpriv"); err != nil {
		command = exec.Command("touch", "--", name)
		command.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}}
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v %s", err, strings.TrimSpace(string(output)))
	}
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("测试文件无法清理: %v", err)
	}
	return nil
}
