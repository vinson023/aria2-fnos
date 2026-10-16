package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// ledgerTestName 复现 Aria2 的命名规则：sha1(元数据原文) + 扩展名。
// 用例独立算出期望值，因此同时校验了实现与上游行为一致。
func ledgerTestName(payload []byte, extension string) string {
	sum := sha1.Sum(payload)
	return hex.EncodeToString(sum[:]) + extension
}

// writeLedgerFile 造出真实的元数据文件。Windows 上 resolvePotentialPath 拒绝含反斜杠的
// 未存在路径，因此涉及授权根校验的用例统一先落盘；Linux 生产环境两种情况都能记账。
func writeLedgerFile(t *testing.T, dir, name string, payload []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// newMethodRPC 起一个按方法名返回固定响应的假 Aria2，用于需要多次 RPC 的台账流程。
func newMethodRPC(t *testing.T, responses map[string]string, captured *map[string]string) int {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var request rpcRequest
		_ = json.Unmarshal(body, &request)
		if captured != nil {
			(*captured)[request.Method] = string(body)
		}
		w.Header().Set("Content-Type", "application/json")
		payload, ok := responses[request.Method]
		if !ok {
			payload = `{"jsonrpc":"2.0","id":"1","result":"OK"}`
		}
		_, _ = io.WriteString(w, payload)
	}))
	t.Cleanup(upstream.Close)
	port, err := strconv.Atoi(strings.TrimPrefix(upstream.URL, "http://127.0.0.1:"))
	if err != nil {
		t.Fatal(err)
	}
	return port
}

func TestParseLedgerLineAcceptsOnlySafeRows(t *testing.T) {
	dir := t.TempDir()
	gid := "3068c5d3471aceda"
	goodName := strings.Repeat("a", 40) + ".torrent"
	goodPath := filepath.Join(dir, goodName)
	cases := []struct {
		name string
		line string
		ok   bool
	}{
		{"合规行", gid + "\t" + goodPath, true},
		{"缺少分隔符", gid + goodPath, false},
		{"gid 非十六进制", strings.Repeat("G", 16) + "\t" + goodPath, false},
		{"gid 长度不符", "abc123" + "\t" + goodPath, false},
		{"路径不是绝对路径", gid + "\t" + goodName, false},
		{"文件名不是元数据规则", gid + "\t" + filepath.Join(dir, "evil.torrent"), false},
		{"文件扩展名不符", gid + "\t" + filepath.Join(dir, strings.Repeat("a", 40)+".txt"), false},
	}
	for _, item := range cases {
		gotGID, gotPath, ok := parseLedgerLine(item.line)
		if ok != item.ok {
			t.Fatalf("%s: ok=%v 期望 %v", item.name, ok, item.ok)
		}
		if !ok {
			continue
		}
		if gotGID != gid {
			t.Fatalf("%s: gid 解析错误 %q", item.name, gotGID)
		}
		if gotPath != filepath.Clean(goodPath) {
			t.Fatalf("%s: 路径未规范化 %q", item.name, gotPath)
		}
	}
	// 行内混入控制字符必须整体拒绝，台账不允许成为注入通道。
	for _, poison := range []string{"\x00", "\n"} {
		if _, _, ok := parseLedgerLine(gid + "\t" + goodPath + poison); ok {
			t.Fatalf("路径含 %q 时必须拒绝", poison)
		}
	}
}

