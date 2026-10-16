package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxTrackerListBytes = 256 << 10
	maxTrackerCount     = 4096
	gatewayPrefix       = "/app/aria2"
	// maxRPCResponseBytes 限制单次 RPC 响应大小，防止极端任务量拖垮网关内存。
	maxRPCResponseBytes = 32 << 20
)

var trackerSources = []string{
	"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best.txt",
	"https://trackerslist.com/all_aria2.txt",
	"https://trackers.p3terx.com/all_aria2.txt",
}

type config struct {
	RPCPort           int      `json:"rpc_port"`
	ListenPort        int      `json:"listen_port"`
	DHTListenPort     int      `json:"dht_listen_port"`
	DHTMessageTimeout string   `json:"dht_message_timeout"`
	EnableDHT         bool     `json:"enable_dht"`
	EnableDHT6        bool     `json:"enable_dht6"`
	GatewayUser       string   `json:"gateway_user"`
	GatewayRole       string   `json:"gateway_role"`
	DownloadRoot      string   `json:"download_root"`
	ArchiveEnabled    bool     `json:"archive_enabled"`
	ArchiveRoot       string   `json:"archive_root"`
	SpecialMode       string   `json:"special_mode"`
	AuthorizedRoots   []string `json:"authorized_roots"`
	TrackerSource     string   `json:"tracker_source"`
	TrackerAutoUpdate bool     `json:"tracker_auto_update"`
	// ConfigVersion 标记配置由哪个版本的安装包写过，升级迁移据此幂等执行。
	ConfigVersion int `json:"config_version"`
}

type rpcRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	ID      json.RawMessage   `json:"id"`
	Params  []json.RawMessage `json:"params"`
}

type configStore struct {
	mu   sync.Mutex
	path string
	cfg  config
}

func validPort(port int) bool { return port >= 1024 && port <= 65535 }

// decodeConfig 是所有配置读取入口的唯一解析路径。环境变量、标准输入、文件描述符和
// 直接读文件必须得到同一份有效配置，否则前台显示、启动生成的 aria2.conf 与持久化配置
// 会各自补齐默认值，出现“界面看到的不是重启后生效的”这类分叉。
func decodeConfig(data []byte) (config, error) {
	cfg, _, err := parseConfigData(data, nil)
	return cfg, err
}

// parseConfigData 解析并校验配置。migrate 非空时先执行升级迁移，并把中文变更说明回传给
// 调用方写日志；常规读取路径不传，保证历史配置不会被悄悄改写（见 ADR-011）。
func parseConfigData(data []byte, targetVersion *int) (config, []string, error) {
	var value config
	if err := json.Unmarshal(data, &value); err != nil {
		return config{}, nil, err
	}
	var changes []string
	if targetVersion != nil {
		changes = migrateLegacyConfig(data, &value)
		value.ConfigVersion = *targetVersion
	}
	if value.DHTMessageTimeout == "" {
		value.DHTMessageTimeout = "10"
	}
	if value.DHTListenPort == 0 {
		value.DHTListenPort = value.ListenPort
	}
	if value.TrackerSource == "" {
		value.TrackerSource = trackerSources[0]
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) == nil {
		if _, ok := raw["tracker_auto_update"]; !ok {
			value.TrackerAutoUpdate = true
		}
	}
	if !validPort(value.RPCPort) || !validPort(value.ListenPort) || !validPort(value.DHTListenPort) || value.RPCPort == value.ListenPort {
		return config{}, nil, errors.New("invalid or conflicting ports")
	}
	if err := prepareConfig(&value); err != nil {
		return config{}, nil, err
	}
	return value, changes, nil
}

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	return decodeConfig(data)
}

// readConfigBytes 只取出配置原始字节，不做解析，供启动自愈在解析失败时复用。
func readConfigBytes(fd int) ([]byte, error) {
	f := os.NewFile(uintptr(fd), "config-fd")
	if f == nil {
		return nil, errors.New("invalid config fd")
	}
	return io.ReadAll(f)
}

func loadConfigFD(fd int) (config, error) {
	data, err := readConfigBytes(fd)
	if err != nil {
		return config{}, err
	}
	return decodeConfig(data)
}

var volumeRootPattern = regexp.MustCompile(`^/vol[0-9]+(?:/|$)`)

