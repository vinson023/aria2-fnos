package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type config struct {
	ArchiveEnabled  bool     `json:"archive_enabled"`
	ArchiveRoot     string   `json:"archive_root"`
	SpecialMode     string   `json:"special_mode"`
	AuthorizedRoots []string `json:"authorized_roots"`
}

// archiveRequested 兼容历史配置：早期前台只写 special_mode 而未同时写
// archive_enabled，导致“下载完成后归档=是”静默失效。两个信号任一成立即归档。
func (value config) archiveRequested() bool {
	return value.ArchiveEnabled || (value.SpecialMode == "move" && value.ArchiveRoot != "")
}

// archiveSettleDelay 给 Aria2 留出停止做种与收尾时间再移动文件。
// 等待发生在 Aria2 fork 出的回调子进程中，不阻塞下载主进程。
var archiveSettleDelay = 2 * time.Second

// renamePath 是 os.Rename 的可替换入口，便于测试覆盖跨卷回退分支。
var renamePath = os.Rename

func rootsFrom(value string) []string {
	var roots []string
	for _, root := range strings.Split(value, ":") {
		if root == "" {
			continue
		}
		resolved, err := filepath.EvalSymlinks(root)
		if err == nil && filepath.IsAbs(resolved) {
			roots = append(roots, resolved)
		}
	}
	return roots
}

func authorizedRoots(value config) []string {
	roots := make([]string, 0, len(value.AuthorizedRoots))
	for _, root := range value.AuthorizedRoots {
		if !strings.HasPrefix(root, "/vol") || strings.ContainsAny(root, "\x00\r\n\\\"") {
			continue
		}
		resolved, err := filepath.EvalSymlinks(root)
		if err != nil || !filepath.IsAbs(resolved) {
			continue
		}
		info, err := os.Stat(resolved)
		if err == nil && info.IsDir() {
			roots = append(roots, resolved)
		}
	}
	return roots
}

