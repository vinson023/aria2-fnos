package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// requireLinux 保护依赖真实 POSIX 目录与冒号分隔共享根的用例，Windows 上直接跳过。
func requireLinux(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fnOS 路径与 TRIM_DATA_SHARE_PATHS 仅在 Linux 生命周期下生效")
	}
}

func TestRuntimeOverrideDeniedKeyBlocksSecurityBoundaries(t *testing.T) {
	denied := []string{
		"rpc-listen-port", "rpc-allow-origin-all", "rpc-secret", "on-bt-download-complete",
		"dir", "conf-path", "input-file", "log", "save-session", "save-session-interval",
		"bt-tracker", "bt-listen-port", "bt-metadata-save-directory",
		"dht-file-path", "dht-file-path6", "dht-listen-port", "dht-message-timeout",
		"enable-dht", "enable-dht6", "enable-rpc", "listen-port",
		"save-upload-metadata-file-dir",
	}
	for _, key := range denied {
		if !runtimeOverrideDeniedKey(key) {
			t.Fatalf("独占项 %s 必须被拒绝，否则前台可绕过端口与授权校验", key)
		}
	}
	// rpc 单独出现不构成 rpc- 前缀，属于可放行的普通键名。
	for _, key := range []string{"max-concurrent-downloads", "split", "max-overall-upload-limit", "seed-time", "rpc", "bt-max-peers"} {
		if runtimeOverrideDeniedKey(key) {
			t.Fatalf("常规调优项 %s 不应被拒绝", key)
		}
	}
}

func TestRuntimeOverrideValueKeepsOnlyAria2Literals(t *testing.T) {
	accepted := map[string]string{
		`"true"`: "true",
		`"20M"`:  "20M",
		`"10"`:   "10",
		`20`:     "20",
		`644`:    "644",
		`1.5`:    "1.5",
		`-1`:     "-1",
		`1e3`:    "1e3",
		`true`:   "true",
		`false`:  "false",
		`"a,b"`:  "a,b",
		`"k=v"`:  "k=v",
	}
	for raw, want := range accepted {
		got, ok := runtimeOverrideValue(json.RawMessage(raw))
		if !ok || got != want {
			t.Fatalf("取值 %s 应接受为 %q，实际 %q ok=%v", raw, want, got, ok)
		}
	}
	longValue := `"` + strings.Repeat("x", maxRuntimeOverrideValueLength+8) + `"`
	rejected := []string{
		``, `null`, `{}`, `{"a":1}`, `[]`, `["x"]`, `""`,
		`not-a-number`, `1;2`, `--dir`, `nan`, `"a\nb"`, longValue,
	}
	for _, raw := range rejected {
		if got, ok := runtimeOverrideValue(json.RawMessage(raw)); ok {
			t.Fatalf("取值 %s 必须被拒绝，实际得到 %q", raw, got)
		}
	}
}

// Aria2 的 changeGlobalOption 只接受一个选项对象，且密钥由网关注入到首位，
// 因此客户端参数里的第一个元素才是真正被应用的选项。去更靠后的对象里找会让覆盖文件
// 记下 Aria2 从未生效的取值，重启后才凭空生效，与界面所见分叉。
func TestChangeGlobalOptionParamsUsesOnlyFirstArgument(t *testing.T) {
	options := json.RawMessage(`{"max-concurrent-downloads":"8"}`)
	got := changeGlobalOptionParams([]json.RawMessage{options})
	if got == nil || got["max-concurrent-downloads"] == nil {
		t.Fatalf("AriaNg 实际形状 [options] 未取到选项: %+v", got)
	}
	if v, ok := runtimeOverrideValue(got["max-concurrent-downloads"]); !ok || v != "8" {
		t.Fatalf("选项取值解析不符: %q ok=%v", v, ok)
	}
	// 客户端自带 token 时，Aria2 会把第二个元素当成选项，取第一个之外的对象就是分叉。
	if got := changeGlobalOptionParams([]json.RawMessage{mustJSON("token:abcdef"), options}); got != nil {
		t.Fatalf("首个参数不是选项对象时不得持久化后面的对象: %+v", got)
	}
	if got := changeGlobalOptionParams([]json.RawMessage{json.RawMessage(`{}`), options}); got != nil {
		t.Fatalf("Aria2 只会应用空对象，此时不得记录后面的取值: %+v", got)
	}
	if got := changeGlobalOptionParams(nil); got != nil {
		t.Fatalf("空参数应返回 nil，实际 %+v", got)
	}
	if got := changeGlobalOptionParams([]json.RawMessage{json.RawMessage(`"not-an-object"`)}); got != nil {
		t.Fatalf("非对象参数应返回 nil，实际 %+v", got)
	}
}