func containsPath(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func resolveDirectory(candidate string) (string, error) {
	if strings.ContainsAny(candidate, "\x00\r\n\\\"") {
		return "", errors.New("directory contains unsupported characters")
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil || !filepath.IsAbs(resolved) {
		return "", errors.New("directory cannot be resolved")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("directory is unavailable")
	}
	return resolved, nil
}

// sharedRoots 给出本机当前可用的下载根候选：飞牛授权的数据共享目录在前，应用私有
// 目录兜底在后。顺序有意义，normalizeConfig 与 effectiveDownloadRoot 都取 roots[0]
// 作为默认值，因此私有目录只在没有任何数据共享目录可用时才可能被选中。
func sharedRoots() []string {
	roots := dataShareRoots()
	// 私有目录只在它已经存在时参与候选；创建动作留给启动自愈，读路径不顺手写盘。
	if private := privateDownloadRoot(); private != "" {
		if _, dup := seenIn(roots, private); !dup {
			roots = append(roots, private)
		}
	}
	return roots
}

// dataShareRoots 只列出飞牛生命周期上下文给到的真实数据共享目录，不含应用私有兜底目录。
// 自愈路径用它判断"这台机器到底还有没有共享目录可用"，因此不能把兜底目录算进去。
func dataShareRoots() []string {
	var roots []string
	for _, root := range strings.Split(os.Getenv("TRIM_DATA_SHARE_PATHS"), ":") {
		if resolved, err := resolveDirectory(root); err == nil {
			roots = append(roots, resolved)
		}
	}
	return roots
}

func normalizeAuthorizedRoots(candidates []string) ([]string, error) {
	// 用空切片而不是 nil，否则清空授权根会持久化成 JSON null。
	roots := make([]string, 0, len(candidates))
	seen := make(map[string]struct{})
	for _, candidate := range candidates {
		resolved, err := resolveDirectory(candidate)
		if err != nil {
			return nil, err
		}
		// fnOS user data volumes are the only extra root class. The SDK grants the
		// app access, while this check prevents configuration from naming system paths.
		if !volumeRootPattern.MatchString(resolved) {
			return nil, errors.New("authorized root is not on an fnOS data volume")
		}
		if _, exists := seen[resolved]; !exists {
			seen[resolved] = struct{}{}
			roots = append(roots, resolved)
		}
	}
	return roots, nil
}

func approvedRoots(candidates []string) ([]string, error) {
	approved := sharedRoots()
	authorized, err := normalizeAuthorizedRoots(candidates)
	if err != nil {
		return nil, err
	}
	approved = append(approved, authorized...)
	return approved, nil
}

func allowedDownloadRoot(candidate string, approvedRoots []string) bool {
	resolved, err := resolveDirectory(candidate)
	if err != nil {
		return false
	}
	for _, root := range approvedRoots {
		if containsPath(root, resolved) {
			return true
		}
	}
	return false
}

// 目录被授权并不等于 aria2 用户拥有实际写权限；在保存配置前用临时文件验证。
func writableDirectory(candidate string) bool {
	resolved, err := resolveDirectory(candidate)
	if err != nil {
		return false
	}
	temporary, err := os.CreateTemp(resolved, ".aria2-write-test-*")
	if err != nil {
		return false
	}
	name := temporary.Name()
	if _, err := temporary.Write([]byte("aria2")); err != nil {
		temporary.Close()
		_ = os.Remove(name)
		return false
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(name)
		return false
	}
	return os.Remove(name) == nil
}

func effectiveDownloadRoot(value config) (string, error) {
	if value.DownloadRoot != "" {
		return resolveDirectory(value.DownloadRoot)
	}
	roots := sharedRoots()
	if len(roots) == 0 {
		return "", errors.New("no shared download directory is available")
	}
	return roots[0], nil
}

// defaultArchiveDirName 是“下载完成后归档”的固定归档目录名，恒定位于下载根目录下。
// 归档目录不再由用户配置，始终是 <download_root>/completed。
const defaultArchiveDirName = "completed"

// normalizeConfig 维护归档相关默认值：下载根默认取飞牛授权的数据共享目录，
// 归档目录恒定派生为“下载根/completed”，不再由用户单独指定；自动归档默认为“是”。
// 历史配置里 special_mode=move 与 archive_enabled=false 并存会导致静默失效，因此这里
// 始终以 special_mode 为唯一事实源派生 archive_enabled。
func normalizeConfig(value *config) {
	if value.DownloadRoot == "" {
		if roots := sharedRoots(); len(roots) > 0 {
			value.DownloadRoot = roots[0]
		}
	}
	// 归档目录是下载目录的派生值：无论历史配置或前台写过什么，都强制回到
	// “下载目录/completed”，保证下载完成后文件总会被搬到下载目录下的 completed，
	// 用户不需要（也无法）再手工指定一个归档路径。没有下载目录时清空，
	// 避免出现“开关为是但没有可用归档目录”的悬空状态。
	if value.DownloadRoot != "" {
		value.ArchiveRoot = filepath.Join(value.DownloadRoot, defaultArchiveDirName)
	} else {
		value.ArchiveRoot = ""
	}
	if value.SpecialMode == "" {
		value.SpecialMode = "move"
	}
	// 只有在既没有下载目录也无法派生时才强制关闭，避免出现“开关为是但任务不动”。
	if value.ArchiveRoot == "" {
		value.SpecialMode = "off"
	}
	value.ArchiveEnabled = value.SpecialMode == "move"
}

// syncArchiveRootWithDownload 让用户改下载目录时，默认归档目录跟着新目录走。
// 仅当归档目录仍是“旧下载目录/completed”这一默认值（或为空）时才重新派生，
// 用户显式自定义过的归档目录保持不变，避免下载与归档目录长期分叉。
func syncArchiveRootWithDownload(value *config, previous config) {
	if value.DownloadRoot == "" || previous.DownloadRoot == "" || value.DownloadRoot == previous.DownloadRoot {
		return
	}
	previousDefault := filepath.Clean(filepath.Join(previous.DownloadRoot, defaultArchiveDirName))
	current := filepath.Clean(value.ArchiveRoot)
	if value.ArchiveRoot == "" {
		current = ""
	}
	if current == "" || current == previousDefault || value.ArchiveRoot == previous.ArchiveRoot {
		value.ArchiveRoot = filepath.Join(value.DownloadRoot, defaultArchiveDirName)
	}
}

// configVersion 是本安装包认可的配置文件版本，升级迁移完成后写入 config.json。
const configVersion = 1

// legacyPackageDownloadRoots 列出历史安装包写过、但本版已不再作为默认值的下载目录。
// 用户在界面能选到的目录必然位于共享目录或授权根内，因此命中这里的取值只可能是机器写入，
// 迁移它不会覆盖任何手工选择。
func legacyPackageDownloadRoots() []string {
	var roots []string
	if pkgvar := strings.TrimSpace(os.Getenv("TRIM_PKGVAR")); pkgvar != "" {
		roots = append(roots, filepath.Join(filepath.Clean(pkgvar), "downloads"))
	}
	return roots
}

// migrateLegacyConfig 在升级与覆盖安装时把历史配置收敛到本版默认：补写缺失的键、
// 替换机器写出的旧默认值、修正早期版本产生的非法取值。它只处理“用户不可能选出来的值”，
// 例如归档开关这类明确的用户意愿一律保持原样，因此可以安全地重复执行。决策见 ADR-011。
func migrateLegacyConfig(data []byte, value *config) []string {
	var changes []string
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return changes
	}
	has := func(key string) bool {
		_, ok := raw[key]
		return ok
	}
	// 早期版本把 DHT 超时写成带单位的 "10s"，aria2 只接受纯秒数。
	if value.DHTMessageTimeout == "10s" {
		value.DHTMessageTimeout = "10"
		changes = append(changes, "DHT 消息超时时间 10s 修正为 10")
	}
	for _, legacy := range legacyPackageDownloadRoots() {
		if value.DownloadRoot == "" {
			break
		}
		if filepath.Clean(value.DownloadRoot) != filepath.Clean(legacy) {
			continue
		}
		// 应用私有目录现在也是候选根之一，命中自己时不要写一条"从 A 更新为 A"的假变更。
		if roots := sharedRoots(); len(roots) > 0 && filepath.Clean(roots[0]) != filepath.Clean(legacy) {
			changes = append(changes, fmt.Sprintf("默认下载目录 %s 更新为 %s", legacy, roots[0]))
			value.DownloadRoot = roots[0]
		}
		break
	}
	// 归档目录仍是历史默认下载目录下的 completed 时跟随到新下载目录；
	// 用户自定义的归档目录名不是 completed，天然不会被这条规则命中。
	if value.DownloadRoot != "" && value.ArchiveRoot != "" {
		current := filepath.Clean(value.ArchiveRoot)
		want := filepath.Join(filepath.Clean(value.DownloadRoot), defaultArchiveDirName)
		if current != want && filepath.Base(current) == defaultArchiveDirName {
			for _, legacy := range legacyPackageDownloadRoots() {
				if current == filepath.Clean(filepath.Join(legacy, defaultArchiveDirName)) {
					changes = append(changes, fmt.Sprintf("默认归档目录 %s 更新为 %s", current, want))
					value.ArchiveRoot = want
					break
				}
			}
		}
	}
	for _, key := range []string{"download_root", "archive_root", "archive_enabled", "special_mode", "authorized_roots", "config_version"} {
		if !has(key) {
			changes = append(changes, "补齐缺失配置项 "+key)
		}
	}
	return changes
}

// migrateConfigFile 原子重写 config.json，使磁盘内容与运行期归一化结果一致。
// 返回值是中文变更说明，供 --migrate-config 输出与升级日志使用。
func migrateConfigFile(path string) (config, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, nil, err
	}
	target := configVersion
	value, changes, err := parseConfigData(data, &target)
	if err != nil {
		return config{}, nil, err
	}
	updated, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return config{}, nil, err
	}
	if string(updated) == strings.TrimRight(string(data), "\r\n") {
		return value, nil, nil
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, append(updated, '\n'), 0o600); err != nil {
		return config{}, nil, err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return config{}, nil, err
	}
	return value, changes, nil
}

