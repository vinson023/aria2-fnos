package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestValidPort(t *testing.T) {
	for _, port := range []int{1024, 6800, 6888, 65535} {
		if !validPort(port) {
			t.Fatalf("expected %d to be valid", port)
		}
	}
	for _, port := range []int{0, 80, 65536} {
		if validPort(port) {
			t.Fatalf("expected %d to be invalid", port)
		}
	}
}

func TestSpecialModeValidation(t *testing.T) {
	if err := validateConfig(config{RPCPort: 6800, ListenPort: 6888, SpecialMode: "rclone"}); err == nil {
		t.Fatal("rclone must not be accepted in native package")
	}
}

func TestNormalizeConfigTiesArchiveToDirectory(t *testing.T) {
	t.Setenv("TRIM_DATA_SHARE_PATHS", "")
	// 历史配置：只写了 special_mode=move 但没有下载目录，无法派生归档目录，必须归一为关闭，
	// 否则“下载完成后归档”会静默失效。
	value := config{SpecialMode: "move", ArchiveEnabled: false}
	normalizeConfig(&value)
	if value.ArchiveRoot != "" || value.SpecialMode != "off" || value.ArchiveEnabled {
		t.Fatalf("archive must stay off without a download directory: %#v", value)
	}
	// 归档目录恒定派生为「下载目录/completed」：历史写入的自定义归档目录会被覆盖，
	// 用户不再需要（也无法）手工指定归档路径。
	value = config{DownloadRoot: "/vol1/1000/downloads", ArchiveRoot: "/vol1/1000/archive", SpecialMode: "move", ArchiveEnabled: false}
	normalizeConfig(&value)
	if value.ArchiveRoot != filepath.Join("/vol1/1000/downloads", defaultArchiveDirName) || !value.ArchiveEnabled {
		t.Fatalf("archive must derive from the download directory: %#v", value)
	}
	// 用户显式关闭开关时必须保持关闭。
	value = config{DownloadRoot: "/vol1/1000/downloads", ArchiveRoot: "/vol1/1000/archive", SpecialMode: "off", ArchiveEnabled: true}
	normalizeConfig(&value)
	if value.SpecialMode != "off" || value.ArchiveEnabled {
		t.Fatalf("disabling the switch must disable archiving: %#v", value)
	}
}

// TestNormalizeConfigDefaultsToShareArchive 落实产品默认值：下载目录取飞牛授权的数据
// 共享目录，归档目录为“下载目录/completed”，自动归档默认为“是”。
func TestNormalizeConfigDefaultsToShareArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TRIM_DATA_SHARE_PATHS 在 fnOS 上以冒号分隔")
	}
	share, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)

	value := config{}
	normalizeConfig(&value)
	if value.DownloadRoot != share {
		t.Fatalf("下载目录应默认取共享目录: %#v", value)
	}
	if value.ArchiveRoot != filepath.Join(share, defaultArchiveDirName) {
		t.Fatalf("归档目录应默认为下载目录/completed: %#v", value)
	}
	if value.SpecialMode != "move" || !value.ArchiveEnabled {
		t.Fatalf("自动归档应默认为开启: %#v", value)
	}

	// 用户显式关闭时不得被默认值覆盖。
	value = config{SpecialMode: "off"}
	normalizeConfig(&value)
	if value.SpecialMode != "off" || value.ArchiveEnabled {
		t.Fatalf("显式关闭归档必须保持关闭: %#v", value)
	}

	// 用户指定下载目录时，归档目录跟随其下的 completed。
	configured := filepath.Join(share, "myfiles")
	if err := os.Mkdir(configured, 0o770); err != nil {
		t.Fatal(err)
	}
	value = config{DownloadRoot: configured}
	normalizeConfig(&value)
	if value.ArchiveRoot != filepath.Join(configured, defaultArchiveDirName) {
		t.Fatalf("归档目录应跟随已配置的下载目录: %#v", value)
	}

	// 没有共享目录也没有下载目录时，必须保持关闭而不是虚构路径。
	t.Setenv("TRIM_DATA_SHARE_PATHS", "")
	value = config{}
	normalizeConfig(&value)
	if value.ArchiveRoot != "" || value.SpecialMode != "off" || value.ArchiveEnabled {
		t.Fatalf("缺少可用目录时不得开启归档: %#v", value)
	}
}

func TestGatewayAdminRequired(t *testing.T) {
	handler := requireGatewayAdmin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), "admin")
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("X-Trim-Username", "admin")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("got %d", response.Code)
	}
	request.Header.Set("X-Trim-Isadmin", "1")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("got %d", response.Code)
	}
}

