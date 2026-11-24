package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// 安装向导与应用中心「应用设置」共用同一套字段。字段名必须保持稳定：飞牛把向导值
// 原样变成环境变量，改名等于让已发布版本的输入失效。决策见 docs/adr/ADR-019。
const (
	wizardFieldRPCPort       = "wizard_rpc_port"
	wizardFieldListenPort    = "wizard_listen_port"
	wizardFieldDHTListenPort = "wizard_dht_listen_port"
	wizardFieldRPCSecret     = "wizard_rpc_secret"
	wizardFieldSpecialMode   = "wizard_special_mode"
	wizardFieldDownloadRoot  = "wizard_download_root"
)

const (
	// minRPCSecretLength 与 aria2 默认配置无关，是本项目的下限：RPC 只监听回环地址，
	// 但密钥同时是网关之外唯一的第二道门槛，16 位随机字符已远超暴力破解成本。
	minRPCSecretLength = 16
	maxRPCSecretLength = 64
)

// rpcSecretPattern 限制密钥字符集。aria2.conf 由 shell 逐行写出，含空格、引号、
// 反斜杠或换行的取值会截断行或引入额外选项，因此这类字符必须在入口处拒绝。
var rpcSecretPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func validRPCSecret(secret string) bool {
	if len(secret) < minRPCSecretLength || len(secret) > maxRPCSecretLength {
		return false
	}
	return rpcSecretPattern.MatchString(secret)
}

// wizardValue 读取一个向导字段。空值视为“用户没有填写该字段”，由调用方决定是保留
// 现有取值还是使用默认值；这与飞牛把未填字段以空字符串注入的行为一致。
func wizardValue(name string) (string, bool) {
	value := strings.TrimSpace(os.Getenv(name))
	return value, value != ""
}

