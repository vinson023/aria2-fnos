package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setShare 把下载候选根固定成一个可控的临时目录，避免测试读取宿主机环境变量。
func setShare(t *testing.T, share string) {
	t.Helper()
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)
	t.Setenv("TRIM_PKGVAR", t.TempDir())
}

// TestRepairStartupConfigRecoversUnresolvableDownloadRoot 复现现场故障：
// 下载目录指向不存在的路径时 decodeConfig 必须失败，而启动自愈必须能恢复出可启动配置。
func TestRepairStartupConfigRecoversUnresolvableDownloadRoot(t *testing.T) {
	requireLinux(t)
	share, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	setShare(t, share)

	raw := `{"rpc_port":6801,"listen_port":6888,"dht_listen_port":6888,"dht_message_timeout":"10",` +
		`"download_root":"/vol1/1000/no-such-share","special_mode":"move","config_version":1}`
	if _, err := decodeConfig([]byte(raw)); err == nil {
		t.Fatal("失效下载目录必须让常规校验失败，否则自愈没有触发机会")
	}
	repaired, notes, err := repairStartupConfig([]byte(raw))
	if err != nil {
		t.Fatalf("自愈必须成功: %v", err)
	}
	if repaired.DownloadRoot != share {
		t.Fatalf("下载目录应回落到本机可用共享目录: got %q want %q", repaired.DownloadRoot, share)
	}
	if repaired.ArchiveRoot != filepath.Join(share, defaultArchiveDirName) {
		t.Fatalf("归档目录应随下载目录重新派生: %#v", repaired.ArchiveRoot)
	}
	// 用户可控设置不得被顺带改写。
	if repaired.RPCPort != 6801 || repaired.ListenPort != 6888 || repaired.DHTListenPort != 6888 {
		t.Fatalf("端口设置被自愈改写: %#v", repaired)
	}
	if repaired.SpecialMode != "move" || repaired.ConfigVersion != 1 {
		t.Fatalf("归档模式或配置版本被自愈改写: %#v", repaired)
	}
	if len(notes) == 0 {
		t.Fatal("自愈必须留下中文变更说明")
	}
	if !strings.Contains(strings.Join(notes, "|"), "不存在") {
		t.Fatalf("变更说明未指出失效目录: %#v", notes)
	}
}

// TestRepairStartupConfigDropsPoisonedAuthorizedRoot 覆盖另一条现场路径：
// 单个越界授权目录连带否决整份配置，必须只丢弃该条目而不是让应用起不来。
func TestRepairStartupConfigDropsPoisonedAuthorizedRoot(t *testing.T) {
	requireLinux(t)
	share, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	setShare(t, share)

	raw := `{"rpc_port":6800,"listen_port":6888,"download_root":"` + share + `","authorized_roots":["/tmp/h1/locked"],"special_mode":"off"}`
	if _, err := decodeConfig([]byte(raw)); err == nil {
		t.Fatal("越界授权目录必须被常规校验拒绝")
	}
	repaired, _, err := repairStartupConfig([]byte(raw))
	if err != nil {
		t.Fatalf("自愈必须成功: %v", err)
	}
	if len(repaired.AuthorizedRoots) != 0 {
		t.Fatalf("越界授权目录应被移出列表: %#v", repaired.AuthorizedRoots)
	}
	if repaired.DownloadRoot != share {
		t.Fatalf("仍可用的下载目录必须保留: got %q want %q", repaired.DownloadRoot, share)
	}
}

// TestRepairStartupConfigKeepsValidConfig 保证自愈不干涉健康配置：
// 可解析的下载目录与端口一律原样保留；归档目录恒定由下载目录派生（历史自定义值会被覆盖）。
func TestRepairStartupConfigKeepsValidConfig(t *testing.T) {
	requireLinux(t)
	share, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	setShare(t, share)
	download := filepath.Join(share, "myfiles")
	if err := os.Mkdir(download, 0o770); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(download, "done")
	if err := os.Mkdir(archive, 0o770); err != nil {
		t.Fatal(err)
	}

	value := config{
		RPCPort: 6801, ListenPort: 6999, DHTListenPort: 6999, DHTMessageTimeout: "20",
		DownloadRoot: download, ArchiveRoot: archive, SpecialMode: "move",
		TrackerSource: trackerSources[0], TrackerAutoUpdate: true, ConfigVersion: 1,
	}
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeConfig(raw)
	if err != nil {
		t.Fatalf("健康配置应通过常规校验: %v", err)
	}
	repaired, notes, err := repairStartupConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Fatalf("健康配置不应产生自愈变更: %#v", notes)
	}
	if repaired.DownloadRoot != decoded.DownloadRoot || repaired.ArchiveRoot != decoded.ArchiveRoot {
		t.Fatalf("目录被自愈改写: %#v", repaired)
	}
	if repaired.RPCPort != 6801 || repaired.ListenPort != 6999 || repaired.DHTMessageTimeout != "20" {
		t.Fatalf("端口或超时被自愈改写: %#v", repaired)
	}
	if !repaired.TrackerAutoUpdate || repaired.TrackerSource != trackerSources[0] {
		t.Fatalf("Tracker 设置被自愈改写: %#v", repaired)
	}
}

