package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Aria2 在 rpc-save-upload-metadata=true（1.36 的出厂默认就是 true）时，把 RPC 上传的
// 种子/metalink 存成 "<sha1(元数据原文)>.torrent"：文件名由元数据内容决定，与任务 gid、
// 与下载出来的文件都无关，源码见 RpcMethodImpl.cc 的 AddTorrentRpcMethod::process。
// on-download-* 回调只拿到 gid、文件数量和下载出来的路径，钩子进程无法反推这个文件名，
// 于是删除任务后下载目录里会留下无人认领的 .torrent。管理器是所有 RPC 的必经网关，
// addTorrent 的元数据原文必然经过它，所以在网关侧按 gid 记账，移除任务时按账清理。
// 详见 docs/adr/ADR-014-rpc-uploaded-metadata-ledger.md。

const (
	// maxTorrentLedgerEntries 限制台账规模。条目只在 addTorrent/addMetalink 时增长，
	// 正常会话远达不到上限；超限时拒绝新增并记录，不静默丢弃记账。
	maxTorrentLedgerEntries = 1024
	// ledgerSeparator 分隔 gid 与元数据文件绝对路径。
	ledgerSeparator = "\t"
	// ledgerScanLimit 是台账收敛时向 Aria2 索要的条目上限，必须不小于台账容量，
	// 否则会把仍在任务列表里的 gid 误判为残留。
	ledgerScanLimit = 4096
)

var (
	// torrentGIDPattern 匹配 Aria2 固定 16 位十六进制的 gid。
	torrentGIDPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)
	// torrentMetadataPattern 只接受“sha1 十六进制 + 官方扩展名”的文件名，其他一律拒绝。
	torrentMetadataPattern = regexp.MustCompile(`^[0-9a-f]{40}\.(torrent|meta4)$`)
)

// torrentLedger 是 gid 到元数据文件绝对路径的映射，落盘格式为每行 "gid<TAB>path"。
type torrentLedger struct {
	mu   sync.Mutex
	path string
	data map[string]string
}

func newTorrentLedger(path string) *torrentLedger {
	ledger := &torrentLedger{path: path, data: make(map[string]string)}
	ledger.load()
	return ledger
}

// load 读取既有台账。个别坏行只丢该行并记录一次：丢记账最坏是退回“残留一个 .torrent”的
// 原状，而整体拒绝加载会让所有条目失去归属，反而扩大残留。
func (l *torrentLedger) load() {
	if l.path == "" {
		return
	}
	payload, err := os.ReadFile(l.path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("读取种子元数据台账失败: %v", err)
		}
		return
	}
	loaded := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(payload))
	scanner.Buffer(make([]byte, 0, 4096), 8192)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			continue
		}
		gid, filePath, ok := parseLedgerLine(line)
		if !ok {
			log.Printf("忽略不合规的种子元数据台账行")
			continue
		}
		if len(loaded) >= maxTorrentLedgerEntries {
			break
		}
		loaded[gid] = filePath
	}
	if err := scanner.Err(); err != nil {
		log.Printf("解析种子元数据台账中断: %v", err)
		return
	}
	l.data = loaded
}

func parseLedgerLine(line string) (string, string, bool) {
	fields := strings.SplitN(line, ledgerSeparator, 2)
	if len(fields) != 2 {
		return "", "", false
	}
	gid, filePath := fields[0], fields[1]
	if !torrentGIDPattern.MatchString(gid) {
		return "", "", false
	}
	if !filepath.IsAbs(filePath) || !torrentMetadataPattern.MatchString(filepath.Base(filePath)) {
		return "", "", false
	}
	if strings.ContainsAny(filePath, "\x00\r\n") {
		return "", "", false
	}
	return gid, filepath.Clean(filePath), true
}