func TestParseTrackersFiltersInvalidAndDeduplicates(t *testing.T) {
	trackers, err := parseTrackers([]byte("udp://one.example:80/announce\nhttps://two.example/announce\nfile:///etc/passwd\nhttps://two.example/announce\nhttp://user@bad.example/announce\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(trackers, ","), "udp://one.example:80/announce,https://two.example/announce"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFetchTrackersRejectsOversizeThenUsesNextSource(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/large":
			w.Header().Set("Content-Length", fmt.Sprint(maxTrackerListBytes+1))
			w.WriteHeader(http.StatusOK)
		case "/good":
			_, _ = io.WriteString(w, "udp://tracker.example:6969/announce\n")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	trackers, source, err := fetchTrackers(server.Client(), []string{server.URL + "/large", server.URL + "/good"})
	if err != nil {
		t.Fatal(err)
	}
	if source != server.URL+"/good" || len(trackers) != 1 {
		t.Fatalf("unexpected result: %q %#v", source, trackers)
	}
}

func TestFetchTrackersRejectsInvalidSourceAndFailure(t *testing.T) {
	client := &http.Client{Timeout: 20 * time.Millisecond}
	if _, _, err := fetchTrackers(client, []string{"http://example.test/list", "https://127.0.0.1:1/list"}); err == nil {
		t.Fatal("expected all sources to fail")
	}
}

func TestTrackerSourcesPreferNgosangBestList(t *testing.T) {
	if len(trackerSources) == 0 || trackerSources[0] != "https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best.txt" {
		t.Fatalf("unexpected default tracker source: %#v", trackerSources)
	}
}

func TestChangeGlobalTrackersInjectsSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Method != "aria2.changeGlobalOption" || string(request.Params[0]) != `"token:test-secret"` {
			t.Fatalf("unexpected RPC request: %#v", request)
		}
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":"OK"}`)
	}))
	defer server.Close()
	port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")
	var parsed int
	if _, err := fmt.Sscanf(port, "%d", &parsed); err != nil {
		t.Fatal(err)
	}
	if err := changeGlobalTrackers(server.Client(), parsed, "test-secret", []string{"udp://tracker.example:80/announce"}); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshConfiguredTrackersWritesAndUpdatesAria2(t *testing.T) {
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "udp://tracker.example:6969/announce\n")
	}))
	defer source.Close()
	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request rpcRequest
		_ = json.NewDecoder(r.Body).Decode(&request)
		if request.Method != "aria2.changeGlobalOption" {
			t.Fatalf("unexpected RPC request: %#v", request)
		}
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":"OK"}`)
	}))
	defer rpc.Close()
	port := strings.TrimPrefix(rpc.URL, "http://127.0.0.1:")
	var parsed int
	if _, err := fmt.Sscanf(port, "%d", &parsed); err != nil {
		t.Fatal(err)
	}
	store := &configStore{cfg: config{RPCPort: parsed, TrackerSource: source.URL}}
	path := filepath.Join(t.TempDir(), "trackers.txt")
	count, usedSource, err := refreshConfiguredTrackers(store, source.Client(), "test-secret", path)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || usedSource != source.URL {
		t.Fatalf("unexpected result: count=%d source=%s", count, usedSource)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "udp://tracker.example:6969/announce\n" {
		t.Fatalf("unexpected tracker file: %q", string(data))
	}
}

func TestLoadConfigRejectsSamePorts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"rpc_port":6800,"listen_port":6800}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(path); err == nil {
		t.Fatal("expected conflicting ports to fail")
	}
}

func TestGatewayHeaderRequired(t *testing.T) {
	handler := requireGatewayUser(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), "admin")
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", response.Code)
	}
	request.Header.Set("X-Trim-Username", "admin")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("got %d", response.Code)
	}
}

func TestNormalizeGatewayPathStripsApplicationPrefix(t *testing.T) {
	var got string
	handler := normalizeGatewayPath(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "http://localhost/app/aria2/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || got != "/" {
		t.Fatalf("got status=%d path=%q", response.Code, got)
	}
	request = httptest.NewRequest(http.MethodGet, "http://localhost/app/aria2/jsonrpc", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if got != "/jsonrpc" {
		t.Fatalf("got path=%q", got)
	}
}

func TestConfigUpdateRejectsConflictingPorts(t *testing.T) {
	store := configStore{path: filepath.Join(t.TempDir(), "config.json")}
	if err := store.update(config{RPCPort: 6800, ListenPort: 6800}); err == nil {
		t.Fatal("expected conflicting ports to fail")
	}
}

func TestConfigUpdateAllowsUnchangedOccupiedPorts(t *testing.T) {
	store := configStore{
		path: filepath.Join(t.TempDir(), "config.json"),
		cfg:  config{RPCPort: 6800, ListenPort: 6888},
	}
	if err := store.update(config{RPCPort: 6800, ListenPort: 6888}); err != nil {
		t.Fatalf("unchanged ports must not be checked as external conflicts: %v", err)
	}
}

func TestConfigUpdateAcceptsIndependentDHTPort(t *testing.T) {
	if err := validateConfig(config{RPCPort: 6800, ListenPort: 6888, DHTListenPort: 6890}); err != nil {
		t.Fatalf("independent DHT port must be accepted: %v", err)
	}
}

func TestAllowedDownloadRootRejectsOutsideShare(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TRIM_DATA_SHARE_PATHS is colon-delimited on fnOS")
	}
	share := t.TempDir()
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)
	if !allowedDownloadRoot(share, []string{share}) {
		t.Fatal("expected share root to be allowed")
	}
	if allowedDownloadRoot(t.TempDir(), []string{share}) {
		t.Fatal("expected unrelated directory to be rejected")
	}
}