func TestTorrentLedgerPersistsAcrossReloads(t *testing.T) {
	dir := t.TempDir()
	gid := "3068c5d3471aceda"
	metadata := []byte("d8:announce20:http://a.example/anne")
	target := filepath.Join(dir, ledgerTestName(metadata, ".torrent"))
	path := filepath.Join(dir, "aria2-torrents.tsv")

	ledger := newTorrentLedger(path)
	changed, err := ledger.record(gid, target)
	if err != nil || !changed {
		t.Fatalf("首次记账应变更: changed=%v err=%v", changed, err)
	}
	if changed, err := ledger.record(gid, target); err != nil || changed {
		t.Fatalf("重复记账不应变更: changed=%v err=%v", changed, err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != gid+"\t"+target+"\n" {
		t.Fatalf("台账落盘内容不符: %q", string(payload))
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("台账含任务路径，权限必须 0600，实际 %v", info.Mode().Perm())
		}
	}

	reloaded := newTorrentLedger(path)
	if got := reloaded.snapshot()[gid]; got != target {
		t.Fatalf("重启后未载入记账: %q", got)
	}
	taken, ok := reloaded.take(gid)
	if !ok || taken != target {
		t.Fatalf("take 未取回记账: %q ok=%v", taken, ok)
	}
	if len(reloaded.snapshot()) != 0 {
		t.Fatalf("take 之后台账应清空: %+v", reloaded.snapshot())
	}
	empty, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("台账清空后文件应为空: %q", string(empty))
	}
	reloaded.restore(gid, target)
	if got := newTorrentLedger(path).snapshot()[gid]; got != target {
		t.Fatalf("restore 未落盘: %q", got)
	}
	// 未知 gid 属于正常情况（普通下载链接任务），不得命中也不得改账。
	if _, ok := reloaded.take("ffffffffffffffff"); ok {
		t.Fatal("不存在的记账不应返回命中")
	}
	if len(newTorrentLedger(path).snapshot()) != 1 {
		t.Fatal("未命中的 take 不应改动台账")
	}
}

func TestTorrentLedgerIgnoresCorruptRowsAndRejectsOversize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aria2-torrents.tsv")
	goodGID := "aaaaaaaaaaaaaaaa"
	goodPath := filepath.Join(dir, strings.Repeat("b", 40)+".torrent")
	broken := "zzzz\t" + filepath.Join(dir, strings.Repeat("c", 40)+".torrent") + "\n"
	if err := os.WriteFile(path, []byte(broken+goodGID+"\t"+goodPath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded := newTorrentLedger(path)
	snapshot := loaded.snapshot()
	if len(snapshot) != 1 || snapshot[goodGID] != goodPath {
		t.Fatalf("坏行应只丢该行并保留其余记账: %+v", snapshot)
	}

	ledger := newTorrentLedger(filepath.Join(dir, "full.tsv"))
	seed := filepath.Join(dir, strings.Repeat("f", 40)+".torrent")
	ledger.mu.Lock()
	for i := 0; i < maxTorrentLedgerEntries; i++ {
		ledger.data[fmt.Sprintf("%016x", i)] = seed
	}
	ledger.mu.Unlock()
	if changed, err := ledger.record(goodGID, seed); err == nil || changed {
		t.Fatalf("达到上限后必须拒绝新增: changed=%v err=%v", changed, err)
	} else if !strings.Contains(err.Error(), "上限") {
		t.Fatalf("超限错误信息缺失: %v", err)
	}
	// 未设置 TRIM_PKGVAR 时台账整体禁用：静默降级，不写任何文件。
	disabled := newTorrentLedger("")
	if changed, err := disabled.record(goodGID, seed); err != nil || changed {
		t.Fatalf("未启用台账时不应记账: changed=%v err=%v", changed, err)
	}
	disabled.restore(goodGID, seed)
	if len(disabled.snapshot()) != 0 {
		t.Fatal("未启用台账时不应回填")
	}
}

func TestTorrentLedgerPruneTreatsNilAsUnknown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aria2-torrents.tsv")
	target := filepath.Join(dir, strings.Repeat("b", 40)+".torrent")
	ledger := newTorrentLedger(path)
	if _, err := ledger.record("aaaaaaaaaaaaaaaa", target); err != nil {
		t.Fatal(err)
	}
	if stale := ledger.prune(nil); stale != nil {
		t.Fatalf("存活集合未知时不得清理: %+v", stale)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("跳过清理时不得改写台账: %v", err)
	}
	if kept := ledger.prune(map[string]bool{"aaaaaaaaaaaaaaaa": true}); len(kept) != 0 {
		t.Fatalf("仍在存活集合里的记账不得被清理: %+v", kept)
	}
	if len(ledger.snapshot()) != 1 {
		t.Fatalf("存活记账不得丢失: %+v", ledger.snapshot())
	}
	// 非 nil 空集合代表 Aria2 确实一个任务都没有。
	stale := ledger.prune(map[string]bool{})
	if len(stale) != 1 || stale["aaaaaaaaaaaaaaaa"] != target {
		t.Fatalf("空存活集合应收敛全部条目: %+v", stale)
	}
	if len(ledger.snapshot()) != 0 {
		t.Fatalf("收敛后台账应空: %+v", ledger.snapshot())
	}
}