// prepareConfig 是所有配置入口的统一处理顺序：补齐默认值、按需自动创建默认归档目录、
// 最后校验端口与目录权限。安装、启动和前台保存必须得到同一份有效配置，否则会出现
// “界面显示已开启归档但运行态没有目录”的分叉。
func prepareConfig(value *config) error {
	normalizeConfig(value)
	if err := ensureConfiguredDirectories(value); err != nil {
		return err
	}
	return validateConfig(*value)
}

// ensureConfiguredDirectories 落实“归档目录不存在就自动创建”。归档目录必须落在飞牛
// 授权的共享目录内：目录已存在时直接校验边界；目录尚未创建时用其父目录判断边界，并且
// 只补建最后一级，避免以 root 身份创建出专属应用用户无法写入的中间目录。
//
// 归档目录创建失败是“尽力而为”的：它只影响“下载完成后归档”这一可选行为，不应阻塞
// 用户保存下载目录等基本配置。失败时记日志并静默关闭归档，避免给前端一个会让用户
// 误以为“下载都不能用”的红色错误。
func ensureConfiguredDirectories(value *config) error {
	if !value.ArchiveEnabled || value.ArchiveRoot == "" {
		return nil
	}
	approved, err := approvedRoots(value.AuthorizedRoots)
	if err != nil {
		return err
	}
	if !disableArchiveOnFailure(value, approved) {
		return nil
	}
	return nil
}

// disableArchiveOnFailure 在归档目录不可用（不存在但父目录不可写、已存在但不是目录、
// 越界）时关闭归档开关、清空归档目录并记日志；返回 true 表示已降级为关闭。
func disableArchiveOnFailure(value *config, approved []string) bool {
	reason := ""
	info, statErr := os.Stat(value.ArchiveRoot)
	switch {
	case statErr == nil:
		if !info.IsDir() {
			reason = "归档路径已存在但不是目录"
		} else if !allowedDownloadRoot(value.ArchiveRoot, approved) {
			reason = "归档目录不在已授权的飞牛共享目录内"
		}
	case errors.Is(statErr, os.ErrNotExist):
		parent := filepath.Dir(value.ArchiveRoot)
		parentInfo, parentErr := os.Stat(parent)
		switch {
		case parentErr != nil || !parentInfo.IsDir():
			reason = "归档目录的上级目录不存在"
		case !allowedDownloadRoot(parent, approved):
			reason = "归档目录不在已授权的飞牛共享目录内"
		default:
			if err := os.Mkdir(value.ArchiveRoot, 0o770); err != nil {
				reason = "自动创建归档目录失败: " + err.Error()
				break
			}
			// umask 可能削掉组写权限，这里显式固定为 0770，保证专属用户可以写入。
			if err := os.Chmod(value.ArchiveRoot, 0o770); err != nil {
				log.Printf("归档目录权限调整失败: %v", err)
			}
			adoptDirectoryOwnership(value.ArchiveRoot, parentInfo)
			log.Printf("已自动创建归档目录: %s", value.ArchiveRoot)
			return false
		}
	default:
		reason = "无法访问归档目录: " + statErr.Error()
	}
	if reason == "" {
		return false
	}
	log.Printf("归档目录不可用，已关闭「下载完成后归档」（%s）：%s", reason, value.ArchiveRoot)
	value.ArchiveEnabled = false
	value.SpecialMode = "off"
	value.ArchiveRoot = ""
	return true
}

// adoptDirectoryOwnership 处理生命周期脚本以 root 运行时补建目录的情况：新目录属主
// 跟随父目录，保证常驻的专属应用用户可以写入。
func adoptDirectoryOwnership(created string, parentInfo os.FileInfo) {
	if os.Geteuid() != 0 {
		return
	}
	uid, gid, ok := directoryOwner(parentInfo)
	if !ok {
		return
	}
	if err := chownDirectory(created, uid, gid); err != nil {
		log.Printf("归档目录属主调整失败: %v", err)
	}
}
func validateConfig(value config) error {
	if value.DHTMessageTimeout == "" {
		value.DHTMessageTimeout = "10"
	}
	if value.DHTListenPort == 0 {
		value.DHTListenPort = value.ListenPort
	}
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(value.DHTMessageTimeout) {
		return errors.New("invalid DHT message timeout")
	}
	if value.SpecialMode != "" && value.SpecialMode != "off" && value.SpecialMode != "move" {
		return errors.New("invalid special mode")
	}
	if value.TrackerSource != "" {
		u, err := url.ParseRequestURI(value.TrackerSource)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
			return errors.New("invalid tracker source")
		}
	}
	if !validPort(value.RPCPort) || !validPort(value.ListenPort) || !validPort(value.DHTListenPort) || value.RPCPort == value.ListenPort {
		return errors.New("invalid or conflicting ports")
	}
	approvedRoots, err := approvedRoots(value.AuthorizedRoots)
	if err != nil {
		return err
	}
	if value.DownloadRoot != "" {
		if !allowedDownloadRoot(value.DownloadRoot, approvedRoots) {
			return errors.New("download root is outside approved shared directories")
		}
		if !writableDirectory(value.DownloadRoot) {
			return errors.New("aria2 用户没有下载目录写入权限，请在飞牛文件管理器中授权该目录")
		}
	}
	if value.ArchiveEnabled {
		if value.ArchiveRoot == "" || !allowedDownloadRoot(value.ArchiveRoot, approvedRoots) {
			return errors.New("archive root is outside approved shared directories")
		}
		if !writableDirectory(value.ArchiveRoot) {
			return errors.New("aria2 用户没有归档目录写入权限")
		}
	}
	return nil
}

func portAvailable(network, address string) bool {
	listener, err := net.Listen(network, address)
	if err != nil {
		return false
	}
	return listener.Close() == nil
}

// taskScope 给出任务清理与种子台账共用的配置快照和授权根集合。
// 调用方拿到的是副本，读不到后续保存，也不会与写路径共享同一份切片。
func (store *configStore) taskScope() (config, []string) {
	store.mu.Lock()
	cfg := store.cfg
	roots := append([]string(nil), cfg.AuthorizedRoots...)
	store.mu.Unlock()
	// 除授权目录外还要覆盖本机可用根（数据共享目录与应用私有兜底目录），否则自愈把
	// 下载目录落到私有目录之后，删除任务会因为"越界"而拒绝清理已下载文件。
	roots = append(roots, sharedRoots()...)
	return cfg, roots
}

