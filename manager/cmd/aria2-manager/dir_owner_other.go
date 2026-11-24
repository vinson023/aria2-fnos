//go:build !unix

package main

import "os"

func directoryOwner(_ os.FileInfo) (int, int, bool) { return -1, -1, false }

func chownDirectory(_ string, _, _ int) error { return nil }

// writableAsUser 只在类 Unix 上用于降权写探测；Windows 上 writableByAppUser 走
// 当前用户路径（os.Geteuid 返回 -1），不会调用到这里。
func writableAsUser(_ string, _, _ int) error { return nil }