func TestMetadataFileNameFollowsAria2Naming(t *testing.T) {
	metadata := []byte("d8:announce20:http://a.example/anne")
	expected := ledgerTestName(metadata, ".torrent")
	padded := base64.StdEncoding.EncodeToString(metadata)
	urlSafeBytes := []byte{0xfb, 0xef, 0xbe}
	cases := []struct {
		name     string
		raw      string
		prefix   string
		expected string
		ok       bool
	}{
		{"标准 base64", padded, "", expected, true},
		{"星号前缀", padded, "*", expected, true},
		{"首尾空白", padded, " ", expected, true},
		{"省略填充", strings.TrimRight(padded, "="), "", expected, true},
		{"URL 安全字符集", base64.URLEncoding.EncodeToString(urlSafeBytes), "", ledgerTestName(urlSafeBytes, ".torrent"), true},
		{"非法 base64", "!!!not-base64!!!", "", "", false},
		{"空内容", "", "", "", false},
		{"只有星号", "", "*", "", false},
	}
	for _, item := range cases {
		name, ok := metadataFileName(mustJSON(item.prefix+item.raw), ".torrent")
		if ok != item.ok {
			t.Fatalf("%s: ok=%v 期望 %v", item.name, ok, item.ok)
		}
		if ok && name != item.expected {
			t.Fatalf("%s: 文件名 %q 期望 %q", item.name, name, item.expected)
		}
	}
	if _, ok := metadataFileName(mustJSON(map[string]string{"a": "b"}), ".torrent"); ok {
		t.Fatal("非字符串元数据不得生成文件名")
	}
	if name, ok := metadataFileName(mustJSON(padded), ".meta4"); !ok || !strings.HasSuffix(name, ".meta4") {
		t.Fatalf("metalink 扩展名错误: %q", name)
	}
}

func TestLedgerPathForRequiresAuthorizedDirectory(t *testing.T) {
	requireLinux(t)
	download := t.TempDir()
	unauthorized := t.TempDir()
	name := strings.Repeat("c", 40) + ".torrent"
	cfg := config{DownloadRoot: download}
	roots := []string{download}
	// 未显式指定 dir 时落到全局下载目录。
	if got, ok := ledgerPathFor(name, "", cfg, roots); !ok || got != filepath.Join(download, name) {
		t.Fatalf("默认下载目录记账失败: %q ok=%v", got, ok)
	}
	// 授权根内的子目录同样允许。
	subdir := filepath.Join(download, "bt")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, ok := ledgerPathFor(name, subdir, cfg, roots); !ok || got != filepath.Join(subdir, name) {
		t.Fatalf("授权子目录记账失败: %q ok=%v", got, ok)
	}
	// 相对路径、越界目录、缺少下载目录一律拒绝。
	if _, ok := ledgerPathFor(name, "relative/dir", cfg, roots); ok {
		t.Fatal("相对目录不得记账")
	}
	if _, ok := ledgerPathFor(name, unauthorized, cfg, roots); ok {
		t.Fatal("未授权目录不得记账")
	}
	if _, ok := ledgerPathFor(name, "", config{}, roots); ok {
		t.Fatal("缺少下载目录不得记账")
	}
	if _, ok := ledgerPathFor(name, "", cfg, nil); ok {
		t.Fatal("没有授权根时不得记账")
	}
}