func (store *configStore) update(value config) error {
	authorizedRoots, err := normalizeAuthorizedRoots(value.AuthorizedRoots)
	if err != nil {
		return err
	}
	value.AuthorizedRoots = authorizedRoots
	syncArchiveRootWithDownload(&value, store.cfg)
	if err := prepareConfig(&value); err != nil {
		return err
	}
	// 保存阶段只校验格式和端口互斥。不同 fnOS 生命周期/网络命名空间下，
	// 普通应用用户的临时 bind 探测可能误报“端口被占用”；真正启动时由
	// ensureAvailablePorts 在同一生命周期上下文选择可用端口并持久化。
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temporary := store.path + ".tmp"
	if err := os.WriteFile(temporary, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(temporary, store.path); err != nil {
		return err
	}
	// 运行时配置位于 /run；同步写入持久配置，避免受控重启时被旧文件覆盖。
	if persistentDir := os.Getenv("TRIM_PKGETC"); persistentDir != "" {
		persistentPath := filepath.Join(persistentDir, "config.json")
		if persistentPath != store.path {
			ptmp := persistentPath + ".tmp"
			if err := os.WriteFile(ptmp, data, 0600); err != nil {
				return err
			}
			if err := os.Rename(ptmp, persistentPath); err != nil {
				return err
			}
		}
	}
	store.cfg = value
	return nil
}

func nextAvailablePort(start int, network string, used map[int]bool) int {
	limit := start + 256
	if limit > 65535 {
		limit = 65535
	}
	for port := start; port <= limit; port++ {
		if used[port] || !portAvailable(network, fmt.Sprintf("0.0.0.0:%d", port)) {
			continue
		}
		return port
	}
	return 0
}

func ensureAvailablePorts(path string, cfg config) (config, error) {
	used := map[int]bool{}
	if !portAvailable("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.RPCPort)) {
		port := nextAvailablePort(cfg.RPCPort+1, "tcp", used)
		if port != 0 {
			cfg.RPCPort = port
		}
	}
	used[cfg.RPCPort] = true
	if !portAvailable("tcp", fmt.Sprintf("0.0.0.0:%d", cfg.ListenPort)) || !portAvailable("udp", fmt.Sprintf("0.0.0.0:%d", cfg.ListenPort)) || cfg.ListenPort == cfg.RPCPort {
		port := nextAvailablePort(cfg.ListenPort+1, "tcp", used)
		if port != 0 && portAvailable("udp", fmt.Sprintf("0.0.0.0:%d", port)) {
			cfg.ListenPort = port
		}
	}
	used[cfg.ListenPort] = true
	if !portAvailable("udp", fmt.Sprintf("0.0.0.0:%d", cfg.DHTListenPort)) {
		port := nextAvailablePort(cfg.DHTListenPort+1, "udp", used)
		if port != 0 {
			cfg.DHTListenPort = port
		}
	}
	if path == "" {
		return cfg, nil
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return cfg, err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return cfg, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func requireGatewayUser(next http.Handler, expectedUser string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Header.Get("X-Trim-Username")
		if user == "" || (expectedUser != "" && user != expectedUser) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireGatewayAdmin(next http.Handler, expectedUser string) http.Handler {
	return requireGatewayUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin := strings.EqualFold(r.Header.Get("X-Trim-Isadmin"), "true") || r.Header.Get("X-Trim-Isadmin") == "1"
		if !isAdmin {
			http.Error(w, "administrator required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}), expectedUser)
}

// fnOS 网关在不同版本中可能将应用前缀保留或剥离后再转发到 Unix Socket。
// 两种形式都归一化到管理器内部路由，避免桌面 iframe 收到 404。
func normalizeGatewayPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == gatewayPrefix || strings.HasPrefix(r.URL.Path, gatewayPrefix+"/") {
			r2 := r.Clone(r.Context())
			r2.URL.Path = strings.TrimPrefix(r.URL.Path, gatewayPrefix)
			if r2.URL.Path == "" {
				r2.URL.Path = "/"
			}
			next.ServeHTTP(w, r2)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isTrackerURI(value string) bool {
	u, err := url.ParseRequestURI(value)
	if err != nil || u.User != nil || u.Host == "" || u.Fragment != "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "udp":
		return true
	default:
		return false
	}
}

func parseTrackers(data []byte) ([]string, error) {
	unique := make(map[string]struct{})
	var trackers []string
	for _, value := range strings.FieldsFunc(string(data), func(r rune) bool { return r == '\n' || r == '\r' || r == ',' }) {
		value = strings.TrimSpace(value)
		if !isTrackerURI(value) {
			continue
		}
		if _, seen := unique[value]; seen {
			continue
		}
		unique[value] = struct{}{}
		trackers = append(trackers, value)
		if len(trackers) > maxTrackerCount {
			return nil, errors.New("too many tracker URIs")
		}
	}
	if len(trackers) == 0 {
		return nil, errors.New("no valid tracker URI")
	}
	return trackers, nil
}

func fetchTrackers(client *http.Client, sources []string) ([]string, string, error) {
	var failures []string
	for _, source := range sources {
		request, err := http.NewRequest(http.MethodGet, source, nil)
		if err != nil || request.URL.Scheme != "https" || request.URL.User != nil || request.URL.Host == "" {
			failures = append(failures, "invalid source")
			continue
		}
		response, err := client.Do(request)
		if err != nil {
			failures = append(failures, "request failed")
			continue
		}
		if response.StatusCode != http.StatusOK || response.ContentLength > maxTrackerListBytes {
			response.Body.Close()
			failures = append(failures, "invalid response")
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(response.Body, maxTrackerListBytes+1))
		response.Body.Close()
		if readErr != nil || len(data) > maxTrackerListBytes {
			failures = append(failures, "response too large")
			continue
		}
		trackers, parseErr := parseTrackers(data)
		if parseErr == nil {
			return trackers, source, nil
		}
		failures = append(failures, "no valid tracker")
	}
	return nil, "", fmt.Errorf("all tracker sources failed: %s", strings.Join(failures, ", "))
}

func writeTrackers(path string, trackers []string) error {
	data := []byte(strings.Join(trackers, "\n") + "\n")
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func changeGlobalTrackers(client *http.Client, port int, secret string, trackers []string) error {
	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  "aria2.changeGlobalOption",
		ID:      json.RawMessage("1"),
		Params: []json.RawMessage{
			mustJSON("token:" + secret),
			mustJSON(map[string]string{"bt-tracker": strings.Join(trackers, ",")}),
		},
	})
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/jsonrpc", port), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("aria2 returned HTTP %d", response.StatusCode)
	}
	var result struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 32<<10)).Decode(&result); err != nil {
		return err
	}
	if len(result.Error) != 0 && string(result.Error) != "null" {
		return errors.New("aria2 rejected tracker update")
	}
	return nil
}

func refreshConfiguredTrackers(store *configStore, client *http.Client, secret, trackerPath string) (int, string, error) {
	store.mu.Lock()
	current := store.cfg
	store.mu.Unlock()
	sources := trackerSources
	if current.TrackerSource != "" {
		sources = append([]string{current.TrackerSource}, trackerSources...)
	}
	trackers, source, err := fetchTrackers(client, sources)
	if err != nil {
		return 0, "", err
	}
	if err := changeGlobalTrackers(client, current.RPCPort, secret, trackers); err != nil {
		return 0, "", err
	}
	if err := writeTrackers(trackerPath, trackers); err != nil {
		return 0, "", err
	}
	return len(trackers), source, nil
}

func changeGlobalOption(port int, secret string, options map[string]string) error {
	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", Method: "aria2.changeGlobalOption", ID: json.RawMessage("1"), Params: []json.RawMessage{mustJSON("token:" + secret), mustJSON(options)}})
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/jsonrpc", port), bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("aria2 returned HTTP %d", response.StatusCode)
	}
	var result struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 32<<10)).Decode(&result); err != nil {
		return err
	}
	if len(result.Error) != 0 && string(result.Error) != "null" {
		return errors.New("aria2 rejected option update")
	}
	return nil
}

// 运行期全局选项的落盘上限，防止前台反复提交把覆盖文件撑爆。
const (
	maxRuntimeOverrideKeys        = 64
	maxRuntimeOverrideKeyLength   = 64
	maxRuntimeOverrideValueLength = 512
)

var runtimeOverrideKeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// runtimeOverrideDenyKeys 由 config.json、生命周期脚本或安全策略独占。放行这些键会让前台
// 绕过端口自动切换、授权目录校验和任务回调，历史上正是这类绕过造成过无法启动。
var runtimeOverrideDenyKeys = map[string]struct{}{
	"bt-listen-port":                {},
	"bt-metadata-save-directory":    {},
	"bt-tracker":                    {},
	"conf-path":                     {},
	"dht-file-path":                 {},
	"dht-file-path6":                {},
	"dht-listen-port":               {},
	"dht-message-timeout":           {},
	"dir":                           {},
	"enable-dht":                    {},
	"enable-dht6":                   {},
	"enable-rpc":                    {},
	"input-file":                    {},
	"listen-port":                   {},
	"log":                           {},
	"save-session":                  {},
	"save-session-interval":         {},
	"save-upload-metadata-file-dir": {},
}

var runtimeOverrideDenyPrefixes = []string{"rpc-", "on-"}

func runtimeOverrideDeniedKey(key string) bool {
	if _, ok := runtimeOverrideDenyKeys[key]; ok {
		return true
	}
	for _, prefix := range runtimeOverrideDenyPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// runtimeOverrides 持久化用户在 AriaNg 里通过 aria2.changeGlobalOption 修改、又不属于
// config.json 管辖的全局选项（决策见 ADR-012）。Aria2 只在内存中保留这类改动，重启后会被重新生成的
// aria2.conf 覆盖，表现为“改完重启就回退”，因此必须在网关侧落盘。
type runtimeOverrides struct {
	mu   sync.Mutex
	path string
	data map[string]string
}

func newRuntimeOverrides(path string) *runtimeOverrides {
	store := &runtimeOverrides{path: path, data: make(map[string]string)}
	store.load()
	return store
}

func (o *runtimeOverrides) load() {
	if o.path == "" {
		return
	}
	data, err := os.ReadFile(o.path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("读取运行期选项失败: %v", err)
		}
		return
	}
	parsed, err := parseRuntimeOverrideLines(data)
	if err != nil {
		log.Printf("运行期选项文件不合规，已忽略: %v", err)
		return
	}
	o.data = parsed
}

// parseRuntimeOverrideLines 按行解析 key=value。遇到不合规的行整体报错而不是静默跳过，
// 避免坏行在下一次写入时被原样带回 aria2.conf。
func parseRuntimeOverrideLines(data []byte) (map[string]string, error) {
	parsed := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 4096), 4096)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			continue
		}
		key, value, ok := splitRuntimeOverrideLine(line)
		if !ok {
			return nil, fmt.Errorf("第 %d 行不是合法的 aria2 选项", lineNumber)
		}
		if runtimeOverrideDeniedKey(key) || len(parsed) >= maxRuntimeOverrideKeys {
			continue
		}
		parsed[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return parsed, nil
}

func splitRuntimeOverrideLine(line string) (string, string, bool) {
	index := strings.Index(line, "=")
	if index <= 0 {
		return "", "", false
	}
	key, value := line[:index], line[index+1:]
	if len(key) > maxRuntimeOverrideKeyLength || !runtimeOverrideKeyPattern.MatchString(key) {
		return "", "", false
	}
	if value == "" || len(value) > maxRuntimeOverrideValueLength || strings.ContainsAny(value, "\x00\r\n") {
		return "", "", false
	}
	return key, value, true
}

// merge 记录一次成功的 aria2.changeGlobalOption，返回实际落盘的选项名。
func (o *runtimeOverrides) merge(values map[string]json.RawMessage) []string {
	if o.path == "" || len(values) == 0 {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	var applied []string
	for rawKey, rawValue := range values {
		key := strings.TrimSpace(rawKey)
		if len(key) > maxRuntimeOverrideKeyLength || !runtimeOverrideKeyPattern.MatchString(key) || runtimeOverrideDeniedKey(key) {
			continue
		}
		value, ok := runtimeOverrideValue(rawValue)
		if !ok {
			continue
		}
		if _, exists := o.data[key]; !exists && len(o.data) >= maxRuntimeOverrideKeys {
			log.Printf("运行期选项数量已达上限 %d，忽略 %s", maxRuntimeOverrideKeys, key)
			continue
		}
		o.data[key] = value
		applied = append(applied, key)
	}
	if len(applied) == 0 {
		return nil
	}
	if err := o.writeLocked(); err != nil {
		log.Printf("写入运行期选项失败: %v", err)
		return nil
	}
	sort.Strings(applied)
	return applied
}

func (o *runtimeOverrides) snapshot() map[string]string {
	o.mu.Lock()
	defer o.mu.Unlock()
	snapshot := make(map[string]string, len(o.data))
	for key, value := range o.data {
		snapshot[key] = value
	}
	return snapshot
}

// reset 清空运行期选项，使下一次生成的 aria2.conf 回到只含随包默认值的状态。
func (o *runtimeOverrides) reset() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.path != "" {
		if err := os.Remove(o.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	o.data = make(map[string]string)
	return nil
}

// writeLocked 以 0600 原子写出，行按字典序排列，保证同一份状态每次落盘字节一致。
func (o *runtimeOverrides) writeLocked() error {
	lines := make([]string, 0, len(o.data))
	for key, value := range o.data {
		lines = append(lines, key+"="+value)
	}
	sort.Strings(lines)
	payload := strings.Join(lines, "\n")
	if len(lines) > 0 {
		payload += "\n"
	}
	temporary := o.path + ".tmp"
	if err := os.WriteFile(temporary, []byte(payload), 0o600); err != nil {
		return err
	}
	if err := os.Rename(temporary, o.path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

// runtimeOverrideValue 保留客户端提交的原始字面量：字符串取出内容，数字与布尔按原样落盘；
// null、对象、数组和空值都不是合法的 Aria2 选项取值。
func runtimeOverrideValue(raw json.RawMessage) (string, bool) {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "", false
	}
	if strings.HasPrefix(text, `"`) {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", false
		}
		if value == "" || len(value) > maxRuntimeOverrideValueLength || strings.ContainsAny(value, "\x00\r\n") {
			return "", false
		}
		return value, true
	}
	if text == "true" || text == "false" {
		return text, true
	}
	if text == "null" || strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
		return "", false
	}
	// 剩下的只可能是数字字面量，其余内容一律拒绝，避免把任意文本写进 aria2.conf。
	if !numericLiteralPattern.MatchString(text) {
		return "", false
	}
	return text, true
}

// numericLiteralPattern 只接受 JSON 数字字面量，拒绝任何可能被解释为选项的任意文本。
var numericLiteralPattern = regexp.MustCompile("^[0-9+\\.\\-eE]+$")

// changeGlobalOptionParams 取出本次要修改的选项对象。RPC 密钥由网关注入并置于首位，因此
// Aria2 实际读取的选项对象恒为客户端参数的第一个。去更靠后的参数里找会与 Aria2 的语义
// 分叉：Aria2 会忽略它并照常返回 OK，我们却把从未生效的取值落盘，重启后才凭空生效。
func changeGlobalOptionParams(params []json.RawMessage) map[string]json.RawMessage {
	if len(params) == 0 {
		return nil
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(params[0], &values); err != nil || len(values) == 0 {
		return nil
	}
	return values
}

// filterGatewayOwnedOptions 把网关专属选项从待转发集合里摘掉，返回保留项与被拒项（按键名排序）。
// 这些键不能转发给 Aria2：aria2.changeGlobalOption 允许在运行期直接改写 dir，一旦放行，
// 前台一次保存就能把下载目录挪出 authorized_roots，整个目录权限与归档模型都被绕过。
// 落盘侧本来就用同一份拒绝清单，转发侧必须保持一致，否则会出现“内存已生效、配置不认”的分叉。
func filterGatewayOwnedOptions(values map[string]json.RawMessage) (map[string]json.RawMessage, []string) {
	kept := make(map[string]json.RawMessage, len(values))
	var denied []string
	for key, value := range values {
		if runtimeOverrideDeniedKey(strings.TrimSpace(key)) {
			denied = append(denied, key)
			continue
		}
		kept[key] = value
	}
	sort.Strings(denied)
	return kept, denied
}

// runtimeResponseAccepted 判断 Aria2 是否真正接受了调用：JSON-RPC 失败时 error 字段非空。
func runtimeResponseAccepted(payload []byte) bool {
	var response struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return false
	}
	return len(response.Error) == 0 || string(response.Error) == "null"
}

func mustJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func resolvePotentialPath(path string) (string, error) {
	if !filepath.IsAbs(path) || strings.ContainsAny(path, "\x00\r\n\\\"") {
		return "", errors.New("path is not absolute")
	}
	remaining := ""
	candidate := filepath.Clean(path)
	for {
		resolved, err := filepath.EvalSymlinks(candidate)
		if err == nil {
			if remaining == "" {
				return resolved, nil
			}
			return filepath.Join(resolved, remaining), nil
		}
		parent := filepath.Dir(candidate)
		if parent == candidate || parent == "." {
			return "", errors.New("path has no existing ancestor")
		}
		remaining = filepath.Join(filepath.Base(candidate), remaining)
		candidate = parent
	}
}

func taskContainedRoot(path string, roots []string) (string, string, error) {
	resolved := path
	if _, err := os.Lstat(path); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return "", "", errors.New("path cannot be inspected")
		}
		var resolveErr error
		resolved, resolveErr = resolvePotentialPath(path)
		if resolveErr != nil || !filepath.IsAbs(resolved) {
			return "", "", errors.New("path cannot be resolved")
		}
	} else {
		resolvedPath, err := filepath.EvalSymlinks(path)
		if err != nil || !filepath.IsAbs(resolvedPath) {
			return "", "", errors.New("path cannot be resolved")
		}
		resolved = resolvedPath
	}
	for _, root := range roots {
		resolvedRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		relative, err := filepath.Rel(resolvedRoot, resolved)
		if err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
			return resolvedRoot, relative, nil
		}
	}
	return "", "", errors.New("path is outside approved roots")
}