// record 记下某个任务对应的元数据文件，返回是否发生变更。
func (l *torrentLedger) record(gid, filePath string) (bool, error) {
	if l.path == "" || !torrentGIDPattern.MatchString(gid) || !torrentMetadataPattern.MatchString(filepath.Base(filePath)) {
		return false, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	target := filepath.Clean(filePath)
	if existing, ok := l.data[gid]; ok && existing == target {
		return false, nil
	}
	if len(l.data) >= maxTorrentLedgerEntries {
		return false, fmt.Errorf("种子元数据台账已达 %d 条上限", maxTorrentLedgerEntries)
	}
	l.data[gid] = target
	return true, l.writeLocked()
}

// take 取出并移除某个 gid 的记账；调用方随后负责删除该文件。
func (l *torrentLedger) take(gid string) (string, bool) {
	if l.path == "" {
		return "", false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	filePath, ok := l.data[gid]
	if !ok {
		return "", false
	}
	delete(l.data, gid)
	if err := l.writeLocked(); err != nil {
		log.Printf("更新种子元数据台账失败: %v", err)
	}
	return filePath, true
}

// restore 在文件删除失败时把记账放回，避免“文件还在、账没了”的永久残留。
func (l *torrentLedger) restore(gid, filePath string) {
	if l.path == "" || !torrentGIDPattern.MatchString(gid) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.data[gid]; ok {
		return
	}
	if len(l.data) >= maxTorrentLedgerEntries {
		return
	}
	l.data[gid] = filePath
	if err := l.writeLocked(); err != nil {
		log.Printf("回填种子元数据台账失败: %v", err)
	}
}

// prune 保留 known 集合内的 gid，返回被移除的 gid 到路径的映射。known 为 nil 表示
// “Aria2 存活任务集合未知”，此时一律不清理：宁可不删，也不能因为一次查询失败就删掉
// 仍在任务列表里的种子。known 是非 nil 空集合时表示 Aria2 确实一个任务都没有。
func (l *torrentLedger) prune(known map[string]bool) map[string]string {
	if l.path == "" || known == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	stale := make(map[string]string)
	for gid, filePath := range l.data {
		if known[gid] {
			continue
		}
		stale[gid] = filePath
		delete(l.data, gid)
	}
	if len(stale) == 0 {
		return nil
	}
	if err := l.writeLocked(); err != nil {
		log.Printf("清理种子元数据台账失败: %v", err)
	}
	return stale
}

// snapshot 返回台账副本，供测试与诊断使用。
func (l *torrentLedger) snapshot() map[string]string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make(map[string]string, len(l.data))
	for gid, filePath := range l.data {
		out[gid] = filePath
	}
	return out
}

// writeLocked 以 0600 原子写出，行按字典序排列，保证同一份状态每次落盘字节一致。
// 调用方必须持有 l.mu。
func (l *torrentLedger) writeLocked() error {
	lines := make([]string, 0, len(l.data))
	for gid, filePath := range l.data {
		lines = append(lines, gid+ledgerSeparator+filePath)
	}
	sort.Strings(lines)
	payload := strings.Join(lines, "\n")
	if len(lines) > 0 {
		payload += "\n"
	}
	temporary := l.path + ".tmp"
	if err := os.WriteFile(temporary, []byte(payload), 0o600); err != nil {
		return err
	}
	if err := os.Rename(temporary, l.path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

// metadataFileName 依据 Aria2 的命名规则算出上传元数据的落盘文件名。
// addTorrent 的 params[0] 是 base64 元数据；Aria2 允许以 "*" 前缀表示按文件名创建，先剥掉。
func metadataFileName(raw json.RawMessage, extension string) (string, bool) {
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return "", false
	}
	encoded = strings.TrimPrefix(strings.TrimSpace(encoded), "*")
	if encoded == "" {
		return "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		// 部分前端会省略填充或使用 URL 安全字符集，按 StdEncoding、RawStdEncoding、
		// URLEncoding 依次回退，全部失败才判定为不合法元数据。
		if decoded, err = base64.RawStdEncoding.DecodeString(encoded); err != nil {
			if decoded, err = base64.URLEncoding.DecodeString(encoded); err != nil {
				return "", false
			}
		}
	}
	if len(decoded) == 0 {
		return "", false
	}
	digest := sha1.Sum(decoded)
	return hex.EncodeToString(digest[:]) + extension, true
}

// ledgerPathFor 把算出的文件名落到任务实际使用的目录，并要求该目录在授权根之内。
// 越界直接放弃记账：宁残留不误删，也绝不让台账变成可以指向任意路径的删除清单。
func ledgerPathFor(fileName string, requestedDir string, cfg config, roots []string) (string, bool) {
	candidateDir := strings.TrimSpace(requestedDir)
	if candidateDir == "" {
		candidateDir = cfg.DownloadRoot
	}
	if candidateDir == "" || !filepath.IsAbs(candidateDir) {
		return "", false
	}
	candidate := filepath.Join(candidateDir, fileName)
	if _, _, err := taskContainedRoot(candidate, roots); err != nil {
		return "", false
	}
	return candidate, true
}

// ledgerMutation 描述一次 RPC 调用对种子元数据台账的影响。必须在注入 RPC 密钥之前、
// 用客户端原始参数构造，因为密钥注入会整体下标 params。
type ledgerMutation struct {
	// addFileName 是本次新增任务会写入下载目录的元数据文件名，空表示不是上传任务。
	addFileName string
	// addDir 是任务选项里显式指定的 dir，空表示沿用全局下载目录。
	addDir string
	// forgetGID 是被移除任务的 gid，空表示不是移除操作。
	forgetGID string
	// purge 表示 Aria2 清空了全部停止结果，需要按存活集合收敛台账。
	purge bool
}

func (m ledgerMutation) empty() bool {
	return m.addFileName == "" && m.forgetGID == "" && !m.purge
}

// ledgerMutationFor 识别台账关心的六个 Aria2 方法。其余方法一律返回空变更。
func ledgerMutationFor(method string, params []json.RawMessage) ledgerMutation {
	var mutation ledgerMutation
	switch method {
	case "aria2.addTorrent", "aria2.addMetalink":
		extension := ".torrent"
		if method == "aria2.addMetalink" {
			extension = ".meta4"
		}
		if len(params) == 0 {
			return mutation
		}
		if name, ok := metadataFileName(params[0], extension); ok {
			mutation.addFileName = name
			mutation.addDir = ledgerRequestedDir(params)
		}
	case "aria2.remove", "aria2.forceRemove", "aria2.removeDownloadResult", "aria2.forceRemoveDownloadResult":
		mutation.forgetGID = ledgerGIDParam(params)
	case "aria2.purgeDownloadResult":
		mutation.purge = true
	}
	return mutation
}

// ledgerGIDParam 取出被移除任务的 gid。Aria2 的移除类方法第一个参数就是 gid 字符串，
// 但少数前端会自己带上密钥，因此只接受形如 16 位十六进制的参数，绝不按位置猜。
func ledgerGIDParam(params []json.RawMessage) string {
	for _, raw := range params {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			continue
		}
		if torrentGIDPattern.MatchString(value) {
			return value
		}
	}
	return ""
}

// ledgerRequestedDir 从任务选项对象里读出 dir（若客户端显式指定）。选项对象是参数里
// 唯一的字典，取不到就按空处理，交给全局下载目录兜底。
func ledgerRequestedDir(params []json.RawMessage) string {
	for _, raw := range params {
		var values map[string]json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			continue
		}
		encoded, ok := values["dir"]
		if !ok {
			continue
		}
		var dir string
		if err := json.Unmarshal(encoded, &dir); err != nil {
			return ""
		}
		return dir
	}
	return ""
}