func TestLedgerMutationForIdentifiesTrackedMethods(t *testing.T) {
	metadata := []byte("d8:announce20:http://a.example/anne")
	encoded := base64.StdEncoding.EncodeToString(metadata)
	name := ledgerTestName(metadata, ".torrent")
	const gid = "3068c5d3471aceda"

	add := ledgerMutationFor("aria2.addTorrent", []json.RawMessage{
		mustJSON(encoded),
		mustJSON([]string{}),
		mustJSON(map[string]string{"dir": "/vol1/1000/download/bt"}),
	})
	if add.addFileName != name || add.addDir != "/vol1/1000/download/bt" || add.forgetGID != "" {
		t.Fatalf("addTorrent 变更识别错误: %+v", add)
	}
	if got := ledgerMutationFor("aria2.addMetalink", []json.RawMessage{mustJSON(encoded)}); got.addFileName != strings.TrimSuffix(name, ".torrent")+".meta4" {
		t.Fatalf("addMetalink 扩展名错误: %+v", got)
	}
	// 少数前端会自带 token，因此只按 gid 形状识别，绝不按位置猜。
	for _, method := range []string{"aria2.remove", "aria2.forceRemove", "aria2.removeDownloadResult", "aria2.forceRemoveDownloadResult"} {
		got := ledgerMutationFor(method, []json.RawMessage{mustJSON("token:secret"), mustJSON(gid)})
		if got.forgetGID != gid || got.empty() {
			t.Fatalf("%s 移除变更识别错误: %+v", method, got)
		}
	}
	if purge := ledgerMutationFor("aria2.purgeDownloadResult", nil); !purge.purge || purge.empty() {
		t.Fatalf("purge 变更识别错误: %+v", purge)
	}
	// 未跟踪方法、参数不足、token 冒充 gid 一律视为无关。
	if got := ledgerMutationFor("aria2.tellActive", []json.RawMessage{mustJSON("token:secret")}); !got.empty() {
		t.Fatalf("无关方法必须空变更: %+v", got)
	}
	if got := ledgerMutationFor("aria2.addTorrent", nil); !got.empty() {
		t.Fatal("缺少元数据参数不得记账")
	}
	if got := ledgerMutationFor("aria2.addTorrent", []json.RawMessage{mustJSON("!!!"), mustJSON(map[string]string{"dir": "/tmp"})}); !got.empty() {
		t.Fatal("非法元数据不得记账")
	}
	if got := ledgerMutationFor("aria2.remove", []json.RawMessage{mustJSON("token:secret")}); got.forgetGID != "" {
		t.Fatal("token 不得被当成 gid")
	}
	if got := ledgerMutationFor("aria2.remove", []json.RawMessage{mustJSON("/../../etc")}); got.forgetGID != "" {
		t.Fatal("非 gid 字符串不得被当成任务")
	}
}