func TestRuntimeResponseAcceptedDistinguishesJSONRPCError(t *testing.T) {
	cases := map[string]bool{
		`{"jsonrpc":"2.0","id":"1","result":"OK"}`:                         true,
		`{"jsonrpc":"2.0","id":"1","result":"OK","error":null}`:            true,
		`{"jsonrpc":"2.0","id":"1","error":{"code":1,"message":"failed"}}`: false,
		`{"error":"x"}`:   false,
		`not json at all`: false,
		``:                false,
	}
	for payload, want := range cases {
		if got := runtimeResponseAccepted([]byte(payload)); got != want {
			t.Fatalf("响应 %s 判定为 %v，期望 %v", payload, got, want)
		}
	}
}

func TestRuntimeOverridesMergePersistAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aria2-overrides.conf")
	store := newRuntimeOverrides(path)
	applied := store.merge(map[string]json.RawMessage{
		"split":                    json.RawMessage(`"8"`),
		"max-concurrent-downloads": json.RawMessage(`5`),
		"dir":                      json.RawMessage(`"/etc"`),
		"rpc-secret":               json.RawMessage(`"leak"`),
		"Bad Key":                  json.RawMessage(`"x"`),
		"empty":                    json.RawMessage(`""`),
	})
	if strings.Join(applied, ",") != "max-concurrent-downloads,split" {
		t.Fatalf("落盘项不符: %v", applied)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("运行期选项未落盘: %v", err)
	}
	if string(data) != "max-concurrent-downloads=5\nsplit=8\n" {
		t.Fatalf("落盘内容不符: %q", string(data))
	}
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("运行期选项权限应为 0600，实际 %v %v", info.Mode(), err)
		}
	}
	reloaded := newRuntimeOverrides(path).snapshot()
	if len(reloaded) != 2 || reloaded["split"] != "8" {
		t.Fatalf("重新加载结果不符: %+v", reloaded)
	}
	if err := store.reset(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("reset 之后文件仍存在")
	}
	if got := store.snapshot(); len(got) != 0 {
		t.Fatalf("reset 之后仍有残留状态: %+v", got)
	}
}

func TestRuntimeOverridesWritesAreSortedAndStable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aria2-overrides.conf")
	store := newRuntimeOverrides(path)
	for _, key := range []string{"split", "max-concurrent-downloads", "bt-max-peers", "seed-time"} {
		if applied := store.merge(map[string]json.RawMessage{key: json.RawMessage(`"7"`)}); len(applied) != 1 {
			t.Fatalf("写入 %s 失败: %v", key, applied)
		}
	}
	if applied := store.merge(map[string]json.RawMessage{"peer-id-prefix": json.RawMessage(`"-FG0001-"`)}); len(applied) != 1 {
		t.Fatalf("追加项失败: %v", applied)
	}
	want := "bt-max-peers=7\nmax-concurrent-downloads=7\npeer-id-prefix=-FG0001-\nseed-time=7\nsplit=7\n"
	for i := 0; i < 5; i++ {
		if err := store.writeLocked(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != want {
			t.Fatalf("第 %d 次重写结果不稳定:\n got %q\nwant %q", i+1, string(data), want)
		}
	}
}

