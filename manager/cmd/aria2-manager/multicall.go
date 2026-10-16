package main

import (
	"encoding/json"
	"errors"
	"strings"
)

// 本文件处理 system.multicall 的受控转发。AriaNg 的任务详情页与重试按钮都用它把
// tellStatus/getPeers/getOption 合并成一次请求；网关此前只放行 aria2. 前缀，这类请求被
// 整体判为非法，表现为详情页打不开、拿不到 peer 列表并反复重试。决策见 ADR-013。

// maxMulticallCalls 限制单次批量调用的数量，避免超大报文把网关和 Core 拖住。
const maxMulticallCalls = 128

// multicallReadOnlyMethods 是嵌套调用白名单，只放读方法，按"默认拒绝"处理。
// 必要性有两条实测依据：Aria2 会执行嵌套的 aria2.changeGlobalOption，改 dir 到 /etc
// 直接返回 OK，顶层 filterGatewayOwnedOptions 的目录守卫对它完全无效；嵌套的 addUri/remove
// 同样会生效，而种子台账只在顶层识别调用，绕过去就会漏掉删除任务时的残留文件清理。
var multicallReadOnlyMethods = map[string]struct{}{
	"aria2.tellStatus":      {},
	"aria2.getPeers":        {},
	"aria2.getFiles":        {},
	"aria2.getServers":      {},
	"aria2.getUris":         {},
	"aria2.tellActive":      {},
	"aria2.tellWaiting":     {},
	"aria2.tellStopped":     {},
	"aria2.getOption":       {},
	"aria2.getGlobalOption": {},
	"aria2.getGlobalStat":   {},
	"aria2.getVersion":      {},
	"aria2.getSessionInfo":  {},
}

// forwardableRPCMethod 判断客户端方法是否允许经网关转发。system.multicall 走
// rewriteMulticallParams 的逐项校验；system.listMethods 只枚举方法名、无参数、无副作用，
// AriaNg 的调试页需要它。其余仍只放行 aria2. 前缀。
func forwardableRPCMethod(method string) bool {
	switch method {
	case "system.multicall", "system.listMethods":
		return true
	}
	return strings.HasPrefix(method, "aria2.")
}

// multicallNestedCall 是 Aria2 嵌套调用的报文结构。
type multicallNestedCall struct {
	MethodName string            `json:"methodName"`
	Params     []json.RawMessage `json:"params"`
}

// rewriteMulticallParams 把客户端的 system.multicall 改写成 Aria2 可执行的形式。
//
// Aria2 对嵌套调用逐个鉴权：顶层 params[0] 放 token 不会被消费，而且该位置要求的是数组，
// 真机实测会直接报 "The parameter at 0 has wrong type"。因此 token 必须注入每个嵌套调用的
// params[0]，顶层不再带 token。客户端可能自带 token（AriaNg 配置了密钥时就会带），必须先剥掉
// 再注入真实密钥，否则前端那份空 token 会抢在真实密钥之前被读到，整批调用全部 Unauthorized。
//
// 改写后的调用列表只含只读方法，台账与运行期选项持久化都不会被触发，保持原有语义不变。
func rewriteMulticallParams(request *rpcRequest, secret string) error {
	calls, ok := parseMulticallCalls(request.Params)
	if !ok || len(calls) == 0 || len(calls) > maxMulticallCalls {
		return errors.New("system.multicall 缺少有效的调用列表")
	}
	for i, call := range calls {
		if _, allowed := multicallReadOnlyMethods[call.MethodName]; !allowed {
			return errors.New("system.multicall 嵌套方法不在只读白名单: " + call.MethodName)
		}
		token, err := json.Marshal("token:" + secret)
		if err != nil {
			return err
		}
		calls[i].Params = append([]json.RawMessage{token}, dropClientTokenParam(call.Params)...)
	}
	request.Params = []json.RawMessage{mustJSON(calls)}
	return nil
}

// parseMulticallCalls 解析调用列表。规范形式是 params[0] 为数组（AriaNg 即此形式），
// 同时容忍开头带 token 的变体与把各调用直接平铺在 params 里的写法，统一归一成数组后再校验。
func parseMulticallCalls(params []json.RawMessage) ([]multicallNestedCall, bool) {
	for _, raw := range params {
		var scalar string
		if json.Unmarshal(raw, &scalar) == nil {
			continue
		}
		var calls []multicallNestedCall
		if json.Unmarshal(raw, &calls) == nil {
			return calls, true
		}
	}
	var flat []multicallNestedCall
	if json.Unmarshal(mustJSON(params), &flat) != nil {
		return nil, false
	}
	return flat, true
}

// dropClientTokenParam 去掉客户端自带的 token 参数。
func dropClientTokenParam(params []json.RawMessage) []json.RawMessage {
	if len(params) == 0 {
		return params
	}
	var first string
	if json.Unmarshal(params[0], &first) != nil || !strings.HasPrefix(first, "token:") {
		return params
	}
	return params[1:]
}