// TestRepairStartupConfigHandlesGarbageInput 覆盖配置文件被写坏或被截断的情况。
func TestRepairStartupConfigHandlesGarbageInput(t *testing.T) {
	requireLinux(t)
	share, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	setShare(t, share)

	for name, raw := range map[string]string{
		"空文件":      "",
		"半截 JSON":  `{"rpc_port":6800,"listen_port":`,
		"端口相同":     `{"rpc_port":6800,"listen_port":6800}`,
		"端口越界":     `{"rpc_port":0,"listen_port":99999}`,
		"归档模式无效":   `{"rpc_port":6800,"listen_port":6888,"special_mode":"rclone"}`,
		"源不是https": `{"rpc_port":6800,"listen_port":6888,"tracker_source":"http://evil/trackers.txt"}`,
	} {
		t.Run(name, func(t *testing.T) {
			repaired, notes, err := repairStartupConfig([]byte(raw))
			if err != nil {
				t.Fatalf("自愈必须成功: %v", err)
			}
			if err := validateConfig(repaired); err != nil {
				t.Fatalf("自愈结果未通过校验: %v", err)
			}
			if repaired.DownloadRoot != share {
				t.Fatalf("下载目录应回落到共享目录: %#v", repaired.DownloadRoot)
			}
			if len(notes) == 0 {
				t.Fatal("自愈必须留下变更说明")
			}
		})
	}
}

// TestConfigStoreUpdateStaysStrict 确认自愈只作用于启动路径：
// 同一份坏配置从前台保存提交仍然必须报错。
func TestConfigStoreUpdateStaysStrict(t *testing.T) {
	requireLinux(t)
	share, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	setShare(t, share)

	store := &configStore{path: filepath.Join(t.TempDir(), "config.json"), cfg: config{RPCPort: 6800, ListenPort: 6888}}
	if err := store.update(config{RPCPort: 6800, ListenPort: 6888, DownloadRoot: "/vol1/1000/no-such-share"}); err == nil {
		t.Fatal("前台保存不得静默接受失效下载目录")
	}
	if _, _, err := repairStartupConfig([]byte(`{"rpc_port":6800,"listen_port":6888,"download_root":"/vol1/1000/no-such-share"}`)); err != nil {
		t.Fatalf("启动路径必须能自愈同一份配置: %v", err)
	}
}

// TestEnsurePrivateDownloadRootFallback 覆盖最后兜底：数据共享目录全部不可用时，
// 启动仍要拿到一个可写目录，并且目录要交还给数据目录的属主。
func TestEnsurePrivateDownloadRootFallback(t *testing.T) {
	requireLinux(t)
	pkgvar, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_PKGVAR", pkgvar)
	t.Setenv("TRIM_DATA_SHARE_PATHS", "/vol1/1000/gone")

	if privateDownloadRoot() != "" {
		t.Fatal("目录尚未创建时读取路径不得顺手建目录")
	}
	created, err := ensurePrivateDownloadRoot()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(pkgvar, fallbackDownloadDirName); created != want {
		t.Fatalf("私有下载目录位置不对: got %q want %q", created, want)
	}
	if privateDownloadRoot() != created {
		t.Fatalf("目录创建后应进入候选列表: %#v", privateDownloadRoot())
	}
	info, err := os.Stat(created)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o770 {
		t.Fatalf("私有下载目录权限应为 0770，实际 %o", perm)
	}
	// 幂等：重复调用不得失败。
	if again, err := ensurePrivateDownloadRoot(); err != nil || again != created {
		t.Fatalf("重复调用应幂等: got %q err %v", again, err)
	}

	// 自愈在共享目录全不可用时应落到私有目录，并给出说明。
	repaired, notes, err := repairStartupConfig([]byte(`{"rpc_port":6800,"listen_port":6888}`))
	if err != nil {
		t.Fatal(err)
	}
	if repaired.DownloadRoot != created {
		t.Fatalf("应回落到私有下载目录: got %q want %q", repaired.DownloadRoot, created)
	}
	if repaired.SpecialMode != "move" || repaired.ArchiveRoot != filepath.Join(created, defaultArchiveDirName) {
		t.Fatalf("兜底目录的归档设置不正确: %#v", repaired)
	}
	if !strings.Contains(strings.Join(notes, "|"), "没有可用的飞牛数据共享目录") {
		t.Fatalf("缺少兜底说明: %#v", notes)
	}
}