// ledgerResultGID 从 Aria2 响应里取出新增任务返回的 gid。
func ledgerResultGID(payload []byte) (string, bool) {
	var response struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return "", false
	}
	if !torrentGIDPattern.MatchString(response.Result) {
		return "", false
	}
	return response.Result, true
}

// applyLedgerMutation 在 Aria2 确认接受请求之后更新台账并清理文件。任何一步失败都只记录
// 日志并保留记账，绝不因为清理失败而扩大删除范围。响应必须在这里再判一次：网关是唯一调用方
// 不是保证，未确认的响应一律不动账，也绝不动文件。
func applyLedgerMutation(ledger *torrentLedger, port int, secret string, mutation ledgerMutation, payload []byte, scope func() (config, []string)) {
	if ledger == nil || ledger.path == "" || mutation.empty() || scope == nil || !runtimeResponseAccepted(payload) {
		return
	}
	cfg, roots := scope()
	switch {
	case mutation.addFileName != "":
		gid, ok := ledgerResultGID(payload)
		if !ok {
			return
		}
		path, ok := ledgerPathFor(mutation.addFileName, mutation.addDir, cfg, roots)
		if !ok {
			log.Printf("跳过种子元数据记账: 目标目录不在授权根内")
			return
		}
		changed, err := ledger.record(gid, path)
		if err != nil {
			log.Printf("记录种子元数据台账失败: %v", err)
		} else if changed {
			log.Printf("已记录种子元数据: %s", filepath.Base(path))
		}
	case mutation.forgetGID != "":
		path, ok := ledger.take(mutation.forgetGID)
		if !ok {
			return
		}
		if err := removeTaskControl(path, roots); err != nil {
			log.Printf("删除种子元数据失败: %v", err)
			ledger.restore(mutation.forgetGID, path)
			return
		}
		log.Printf("已删除种子元数据: %s", filepath.Base(path))
	case mutation.purge:
		pruneLedgerStale(ledger, port, secret, roots)
	}
}

