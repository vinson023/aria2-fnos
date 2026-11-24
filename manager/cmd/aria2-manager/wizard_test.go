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
// aria2.conf 由 shell 逐行写出，密钥里出现空白或引号会破坏该文件。
func TestValidRPCSecretPattern(t *testing.T) {
	good := []string{
		"abcdefghijklmnop",
		"0123456789abcdef0123456789abcdef0123456789abcdef",
		"Abc-123_def.4567",
	}
	for _, secret := range good {
		if !validRPCSecret(secret) {
			t.Fatalf("%q must be accepted", secret)
		}
	}
	bad := []string{
		"",
		"short",
		"abcdefghijklmno",    // 15 位
		"abcdefgh ijklmnop",  // 空格
		"abcdefgh\"ijklmnop", // 引号
		"abcdefgh#ijklmnop",  // 井号会被当成注释
		"abcdefghijklmnop$",  // 会触发 shell 展开
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

func TestApplyWizardCreatesConfigWithSubmittedValues(t *testing.T) {
	configPath, secretPath, share := wizardEnv(t)
	t.Setenv(wizardFieldRPCPort, "6900")
	t.Setenv(wizardFieldListenPort, "6999")
	t.Setenv(wizardFieldDHTListenPort, "6998")
	t.Setenv(wizardFieldSpecialMode, "off")
	t.Setenv(wizardFieldDownloadRoot, filepath.Join(share, "downloads"))
	t.Setenv(wizardFieldRPCSecret, "abcdefghijklmnop1234")

	notes, err := applyWizardInput(configPath, secretPath, true)
	if err != nil {
		t.Fatalf("apply wizard: %v", err)
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

	if _, err := applyWizardInput(configPath, secretPath, true); err != nil {
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
			_, err := applyWizardInput(configPath, secretPath, true)
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
	if _, err := applyWizardInput(configPath, secretPath, true); err != nil {
		t.Fatalf("initial apply: %v", err)
	}

	// 应用设置只改归档开关：端口、密钥、下载目录都不填，必须保持原值。
	t.Setenv(wizardFieldRPCPort, "")
	t.Setenv(wizardFieldListenPort, "")
	t.Setenv(wizardFieldRPCSecret, "")
	t.Setenv(wizardFieldSpecialMode, "off")
	if _, err := applyWizardInput(configPath, secretPath, false); err != nil {
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

func TestApplyWizardRejectsDownloadRootOutsideShares(t *testing.T) {
	configPath, secretPath, _ := wizardEnv(t)
	outside := t.TempDir()
	t.Setenv(wizardFieldDownloadRoot, filepath.Join(outside, "downloads"))
	if _, err := applyWizardInput(configPath, secretPath, true); err == nil {
		t.Fatal("expected outside directory to be rejected")
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatal("config must not be written when validation fails")
	}
}

func TestApplyWizardWithoutConfigRequiresInitial(t *testing.T) {
	configPath, secretPath, _ := wizardEnv(t)
	if _, err := applyWizardInput(configPath, secretPath, false); err == nil {
		t.Fatal("expected missing config to be rejected outside install flow")
	}
}

// TestWizardConfigRoundTrip 保证写出的配置文件能被常规读取路径原样接受，避免
// 向导写入的字段名与 config 结构体标签漂移。
func TestWizardConfigRoundTrip(t *testing.T) {
	configPath, secretPath, _ := wizardEnv(t)
	t.Setenv(wizardFieldSpecialMode, "move")
	if _, err := applyWizardInput(configPath, secretPath, true); err != nil {
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