func removeTaskControl(path string, roots []string) error {
	if _, _, err := taskContainedRoot(path, roots); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("unsafe control file")
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func pruneEmptyParents(path, root string) {
	for directory := filepath.Dir(path); directory != root; directory = filepath.Dir(directory) {
		info, err := os.Lstat(directory)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return
		}
		entries, err := os.ReadDir(directory)
		if err != nil || len(entries) != 0 || os.Remove(directory) != nil {
			return
		}
	}
}

func removeTaskPath(path string, roots []string) error {
	root, _, err := taskContainedRoot(path, roots)
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		for _, suffix := range []string{".aria2", ".torrent"} {
			if err := removeTaskControl(path+suffix, roots); err != nil {
				return err
			}
		}
		return nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("unsafe download path")
	}
	if info.IsDir() {
		if err := filepath.WalkDir(path, func(p string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return errors.New("symlink in download path")
			}
			return nil
		}); err != nil {
			return err
		}
		if err := os.RemoveAll(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	} else if info.Mode().IsRegular() {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	} else {
		return errors.New("unsafe download path")
	}
	for _, suffix := range []string{".aria2", ".torrent"} {
		if err := removeTaskControl(path+suffix, roots); err != nil {
			return err
		}
	}
	pruneEmptyParents(path, root)
	return nil
}

// rpcTransport 复用长连接到回环 Aria2。默认 Transport 的 MaxIdleConnsPerHost=2，
// 在任务很多、AriaNg 每秒轮询 tellActive/tellWaiting/tellStopped 时会不断重建连接，
// 叠加大量任务的响应体后表现为界面卡死。这里放宽空闲连接池并延长空闲超时。
func rpcTransport() *http.Transport {
	return &http.Transport{
		Proxy:               nil,
		MaxIdleConns:        128,
		MaxIdleConnsPerHost: 128,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	}
}

func proxyRPC(port int, secret string, overrides *runtimeOverrides, ledger *torrentLedger, store *configStore) http.Handler {
	// 超时放宽到 60s：任务数很多时 tellStopped 的响应体可以到数 MB，
	// 30s 上限在慢盘或高负载机器上会截断响应，前台拿到残缺 JSON 后反复重试。
	client := &http.Client{Timeout: 60 * time.Second, Transport: rpcTransport()}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// 与 Aria2 默认 rpc-max-request-size=10M 对齐，避免前台上传较大种子文件被截断。
		body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
		if err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		var request rpcRequest
		if err := json.Unmarshal(body, &request); err != nil || !forwardableRPCMethod(request.Method) {
			http.Error(w, "invalid RPC request", http.StatusBadRequest)
			return
		}
		// 令牌注入会改变参数下标，必须先取出本次要修改的选项对象。
		var changedOptions map[string]json.RawMessage
		if request.Method == "aria2.changeGlobalOption" && len(request.Params) > 0 {
			changedOptions = changeGlobalOptionParams(request.Params)
			if len(changedOptions) > 0 {
				// 网关专属选项一律不转发：Aria2 会在运行期直接接受 dir，
				// 一次保存就能把下载目录挪出 authorized_roots，绕过目录授权与归档。
				if kept, denied := filterGatewayOwnedOptions(changedOptions); len(denied) > 0 {
					request.Params[0] = mustJSON(kept)
					log.Printf("已拒绝网关专属选项: %s", strings.Join(denied, ", "))
				}
			}
		}
		// 台账只认客户端原始参数：密钥注入会让参数整体后移，必须在此之前识别本次调用。
		// 改写后的 system.multicall 只含只读方法，台账变更恒为空，既不漏记账也不漏清理。
		mutation := ledgerMutationFor(request.Method, request.Params)
		// 只有 aria2. 方法需要顶层注入密钥；system. 方法（当前只有 listMethods）无需密钥，原样转发。
		switch {
		case request.Method == "system.multicall":
			// 批量调用的密钥逐个注入，顶层不带 token，原因见 rewriteMulticallParams。
			if mcErr := rewriteMulticallParams(&request, secret); mcErr != nil {
				log.Printf("拒绝 system.multicall: %v", mcErr)
				http.Error(w, "invalid system.multicall request", http.StatusBadRequest)
				return
			}
		case strings.HasPrefix(request.Method, "aria2."):
			token, _ := json.Marshal("token:" + secret)
			request.Params = append([]json.RawMessage{token}, request.Params...)
		}
		if request.Params == nil {
			// system.listMethods 可以不带参数，补成空数组，避免序列化成 null 后被 Core 拒绝。
			request.Params = []json.RawMessage{}
		}
		body, err = json.Marshal(request)
		if err != nil {
			http.Error(w, "invalid RPC request", http.StatusBadRequest)
			return
		}
		upstream, err := http.NewRequestWithContext(r.Context(), http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/jsonrpc", port), bytes.NewReader(body))
		if err != nil {
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
			return
		}
		upstream.Header.Set("Content-Type", "application/json")
		response, err := client.Do(upstream)
		if err != nil {
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
			return
		}
		defer response.Body.Close()
		// 任务很多时 tellStopped 响应可能很大。此前直接截断到 8 MiB 会让前台拿到
		// 残缺 JSON 并反复重试，表现为界面卡顿；现在整体读取并显式拒绝超限响应。
		payload, err := io.ReadAll(io.LimitReader(response.Body, maxRPCResponseBytes+1))
		if err != nil {
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
			return
		}
		if len(payload) > maxRPCResponseBytes {
			log.Printf("RPC response exceeds %d bytes, rejected", maxRPCResponseBytes)
			http.Error(w, "upstream response too large", http.StatusBadGateway)
			return
		}
		// 只有上游真正接受时才落盘，避免把 aria2 拒绝过的取值反复写进 aria2.conf。
		if len(changedOptions) > 0 && response.StatusCode == http.StatusOK && runtimeResponseAccepted(payload) {
			if applied := overrides.merge(changedOptions); len(applied) > 0 {
				log.Printf("已持久化运行期选项: %s", strings.Join(applied, ", "))
			}
		}
		// 上游确认接受之后才记账/清账，避免把 Aria2 拒绝过的任务当成已处理。
		if response.StatusCode == http.StatusOK && runtimeResponseAccepted(payload) {
			applyLedgerMutation(ledger, port, secret, mutation, payload, store.taskScope)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		w.WriteHeader(response.StatusCode)
		_, _ = w.Write(payload)
	})
}

