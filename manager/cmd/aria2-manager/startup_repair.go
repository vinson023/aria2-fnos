package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// 启动自愈。设计见 docs/adr/ADR-015-startup-self-repair.md。
//
// config.json 里的下载目录、归档目录和授权目录都是"机器相关"取值：换一台 NAS、
// 卷号不同、共享目录被改名或删除、配置从别的机器拷贝过来，都会让这些路径无法解析。
// resolveDirectory 对不存在的路径直接报错，于是校验失败、--ensure-ports 非零退出，
// 而 fpk/cmd/main 是 set -eu，整条启动当场中止，前台只留下一句"无法启动"。
// 更糟的是此时界面也起不来，用户没有任何入口改回配置，形成死锁。
//
// 因此区分两条路径：前台保存仍然严格，改错了就明确报错，绝不静默搬迁用户数据；
// 启动路径则尽最大努力起来，把机器相关字段回落到当前生命周期上下文里的安全默认，
// 逐条把改动打到日志，让应用先能用、再让用户在界面上重新选定目录。

const (
	// fallbackDownloadDirName 是应用私有下载目录名，位于 $TRIM_PKGVAR 下。
	fallbackDownloadDirName = "downloads"
	// defaultStartupRPCPort 与 defaultStartupListenPort 是自愈可用的出厂端口，口径同 ADR-006。
	defaultStartupRPCPort    = 6800
	defaultStartupListenPort = 6888
)

// privateDownloadRoot 只在目录已经存在时返回它的绝对路径。sharedRoots 会被校验、
// 清理和前台接口反复读取，读取路径不能顺手创建目录。
func privateDownloadRoot() string {
	pkgvar := strings.TrimSpace(os.Getenv("TRIM_PKGVAR"))
	if pkgvar == "" {
		return ""
	}
	resolved, err := resolveDirectory(filepath.Join(pkgvar, fallbackDownloadDirName))
	if err != nil {
		return ""
	}
	return resolved
}

// ensurePrivateDownloadRoot 创建应用私有下载目录，供数据共享目录全部不可用时兜底。
// 生命周期探针通常以 root 运行，新建目录必须交还给应用用户，否则 aria2c 写不进去。
func ensurePrivateDownloadRoot() (string, error) {
	pkgvar := strings.TrimSpace(os.Getenv("TRIM_PKGVAR"))
	if pkgvar == "" {
		return "", errors.New("TRIM_PKGVAR 未设置，无法推导私有下载目录")
	}
	parent, err := resolveDirectory(pkgvar)
	if err != nil {
		return "", fmt.Errorf("应用数据目录不可用: %w", err)
	}
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return "", err
	}
	uid, gid, ok := directoryOwner(parentInfo)
	dir := filepath.Join(parent, fallbackDownloadDirName)
	if err := os.MkdirAll(dir, 0770); err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0770); err != nil {
		return "", err
	}
	if ok {
		if err := chownDirectory(dir, uid, gid); err != nil {
			return "", err
		}
	}
	resolved, err := resolveDirectory(dir)
	if err != nil {
		return "", err
	}
	if !writableDirectory(resolved) {
		return "", fmt.Errorf("私有下载目录 %s 对应用用户不可写", resolved)
	}
	return resolved, nil
}

// validHTTPSTrackerSource 与 validateConfig 的口径保持一致：只允许无凭据的 https 源。
func validHTTPSTrackerSource(candidate string) bool {
	if candidate == "" {
		return true
	}
	u, err := url.ParseRequestURI(candidate)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.Fragment == ""
}