func TestRuntimeOverridesEnforceKeyCap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aria2-overrides.conf")
	store := newRuntimeOverrides(path)
	for i := 0; i < maxRuntimeOverrideKeys; i++ {
		key := fmt.Sprintf("user-opt-%03d", i)
		if got := store.merge(map[string]json.RawMessage{key: json.RawMessage(`"1"`)}); len(got) != 1 {
			t.Fatalf("第 %d 项写入失败: %v", i, got)
		}
	}
	if got := store.merge(map[string]json.RawMessage{"user-opt-999": json.RawMessage(`"1"`)}); got != nil {
		t.Fatalf("超过上限的项必须被忽略，实际 %v", got)
	}
	// 覆盖已有键不受数量上限约束。
	if got := store.merge(map[string]json.RawMessage{"user-opt-000": json.RawMessage(`"2"`)}); len(got) != 1 {
		t.Fatalf("覆盖已有键被误拒: %v", got)
	}
	if got := store.snapshot(); len(got) != maxRuntimeOverrideKeys || got["user-opt-000"] != "2" {
		t.Fatalf("快照状态不符: %d 项", len(got))
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("原子写入残留临时文件")
	}
}

func TestRuntimeOverridesDisabledWithoutPath(t *testing.T) {
	store := newRuntimeOverrides("")
	if got := store.merge(map[string]json.RawMessage{"split": json.RawMessage(`"8"`)}); got != nil {
		t.Fatalf("未配置文件路径时不应落盘: %v", got)
	}
	if got := store.snapshot(); len(got) != 0 {
		t.Fatalf("未配置文件路径时快照应为空: %+v", got)
	}
	if err := store.reset(); err != nil {
		t.Fatalf("未配置文件路径时 reset 不应报错: %v", err)
	}
}

