package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// wizardEnv 在测试里准备一套安装上下文：数据共享目录 + 持久配置目录。
// resolveDirectory 明确拒绝含反斜杠的路径（避免 aria2.conf 被注入），Windows 路径
// 天然带反斜杠，因此这些用例只在类 Unix 环境执行，由 CI 的 ubuntu runner 覆盖。
func wizardEnv(t *testing.T) (configPath, secretPath, share string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("resolveDirectory 拒绝反斜杠路径，需在 Linux 上执行")
	}
	share, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	etc := t.TempDir()
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)
	t.Setenv("TRIM_PKGETC", etc)
	return filepath.Join(etc, "config.json"), filepath.Join(etc, "rpc.secret"), share
}

// TestValidRPCSecretPattern 与平台无关，任何架构都必须通过同一套字符集校验：
// aria2.conf 的取值按行原样保留（只有行首 '#' 是注释），因此换行与空白是真正的
// 破坏因素；符号（@ ! # $ 等）是安全的，曾经把它们拒之门外反而导致用户改不了密钥。
func TestValidRPCSecretPattern(t *testing.T) {
	good := []string{
		"abcdefgh",    // 下限 8 位
		"vinson520",   // 常见的 9 位口令
		"Vinson@2024", // 符号
		"Abc-123_def.4567",
		"p@ss!w0rd#1$", // # 与 $ 在配置行里是普通字符
		strings.Repeat("a", maxRPCSecretLength),
	}
	for _, secret := range good {
		if !validRPCSecret(secret) {
			t.Fatalf("%q must be accepted", secret)
		}
	}
	bad := []string{
		"",
		"short",                  // 5 位，低于下限
		"abcdefg",                // 7 位
		"abcdefgh ijklmnop",      // 空格
		"abcdefgh\tijklmnop",     // 制表符
		"abcdefghijklmnop\r",     // 回车会插入新行
		"abcdefghijklmnop\npath", // 换行会引入额外配置项
		"密钥密钥密钥密钥",               // 非 ASCII
		strings.Repeat("a", maxRPCSecretLength+1), // 超长
	}
	for _, secret := range bad {
		if validRPCSecret(secret) {
			t.Fatalf("%q must be rejected", secret)
		}
	}
}

func TestParseWizardPort(t *testing.T) {
	if port, err := parseWizardPort("6888", "BT 监听端口"); err != nil || port != 6888 {
		t.Fatalf("valid port rejected: %d %v", port, err)
	}
	for _, raw := range []string{"", "80", "65536", "abc", "-1", "6888.0"} {
		if _, err := parseWizardPort(raw, "BT 监听端口"); err == nil {
			t.Fatalf("%q must be rejected", raw)
		}
	}
}

