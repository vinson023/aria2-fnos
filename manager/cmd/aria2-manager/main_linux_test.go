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

// volumeFixture 在 /vol<N> 下准备一个可写测试目录。这组用例必须使用真实的数据卷
// 路径（volumeRootPattern 只接受 /vol<N>/），因此在没有 /vol1 写权限的环境（例如
// 普通用户的 CI runner）里直接跳过，而不是让整个测试套件失败。
func volumeFixture(t *testing.T) string {
	t.Helper()
	parent := "/vol1/@appdata/aria2"
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Skipf("需要可写的 /vol1 数据卷路径，当前环境不具备: %v", err)
	}
	dir, err := os.MkdirTemp(parent, "test-fixture-")
	if err != nil {
		t.Skipf("无法在 %s 下创建 fixture: %v", parent, err)
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(resolved, 0o700)
		_ = os.RemoveAll(dir)
	})
	return resolved
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
	share := volumeFixture(t)
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)
	if err := os.Chmod(share, 0500); err != nil {
		t.Fatal(err)
	}

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
	share := volumeFixture(t)
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

// 用户在安装向导里填写已有共享目录（如 /vol1/1000/Download）时，该目录属主是管理员、
// 权限 755，飞牛的 trimacl 又不认 POSIX ACL，专属应用用户写不进去。ensureAppWritable
// 应当把目录划给专属应用组并补组写权限，同时保持属主不变。
//
// 需要 root 才能改属组，且依赖飞牛注入的 TRIM_GID，因此在普通 CI 上跳过。
func TestEnsureAppWritableAdoptsGroupWithoutChangingOwner(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("需要 root 才能修改目录属组")
	}
	if _, ok := dedicatedGroupGID(); !ok {
		t.Skip("缺少 TRIM_GID，非飞牛环境")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureAppWritable(dir); err != nil {
		t.Fatalf("ensureAppWritable: %v", err)
	}
	after, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if after.Mode().Perm()&0o070 != 0o070 {
		t.Fatalf("group write not granted: %v", after.Mode().Perm())
	}
	if uid, _, ok := directoryOwner(before); ok {
		if uidAfter, _, okAfter := directoryOwner(after); !okAfter || uidAfter != uid {
			t.Fatalf("owner changed: %d -> %d", uid, uidAfter)
		}
	}
}