func TestParseRuntimeOverrideLinesRejectsCorruption(t *testing.T) {
	ok := map[string]map[string]string{
		"split=8\n": {"split": "8"},
		"split=8\r\nmax-concurrent-downloads=5\r\n": {"split": "8", "max-concurrent-downloads": "5"},
		"peer-id-prefix=-FG0001-\n":                 {"peer-id-prefix": "-FG0001-"},
		"split=8\nsplit=9\n":                        {"split": "9"},
	}
	for payload, want := range ok {
		got, err := parseRuntimeOverrideLines([]byte(payload))
		if err != nil {
			t.Fatalf("合法内容 %q 解析失败: %v", payload, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("合法内容 %q 结果不符:\n got %+v\nwant %+v", payload, got, want)
		}
	}
	// 独占项在解析阶段静默丢弃，坏行则整体报错，避免带病写回 aria2.conf。
	skipped, err := parseRuntimeOverrideLines([]byte("rpc-secret=leak\ndir=/etc\nsplit=8\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 1 || skipped["split"] != "8" {
		t.Fatalf("独占项未被丢弃: %+v", skipped)
	}
	bad := []string{
		"# 注释\n",
		"noequals\n",
		"=8\n",
		"split=\n",
		"Split=8\n",
		"-split=8\n",
		strings.Repeat("k", maxRuntimeOverrideKeyLength+1) + "=8\n",
		"split=" + strings.Repeat("v", maxRuntimeOverrideValueLength+1) + "\n",
	}
	for _, payload := range bad {
		if got, err := parseRuntimeOverrideLines([]byte(payload)); err == nil {
			t.Fatalf("坏内容 %q 应当报错，实际 %+v", payload, got)
		}
	}
}

func TestRuntimeOverridesIgnoreMalformedFileOnDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aria2-overrides.conf")
	if err := os.WriteFile(path, []byte("split=8\nbroken line without equals\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := newRuntimeOverrides(path)
	if got := store.snapshot(); len(got) != 0 {
		t.Fatalf("损坏文件应被整体忽略而不是部分接受: %+v", got)
	}
	// 忽略之后仍可正常写入，损坏文件不会永久卡住功能。
	if applied := store.merge(map[string]json.RawMessage{"seed-time": json.RawMessage(`"0"`)}); len(applied) != 1 {
		t.Fatalf("忽略损坏文件后无法继续写入: %v", applied)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "seed-time=0\n" {
		t.Fatalf("重写结果不符: %q", string(data))
	}
}

// newRPCUpstream 起一个假的 Aria2 RPC，返回值可按用例改写。
func newRPCUpstream(t *testing.T, captured *string, response *string) int {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*captured = string(body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, *response)
	}))
	t.Cleanup(upstream.Close)
	port, err := strconv.Atoi(strings.TrimPrefix(upstream.URL, "http://127.0.0.1:"))
	if err != nil {
		t.Fatal(err)
	}
	return port
}

func TestProxyRPCPersistsOnlyAcceptedGlobalOptions(t *testing.T) {
	var upstreamBody string
	response := `{"jsonrpc":"2.0","id":"1","result":"OK"}`
	port := newRPCUpstream(t, &upstreamBody, &response)
	path := filepath.Join(t.TempDir(), "aria2-overrides.conf")
	store := newRuntimeOverrides(path)
	handler := proxyRPC(port, "unit-test-secret", store, nil, nil)

	post := func(method string, params ...any) *httptest.ResponseRecorder {
		raw := make([]json.RawMessage, 0, len(params))
		for _, p := range params {
			raw = append(raw, mustJSON(p))
		}
		body, marshalErr := json.Marshal(rpcRequest{JSONRPC: "2.0", Method: method, ID: mustJSON("1"), Params: raw})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/jsonrpc", strings.NewReader(string(body))))
		return recorder
	}

	// AriaNg 不发 token（密钥由网关注入），因此选项对象就是第一个参数。
	recorder := post("aria2.changeGlobalOption", map[string]string{"split": "8"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("网关应转发上游 200，实际 %d %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(upstreamBody, `"token:unit-test-secret"`) {
		t.Fatal("网关未注入 RPC 密钥")
	}
	if got := store.snapshot(); got["split"] != "8" {
		t.Fatalf("已接受的全局选项未落盘: %+v", got)
	}

	// 上游返回 JSON-RPC error 时不得落盘，否则会把 aria2 拒绝过的值反复写回配置。
	response = `{"jsonrpc":"2.0","id":"1","error":{"code":1,"message":"failed"}}`
	recorder = post("aria2.changeGlobalOption", map[string]string{"max-concurrent-downloads": "99"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("上游错误响应也应原样转发，实际 %d", recorder.Code)
	}
	if got := store.snapshot(); got["max-concurrent-downloads"] != "" {
		t.Fatalf("被拒绝的选项被错误持久化: %+v", got)
	}

	// 首个参数不是选项对象时，Aria2 应用的也不是后面那个对象，不得持久化。
	response = `{"jsonrpc":"2.0","id":"1","result":"OK"}`
	recorder = post("aria2.changeGlobalOption", map[string]string{}, map[string]string{"seed-time": "0"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("空对象调用失败: %d", recorder.Code)
	}
	if got := store.snapshot(); got["seed-time"] != "" {
		t.Fatalf("Aria2 未应用的选项被持久化: %+v", got)
	}

	// 任务级方法即使带选项对象也不持久化。
	response = `{"jsonrpc":"2.0","id":"1","result":"gid"}`
	recorder = post("aria2.addUri", []string{"http://example.invalid/a.bin"}, map[string]string{"out": "a.bin"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("addUri 转发失败: %d", recorder.Code)
	}
	if got := store.snapshot(); len(got) != 1 {
		t.Fatalf("任务级选项被错误持久化: %+v", got)
	}
}

func TestConfigEntryPointsAgreeOnTheSamePayload(t *testing.T) {
	requireLinux(t)
	payload := `{"rpc_port":6800,"listen_port":6888,"dht_message_timeout":"10","special_mode":"move","gateway_user":"admin","gateway_role":"admin"}`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := decodeConfig(nil); err == nil {
		t.Fatal("空内容不应解析成功")
	}
	viaEnv, err := decodeConfig([]byte(payload))
	if err != nil {
		t.Fatalf("环境变量入口失败: %v", err)
	}
	viaFile, err := loadConfig(path)
	if err != nil {
		t.Fatalf("文件入口失败: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	viaFD, err := loadConfigFD(int(file.Fd()))
	if err != nil {
		t.Fatalf("文件描述符入口失败: %v", err)
	}
	viaStdin, err := decodeConfig([]byte(payload))
	if err != nil {
		t.Fatalf("标准输入入口失败: %v", err)
	}
	for name, got := range map[string]config{"env": viaEnv, "fd": viaFD, "stdin": viaStdin} {
		if !reflect.DeepEqual(got, viaFile) {
			t.Fatalf("%s 入口与文件入口不一致:\n %+v\n vs\n %+v", name, got, viaFile)
		}
	}
	// 历史遗留的非法取值必须被常规入口拒绝，只有迁移模式能修复它。
	legacy := `{"rpc_port":6800,"listen_port":6888,"dht_message_timeout":"10s"}`
	if _, err := decodeConfig([]byte(legacy)); err == nil {
		t.Fatal("常规读取不应静默接受 10s 这类非法超时")
	}
	target := configVersion
	migrated, changes, err := parseConfigData([]byte(legacy), &target)
	if err != nil {
		t.Fatal(err)
	}
	if migrated.DHTMessageTimeout != "10" || migrated.ConfigVersion != configVersion {
		t.Fatalf("迁移未修正历史取值: %+v", migrated)
	}
	if len(changes) == 0 {
		t.Fatal("迁移必须回传中文变更说明供日志使用")
	}
}

func TestAuthorizedRootsRoundTripKeepsEmptyArray(t *testing.T) {
	requireLinux(t)
	t.Setenv("TRIM_DATA_SHARE_PATHS", "")
	t.Setenv("TRIM_PKGVAR", "")
	t.Setenv("TRIM_PKGETC", "")
	store := &configStore{path: filepath.Join(t.TempDir(), "config.json"), cfg: config{RPCPort: 6800, ListenPort: 6888}}
	if err := store.update(config{RPCPort: 6800, ListenPort: 6888, AuthorizedRoots: []string{}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"authorized_roots": []`) {
		t.Fatalf("清空授权根被写成 null，前台会再次解析失败:\n%s", string(data))
	}
	reloaded, err := loadConfig(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.AuthorizedRoots == nil {
		t.Fatal("重新加载后授权根变成 nil")
	}
	again, err := json.Marshal(reloaded)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(again), `"authorized_roots":null`) {
		t.Fatalf("二次保存又退化为 null: %s", string(again))
	}
}

func TestMigrateLegacyConfigOnlyTouchesMachineWrittenValues(t *testing.T) {
	requireLinux(t)
	dir := t.TempDir()
	share := filepath.Join(dir, "share")
	if err := os.MkdirAll(share, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyRoot := filepath.Join(dir, "downloads")
	t.Setenv("TRIM_PKGVAR", dir)
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)
	wantRoot := sharedRoots()[0]
	customArchive := filepath.Join(legacyRoot, "mine")

	data := []byte(`{"rpc_port":6800,"listen_port":6888,"dht_message_timeout":"10s","download_root":"` + legacyRoot +
		`","archive_root":"` + filepath.Join(legacyRoot, defaultArchiveDirName) +
		`","archive_enabled":true,"special_mode":"move","authorized_roots":[],"tracker_source":"https://example.invalid/t.txt","tracker_auto_update":true,"config_version":0}`)
	var value config
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	changes := migrateLegacyConfig(data, &value)
	if value.DHTMessageTimeout != "10" {
		t.Fatalf("DHT 超时未被修正: %q", value.DHTMessageTimeout)
	}
	if value.DownloadRoot != wantRoot {
		t.Fatalf("历史下载目录未迁移: %q want %q", value.DownloadRoot, wantRoot)
	}
	if value.ArchiveRoot != filepath.Join(wantRoot, defaultArchiveDirName) {
		t.Fatalf("默认归档目录未跟随: %q", value.ArchiveRoot)
	}
	joined := strings.Join(changes, "\n")
	for _, want := range []string{"DHT 消息超时时间", "默认下载目录", "默认归档目录"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("变更说明缺少 %s:\n%s", want, joined)
		}
	}

	// 幂等：已迁移的配置再跑一次不产生任何变更。
	already := []byte(`{"rpc_port":6800,"listen_port":6888,"dht_message_timeout":"10","download_root":"` + wantRoot +
		`","archive_root":"` + filepath.Join(wantRoot, defaultArchiveDirName) +
		`","archive_enabled":true,"special_mode":"move","authorized_roots":[],"tracker_source":"https://example.invalid/t.txt","tracker_auto_update":true,"config_version":1}`)
	var second config
	if err := json.Unmarshal(already, &second); err != nil {
		t.Fatal(err)
	}
	if got := migrateLegacyConfig(already, &second); len(got) != 0 {
		t.Fatalf("迁移不幂等，第二次仍报告变更: %v", got)
	}
	if second.DownloadRoot != wantRoot {
		t.Fatalf("第二次迁移改动了下载目录: %q", second.DownloadRoot)
	}

	// 迁移阶段只认「用户显式关闭归档」这一意愿；归档目录本身恒由下载目录派生，
	// 因此这里的自定义归档目录名不参与迁移，保持原样由 normalizeConfig 统一覆盖。
	userIntent := []byte(`{"rpc_port":6800,"listen_port":6888,"dht_message_timeout":"10","download_root":"` + wantRoot +
		`","archive_root":"` + customArchive +
		`","archive_enabled":false,"special_mode":"off","authorized_roots":[],"tracker_source":"https://example.invalid/t.txt","tracker_auto_update":false,"config_version":1}`)
	var third config
	if err := json.Unmarshal(userIntent, &third); err != nil {
		t.Fatal(err)
	}
	if got := migrateLegacyConfig(userIntent, &third); len(got) != 0 {
		t.Fatalf("用户意愿项被机器改写: %v", got)
	}
	if third.SpecialMode != "off" || third.ArchiveEnabled || third.ArchiveRoot != customArchive {
		t.Fatalf("用户设置未保持不变: %+v", third)
	}
}

func TestMigrateConfigFileRewritesThenSettles(t *testing.T) {
	requireLinux(t)
	dir := t.TempDir()
	share := filepath.Join(dir, "share")
	legacyRoot := filepath.Join(dir, "downloads")
	if err := os.MkdirAll(share, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(legacyRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_PKGVAR", dir)
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)
	t.Setenv("TRIM_PKGETC", "")
	path := filepath.Join(dir, "config.json")
	legacy := `{"rpc_port":6800,"listen_port":6888,"dht_message_timeout":"10s","download_root":"` + legacyRoot +
		`","archive_root":"` + filepath.Join(legacyRoot, defaultArchiveDirName) + `","special_mode":"move","tracker_source":"https://example.invalid/t.txt"}`
	if err := os.WriteFile(path, []byte(legacy+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	value, changes, err := migrateConfigFile(path)
	if err != nil {
		t.Fatalf("迁移非法历史配置失败，升级回调将无从修复: %v", err)
	}
	if len(changes) == 0 {
		t.Fatal("首次迁移必须报告变更")
	}
	if value.ConfigVersion != configVersion || value.DHTMessageTimeout != "10" {
		t.Fatalf("迁移结果不符: %+v", value)
	}
	rewritten, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rewritten), `"config_version": 1`) {
		t.Fatalf("迁移未写回版本号，下次升级会重复迁移:\n%s", string(rewritten))
	}
	if strings.Contains(string(rewritten), "10s") || strings.Contains(string(rewritten), legacyRoot) {
		t.Fatalf("磁盘上仍留有历史取值:\n%s", string(rewritten))
	}
	// 第二次执行必须完全静默，升级日志才不会每次都刷出假变更。
	second, secondChanges, err := migrateConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondChanges) != 0 {
		t.Fatalf("重复迁移报告了变更: %v", secondChanges)
	}
	again, err := loadConfig(path)
	if err != nil {
		t.Fatalf("迁移后的文件必须能被常规入口原样读取: %v", err)
	}
	if !reflect.DeepEqual(again, second) {
		t.Fatalf("迁移结果与重新读取结果不一致:\n %+v\n %+v", again, second)
	}
}

// TestProxyRPCStripsGatewayOwnedOptions 验证网关专属选项不会到达 Aria2。
// aria2.changeGlobalOption 能在运行期直接改写 dir，放行就等于绕过目录授权与归档边界。
func TestProxyRPCStripsGatewayOwnedOptions(t *testing.T) {
	var upstreamBody string
	response := `{"jsonrpc":"2.0","id":"1","result":"OK"}`
	port := newRPCUpstream(t, &upstreamBody, &response)
	store := newRuntimeOverrides(filepath.Join(t.TempDir(), "aria2-overrides.conf"))
	handler := proxyRPC(port, "unit-test-secret", store, nil, nil)

	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", Method: "aria2.changeGlobalOption", ID: mustJSON("1"), Params: []json.RawMessage{
		mustJSON(map[string]string{
			"dir":                      "/etc",
			"rpc-listen-port":          "9999",
			"on-download-complete":     "/bin/sh",
			"bt-tracker":               "http://evil.invalid:6969/announce",
			"max-concurrent-downloads": "7",
			"split":                    "8",
		}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/jsonrpc", strings.NewReader(string(body))))
	if recorder.Code != http.StatusOK {
		t.Fatalf("转发失败: %d %s", recorder.Code, recorder.Body.String())
	}

	for _, forbidden := range []string{"/etc", "9999", "/bin/sh", "evil.invalid", `"dir"`, "rpc-listen-port", "on-download-complete", "bt-tracker"} {
		if strings.Contains(upstreamBody, forbidden) {
			t.Fatalf("网关专属选项被转发给 Aria2: %s\n%s", forbidden, upstreamBody)
		}
	}
	if !strings.Contains(upstreamBody, `"split":"8"`) || !strings.Contains(upstreamBody, `"max-concurrent-downloads":"7"`) {
		t.Fatalf("过滤专属项时误伤普通选项: %s", upstreamBody)
	}
	if !strings.Contains(upstreamBody, `"token:unit-test-secret"`) {
		t.Fatalf("网关未注入 RPC 密钥: %s", upstreamBody)
	}

	got := store.snapshot()
	if got["split"] != "8" || got["max-concurrent-downloads"] != "7" {
		t.Fatalf("普通选项未落盘: %+v", got)
	}
	if got["dir"] != "" || got["bt-tracker"] != "" {
		t.Fatalf("专属选项被持久化: %+v", got)
	}
}

// TestFilterGatewayOwnedOptionsKeepsLegitimateKeys 覆盖全部专属项被拒绝时仍转发空对象，
// 让客户端拿到与 Aria2 一致的 OK，而不是一个凭空生效的取值。
func TestFilterGatewayOwnedOptionsKeepsLegitimateKeys(t *testing.T) {
	kept, denied := filterGatewayOwnedOptions(map[string]json.RawMessage{
		"save-session":  mustJSON("/tmp/x"),
		"dht-file-path": mustJSON("/tmp/y"),
		"seed-time":     mustJSON("0"),
	})
	if len(denied) != 2 || denied[0] != "dht-file-path" || denied[1] != "save-session" {
		t.Fatalf("被拒清单不正确: %+v", denied)
	}
	if len(kept) != 1 {
		t.Fatalf("应只保留 seed-time: %+v", kept)
	}

	kept, denied = filterGatewayOwnedOptions(map[string]json.RawMessage{"dir": mustJSON("/etc")})
	if len(denied) != 1 {
		t.Fatalf("dir 必须被拒绝: %+v", denied)
	}
	if kept == nil || len(kept) != 0 {
		t.Fatalf("全部专属项时应返回空集合: %+v", kept)
	}
}
