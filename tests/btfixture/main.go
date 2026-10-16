// btfixture 是测试夹具，不进入发布包，也不联网。
//
// 它提供两个子命令，用来在一台机器上离线跑通真实的 BitTorrent 传输，
// 从而验证 Core 的 BT 数据面（不只是"能建立监听端口"）：
//
//	mktorrent  为一个已存在的文件生成 .torrent（bencode + 分片 SHA-1）
//	tracker    最小 HTTP Tracker，把同一 info_hash 下的其他对等体回给请求方
package main

import (
	"crypto/sha1"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

// --- bencode 编码 ---

func bencStr(s []byte) []byte {
	return []byte(fmt.Sprintf("%d:", len(s)) + string(s))
}

func bencInt(n int64) []byte {
	return []byte("i" + strconv.FormatInt(n, 10) + "e")
}

// bencDict 按字典序拼接键值，bencode 要求字典键有序。
func bencDict(kv [][2][]byte) []byte {
	sort.Slice(kv, func(i, j int) bool { return string(kv[i][0]) < string(kv[j][0]) })
	out := []byte("d")
	for _, p := range kv {
		out = append(out, bencStr(p[0])...)
		out = append(out, p[1]...)
	}
	return append(out, 'e')
}

func cmdMkTorrent(args []string) error {
	fs := flag.NewFlagSet("mktorrent", flag.ContinueOnError)
	file := fs.String("file", "", "需要被种的文件")
	out := fs.String("out", "", "输出的 .torrent 路径")
	announce := fs.String("announce", "", "Tracker announce URL")
	pieceSize := fs.Int64("piece", 256*1024, "分片大小（字节）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" || *out == "" || *announce == "" {
		return fmt.Errorf("mktorrent 需要 -file -out -announce")
	}
	info, err := os.Stat(*file)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("只支持单文件种子")
	}
	f, err := os.Open(*file)
	if err != nil {
		return err
	}
	defer f.Close()

	var pieces []byte
	buf := make([]byte, *pieceSize)
	for {
		n, err := io.ReadFull(f, buf)
		if n > 0 {
			sum := sha1.Sum(buf[:n])
			pieces = append(pieces, sum[:]...)
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return err
		}
	}

	infoDict := bencDict([][2][]byte{
		{[]byte("length"), bencInt(info.Size())},
		{[]byte("name"), bencStr([]byte(filepath.Base(*file)))},
		{[]byte("piece length"), bencInt(*pieceSize)},
		{[]byte("pieces"), bencStr(pieces)},
	})
	top := bencDict([][2][]byte{
		{[]byte("announce"), bencStr([]byte(*announce))},
		{[]byte("creation date"), bencInt(time.Now().Unix())},
		{[]byte("info"), infoDict},
	})
	if err := os.WriteFile(*out, top, 0o600); err != nil {
		return err
	}
	// 打印 info_hash 便于日志核对，它只是内容摘要，不是凭据。
	sum := sha1.Sum(infoDict)
	fmt.Printf("mktorrent ok pieces=%d info_hash=%x\n", len(pieces)/sha1.Size, sum)
	return nil
}

type peerKey struct {
	infoHash string
	peerID   string
}

// trackedPeer 同时记住地址与 BT 监听端口：Tracker 的 compact peers 需要两者，
// 而端口只在 announce 查询参数里出现，必须与 peer_id 一起存下来。
type trackedPeer struct {
	ip   net.IP
	port int
	last time.Time
}

func cmdTracker(args []string) error {
	fs := flag.NewFlagSet("tracker", flag.ContinueOnError)
	listen := fs.String("listen", "127.0.0.1:0", "监听地址")
	outFile := fs.String("port-file", "", "把实际监听端口写到这里，便于脚本读取")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var (
		mu    sync.Mutex
		peers = map[peerKey]trackedPeer{}
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/announce", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		infoHash := q.Get("info_hash")
		peerID := q.Get("peer_id")
		if len(infoHash) != sha1.Size || len(peerID) < 8 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		port, _ := strconv.Atoi(q.Get("port"))
		key := peerKey{infoHash, peerID}
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = "127.0.0.1"
		}
		ip := net.ParseIP(host)
		if ip == nil {
			ip = net.IPv4(127, 0, 0, 1)
		}
		if v := q.Get("event"); v == "stopped" {
			mu.Lock()
			delete(peers, key)
			mu.Unlock()
		} else if port > 0 {
			mu.Lock()
			peers[key] = trackedPeer{ip: ip, port: port, last: time.Now()}
			mu.Unlock()
		}

		// compact peers：每个对等体 4 字节 IPv4 + 2 字节端口，按 info_hash 汇总并排除请求者自身。
		var compact []byte
		deadline := time.Now().Add(-90 * time.Second)
		mu.Lock()
		for k, tp := range peers {
			if k.infoHash != infoHash || k.peerID == peerID || tp.port == 0 {
				continue
			}
			if tp.last.Before(deadline) {
				delete(peers, k)
				continue
			}
			v4 := tp.ip.To4()
			if v4 == nil {
				continue
			}
			compact = append(compact, v4...)
			var pb [2]byte
			binary.BigEndian.PutUint16(pb[:], uint16(tp.port))
			compact = append(compact, pb[:]...)
		}
		mu.Unlock()

		body := bencDict([][2][]byte{
			{[]byte("interval"), bencInt(5)},
			{[]byte("peers"), bencStr(compact)},
		})
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write(body)
	})

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		return err
	}
	if *outFile != "" {
		_, portS, _ := net.SplitHostPort(ln.Addr().String())
		if err := os.WriteFile(*outFile, []byte(portS), 0o600); err != nil {
			return err
		}
	}
	fmt.Printf("tracker listening on %s\n", ln.Addr().String())
	return http.Serve(ln, mux)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "用法: btfixture <mktorrent|tracker> [参数]")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "mktorrent":
		err = cmdMkTorrent(os.Args[2:])
	case "tracker":
		err = cmdTracker(os.Args[2:])
	case "fileserver":
		err = cmdFileServer(os.Args[2:])
	case "gateway":
		err = cmdGateway(os.Args[2:])
	case "waitfile":
		err = cmdWaitFile(os.Args[2:])
	default:
		err = fmt.Errorf("未知子命令 %q", os.Args[1])
	}
	if err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}
