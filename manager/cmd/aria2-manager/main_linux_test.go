//go:build linux

package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureConfiguredDirectoriesCreatesArchiveDir(t *testing.T) {
	share, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)

	value := config{}
	normalizeConfig(&value)
	if err := ensureConfiguredDirectories(&value); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(value.ArchiveRoot)
	if err != nil || !info.IsDir() {
		t.Fatalf("归档目录应被自动创建: %v", err)
	}
	// 重复执行不得报错，且已存在的目录保持原样。
	if err := ensureConfiguredDirectories(&value); err != nil {
		t.Fatalf("归档目录已存在时应通过: %v", err)
	}
}

// 共享目录之外的归档目录不再硬性拒绝：归档只影响"下载完成后归档"这一可选行为，
// 不应阻塞用户保存下载目录等基本配置。失败时降级为关闭归档、清空归档目录。
func TestEnsureConfiguredDirectoriesRejectsOutsideShare(t *testing.T) {
	share, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)

	value := config{DownloadRoot: share, ArchiveRoot: filepath.Join(outside, defaultArchiveDirName), SpecialMode: "move", ArchiveEnabled: true}
	if err := ensureConfiguredDirectories(&value); err != nil {
		t.Fatalf("归档失败必须降级而非报错: %v", err)
	}
	if value.ArchiveEnabled {
		t.Fatal("归档失败后必须自动关闭开关")
	}
	if value.SpecialMode != "off" {
		t.Fatalf("special_mode 必须为 off，实际 %q", value.SpecialMode)
	}
	if value.ArchiveRoot != "" {
		t.Fatalf("归档目录必须清空，实际 %q", value.ArchiveRoot)
	}
	if _, err := os.Stat(filepath.Join(outside, defaultArchiveDirName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("被拒绝的归档目录不得被创建")
	}
}

func TestEnsureConfiguredDirectoriesNoopWhenDisabled(t *testing.T) {
	share, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)

	value := config{DownloadRoot: share, ArchiveRoot: filepath.Join(share, defaultArchiveDirName), SpecialMode: "off"}
	if err := ensureConfiguredDirectories(&value); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(value.ArchiveRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("关闭归档时不应创建归档目录")
	}
}

// 复现用户截图里 "/vol2/1000/downloads 不是 aria2 用户可写" 的故障模式：
// 归档目录的父目录 aria2 不可写 → 创建失败。这种情况必须降级为关闭归档，
// 而不是把整个下载目录保存给堵死。
func TestEnsureConfiguredDirectoriesDegradesWhenParentUnwritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root 可绕过文件权限位，无法在该环境下复现不可写场景")
	}
	// 落在 /vol<N>/ 下才能通过 volumeRootPattern 授权根校验，同时 aria2 用户
	// 必须能在该路径下创建子目录（否则测试本身就因 fixture 创建失败而崩）。
	fixture := "/vol1/@appdata/aria2/test-fixture"
	if err := os.MkdirAll(fixture, 0755); err != nil {
		t.Fatal(err)
	}
	share, err := filepath.EvalSymlinks(fixture)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)
	if err := os.Chmod(share, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(share, 0700)
		_ = os.RemoveAll(fixture)
	})

	value := config{
		DownloadRoot:   share,
		ArchiveRoot:    filepath.Join(share, defaultArchiveDirName),
		SpecialMode:    "move",
		ArchiveEnabled: true,
	}
	if err := ensureConfiguredDirectories(&value); err != nil {
		t.Fatalf("父目录不可写时必须降级而非报错: %v", err)
	}
	if value.ArchiveEnabled || value.SpecialMode != "off" {
		t.Fatalf("归档失败后必须关闭，实际 %#v", value)
	}
	if value.ArchiveRoot != "" {
		t.Fatalf("归档目录必须清空，实际 %q", value.ArchiveRoot)
	}
	if _, err := os.Stat(filepath.Join(share, defaultArchiveDirName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("降级后不得创建归档目录")
	}
}

// 复现用户截图里的故障：/vol2/1000/downloads 这类共享挂载对 aria2 不可写，
// 归档子目录创建失败之前会整笔 PUT 被拒。修复后归档必须静默降级为关闭，
// 下载目录仍能保存成功。
func TestConfigUpdateKeepsDownloadRootWhenArchiveUnwritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root 可绕过文件权限位")
	}
	// 落在 /vol<N>/ 下才能通过 volumeRootPattern 授权根校验。
	fixture := "/vol1/@appdata/aria2/test-fixture"
	if err := os.MkdirAll(fixture, 0755); err != nil {
		t.Fatal(err)
	}
	share, err := filepath.EvalSymlinks(fixture)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(share, 0700)
		_ = os.RemoveAll(fixture)
	})
	if err := os.Chmod(share, 0500); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)
	t.Setenv("TRIM_PKGETC", "")

	store := configStore{path: filepath.Join(t.TempDir(), "config.json")}
	if err := store.update(config{
		RPCPort:           6800,
		ListenPort:        6888,
		DHTMessageTimeout: "10",
		DownloadRoot:      share,
		AuthorizedRoots:   []string{share},
	}); err != nil {
		t.Fatalf("下载目录的保存必须成功，归档失败应降级: %v", err)
	}
	if store.cfg.DownloadRoot != share {
		t.Fatalf("下载目录必须保留: got %q", store.cfg.DownloadRoot)
	}
	if store.cfg.ArchiveEnabled {
		t.Fatal("归档必须已降级为关闭")
	}
	if store.cfg.ArchiveRoot != "" {
		t.Fatalf("归档目录必须清空: got %q", store.cfg.ArchiveRoot)
	}
}