func TestApplyLedgerMutationRecordsThenDeletes(t *testing.T) {
	download := t.TempDir()
	metadata := []byte("d8:announce20:http://a.example/anne")
	name := ledgerTestName(metadata, ".torrent")
	target := writeLedgerFile(t, download, name, metadata)
	const gid = "3068c5d3471aceda"
	scope := func() (config, []string) { return config{DownloadRoot: download}, []string{download} }

	responses := map[string]string{"aria2.addTorrent": `{"jsonrpc":"2.0","id":"1","result":"` + gid + `"}`}
	port := newMethodRPC(t, responses, nil)
	ledger := newTorrentLedger(filepath.Join(t.TempDir(), "aria2-torrents.tsv"))
	addParams := []json.RawMessage{mustJSON(base64.StdEncoding.EncodeToString(metadata)), mustJSON(map[string]string{})}
	applyLedgerMutation(ledger, port, "secret", ledgerMutationFor("aria2.addTorrent", addParams), []byte(responses["aria2.addTorrent"]), scope)
	if got := ledger.snapshot()[gid]; got != target {
		t.Fatalf("网关未记账上传元数据: %+v", ledger.snapshot())
	}

	applyLedgerMutation(ledger, port, "secret", ledgerMutationFor("aria2.remove", []json.RawMessage{mustJSON(gid)}),
		[]byte(`{"jsonrpc":"2.0","id":"1","result":"OK"}`), scope)
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("删除任务后种子元数据仍在: %v", err)
	}
	if len(ledger.snapshot()) != 0 {
		t.Fatalf("删除后台账未清: %+v", ledger.snapshot())
	}

	// Aria2 返回 JSON-RPC error 时不得删文件，也不得丢记账，等下一次清理再收敛。
	other := []byte("second metadata")
	otherPath := writeLedgerFile(t, download, ledgerTestName(other, ".torrent"), other)
	const otherGID = "bbbbbbbbbbbbbbbb"
	if _, err := ledger.record(otherGID, otherPath); err != nil {
		t.Fatal(err)
	}
	applyLedgerMutation(ledger, port, "secret", ledgerMutationFor("aria2.remove", []json.RawMessage{mustJSON(otherGID)}),
		[]byte(`{"jsonrpc":"2.0","id":"1","error":{"code":1,"message":"failed"}}`), scope)
	if _, err := os.Stat(otherPath); err != nil {
		t.Fatalf("Aria2 未确认的移除请求不得删文件: %v", err)
	}
	if got := ledger.snapshot()[otherGID]; got != otherPath {
		t.Fatalf("Aria2 未确认的移除请求不得清账: %q", got)
	}
	// 空变更与台账禁用都必须是纯无操作。
	applyLedgerMutation(ledger, port, "secret", ledgerMutation{}, []byte(`{}`), scope)
	applyLedgerMutation(ledger, port, "secret", ledgerMutationFor("aria2.pause", []json.RawMessage{mustJSON(otherGID)}), []byte(`{"result":"OK"}`), scope)
	applyLedgerMutation(nil, port, "secret", ledgerMutation{purge: true}, nil, scope)
	applyLedgerMutation(ledger, port, "secret", ledgerMutation{forgetGID: otherGID}, nil, nil)
	if _, err := os.Stat(otherPath); err != nil {
		t.Fatalf("越界调用不得删文件: %v", err)
	}
}

func TestApplyLedgerMutationSkipsUnauthorizedDirectory(t *testing.T) {
	download := t.TempDir()
	outside := t.TempDir()
	metadata := []byte("d8:announce20:http://a.example/anne")
	responses := map[string]string{"aria2.addTorrent": `{"jsonrpc":"2.0","id":"1","result":"3068c5d3471aceda"}`}
	port := newMethodRPC(t, responses, nil)
	ledger := newTorrentLedger(filepath.Join(t.TempDir(), "aria2-torrents.tsv"))
	params := []json.RawMessage{mustJSON(base64.StdEncoding.EncodeToString(metadata)), mustJSON(map[string]string{"dir": outside})}
	scope := func() (config, []string) { return config{DownloadRoot: download}, []string{download} }
	applyLedgerMutation(ledger, port, "secret", ledgerMutationFor("aria2.addTorrent", params), []byte(responses["aria2.addTorrent"]), scope)
	if len(ledger.snapshot()) != 0 {
		t.Fatalf("越界目录不得写入台账: %+v", ledger.snapshot())
	}
	if _, err := os.Stat(ledger.path); !os.IsNotExist(err) {
		t.Fatalf("记账失败时不得留下台账文件: %v", err)
	}
	// 台账未启用时整条链路必须无副作用。
	applyLedgerMutation(newTorrentLedger(""), port, "secret", ledgerMutationFor("aria2.addTorrent", params), []byte(responses["aria2.addTorrent"]), scope)
}