func main() {
	var socket, configPath, secretPath, uiRoot, trackerPath, lifecycle, runtimeOptionsPath, torrentLedgerPath string
	var dropUser string
	var configFD, secretFD int
	var configStdin bool
	var configEnv string
	var secretEnv string
	var printDownloadRoot, ensurePorts, migrateConfig bool
	// 运行期覆盖文件与种子元数据台账写在持久化目录，卸载选择清空配置时由 uninstall_callback 一并删除。
	defaultRuntimeOptions := ""
	defaultTorrentLedger := ""
	if pkgvar := strings.TrimSpace(os.Getenv("TRIM_PKGVAR")); pkgvar != "" {
		defaultRuntimeOptions = filepath.Join(pkgvar, "aria2-overrides.conf")
		defaultTorrentLedger = filepath.Join(pkgvar, "aria2-torrents.tsv")
	}
	flag.StringVar(&socket, "socket", os.Getenv("TRIM_APPDEST")+"/aria2.sock", "Unix socket")
	flag.StringVar(&configPath, "config", os.Getenv("TRIM_PKGETC")+"/config.json", "configuration file")
	flag.StringVar(&secretPath, "secret", os.Getenv("TRIM_PKGETC")+"/rpc.secret", "RPC secret file")
	flag.StringVar(&uiRoot, "ui", os.Getenv("TRIM_APPDEST")+"/www", "AriaNg static directory")
	flag.StringVar(&trackerPath, "trackers", os.Getenv("TRIM_PKGVAR")+"/trackers.txt", "persisted tracker list")
	flag.StringVar(&lifecycle, "lifecycle", os.Getenv("TRIM_APPCTL"), "lifecycle script")
	flag.StringVar(&dropUser, "drop-user", "", "启动监听前降权到指定用户")
	flag.IntVar(&configFD, "config-fd", -1, "从已打开的文件描述符读取配置")
	flag.IntVar(&secretFD, "secret-fd", -1, "从已打开的文件描述符读取 RPC 密钥")
	flag.BoolVar(&configStdin, "config-stdin", false, "从标准输入读取配置")
	flag.StringVar(&configEnv, "config-env", "", "从环境变量读取配置 JSON")
	flag.StringVar(&secretEnv, "secret-env", "", "从环境变量读取 RPC 密钥")
	flag.BoolVar(&printDownloadRoot, "print-download-root", false, "validate configuration and print the effective download root")
	flag.BoolVar(&ensurePorts, "ensure-ports", false, "自动切换被占用的端口并持久化")
	flag.BoolVar(&migrateConfig, "migrate-config", false, "按本版默认值迁移已有 config.json 后退出")
	flag.StringVar(&runtimeOptionsPath, "runtime-options", defaultRuntimeOptions, "AriaNg 运行期全局选项的持久化文件")
	flag.StringVar(&torrentLedgerPath, "torrent-ledger", defaultTorrentLedger, "RPC 上传种子元数据的 gid 台账文件")
	flag.Parse()

	// 升级与覆盖安装时先把历史配置收敛到本版默认：补写缺失键、替换机器写出的旧取值。
	// 这一步必须发生在常规读取之前，否则旧文件里的非法值会让进程直接退出，迁移没机会执行。
	if migrateConfig {
		_, changes, migrateErr := migrateConfigFile(configPath)
		if migrateErr != nil {
			log.Fatal(migrateErr)
		}
		for _, change := range changes {
			fmt.Println(change)
		}
		return
	}
	configReadPath := configPath
	// 启动探针（--ensure-ports、--print-download-root）由 fpk/cmd/main 以 set -eu 调用，
	// 这里只要非零退出整个应用就起不来；而界面起不来时用户根本没有入口改回失效的目录。
	// 所以启动路径先拿到原始字节，解析或校验失败时执行启动自愈；其余模式保持严格报错，
	// 避免前台保存把用户写错的路径静默改掉。设计见 docs/adr/ADR-015-startup-self-repair.md。
	startupProbe := ensurePorts || printDownloadRoot
	var rawConfig []byte
	var err error
	switch {
	case configEnv != "":
		rawConfig = []byte(os.Getenv(configEnv))
	case configStdin:
		rawConfig, err = io.ReadAll(os.Stdin)
	case configFD >= 0:
		configReadPath = fmt.Sprintf("/proc/self/fd/%d", configFD)
		rawConfig, err = readConfigBytes(configFD)
	default:
		rawConfig, err = os.ReadFile(configReadPath)
	}
	var cfg config
	if err == nil {
		cfg, err = decodeConfig(rawConfig)
	}
	if err != nil && startupProbe {
		repaired, notes, repairErr := repairStartupConfig(rawConfig)
		if repairErr != nil {
			log.Fatalf("启动配置自动修复失败: %v（原始错误: %v）", repairErr, err)
		}
		logRepairNotes(notes)
		cfg = repaired
		err = nil
	}
	secretReadPath := secretPath
	var secretBytes []byte
	if secretEnv != "" {
		secretBytes = []byte(os.Getenv(secretEnv))
	} else if secretFD >= 0 {
		secretReadPath = fmt.Sprintf("/proc/self/fd/%d", secretFD)
	}
	if err != nil {
		log.Fatal(err)
	}
	if ensurePorts {
		writePath := configPath
		if configEnv != "" || configStdin {
			writePath = ""
		}
		if cfg, err = ensureAvailablePorts(writePath, cfg); err != nil {
			log.Fatal(err)
		}
		if writePath == "" {
			data, marshalErr := json.MarshalIndent(cfg, "", "  ")
			if marshalErr != nil {
				log.Fatal(marshalErr)
			}
			fmt.Println(string(data))
			return
		}
		fmt.Printf("%d %d\n", cfg.RPCPort, cfg.ListenPort)
		return
	}
	if printDownloadRoot {
		root, err := effectiveDownloadRoot(cfg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(root)
		return
	}
	if secretEnv != "" {
		// 已在上方从环境变量读取。
	} else if secretFD >= 0 {
		secretFile := os.NewFile(uintptr(secretFD), "secret-fd")
		if secretFile == nil {
			log.Fatal("invalid secret fd")
		}
		secretBytes, err = io.ReadAll(secretFile)
	} else {
		secretBytes, err = os.ReadFile(secretReadPath)
	}
	if err != nil {
		log.Fatal(err)
	}
	secret := strings.TrimSpace(string(secretBytes))
	if len(secret) < 32 {
		log.Fatal("invalid RPC secret")
	}
	if dropUser != "" {
		args := make([]string, 0, len(os.Args[1:]))
		for _, arg := range os.Args[1:] {
			if arg != "--drop-user" && !strings.HasPrefix(arg, "--drop-user=") {
				args = append(args, arg)
			}
		}
		cmd := exec.Command("setpriv", append([]string{"--reuid=" + dropUser, "--regid=" + dropUser, "--init-groups", os.Args[0]}, args...)...)
		cmd.Env = os.Environ()
		cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := os.MkdirAll(filepath.Dir(socket), 0750); err != nil {
		log.Fatal(err)
	}
	_ = os.Remove(socket)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod(socket, 0660); err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	// 启动即载入已落盘的运行期选项，保证前台读到的是重启后真正生效的取值。
	overrides := newRuntimeOverrides(runtimeOptionsPath)
	// 台账在启动时载入，保证管理器重启后仍能认领此前 RPC 上传的 .torrent/.meta4。
	ledger := newTorrentLedger(torrentLedgerPath)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	store := &configStore{path: configPath, cfg: cfg}
	mux.Handle("/api/v1/config", requireGatewayAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			store.mu.Lock()
			defer store.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(store.cfg)
		case http.MethodPut:
			var next config
			if err := json.NewDecoder(io.LimitReader(r.Body, 32<<10)).Decode(&next); err != nil {
				http.Error(w, "invalid configuration", http.StatusBadRequest)
				return
			}
			store.mu.Lock()
			defer store.mu.Unlock()
			old := store.cfg
			if old.DownloadRoot != next.DownloadRoot && old.RPCPort > 0 {
				if err := changeGlobalOption(old.RPCPort, secret, map[string]string{"dir": next.DownloadRoot}); err != nil {
					http.Error(w, "aria2 download directory update failed", http.StatusBadGateway)
					return
				}
			}
			if err := store.update(next); err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			// 归档与特殊模式由完成回调在每次任务结束时读取配置文件生效，不参与
			// aria2.conf，因此不再触发受控重启，避免打断正在进行的下载。
			if old.RPCPort != next.RPCPort || old.ListenPort != next.ListenPort || old.DHTListenPort != next.DHTListenPort || old.DHTMessageTimeout != next.DHTMessageTimeout || old.EnableDHT != next.EnableDHT || old.EnableDHT6 != next.EnableDHT6 || old.TrackerSource != next.TrackerSource || old.TrackerAutoUpdate != next.TrackerAutoUpdate {
				// 端口只在生命周期脚本重启后由 aria2.conf 生效；延迟执行避免截断当前响应。
				go func() {
					time.Sleep(150 * time.Millisecond)
					cmd := exec.Command(lifecycle, "restart")
					cmd.Env = os.Environ()
					_ = cmd.Start()
				}()
			}
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}), cfg.GatewayUser))
	mux.Handle("/api/v1/restart", requireGatewayAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if lifecycle == "" {
			http.Error(w, "lifecycle unavailable", http.StatusConflict)
			return
		}
		if _, err := os.Stat(lifecycle); err != nil {
			http.Error(w, "lifecycle unavailable", http.StatusConflict)
			return
		}
		go func() {
			time.Sleep(100 * time.Millisecond)
			cmd := exec.Command(lifecycle, "restart")
			cmd.Env = os.Environ()
			_ = cmd.Start()
		}()
		w.WriteHeader(http.StatusAccepted)
	}), cfg.GatewayUser))
	mux.Handle("/api/v1/cleanup", requireGatewayAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			Paths []string `json:"paths"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 128<<10)).Decode(&request); err != nil || len(request.Paths) == 0 || len(request.Paths) > 4096 {
			http.Error(w, "invalid cleanup request", http.StatusBadRequest)
			return
		}
		_, roots := store.taskScope()
		for _, path := range request.Paths {
			if err := removeTaskPath(path, roots); err != nil {
				http.Error(w, "cleanup rejected", http.StatusConflict)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}), cfg.GatewayUser))
	mux.Handle("/jsonrpc", requireGatewayAdmin(proxyRPC(cfg.RPCPort, secret, overrides, ledger, store), cfg.GatewayUser))
	// 前台可查看已经持久化的运行期选项；删除后重启即恢复随包默认值。
	mux.Handle("/api/v1/runtime-options", requireGatewayAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(overrides.snapshot())
		case http.MethodDelete:
			if err := overrides.reset(); err != nil {
				http.Error(w, "清除运行期选项失败", http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}), cfg.GatewayUser))
	trackerClient := &http.Client{
		Timeout:       15 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("redirect is not allowed") },
	}
	mux.Handle("/api/v1/trackers/refresh", requireGatewayAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		count, source, err := refreshConfiguredTrackers(store, trackerClient, secret, trackerPath)
		if err != nil {
			http.Error(w, "tracker refresh failed", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"count": count, "source": source})
	}), cfg.GatewayUser))
	if cfg.TrackerAutoUpdate {
		go func() {
			time.Sleep(6 * time.Second)
			store.mu.Lock()
			initialAutoUpdate := store.cfg.TrackerAutoUpdate
			store.mu.Unlock()
			if initialAutoUpdate {
				if count, source, err := refreshConfiguredTrackers(store, trackerClient, secret, trackerPath); err != nil {
					log.Printf("initial tracker refresh failed: %v", err)
				} else {
					log.Printf("initial tracker refresh ok: %d trackers from %s", count, source)
				}
			}
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				store.mu.Lock()
				autoUpdate := store.cfg.TrackerAutoUpdate
				store.mu.Unlock()
				if !autoUpdate {
					continue
				}
				if count, source, err := refreshConfiguredTrackers(store, trackerClient, secret, trackerPath); err != nil {
					log.Printf("periodic tracker refresh failed: %v", err)
				} else {
					log.Printf("periodic tracker refresh ok: %d trackers from %s", count, source)
				}
			}
		}()
	}
	staticHandler := requireGatewayUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 入口和静态资源禁止复用旧版本，避免升级后继续加载旧语言 bundle。
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		http.FileServer(http.Dir(uiRoot)).ServeHTTP(w, r)
	}), cfg.GatewayUser)
	mux.Handle("/", staticHandler)
	// WriteTimeout 放宽到 60s，与代理侧一致：大量任务时单个 RPC 响应体较大，
	// 30s 会在写出中途被切断，前台表现为界面卡死。
	server := &http.Server{Handler: normalizeGatewayPath(mux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second}
	log.Fatal(server.Serve(listener))
}
