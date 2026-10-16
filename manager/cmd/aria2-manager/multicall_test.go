package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// multicallRequestBody 构造 AriaNg 实际发出的报文：调用数组位于 params[0]，
// 顶层和嵌套都不带 token，因为前端密钥被适配层固定为空。
func multicallRequestBody(calls string) string {
	return `{"jsonrpc":"2.0","id":7,"method":"system.multicall","params":[[` + calls + `]]}`
}

func parseMulticallRequest(t *testing.T, body string) *rpcRequest {
	t.Helper()
	var request rpcRequest
	if err := json.Unmarshal([]byte(body), &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	return &request
}

// nestedCalls 解出改写后发给 Core 的调用列表。
func nestedCalls(t *testing.T, request *rpcRequest) []multicallNestedCall {
	t.Helper()
	if len(request.Params) != 1 {
		t.Fatalf("改写后应只剩一个参数（调用数组），实际 %d 个: %s", len(request.Params), request.Params)
	}
	var calls []multicallNestedCall
	if err := json.Unmarshal(request.Params[0], &calls); err != nil {
		t.Fatalf("调用数组解析失败: %v (%s)", err, request.Params[0])
	}
	return calls
}

func firstParam(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("首参数不是字符串: %s", raw)
	}
	return value
}

func TestRewriteMulticallInjectsTokenPerNestedCall(t *testing.T) {
	request := parseMulticallRequest(t, multicallRequestBody(
		`{"methodName":"aria2.tellStatus","params":["g1",["gid","status"]]},{"methodName":"aria2.getPeers","params":["g1"]}`))
	if err := rewriteMulticallParams(request, "test-secret"); err != nil {
		t.Fatalf("改写失败: %v", err)
	}
	calls := nestedCalls(t, request)
	if len(calls) != 2 {
		t.Fatalf("调用数量应为 2，实际 %d", len(calls))
	}
	for i, call := range calls {
		if len(call.Params) == 0 || firstParam(t, call.Params[0]) != "token:test-secret" {
			t.Fatalf("第 %d 个嵌套调用未在首位注入密钥: %s", i, call.Params)
		}
	}
	// 原有业务参数必须整体后移而不是被覆盖，否则 tellStatus 会丢掉 gid。
	if got := firstParam(t, calls[0].Params[1]); got != "g1" {
		t.Fatalf("tellStatus 的 gid 被改写: %s", got)
	}
	if len(calls[1].Params) != 2 {
		t.Fatalf("getPeers 参数数量应为 2，实际 %d", len(calls[1].Params))
	}
}

func TestRewriteMulticallDropsClientTokenBeforeInjecting(t *testing.T) {
	request := parseMulticallRequest(t, multicallRequestBody(
		`{"methodName":"aria2.getGlobalStat","params":["token:stale","token:other"]}`))
	if err := rewriteMulticallParams(request, "test-secret"); err != nil {
		t.Fatalf("改写失败: %v", err)
	}
	calls := nestedCalls(t, request)
	if len(calls[0].Params) != 2 {
		t.Fatalf("客户端自带的 token 未剥净: %s", calls[0].Params)
	}
	if got := firstParam(t, calls[0].Params[0]); !strings.HasPrefix(got, "token:test-secret") {
		t.Fatalf("首位不是真实密钥: %s", got)
	}
	if got := firstParam(t, calls[0].Params[1]); !strings.HasPrefix(got, "token:other") {
		t.Fatalf("非首位的伪造 token 应作为普通参数保留: %s", got)
	}
}

func TestRewriteMulticallRejectsNonReadOnlyNestedMethods(t *testing.T) {
	// changeGlobalOption 是真机实测过的目录守卫绕过路径，addUri/remove 会绕过种子台账。
	for _, method := range []string{
		"aria2.changeGlobalOption", "aria2.changeOption", "aria2.addUri", "aria2.addTorrent",
		"aria2.remove", "aria2.forceRemove", "aria2.purgeDownloadResult", "aria2.saveSession",
		"aria2.shutdown", "system.multicall", "system.listMethods", "aria2.unknownMethod",
	} {
		request := parseMulticallRequest(t, multicallRequestBody(`{"methodName":"`+method+`","params":[]}`))
		err := rewriteMulticallParams(request, "test-secret")
		if err == nil {
			t.Fatalf("嵌套的 %s 必须被拒绝", method)
		}
		if !strings.Contains(err.Error(), method) {
			t.Fatalf("拒绝原因未点明方法名: %v", err)
		}
	}
}

func TestRewriteMulticallRejectsMalformedLists(t *testing.T) {
	for name, body := range map[string]string{
		"空数组":  `{"jsonrpc":"2.0","id":1,"method":"system.multicall","params":[[]]}`,
		"无参数":  `{"jsonrpc":"2.0","id":2,"method":"system.multicall","params":[]}`,
		"缺方法名": `{"jsonrpc":"2.0","id":3,"method":"system.multicall","params":[{"params":[]}]}`,
		"非对象项": `{"jsonrpc":"2.0","id":4,"method":"system.multicall","params":[["aria2.getVersion"]]}`,
	} {
		request := parseMulticallRequest(t, body)
		if err := rewriteMulticallParams(request, "test-secret"); err == nil {
			t.Fatalf("%s 应当被拒绝", name)
		}
	}
}

