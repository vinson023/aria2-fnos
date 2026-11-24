//go:build unix

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// dedicatedGroupGID 返回专属应用组的 gid（飞牛注入的 TRIM_GID）。
// 飞牛为每个应用建了与应用同名的组，组内只有该应用的运行用户，
// 因此把目录划给这个组比划给 AppUsers 更收敛。
func dedicatedGroupGID() (int, bool) {
	raw := strings.TrimSpace(os.Getenv("TRIM_GID"))
	if raw == "" {
		return 0, false
	}
	gid, err := strconv.Atoi(raw)
	if err != nil || gid <= 0 {
		return 0, false
	}
	return gid, true
}

// adoptDirectoryGroup 把目录的组改为专属应用组并补上组写权限，属主保持不变。
//
// 场景：用户在安装向导或应用设置里填了一个已存在的目录（例如 /vol1/1000/Download）。
// 该目录属主是管理员、权限 755，而飞牛的 trimacl 不认 POSIX ACL（设备实测：
// `setfacl -m u:aria2:rwx` 之后 aria2 依然写不进去），所以专属应用用户无法写入。
// 直接 777 会把目录对所有人开放；chown 会改变用户在文件管理器里看到的属主。
// 改组 + 补组写权限的副作用最小：属主与 other 位不变，写权限只给到本应用的专属组。
func adoptDirectoryGroup(path string) error {
	gid, ok := dedicatedGroupGID()
	if !ok {
		return errors.New("未取得专属应用组，无法自动授权")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s 不是目录", path)
	}
	// uid 传 -1 表示保持属主不变，只改组。
	if err := os.Chown(path, -1, gid); err != nil {
		return err
	}
	// 回收 setgid 等特殊位，只保留原有权限并在组位上补 rwx。
	return os.Chmod(path, info.Mode().Perm()|0o070)
}