func TestPurgeReconcilesLedgerWithLiveTasks(t *testing.T) {
	download := t.TempDir()
	const liveGID = "aaaaaaaaaaaaaaaa"
	const deadGID = "bbbbbbbbbbbbbbbb"
	live := []byte("live metadata")
	dead := []byte("dead metadata")
	livePath := writeLedgerFile(t, download, ledgerTestName(live, ".torrent"), live)
	deadPath := writeLedgerFile(t, download, ledgerTestName(dead, ".torrent"), dead)
	ledger := newTorrentLedger(filepath.Join(t.TempDir(), "aria2-torrents.tsv"))
	for gid, path := range map[string]string{liveGID: livePath, deadGID: deadPath} {
		if _, err := ledger.record(gid, path); err != nil {
			t.Fatal(err)
		}
	}
	scope := func() (config, []string) { return config{DownloadRoot: download}, []string{download} }

	port := newMethodRPC(t, map[string]string{
		"aria2.tellActive":  `{"jsonrpc":"2.0","id":"1","result":[{"gid":"` + liveGID + `"}]}`,
		"aria2.tellWaiting": `{"jsonrpc":"2.0","id":"1","result":[]}`,
		"aria2.tellStopped": `{"jsonrpc":"2.0","id":"1","result":[]}`,
	}, nil)
	applyLedgerMutation(ledger, port, "secret", ledgerMutation{purge: true}, []byte(`{"jsonrpc":"2.0","id":"1","result":"OK"}`), scope)
	if _, err := os.Stat(livePath); err != nil {
		t.Fatalf("仍在任务列表中的种子被误删: %v", err)
	}
	if _, err := os.Stat(deadPath); !os.IsNotExist(err) {
		t.Fatalf("清空已停止任务后残留种子: %v", err)
	}
	snapshot := ledger.snapshot()
	if _, ok := snapshot[liveGID]; !ok {
		t.Fatalf("存活记账被清除: %+v", snapshot)
	}
	if _, ok := snapshot[deadGID]; ok {
		t.Fatalf("残留记账未收敛: %+v", snapshot)
	}

	// Aria2 明确拒绝清空时，同样不得清账、不得删文件。
	applyLedgerMutation(ledger, port, "secret", ledgerMutation{purge: true},
		[]byte(`{"jsonrpc":"2.0","id":"1","error":{"code":1,"message":"failed"}}`), scope)
	if len(ledger.snapshot()) != 1 {
		t.Fatalf("被拒绝的清空请求改了台账: %+v", ledger.snapshot())
	}

	// 任何一类查询失败都当成存活集合未知：本轮完全放弃清理。
	unknown := []byte("unknown metadata")
	unknownPath := writeLedgerFile(t, download, ledgerTestName(unknown, ".torrent"), unknown)
	if _, err := ledger.record("cccccccccccccccc", unknownPath); err != nil {
		t.Fatal(err)
	}
	failing := newMethodRPC(t, map[string]string{
		"aria2.tellStopped": `{"jsonrpc":"2.0","id":"1","error":{"code":1,"message":"resource not found"}}`,
	}, nil)
	applyLedgerMutation(ledger, failing, "secret", ledgerMutation{purge: true}, []byte(`{"jsonrpc":"2.0","id":"1","result":"OK"}`), scope)
	for _, path := range []string{livePath, unknownPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("存活集合未知时不得删除任何文件: %v", err)
		}
	}
	if len(ledger.snapshot()) != 2 {
		t.Fatalf("存活集合未知时不得清账: %+v", ledger.snapshot())
	}
}