func TestRewriteMulticallCapsCallCount(t *testing.T) {
	var builder strings.Builder
	for i := 0; i <= maxMulticallCalls; i++ {
		if i > 0 {
			builder.WriteByte(',')
		}
		fmt.Fprintf(&builder, `{"methodName":"aria2.getVersion","params":[]}`)
	}
	request := parseMulticallRequest(t, multicallRequestBody(builder.String()))
	if err := rewriteMulticallParams(request, "test-secret"); err == nil {
		t.Fatalf("超过 %d 个调用应当被拒绝", maxMulticallCalls)
	}
}

func TestRewriteMulticallAcceptsFlatShape(t *testing.T) {
	// 少数客户端把各调用直接平铺在 params 里，Core 不接受这种形状，网关需归一后再转发。
	request := parseMulticallRequest(t,
		`{"jsonrpc":"2.0","id":5,"method":"system.multicall","params":[{"methodName":"aria2.getVersion","params":[]}]}`)
	if err := rewriteMulticallParams(request, "test-secret"); err != nil {
		t.Fatalf("平铺形式应被接受: %v", err)
	}
	calls := nestedCalls(t, request)
	if len(calls) != 1 || calls[0].MethodName != "aria2.getVersion" {
		t.Fatalf("归一结果不符: %s", request.Params[0])
	}
}

func TestForwardableRPCMethod(t *testing.T) {
	for _, method := range []string{"aria2.tellStatus", "system.multicall", "system.listMethods"} {
		if !forwardableRPCMethod(method) {
			t.Fatalf("%s 应当可以转发", method)
		}
	}
	for _, method := range []string{"system.multicallX", "aria2x.foo", "", "foo.listMethods"} {
		if forwardableRPCMethod(method) {
			t.Fatalf("%q 不应被放行", method)
		}
	}
}

// proxyThroughGateway 用假 Core 跑一次真实的 proxyRPC，回传 Core 收到的报文。
func proxyThroughGateway(t *testing.T, secret string, body string) (*httptest.ResponseRecorder, string, int) {
	t.Helper()
	var forwarded string
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received, _ := io.ReadAll(r.Body)
		forwarded = string(received)
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":7,"result":[["OK"]]}`)
	}))
	defer core.Close()
	port := strings.TrimPrefix(core.URL, "http://127.0.0.1:")
	parsed := 0
	if _, err := fmt.Sscanf(port, "%d", &parsed); err != nil {
		t.Fatal(err)
	}
	handler := proxyRPC(parsed, secret, nil, nil, &configStore{cfg: config{RPCPort: parsed}})
	request := httptest.NewRequest(http.MethodPost, "/jsonrpc", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response, forwarded, parsed
}

func TestProxyRPCForwardsMulticallForTaskDetailPage(t *testing.T) {
	body := multicallRequestBody(`{"methodName":"aria2.tellStatus","params":["g1",["gid","status"]]},{"methodName":"aria2.getPeers","params":["g1"]}`)
	response, forwarded, _ := proxyThroughGateway(t, "test-secret", body)
	if response.Code != http.StatusOK {
		t.Fatalf("任务详情页的批量查询应被放行，实际 %d: %s", response.Code, response.Body.String())
	}
	var upstream rpcRequest
	if err := json.Unmarshal([]byte(forwarded), &upstream); err != nil {
		t.Fatalf("Core 收到的报文无法解析: %v (%s)", err, forwarded)
	}
	if upstream.Method != "system.multicall" {
		t.Fatalf("方法名被改写: %s", upstream.Method)
	}
	if strings.Contains(forwarded, "token:stale") {
		t.Fatalf("顶层不应带客户端 token")
	}
	for _, call := range nestedCalls(t, &upstream) {
		if len(call.Params) == 0 || firstParam(t, call.Params[0]) != "token:test-secret" {
			t.Fatalf("转发给 Core 的嵌套调用未注入密钥: %s", call.Params)
		}
	}
}

func TestProxyRPCRejectsNestedDirEscape(t *testing.T) {
	// 真机实测 Core 会接受嵌套的 changeGlobalOption dir=/etc，必须在网关拦掉。
	body := multicallRequestBody(`{"methodName":"aria2.changeGlobalOption","params":[{"dir":"/etc"}]}`)
	response, forwarded, _ := proxyThroughGateway(t, "test-secret", body)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("嵌套目录越权应返回 400，实际 %d", response.Code)
	}
	if strings.Contains(forwarded, "/etc") {
		t.Fatalf("越权参数不得转发给 Core: %s", forwarded)
	}
}

func TestProxyRPCForwardsListMethodsWithoutToken(t *testing.T) {
	response, forwarded, _ := proxyThroughGateway(t, "test-secret",
		`{"jsonrpc":"2.0","id":8,"method":"system.listMethods"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("system.listMethods 应被放行，实际 %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(forwarded, `"params":[]`) {
		t.Fatalf("params 应补成空数组而不是 null: %s", forwarded)
	}
	if strings.Contains(forwarded, "test-secret") {
		t.Fatalf("system.listMethods 无需密钥，不应注入: %s", forwarded)
	}
}