func containedRoot(path string, roots []string) (string, string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || !filepath.IsAbs(resolved) {
		return "", "", errors.New("path cannot be resolved")
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

func findDownloadPath(args []string, roots []string) (string, error) {
	for _, candidate := range args {
		if _, _, err := containedRoot(candidate, roots); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("download path cannot be resolved")
}

// taskFileCount 读取 Aria2 回调第 2 个参数（文件数量），缺失时按单文件处理。
func taskFileCount(args []string) int {
	if len(args) < 2 {
		return 1
	}
	count, err := strconv.Atoi(args[1])
	if err != nil || count < 1 {
		return 1
	}
	return count
}

// taskContainer 对齐上游 move.sh：多文件任务的第一个文件通常位于任务容器目录内，
// 此时归档整个容器；容器就是下载根目录时只归档文件本身，绝不移动下载根目录。
func taskContainer(path string, numFiles int, roots []string) string {
	if numFiles < 2 {
		return path
	}
	parent := filepath.Dir(path)
	root, relative, err := containedRoot(parent, roots)
	if err != nil || parent == root || relative == "." || relative == "" || relative == ".." {
		return path
	}
	return parent
}

func removeRegular(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil || !info.Mode().IsRegular() {
		return err
	}
	return os.Remove(path)
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

func removeDownloadData(path string, roots []string) error {
	root, _, err := containedRoot(path, roots)
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("unsafe download path")
	}
	if info.IsDir() {
		err = filepath.WalkDir(path, func(p string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return errors.New("symlink in download path")
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	for _, suffix := range []string{".aria2", ".torrent"} {
		if err := removeRegular(path + suffix); err != nil {
			return err
		}
	}
	pruneEmptyParents(path, root)
	return nil
}

func clean(path string, roots []string, removeData bool) error {
	if removeData {
		return removeDownloadData(path, roots)
	}
	root, _, err := containedRoot(path, roots)
	if err != nil {
		return err
	}
	for _, suffix := range []string{".aria2", ".torrent"} {
		if err := removeRegular(path + suffix); err != nil {
			return err
		}
	}
	pruneEmptyParents(path, root)
	return nil
}

// splitExtension 只在扩展名较短且不含分隔符时拆分，避免误判目录名中的点。
func splitExtension(base string) (string, string) {
	index := strings.LastIndex(base, ".")
	if index <= 0 {
		return base, ""
	}
	extension := base[index:]
	if len(extension) > 12 || strings.ContainsAny(extension, `/\`) {
		return base, ""
	}
	return base[:index], extension
}

// freeDestination 在目标已存在时追加 " (1)"、" (2)" 序号，避免同名互相覆盖。
func freeDestination(destination string) (string, error) {
	if _, err := os.Lstat(destination); errors.Is(err, fs.ErrNotExist) {
		return destination, nil
	} else if err != nil {
		return "", err
	}
	dir, base := filepath.Split(destination)
	name, extension := splitExtension(base)
	for index := 1; index <= 99; index++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", name, index, extension))
		if _, err := os.Lstat(candidate); errors.Is(err, fs.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", err
		}
	}
	return "", errors.New("no free archive destination name")
}

func isCrossDevice(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) && errno == syscall.Errno(18) {
		return true
	}
	return err != nil && strings.Contains(err.Error(), "cross-device")
}

func copyFile(source, destination string, mode os.FileMode) error {
	from, err := os.Open(source)
	if err != nil {
		return err
	}
	defer from.Close()
	to, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(to, from); err != nil {
		to.Close()
		return err
	}
	if err := to.Close(); err != nil {
		return err
	}
	return os.Chmod(destination, mode.Perm())
}

// copyTree 用于跨挂载点归档；复制阶段的任何错误都不会删除源文件。
func copyTree(source, destination string) error {
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if sourceInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("unsafe archive path")
	}
	if !sourceInfo.IsDir() {
		if err := os.MkdirAll(filepath.Dir(destination), 0750); err != nil {
			return err
		}
		return copyFile(source, destination, sourceInfo.Mode())
	}
	if err := os.MkdirAll(destination, sourceInfo.Mode().Perm()); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("symlink in archive path")
		}
		if err := copyTree(filepath.Join(source, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
			return err
		}
	}
	return os.Chmod(destination, sourceInfo.Mode().Perm())
}

// movePath 优先原子重命名；跨卷时回退为复制后删除源。
func movePath(source, destination string) error {
	err := renamePath(source, destination)
	if err == nil {
		return nil
	}
	if !isCrossDevice(err) {
		return err
	}
	if err := copyTree(source, destination); err != nil {
		_ = os.RemoveAll(destination)
		return err
	}
	return os.RemoveAll(source)
}

// pathWithinRoots 判断目录是否等于或位于某个授权根内。默认归档目录是下载目录下的
// completed，其父目录恰好就是下载根本身，因此这里必须允许“等于根目录”，
// 否则归档会在这一步被误判为越界而静默失败。
func pathWithinRoots(dir string, roots []string) bool {
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil || !filepath.IsAbs(resolvedDir) {
		return false
	}
	for _, root := range roots {
		resolvedRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		relative, err := filepath.Rel(resolvedRoot, resolvedDir)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// ensureArchiveDir 自愈归档目录：默认归档目录是派生路径，用户可能手动删除，
// 旧配置升级时也可能尚未创建。这里只在授权共享目录内补建最后一级，
// 不递归创建，避免以 root 建出专属用户无法写入的中间目录。
func ensureArchiveDir(candidate string, roots []string) error {
	info, err := os.Stat(candidate)
	if err == nil {
		if !info.IsDir() {
			return errors.New("archive root is not a directory")
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	parent := filepath.Dir(candidate)
	if !pathWithinRoots(parent, roots) {
		return errors.New("archive root is outside approved roots")
	}
	if err := os.Mkdir(candidate, 0o770); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	log.Printf("已自动重建归档目录: %s", candidate)
	return nil
}

func archive(path string, value config, roots []string) error {
	if !value.archiveRequested() {
		return nil
	}
	if value.ArchiveRoot == "" {
		return errors.New("archive directory is not selected")
	}
	sourceRoot, relative, err := containedRoot(path, roots)
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("unsafe archive path")
	}
	if err := ensureArchiveDir(value.ArchiveRoot, roots); err != nil {
		return err
	}
	if _, _, err := containedRoot(value.ArchiveRoot, roots); err != nil {
		return errors.New("archive root is outside approved roots")
	}
	archiveRoot, err := filepath.EvalSymlinks(value.ArchiveRoot)
	if err != nil {
		return err
	}
	// 默认归档目录就在下载目录内，重复回调时文件可能已经位于归档目录，
	// 再移动一次会产生 completed/completed 套娃，这里直接跳过。
	if resolvedPath, pathErr := filepath.EvalSymlinks(path); pathErr == nil {
		if inner, relErr := filepath.Rel(archiveRoot, resolvedPath); relErr == nil && inner != "." &&
			inner != ".." && !strings.HasPrefix(inner, ".."+string(os.PathSeparator)) {
			return nil
		}
	}
	destination := filepath.Join(archiveRoot, relative)
	if sourceRoot == archiveRoot || destination == path {
		return errors.New("archive destination conflicts with source")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0750); err != nil {
		return err
	}
	if destination, err = freeDestination(destination); err != nil {
		return err
	}
	if err := movePath(path, destination); err != nil {
		return err
	}
	pruneEmptyParents(path, sourceRoot)
	log.Printf("archive done: %s -> %s", path, destination)
	return nil
}

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return config{}, err
	}
	var value config
	if encoded, ok := raw["archive_enabled"]; ok {
		if err := json.Unmarshal(encoded, &value.ArchiveEnabled); err != nil {
			return config{}, err
		}
	}
	if encoded, ok := raw["archive_root"]; ok {
		if err := json.Unmarshal(encoded, &value.ArchiveRoot); err != nil {
			return config{}, err
		}
	}
	if encoded, ok := raw["special_mode"]; ok {
		if err := json.Unmarshal(encoded, &value.SpecialMode); err != nil {
			return config{}, err
		}
	}
	if encoded, ok := raw["authorized_roots"]; ok {
		if err := json.Unmarshal(encoded, &value.AuthorizedRoots); err != nil {
			return config{}, err
		}
	}
	return value, nil
}

func main() {
	var mode, configPath, shares string
	flag.StringVar(&mode, "mode", "", "callback mode")
	flag.StringVar(&configPath, "config", "", "configuration file")
	flag.StringVar(&shares, "shares", os.Getenv("TRIM_DATA_SHARE_PATHS"), "approved shared directories")
	flag.Parse()
	args := flag.Args()
	if (mode != "complete" && mode != "error" && mode != "removed") || len(args) < 3 || configPath == "" {
		log.Fatal("invalid callback invocation")
	}
	value, err := loadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}
	roots := append(rootsFrom(shares), authorizedRoots(value)...)
	if len(roots) == 0 {
		log.Fatal("no approved shared directory")
	}
	path, err := findDownloadPath(args, roots)
	if err != nil {
		log.Printf("cleanup skipped: %v", err)
		return
	}
	if err := clean(path, roots, mode == "removed"); err != nil {
		log.Printf("cleanup skipped: %v", err)
	}
	if mode != "complete" || !value.archiveRequested() {
		return
	}
	if value.ArchiveRoot == "" {
		log.Printf("archive skipped: archive directory is not selected")
		return
	}
	time.Sleep(archiveSettleDelay)
	if err := archive(taskContainer(path, taskFileCount(args), roots), value, roots); err != nil {
		log.Printf("archive skipped: %v", err)
	}
}