func TestEffectiveDownloadRootUsesConfiguredRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TRIM_DATA_SHARE_PATHS is colon-delimited on fnOS")
	}
	share := t.TempDir()
	configured := filepath.Join(share, "selected")
	if err := os.Mkdir(configured, 0750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TRIM_DATA_SHARE_PATHS", share)
	root, err := effectiveDownloadRoot(config{DownloadRoot: configured})
	if err != nil || root != configured {
		t.Fatalf("got %q, %v", root, err)
	}
}

func TestValidateConfigRejectsUnsafeDirectoryCharacters(t *testing.T) {
	if err := validateConfig(config{RPCPort: 6800, ListenPort: 6888, DownloadRoot: "/vol1/downloads\nlisten-port=22"}); err == nil {
		t.Fatal("expected unsafe directory to be rejected")
	}
}

func TestNormalizeAuthorizedRootsRejectsSystemDirectory(t *testing.T) {
	if _, err := normalizeAuthorizedRoots([]string{"/etc"}); err == nil {
		t.Fatal("expected system directory to be rejected")
	}
}

func TestRemoveTaskPathFileAndControls(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fnOS cleanup paths use POSIX separators")
	}
	root := t.TempDir()
	path := filepath.Join(root, "download.bin")
	if err := os.WriteFile(path, []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".aria2", []byte("control"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".torrent", []byte("torrent"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := removeTaskPath(path, []string{root}); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []string{path, path + ".aria2", path + ".torrent"} {
		if _, err := os.Stat(candidate); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed", candidate)
		}
	}
}

func TestRemoveTaskPathDirectoryAndControls(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fnOS cleanup paths use POSIX separators")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "torrent-dir")
	nested := filepath.Join(dir, "sub")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "piece.bin"), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "piece.bin.aria2"), []byte("control"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+".torrent", []byte("torrent"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := removeTaskPath(dir, []string{root}); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []string{dir, nested, dir + ".torrent"} {
		if _, err := os.Stat(candidate); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed", candidate)
		}
	}
}

func TestRemoveTaskPathOutsideRootRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fnOS cleanup paths use POSIX separators")
	}
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "download.bin")
	if err := os.WriteFile(outside, []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := removeTaskPath(outside, []string{root}); err == nil {
		t.Fatal("expected outside path to be rejected")
	}
}

func TestRemoveTaskPathMissingPathNoops(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fnOS cleanup paths use POSIX separators")
	}
	root := t.TempDir()
	if err := removeTaskPath(filepath.Join(root, "missing.bin"), []string{root}); err != nil {
		t.Fatal(err)
	}
}

// TestSyncArchiveRootWithDownload 覆盖用户改下载目录后的归档目录跟随行为：
// 默认归档跟随新目录，自定义归档保持不变，未改目录时不产生副作用。
func TestSyncArchiveRootWithDownload(t *testing.T) {
	previous := config{DownloadRoot: "/vol1/@appshare/aria2/downloads", ArchiveRoot: "/vol1/@appshare/aria2/downloads/completed", SpecialMode: "move"}

	follow := previous
	follow.DownloadRoot = "/vol1/1000/download"
	follow.ArchiveRoot = previous.ArchiveRoot
	syncArchiveRootWithDownload(&follow, previous)
	if want := filepath.Join("/vol1/1000/download", defaultArchiveDirName); follow.ArchiveRoot != want {
		t.Fatalf("默认归档未跟随新下载目录: got %q want %q", follow.ArchiveRoot, want)
	}

	blank := previous
	blank.DownloadRoot = "/vol1/1000/download"
	blank.ArchiveRoot = ""
	syncArchiveRootWithDownload(&blank, previous)
	if want := filepath.Join("/vol1/1000/download", defaultArchiveDirName); blank.ArchiveRoot != want {
		t.Fatalf("空归档未派生: got %q want %q", blank.ArchiveRoot, want)
	}

	custom := previous
	custom.DownloadRoot = "/vol1/1000/download"
	custom.ArchiveRoot = "/vol1/1000/other"
	syncArchiveRootWithDownload(&custom, previous)
	if custom.ArchiveRoot != "/vol1/1000/other" {
		t.Fatalf("自定义归档被覆盖: got %q", custom.ArchiveRoot)
	}

	unchanged := previous
	unchanged.ArchiveRoot = previous.ArchiveRoot
	syncArchiveRootWithDownload(&unchanged, previous)
	if unchanged.ArchiveRoot != previous.ArchiveRoot {
		t.Fatalf("未改下载目录时归档被改动: got %q", unchanged.ArchiveRoot)
	}
}