// pruneLedgerStale 在“清空已停止任务”之后收敛台账：向 Aria2 确认仍然存活的 gid 集合，
// 剩下的都是永不可能再被引用的孤儿，连带删掉它们的元数据文件。查询失败直接放弃本轮，
// 保持原样，等下一次批量清理再收敛。
func pruneLedgerStale(ledger *torrentLedger, port int, secret string, roots []string) {
	if len(ledger.snapshot()) == 0 {
		return
	}
	known, err := ledgerKnownGIDs(port, secret)
	if err != nil {
		log.Printf("跳过种子元数据清理: 无法确认存活任务: %v", err)
		return
	}
	stale := ledger.prune(known)
	for gid, path := range stale {
		if err := removeTaskControl(path, roots); err != nil {
			log.Printf("清理种子元数据失败: %v", err)
			ledger.restore(gid, path)
			continue
		}
		log.Printf("已清理残留种子元数据: %s", filepath.Base(path))
	}
}

// ledgerKnownGIDs 汇总 Aria2 仍然认识的全部 gid：活动中、排队或暂停中、以及停止结果。
// 只要有一类查询失败就返回错误，调用方据此放弃清理，避免把未知当成“不存在”。
func ledgerKnownGIDs(port int, secret string) (map[string]bool, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	known := make(map[string]bool)
	calls := []struct {
		method string
		extra  []any
	}{
		{method: "aria2.tellActive", extra: []any{[]string{"gid"}}},
		{method: "aria2.tellWaiting", extra: []any{0, ledgerScanLimit, []string{"gid"}}},
		{method: "aria2.tellStopped", extra: []any{0, ledgerScanLimit, []string{"gid"}}},
	}
	for _, call := range calls {
		params := make([]json.RawMessage, 0, len(call.extra)+1)
		params = append(params, mustJSON("token:"+secret))
		for _, value := range call.extra {
			params = append(params, mustJSON(value))
		}
		body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", Method: call.method, ID: mustJSON("1"), Params: params})
		if err != nil {
			return nil, err
		}
		request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/jsonrpc", port), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := client.Do(request)
		if err != nil {
			return nil, err
		}
		payload, readErr := io.ReadAll(io.LimitReader(response.Body, maxRPCResponseBytes+1))
		response.Body.Close()
		if readErr != nil || len(payload) > maxRPCResponseBytes || !runtimeResponseAccepted(payload) {
			return nil, fmt.Errorf("%s 查询失败", call.method)
		}
		var entries struct {
			Result []struct {
				GID string `json:"gid"`
			} `json:"result"`
		}
		if err := json.Unmarshal(payload, &entries); err != nil {
			return nil, err
		}
		for _, entry := range entries.Result {
			if torrentGIDPattern.MatchString(entry.GID) {
				known[entry.GID] = true
			}
		}
	}
	return known, nil
}