func TestLedgerKnownGIDsMergesAllLists(t *testing.T) {
	captured := map[string]string{}
	port := newMethodRPC(t, map[string]string{
		"aria2.tellActive":  `{"jsonrpc":"2.0","id":"1","result":[{"gid":"aaaaaaaaaaaaaaaa"},{"gid":"不合法"}]}`,
		"aria2.tellWaiting": `{"jsonrpc":"2.0","id":"1","result":[{"gid":"bbbbbbbbbbbbbbbb"}]}`,
		"aria2.tellStopped": `{"jsonrpc":"2.0","id":"1","result":[{"gid":"cccccccccccccccc"}]}`,
	}, &captured)
	known, err := ledgerKnownGIDs(port, "unit-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 3 {
		t.Fatalf("应合并三类列表且忽略非法 gid: %+v", known)
	}
	for _, gid := range []string{"aaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbb", "cccccccccccccccc"} {
		if !known[gid] {
			t.Fatalf("缺少存活任务 %s", gid)
		}
	}
	for _, method := range []string{"aria2.tellActive", "aria2.tellWaiting", "aria2.tellStopped"} {
		if !strings.Contains(captured[method], `"token:unit-test-secret"`) {
			t.Fatalf("%s 未带 RPC 密钥: %q", method, captured[method])
		}
	}
	// 端口不通时返回错误，调用方据此放弃清理。
	if _, err := ledgerKnownGIDs(1, "secret"); err == nil {
		t.Fatal("无法联系 Aria2 时必须返回错误")
	}
}

func TestProxyRPCRecordsAndCleansThroughGateway(t *testing.T) {
	download := t.TempDir()
	metadata := []byte("d8:announce20:http://a.example/anne")
	name := ledgerTestName(metadata, ".torrent")
	target := writeLedgerFile(t, download, name, metadata)
	const gid = "3068c5d3471aceda"
	responses := map[string]string{"aria2.addTorrent": `{"jsonrpc":"2.0","id":"1","result":"` + gid + `"}`}
	captured := map[string]string{}
	port := newMethodRPC(t, responses, &captured)

	store := &configStore{
		path: filepath.Join(t.TempDir(), "config.json"),
		cfg: config{
			DownloadRoot:    download,
			AuthorizedRoots: []string{download},
			GatewayUser:     "admin",
		},
	}
	ledger := newTorrentLedger(filepath.Join(t.TempDir(), "aria2-torrents.tsv"))
	handler := proxyRPC(port, "unit-test-secret", newRuntimeOverrides(filepath.Join(t.TempDir(), "overrides.conf")), ledger, store)

	post := func(method string, params ...any) *httptest.ResponseRecorder {
		raw := make([]json.RawMessage, 0, len(params))
		for _, item := range params {
			raw = append(raw, mustJSON(item))
		}
		body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", Method: method, ID: mustJSON("1"), Params: raw})
		if err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/jsonrpc", strings.NewReader(string(body))))
		return recorder
	}

	if recorder := post("aria2.addTorrent", base64.StdEncoding.EncodeToString(metadata), map[string]string{}); recorder.Code != http.StatusOK {
		t.Fatalf("addTorrent 网关响应异常: %d %s", recorder.Code, recorder.Body.String())
	}
	if got := ledger.snapshot()[gid]; got != target {
		t.Fatalf("网关未按 gid 记账: %+v", ledger.snapshot())
	}
	if !strings.Contains(captured["aria2.addTorrent"], `"token:unit-test-secret"`) {
		t.Fatal("网关未注入 RPC 密钥")
	}
	// 密钥注入会让参数整体后移，台账在注入前识别，因此移除请求仍能对上账。
	if recorder := post("aria2.remove", gid); recorder.Code != http.StatusOK {
		t.Fatalf("remove 网关响应异常: %d", recorder.Code)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("经网关删除任务后种子元数据残留: %v", err)
	}
	if len(ledger.snapshot()) != 0 {
		t.Fatalf("经网关删除后台账未清: %+v", ledger.snapshot())
	}
	// 无关方法不得改动台账文件。
	persisted, err := os.ReadFile(ledger.path)
	if err != nil {
		t.Fatal(err)
	}
	if recorder := post("aria2.tellActive", []string{"gid"}); recorder.Code != http.StatusOK {
		t.Fatalf("无关方法响应异常: %d", recorder.Code)
	}
	after, err := os.ReadFile(ledger.path)
	if err != nil || string(after) != string(persisted) {
		t.Fatalf("无关方法改动了台账: %q -> %q", string(persisted), string(after))
	}
}