// TestDownloadRootAllowed 固化“哪些目录可以作为下载目录”的判定：已授权根内一律允许；
// 授权根之外的路径只在飞牛数据卷（/vol<N>）内才继续考察写入权限，系统目录与临时目录
// 直接拒绝。该函数是纯判定，不访问文件系统，因此各平台都能跑。
func TestDownloadRootAllowed(t *testing.T) {
	approved := []string{"/vol1/@appshare/aria2/downloads"}
	cases := map[string]bool{
		"/vol1/@appshare/aria2/downloads":        true,
		"/vol1/@appshare/aria2/downloads/movies": true,
		"/vol1/1000/downloads":                   true,
		"/vol2/1000/downloads":                   true,
		"/etc/aria2":                             false,
		"/tmp/aria2":                             false,
		"":                                       false,
		"/vol1":                                  true,
	}
	for path, want := range cases {
		if got := downloadRootAllowed(path, approved); got != want {
			t.Fatalf("downloadRootAllowed(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestApplyWizardCreatesConfigWithSubmittedValues(t *testing.T) {
	configPath, secretPath, share := wizardEnv(t)
	t.Setenv(wizardFieldRPCPort, "6900")
	t.Setenv(wizardFieldListenPort, "6999")
	t.Setenv(wizardFieldDHTListenPort, "6998")
	t.Setenv(wizardFieldSpecialMode, "off")
	t.Setenv(wizardFieldDownloadRoot, filepath.Join(share, "downloads"))
	t.Setenv(wizardFieldRPCSecret, "abcdefghijklmnop1234")

	notes, changed, err := applyWizardInput(configPath, secretPath, true)
	if err != nil {
		t.Fatalf("apply wizard: %v", err)
	}
	if !changed {
		t.Fatal("首次写入必须报告配置已变更")
	}
	if len(notes) == 0 {
		t.Fatal("expected human readable notes")
	}

	value, err := loadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if value.RPCPort != 6900 || value.ListenPort != 6999 || value.DHTListenPort != 6998 {
		t.Fatalf("ports not applied: %#v", value)
	}
	if value.SpecialMode != "off" || value.ArchiveEnabled {
		t.Fatalf("archive must be off: %#v", value)
	}
	if want := filepath.Join(share, "downloads"); value.DownloadRoot != want {
		t.Fatalf("download root = %q, want %q", value.DownloadRoot, want)
	}
	if info, err := os.Stat(value.DownloadRoot); err != nil || !info.IsDir() {
		t.Fatalf("download root was not created: %v", err)
	}

	secret, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(secret) != "abcdefghijklmnop1234" {
		t.Fatalf("secret not persisted: %q", secret)
	}
}

func TestApplyWizardGeneratesSecretWhenMissing(t *testing.T) {
	configPath, secretPath, _ := wizardEnv(t)
	t.Setenv(wizardFieldRPCPort, "6800")
	t.Setenv(wizardFieldListenPort, "6888")

	if _, _, err := applyWizardInput(configPath, secretPath, true); err != nil {
		t.Fatalf("apply wizard: %v", err)
	}
	secret, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatal(err)
	}
	if !validRPCSecret(string(secret)) {
		t.Fatalf("generated secret is not valid: %q", secret)
	}
}

func TestApplyWizardRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name  string
		env   map[string]string
		error string
	}{
		{"rpc port out of range", map[string]string{wizardFieldRPCPort: "80"}, "RPC 监听端口"},
		{"port conflict", map[string]string{wizardFieldListenPort: "6800"}, "不能与 BT 监听端口相同"},
		{"bad dht port", map[string]string{wizardFieldDHTListenPort: "70000"}, "DHT 监听端口"},
		{"bad special mode", map[string]string{wizardFieldSpecialMode: "rclone"}, "归档"},
		{"relative download root", map[string]string{wizardFieldDownloadRoot: "downloads"}, "绝对路径"},
		{"secret too short", map[string]string{wizardFieldRPCSecret: "short"}, "RPC 密钥"},
		{"secret with space", map[string]string{wizardFieldRPCSecret: "abcdefgh ijklmnop"}, "RPC 密钥"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configPath, secretPath, _ := wizardEnv(t)
			t.Setenv(wizardFieldRPCPort, "6800")
			t.Setenv(wizardFieldListenPort, "6888")
			for key, value := range tc.env {
				t.Setenv(key, value)
			}
			_, _, err := applyWizardInput(configPath, secretPath, true)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.error) {
				t.Fatalf("error %q does not mention %q", err, tc.error)
			}
		})
	}
}