// repairStartupConfig 从一份可能已经不合法的配置里恢复出可启动的配置，第二个返回值是
// 面向运维的中文变更说明，调用方逐条写日志。解析彻底失败时退回出厂默认，因为
// "起不来"比"回到默认目录"代价高得多，而 RPC 密钥、端口等仍可另见持久配置。
func repairStartupConfig(rawData []byte) (config, []string, error) {
	var notes []string
	var value config
	if len(strings.TrimSpace(string(rawData))) == 0 {
		notes = append(notes, "配置为空，已改用出厂默认值")
	} else if err := json.Unmarshal(rawData, &value); err != nil {
		value = config{}
		notes = append(notes, fmt.Sprintf("配置无法解析（%v），已改用出厂默认值", err))
	}

	if !validPort(value.RPCPort) {
		notes = append(notes, fmt.Sprintf("RPC 端口 %d 无效，已改用 %d", value.RPCPort, defaultStartupRPCPort))
		value.RPCPort = defaultStartupRPCPort
	}
	if !validPort(value.ListenPort) || value.ListenPort == value.RPCPort {
		// 出厂 BT 端口可能正好等于用户自定义的 RPC 端口（例如 rpc=6888），此时把 BT
		// 复位到 6888 等于什么都没修，validateConfig 仍会判为端口冲突而拒绝启动。
		// 因此优先保住用户可见的 RPC 端口不动，只在 BT 侧挑一个不冲突的取值；
		// 端口是否真的空闲交给随后的 ensureAvailablePorts 处理。
		repaired := 0
		for _, candidate := range []int{defaultStartupListenPort, defaultStartupListenPort + 1, defaultStartupRPCPort, defaultStartupRPCPort + 1} {
			if validPort(candidate) && candidate != value.RPCPort {
				repaired = candidate
				break
			}
		}
		if repaired == 0 {
			return config{}, notes, fmt.Errorf("RPC 端口 %d 占用了全部出厂端口，无法为 BT 监听选出可用端口", value.RPCPort)
		}
		notes = append(notes, fmt.Sprintf("BT 监听端口 %d 无效或与 RPC 端口 %d 冲突，已改用 %d", value.ListenPort, value.RPCPort, repaired))
		value.ListenPort = repaired
	}
	if !validPort(value.DHTListenPort) {
		value.DHTListenPort = value.ListenPort
	}
	switch value.SpecialMode {
	case "", "off", "move":
	default:
		notes = append(notes, fmt.Sprintf("归档模式 %q 无效，已改用 move", value.SpecialMode))
		value.SpecialMode = "move"
	}
	if !validHTTPSTrackerSource(value.TrackerSource) {
		notes = append(notes, fmt.Sprintf("Tracker 源 %q 不合法，已改用随包默认源", value.TrackerSource))
		value.TrackerSource = ""
	}

	// 授权目录：逐条保留仍然可用的，丢弃无法解析的。整体拒绝会让一个失效目录
	// 连带否决其余可用目录，代价是应用完全起不来。
	approved := sharedRoots()
	kept := make([]string, 0, len(value.AuthorizedRoots))
	var dropped []string
	for _, candidate := range value.AuthorizedRoots {
		resolved, err := resolveDirectory(candidate)
		if err != nil || !volumeRootPattern.MatchString(resolved) {
			dropped = append(dropped, candidate)
			continue
		}
		if _, dup := seenIn(kept, resolved); !dup {
			kept = append(kept, resolved)
		}
	}
	if len(dropped) > 0 {
		notes = append(notes, "以下授权目录在当前机器上不可用，已移出授权列表: "+strings.Join(dropped, "、"))
	}
	value.AuthorizedRoots = kept
	roots := append(append([]string(nil), approved...), kept...)

	// 下载目录：不可解析、越界或不可写时清空，交给 normalizeConfig 重新派生默认值。
	if value.DownloadRoot != "" {
		resolved, err := resolveDirectory(value.DownloadRoot)
		switch {
		case err != nil:
			notes = append(notes, fmt.Sprintf("下载目录 %s 在当前机器上不存在，已改用默认目录", value.DownloadRoot))
			value.DownloadRoot = ""
		case !allowedDownloadRoot(resolved, roots):
			notes = append(notes, fmt.Sprintf("下载目录 %s 不在授权范围内，已改用默认目录", value.DownloadRoot))
			value.DownloadRoot = ""
		case !writableDirectory(resolved):
			notes = append(notes, fmt.Sprintf("下载目录 %s 对 aria2 用户不可写，已改用默认目录", value.DownloadRoot))
			value.DownloadRoot = ""
		default:
			value.DownloadRoot = resolved
		}
	}
	// 归档目录自本版起恒定为「下载目录/completed」，由 normalizeConfig 统一派生，
	// 这里不再区分「默认值」与「用户自定义值」。下载目录失效时先清空归档目录，
	// 待下载目录重新确定后（含私有目录兜底）再统一重新派生。
	if value.DownloadRoot == "" {
		value.ArchiveRoot = ""
	}

	// 兜底目录必须在 normalizeConfig 之前建好：该函数见到空的下载目录会连带把归档判成
	// 关闭，那样即使用户开着"下载完成后归档"，兜底启动后也会静默变成不归档。
	hasDataShare := len(dataShareRoots()) > 0
	fallbackNote := ""
	if value.DownloadRoot == "" && !hasDataShare {
		// 数据共享目录整体缺失（生命周期上下文没给出 TRIM_DATA_SHARE_PATHS，或共享目录
		// 尚未创建）时，退到应用私有数据目录，保证 aria2c 一定有可写的 dir。
		fallback, err := ensurePrivateDownloadRoot()
		if err != nil {
			return config{}, notes, err
		}
		fallbackNote = fmt.Sprintf("没有可用的飞牛数据共享目录，下载目录改用 %s", fallback)
		value.DownloadRoot = fallback
	}
	normalizeConfig(&value)
	if fallbackNote != "" {
		notes = append(notes, fallbackNote)
	} else if !hasDataShare {
		// 兜底目录是历史某次启动建出来的，这次同样是"无共享目录"降级，必须留痕。
		if private := privateDownloadRoot(); private != "" && value.DownloadRoot == private {
			notes = append(notes, fmt.Sprintf("没有可用的飞牛数据共享目录，下载目录沿用应用私有目录 %s；请在设置中重新选择共享目录", value.DownloadRoot))
		}
	}
	if err := prepareConfig(&value); err != nil {
		// 连默认目录都过不了校验，说明这台机器上应用自己的数据目录也不可用；
		// 这种情况不再猜，直接带上原因失败，让日志能指到真正的病灶。
		return config{}, notes, fmt.Errorf("自动修复后仍无法启动: %w", err)
	}
	return value, notes, nil
}

// seenIn 是切片存在性判断的小工具，避免自愈路径引入 map 分配噪声。
func seenIn(items []string, candidate string) (int, bool) {
	for i, item := range items {
		if item == candidate {
			return i, true
		}
	}
	return -1, false
}

// logRepairNotes 把自愈变更逐条写入日志（stderr），由 fpk/cmd/main 归集到
// $TRIM_PKGVAR/startup.log。禁止在此输出 RPC 密钥等敏感取值。
func logRepairNotes(notes []string) {
	for _, note := range notes {
		log.Printf("启动配置自动修复: %s", note)
	}
}
