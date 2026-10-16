package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestCleanRemovesOnlyControlFilesInsideApprovedRoot(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "nested", "item")
	if err := os.MkdirAll(filepath.Dir(file), 0750); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{file, file + ".aria2", file + ".torrent"} {
		if err := os.WriteFile(path, []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := clean(file, []string{root}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("download file must remain: %v", err)
	}
	for _, path := range []string{file + ".aria2", file + ".torrent"} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("control file was not removed: %s", path)
		}
	}
}

func TestCleanRemovedDeletesIncompleteFile(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "incomplete.bin")
	if err := os.WriteFile(file, []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := clean(file, []string{root}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("file remains: %v", err)
	}
}

func TestArchiveRequiresApprovedDestination(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "item")
	if err := os.WriteFile(file, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := archive(file, config{ArchiveEnabled: true, ArchiveRoot: t.TempDir()}, []string{root}); err == nil {
		t.Fatal("expected unapproved archive root to fail")
	}
}

func TestArchiveMovesToApprovedSubdirectory(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "downloads", "item")
	archiveRoot := filepath.Join(root, "archive")
	if err := os.MkdirAll(filepath.Dir(source), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(archiveRoot, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := archive(source, config{ArchiveEnabled: true, ArchiveRoot: archiveRoot}, []string{root}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(archiveRoot, "downloads", "item")); err != nil {
		t.Fatalf("archive destination missing: %v", err)
	}
}

func TestAuthorizedRootsIgnoreUnsafeConfigValues(t *testing.T) {
	if roots := authorizedRoots(config{AuthorizedRoots: []string{"/etc", "/vol1/unsafe\npath"}}); len(roots) != 0 {
		t.Fatalf("unexpected roots: %#v", roots)
	}
}

func TestSpecialModeMoveEnablesArchive(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "downloads", "movie.mkv")
	archiveRoot := filepath.Join(root, "archive")
	if err := os.MkdirAll(filepath.Dir(source), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(archiveRoot, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	// 历史配置只写 special_mode=move，也必须真正完成归档。
	value := config{SpecialMode: "move", ArchiveRoot: archiveRoot}
	if !value.archiveRequested() {
		t.Fatal("special_mode=move must request archiving")
	}
	if err := archive(source, value, []string{root}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(archiveRoot, "downloads", "movie.mkv")); err != nil {
		t.Fatalf("archived file missing: %v", err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source file remains: %v", err)
	}
}

func TestArchiveSkippedWithoutEnabledMode(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "keep.bin")
	if err := os.WriteFile(source, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := archive(source, config{ArchiveRoot: filepath.Join(root, "archive")}, []string{root}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("file must stay in place: %v", err)
	}
}

func TestArchiveRenamesInsteadOfOverwriting(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "downloads", "movie.mkv")
	archiveRoot := filepath.Join(root, "archive")
	if err := os.MkdirAll(filepath.Dir(source), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(archiveRoot, 0750); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{source, filepath.Join(archiveRoot, "downloads", "movie.mkv")} {
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive(source, config{ArchiveEnabled: true, ArchiveRoot: archiveRoot}, []string{root}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(archiveRoot, "downloads", "movie (1).mkv")); err != nil {
		t.Fatalf("collision copy was not renamed: %v", err)
	}
}

func TestArchiveMovesTaskContainerForMultiFileTasks(t *testing.T) {
	root := t.TempDir()
	container := filepath.Join(root, "downloads", "Movie.1080p")
	first := filepath.Join(container, "disc1.mkv")
	second := filepath.Join(container, "disc2.mkv")
	archiveRoot := filepath.Join(root, "archive")
	if err := os.MkdirAll(container, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(archiveRoot, 0750); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if got := taskContainer(first, 2, []string{root}); got != container {
		t.Fatalf("container resolution got %s want %s", got, container)
	}
	if got := taskContainer(first, 1, []string{root}); got != first {
		t.Fatalf("single file task must stay a file, got %s", got)
	}
	// 多文件任务直接落在下载根目录时，不允许移动下载根目录本身。
	loose := filepath.Join(root, "a.bin")
	if err := os.WriteFile(loose, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := taskContainer(loose, 3, []string{root}); got != loose {
		t.Fatalf("must not escalate to the download root, got %s", got)
	}
	if err := archive(taskContainer(first, 2, []string{root}), config{ArchiveEnabled: true, ArchiveRoot: archiveRoot}, []string{root}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(archiveRoot, "downloads", "Movie.1080p", "disc2.mkv")); err != nil {
		t.Fatalf("container was not moved: %v", err)
	}
}

func TestMovePathFallsBackToCopyAcrossDevices(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "vol1", "big.bin")
	destination := filepath.Join(root, "vol2", "big.bin")
	if err := os.MkdirAll(filepath.Dir(source), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}
	if !isCrossDevice(syscall.Errno(18)) {
		t.Fatal("errno 18 must be treated as cross-device")
	}
	original := renamePath
	renamePath = func(_, _ string) error { return syscall.Errno(18) }
	defer func() { renamePath = original }()
	if err := movePath(source, destination); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("copy fallback did not write destination: %v", err)
	}
	if string(data) != "payload" {
		t.Fatalf("unexpected content: %s", data)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source must be removed after a successful copy: %v", err)
	}
}

func TestMovePathKeepsSourceWhenCopyFails(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "keep.bin")
	if err := os.WriteFile(source, []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}
	// 目标父路径是普通文件，复制必然失败，此时源文件必须保留。
	blocker := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	original := renamePath
	renamePath = func(_, _ string) error { return syscall.Errno(18) }
	defer func() { renamePath = original }()
	if err := movePath(source, filepath.Join(blocker, "nested", "keep.bin")); err == nil {
		t.Fatal("expected copy failure")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source must survive a failed archive: %v", err)
	}
}

// 默认归档目录是下载目录下的 completed，可能被用户手动删除。
// archive 必须在授权根内自愈补建，否则任务会永久停留在下载目录。
func TestArchiveRecreatesMissingArchiveDirectory(t *testing.T) {
	downloads := t.TempDir()
	source := filepath.Join(downloads, "movie.mkv")
	archiveRoot := filepath.Join(downloads, "completed")
	if err := os.WriteFile(source, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	value := config{SpecialMode: "move", ArchiveEnabled: true, ArchiveRoot: archiveRoot}
	if err := archive(source, value, []string{downloads}); err != nil {
		t.Fatalf("archive with missing directory: %v", err)
	}
	info, err := os.Stat(archiveRoot)
	if err != nil {
		t.Fatalf("archive directory not recreated: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("archive path is not a directory")
	}
	if _, err := os.Stat(filepath.Join(archiveRoot, "movie.mkv")); err != nil {
		t.Fatalf("archived file missing: %v", err)
	}
}

func TestLoadConfigIncludesAuthorizedRoots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	want := []string{filepath.Join(t.TempDir(), "download")}
	data, err := json.Marshal(map[string]any{
		"archive_enabled":  true,
		"archive_root":     "/vol1/archive",
		"authorized_roots": want,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.AuthorizedRoots) != 1 || got.AuthorizedRoots[0] != want[0] {
		t.Fatalf("authorized roots not loaded: %#v", got.AuthorizedRoots)
	}
}

func TestLoadConfigReadsSpecialMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"special_mode":"move","archive_root":"/vol1/1000/archive"}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := loadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.SpecialMode != "move" || !got.archiveRequested() {
		t.Fatalf("special mode not loaded: %#v", got)
	}
}