func TestApplyWizardKeepsExistingValuesWhenFieldsEmpty(t *testing.T) {
	configPath, secretPath, share := wizardEnv(t)
	t.Setenv(wizardFieldRPCPort, "6900")
	t.Setenv(wizardFieldListenPort, "6999")
	t.Setenv(wizardFieldRPCSecret, "abcdefghijklmnop1234")
	if _, _, err := applyWizardInput(configPath, secretPath, true); err != nil {
		t.Fatalf("initial apply: %v", err)
	}

	// 应用设置只改归档开关：端口、密钥、下载目录都不填，必须保持原值。
	t.Setenv(wizardFieldRPCPort, "")
	t.Setenv(wizardFieldListenPort, "")
	t.Setenv(wizardFieldRPCSecret, "")
	t.Setenv(wizardFieldSpecialMode, "off")
	if _, _, err := applyWizardInput(configPath, secretPath, false); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	value, err := loadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if value.RPCPort != 6900 || value.ListenPort != 6999 {
		t.Fatalf("ports must stay unchanged: %#v", value)
	}
	if value.DownloadRoot != share {
		t.Fatalf("download root must stay unchanged: %q", value.DownloadRoot)
	}
	if value.SpecialMode != "off" {
		t.Fatalf("special mode = %q, want off", value.SpecialMode)
	}
	secret, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(secret) != "abcdefghijklmnop1234" {
		t.Fatalf("secret must stay unchanged: %q", secret)
	}
}

// TestApplyWizardReportsNoChangeForIdenticalInput 保证“原样保存”不会触发重启：
// 配置与密钥都没变化时必须返回 changed=false。飞牛会把上次提交的密钥原样回填并再次
// 提交，因此“提交了密钥”不能直接等同于“密钥变了”。
func TestApplyWizardReportsNoChangeForIdenticalInput(t *testing.T) {
	configPath, secretPath, _ := wizardEnv(t)
	t.Setenv(wizardFieldRPCPort, "6900")
	t.Setenv(wizardFieldListenPort, "6999")
	t.Setenv(wizardFieldSpecialMode, "off")
	t.Setenv(wizardFieldRPCSecret, "abcdefghijklmnop1234")
	if _, changed, err := applyWizardInput(configPath, secretPath, true); err != nil || !changed {
		t.Fatalf("首次写入应有变更: changed=%v err=%v", changed, err)
	}
	// 第二次提交完全相同的取值，包括被回填的密钥。
	if _, changed, err := applyWizardInput(configPath, secretPath, false); err != nil {
		t.Fatalf("二次写入: %v", err)
	} else if changed {
		t.Fatal("取值未变化时不应报告变更")
	}
	// 真正换一个密钥时必须报告变更。
	t.Setenv(wizardFieldRPCSecret, "zyxwvutsrqponmlk9876")
	if _, changed, err := applyWizardInput(configPath, secretPath, false); err != nil {
		t.Fatalf("三次写入: %v", err)
	} else if !changed {
		t.Fatal("密钥变化时必须报告变更")
	}
	secret, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(secret) != "zyxwvutsrqponmlk9876" {
		t.Fatalf("secret not rotated: %q", secret)
	}
}

func TestApplyWizardRejectsDownloadRootOutsideShares(t *testing.T) {
	configPath, secretPath, _ := wizardEnv(t)
	outside := t.TempDir()
	t.Setenv(wizardFieldDownloadRoot, filepath.Join(outside, "downloads"))
	if _, _, err := applyWizardInput(configPath, secretPath, true); err == nil {
		t.Fatal("expected outside directory to be rejected")
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatal("config must not be written when validation fails")
	}
}

func TestApplyWizardWithoutConfigRequiresInitial(t *testing.T) {
	configPath, secretPath, _ := wizardEnv(t)
	if _, _, err := applyWizardInput(configPath, secretPath, false); err == nil {
		t.Fatal("expected missing config to be rejected outside install flow")
	}
}

// TestWizardConfigRoundTrip 保证写出的配置文件能被常规读取路径原样接受，避免
// 向导写入的字段名与 config 结构体标签漂移。
func TestWizardConfigRoundTrip(t *testing.T) {
	configPath, secretPath, _ := wizardEnv(t)
	t.Setenv(wizardFieldSpecialMode, "move")
	if _, _, err := applyWizardInput(configPath, secretPath, true); err != nil {
		t.Fatalf("apply wizard: %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"rpc_port", "listen_port", "dht_listen_port", "download_root", "archive_root", "special_mode", "config_version"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("config is missing %q: %s", key, data)
		}
	}
}