func generateRPCSecret() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("生成 RPC 密钥失败: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

// applyWizardInput 把安装向导或应用设置提交的取值写入持久配置。
//
// 它刻意复用常规配置路径的全部校验：端口范围与互斥、下载目录是否位于飞牛共享目录、
// 归档目录派生与权限探测。差异只有两点：安装阶段（initial）允许在配置文件不存在时
// 按默认值创建；下载目录允许尚不存在，但必须能在已授权目录内安全创建。
//
// 返回的 changed 表示配置或密钥是否真的发生了变化。调用方据此决定要不要重启服务：
// 用户在设置页原样保存一次不应该打断正在进行的下载。
func applyWizardInput(configPath, secretPath string, initial bool) ([]string, bool, error) {
	var notes []string

	value, existed, err := loadWizardBase(configPath, initial)
	if err != nil {
		return nil, false, err
	}

	if raw, ok := wizardValue(wizardFieldRPCPort); ok {
		port, err := parseWizardPort(raw, "RPC 监听端口")
		if err != nil {
			return nil, false, err
		}
		value.RPCPort = port
	}
	if raw, ok := wizardValue(wizardFieldListenPort); ok {
		port, err := parseWizardPort(raw, "BT 监听端口")
		if err != nil {
			return nil, false, err
		}
		value.ListenPort = port
	}
	if raw, ok := wizardValue(wizardFieldDHTListenPort); ok {
		port, err := parseWizardPort(raw, "DHT 监听端口")
		if err != nil {
			return nil, false, err
		}
		value.DHTListenPort = port
	} else if value.DHTListenPort == 0 {
		// 未单独给出 DHT 端口时与 BT 监听端口保持一致，避免落到随机取值。
		value.DHTListenPort = value.ListenPort
	}
	if value.RPCPort == value.ListenPort {
		return nil, false, errors.New("RPC 监听端口不能与 BT 监听端口相同")
	}
	if raw, ok := wizardValue(wizardFieldSpecialMode); ok {
		switch raw {
		case "move", "off":
			value.SpecialMode = raw
		case "keep":
			// 应用设置表单的默认选项：留空或显式“保持当前设置”都不改动归档开关。
		default:
			return nil, false, errors.New("下载完成后归档只能选择开启、关闭或保持当前设置")
		}
	}
	if raw, ok := wizardValue(wizardFieldDownloadRoot); ok {
		if !filepath.IsAbs(raw) {
			return nil, false, errors.New("下载文件夹必须是绝对路径，例如 /vol1/1000/downloads")
		}
		value.DownloadRoot = filepath.Clean(raw)
	}

	approved, err := approvedRoots(value.AuthorizedRoots)
	if err != nil {
		return nil, false, err
	}
	// 下载目录必须先落地再走 prepareConfig：归档目录恒定派生为「下载目录/completed」，
	// 上级目录不存在会让归档被判定为不可用而静默关闭。
	if err := ensureWizardDownloadRoot(value.DownloadRoot, approved); err != nil {
		return nil, false, err
	}
	if err := prepareConfig(&value); err != nil {
		return nil, false, describeWizardError(value, err)
	}
	// prepareConfig 会按实际情况修正归档开关，这里以修正后的结果为准。
	if value.DownloadRoot != "" {
		resolved, resolveErr := resolveDirectory(value.DownloadRoot)
		if resolveErr != nil || !downloadRootAllowed(resolved, approved) {
			return nil, false, errors.New("下载文件夹必须位于飞牛数据卷内")
		}
	}
	if value.AuthorizedRoots == nil {
		// 用空数组而不是 null 落盘，保持与安装脚本写出的形态一致。
		value.AuthorizedRoots = []string{}
	}

	// 密钥先校验再落盘配置文件：密钥不合法时不应留下“配置已改、重启后仍连不上”的中间态。
	submittedSecret, secretProvided := wizardValue(wizardFieldRPCSecret)
	if secretProvided && !validRPCSecret(submittedSecret) {
		return nil, false, fmt.Errorf("RPC 密钥需为 16-%d 位字母、数字、点、下划线或连字符", maxRPCSecretLength)
	}

	configChanged, err := writeWizardConfig(configPath, value)
	if err != nil {
		return nil, false, err
	}
	if !existed {
		notes = append(notes, "已按向导取值创建配置文件")
	}

	secretChanged, secretNote, err := applyWizardSecret(secretPath, submittedSecret)
	if err != nil {
		return nil, false, err
	}
	if secretNote != "" {
		notes = append(notes, secretNote)
	}

	notes = append(notes, fmt.Sprintf(
		"当前配置：RPC 端口 %d，BT 监听端口 %d，DHT 监听端口 %d，自动归档 %s，下载目录 %s",
		value.RPCPort, value.ListenPort, value.DHTListenPort, wizardArchiveLabel(value.SpecialMode), value.DownloadRoot,
	))
	if secretChanged {
		notes = append(notes, "RPC 密钥已更新，客户端需要使用新密钥重新连接")
	}
	return notes, configChanged || secretChanged, nil
}

// loadWizardBase 读取现有配置作为基础。配置损坏时走启动自愈同一套修复逻辑，
// 保证“能改设置”不会被一个与本字段无关的历史坏值彻底堵死。
func loadWizardBase(configPath string, initial bool) (config, bool, error) {
	data, err := os.ReadFile(configPath)
	switch {
	case err == nil:
		value, _, parseErr := parseConfigData(data, nil)
		if parseErr == nil {
			return value, true, nil
		}
		repaired, repairNotes, repairErr := repairStartupConfig(data)
		if repairErr != nil {
			return config{}, true, fmt.Errorf("现有配置无法解析: %v", parseErr)
		}
		for _, note := range repairNotes {
			fmt.Fprintln(os.Stderr, "配置自动修复: "+note)
		}
		return repaired, true, nil
	case errors.Is(err, os.ErrNotExist):
		if !initial {
			return config{}, false, errors.New("配置文件不存在，无法修改设置")
		}
		return defaultWizardConfig(), false, nil
	default:
		return config{}, false, err
	}
}

// defaultWizardConfig 是全新安装的出厂配置，取值与早期 install_callback 写出的
// 默认值保持一致，避免升级前后行为漂移。
func defaultWizardConfig() config {
	value := config{
		RPCPort:           defaultStartupRPCPort,
		ListenPort:        defaultStartupListenPort,
		DHTListenPort:     defaultStartupListenPort,
		DHTMessageTimeout: "10",
		EnableDHT:         true,
		EnableDHT6:        true,
		SpecialMode:       "move",
		AuthorizedRoots:   []string{},
	}
	if roots := sharedRoots(); len(roots) > 0 {
		value.DownloadRoot = roots[0]
	}
	// 空字节表示“所有键都未写入”，据此补齐 Tracker 源与自动更新等默认值。
	applyConfigDefaults(&value, nil)
	value.ConfigVersion = configVersion
	return value
}

func parseWizardPort(raw, label string) (int, error) {
	port, err := strconv.Atoi(raw)
	if err != nil || !validPort(port) {
		return 0, fmt.Errorf("%s必须是 1024-65535 之间的整数", label)
	}
	return port, nil
}

func wizardArchiveLabel(mode string) string {
	if mode == "move" {
		return "开启"
	}
	return "关闭"
}

// describeWizardError 把配置校验的英文内部错误翻译成用户可执行的提示。
func describeWizardError(value config, err error) error {
	message := err.Error()
	switch {
	case strings.Contains(message, "download root is outside approved shared directories"):
		return errors.New("下载文件夹必须位于飞牛数据卷（/vol1、/vol2 等）内")
	case strings.Contains(message, "invalid or conflicting ports"):
		return errors.New("端口取值不合法，或 RPC 端口与 BT 监听端口相同")
	default:
		return err
	}
}

// ensureWizardDownloadRoot 允许用户指定尚不存在的下载子目录：逐级创建并把新建目录
// 交还给上级目录的属主，避免以 root 身份创建出应用用户无法穿越的中间目录。
func ensureWizardDownloadRoot(root string, approved []string) error {
	if root == "" {
		return nil
	}
	info, err := os.Stat(root)
	if err == nil {
		if !info.IsDir() {
			return errors.New("下载文件夹已存在但不是目录")
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var missing []string
	for current := filepath.Clean(root); ; {
		if _, statErr := os.Stat(current); statErr == nil {
			break
		}
		missing = append([]string{current}, missing...)
		parent := filepath.Dir(current)
		if parent == current || parent == "." {
			break
		}
		current = parent
	}
	if len(missing) == 0 {
		return nil
	}
	ancestor := filepath.Dir(missing[0])
	ancestorInfo, statErr := os.Stat(ancestor)
	if statErr != nil || !ancestorInfo.IsDir() {
		return errors.New("下载文件夹的上级目录不存在")
	}
	if !downloadRootAllowed(ancestor, approved) {
		return errors.New("下载文件夹必须位于飞牛数据卷内")
	}
	parentInfo := ancestorInfo
	for _, dir := range missing {
		if mkErr := os.Mkdir(dir, 0o770); mkErr != nil && !errors.Is(mkErr, os.ErrExist) {
			return fmt.Errorf("创建下载文件夹 %s 失败: %v", dir, mkErr)
		}
		// umask 可能削掉组写权限，显式固定为 0770 保证专属用户可以写入。
		_ = os.Chmod(dir, 0o770)
		// 新建目录必须归专属用户所有：在 /vol1/1000 这类由管理员拥有的共享目录下，
		// 若沿用上级目录属主，aria2 依然写不进去。
		if uid, gid, ok := appUserIDs(); ok && os.Geteuid() == 0 {
			if err := chownDirectory(dir, uid, gid); err != nil {
				return fmt.Errorf("设置下载文件夹属主失败: %v", err)
			}
		} else {
			adoptDirectoryOwnership(dir, parentInfo)
		}
		if created, statErr := os.Stat(dir); statErr == nil {
			parentInfo = created
		}
	}
	return nil
}

// applyWizardSecret 处理 RPC 密钥。未填写时保持现有密钥不动；密钥文件缺失时补一个
// 随机值，保证首次安装不需要用户必须理解“RPC 密钥”是什么也能正常启动。
//
// 飞牛会把上次提交的密钥原样回填到应用设置表单并随表单再次提交，因此“提交了密钥”
// 不等于“密钥变了”。只有与磁盘内容不同才视为变更，否则一次原样保存就会触发重启。
func applyWizardSecret(secretPath, submitted string) (bool, string, error) {
	if secretPath == "" {
		return false, "", nil
	}
	if submitted != "" {
		if !validRPCSecret(submitted) {
			return false, "", fmt.Errorf("RPC 密钥需为 16-%d 位字母、数字、点、下划线或连字符", maxRPCSecretLength)
		}
		if existing, err := os.ReadFile(secretPath); err == nil && strings.TrimSpace(string(existing)) == submitted {
			return false, "", nil
		}
		if err := writeSecretFile(secretPath, submitted); err != nil {
			return false, "", err
		}
		return true, "", nil
	}
	if _, err := os.Stat(secretPath); err == nil {
		return false, "", nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, "", err
	}
	generated, err := generateRPCSecret()
	if err != nil {
		return false, "", err
	}
	if err := writeSecretFile(secretPath, generated); err != nil {
		return false, "", err
	}
	return false, "未填写 RPC 密钥，已自动生成随机密钥", nil
}

func writeSecretFile(path, secret string) error {
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, []byte(secret), 0o600); err != nil {
		return err
	}
	if err := os.Chmod(temporary, 0o600); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

// writeWizardConfig 原子写出配置，返回内容是否真的发生变化。取值完全一致时不重写文件，
// 调用方据此跳过重启，避免用户原样保存一次就打断正在进行的下载。
func writeWizardConfig(path string, value config) (bool, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	if previous, err := os.ReadFile(path); err == nil {
		if strings.TrimRight(string(previous), "\r\n") == strings.TrimRight(string(data), "\n") {
			return false, nil
		}
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return false, err
	}
	if err := os.Chmod(temporary, 0o600); err != nil {
		_ = os.Remove(temporary)
		return false, err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return false, err
	}
	return true, nil
}