// TestRepairStartupConfigFallbackCreatesUsableArchive 覆盖容器实测抓到的回归：私有兜底目录
// 首次创建时，normalizeConfig 会先看到空的下载目录而把归档判成关闭，导致"下载完成后归档"
// 在兜底启动后静默失效。兜底目录必须在派生归档之前建好。
func TestRepairStartupConfigFallbackCreatesUsableArchive(t *testing.T) {
	requireLinux(t)
	pkgvar, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_PKGVAR", pkgvar)
	t.Setenv("TRIM_DATA_SHARE_PATHS", "/vol1/1000/never-created")

	repaired, notes, err := repairStartupConfig([]byte(`{"rpc_port":6800,"listen_port":6888,"special_mode":"move"}`))
	if err != nil {
		t.Fatalf("兜底路径必须能启动: %v", err)
	}
	want := filepath.Join(pkgvar, fallbackDownloadDirName)
	if repaired.DownloadRoot != want {
		t.Fatalf("下载目录未落到兜底位置: got %q want %q", repaired.DownloadRoot, want)
	}
	if repaired.SpecialMode != "move" || !repaired.ArchiveEnabled {
		t.Fatalf("兜底启动不得静默关闭自动归档: %#v", repaired)
	}
	if repaired.ArchiveRoot != filepath.Join(want, defaultArchiveDirName) {
		t.Fatalf("归档目录未随兜底下载目录派生: %#v", repaired.ArchiveRoot)
	}
	if info, statErr := os.Stat(repaired.ArchiveRoot); statErr != nil || !info.IsDir() {
		t.Fatalf("归档目录应已自动创建: %v", statErr)
	}
	if !strings.Contains(strings.Join(notes, "|"), "没有可用的飞牛数据共享目录") {
		t.Fatalf("缺少兜底说明: %#v", notes)
	}
}

// TestRepairStartupConfigSplitsConflictingPorts 覆盖 rpc 自己占用出厂 BT 端口的情况：
// 只把 BT 复位到 6888 等于没修，必须挑一个真正不冲突的端口，否则自愈后仍无法启动。
func TestRepairStartupConfigSplitsConflictingPorts(t *testing.T) {
	requireLinux(t)
	// 共享目录取值本身无关紧要，这里只验证端口自愈；兜底目录需要 TRIM_PKGVAR 可建目录，
	// 因此与其余自愈用例一样只在 Linux 下断言。
	setShare(t, "/vol1/1000/download")
	repaired, notes, err := repairStartupConfig([]byte(`{"rpc_port":6888,"listen_port":6888}`))
	if err != nil {
		t.Fatalf("端口自冲突必须能被自愈: %v", err)
	}
	if repaired.RPCPort != 6888 {
		t.Fatalf("用户可见的 RPC 端口不得被顺带改掉: %#v", repaired.RPCPort)
	}
	if repaired.ListenPort == repaired.RPCPort || !validPort(repaired.ListenPort) {
		t.Fatalf("自愈后端口仍然冲突或非法: %#v", repaired)
	}
	if len(notes) == 0 || !strings.Contains(notes[0], "冲突") {
		t.Fatalf("缺少端口冲突说明: %#v", notes)
	}
}

// TestEnsurePrivateDownloadRootRejectsMissingPkgvar 保证 TRIM_PKGVAR 缺失时明确失败，
// 而不是回落到某个未受控的相对路径。
func TestEnsurePrivateDownloadRootRejectsMissingPkgvar(t *testing.T) {
	t.Setenv("TRIM_PKGVAR", "")
	if _, err := ensurePrivateDownloadRoot(); err == nil {
		t.Fatal("缺少 TRIM_PKGVAR 时必须失败")
	}
	if privateDownloadRoot() != "" {
		t.Fatal("缺少 TRIM_PKGVAR 时私有目录候选必须为空")
	}
}
