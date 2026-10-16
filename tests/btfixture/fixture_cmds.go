// 以下子命令服务于"发货产物全链路"容器测试：容器里既没有 curl 也没有 python，
// 因此驱动、HTTP 源和轮询都由本夹具提供。测试专用，不进入发布包。
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// cmdFileServer 在回环地址上伺服一个目录，给下载任务提供本地 HTTP 源。
func cmdFileServer(args []string) error {
	fs := flag.NewFlagSet("fileserver", flag.ContinueOnError)
	dir := fs.String("dir", "", "要伺服的目录")
	listen := fs.String("listen", "127.0.0.1:0", "监听地址")
	portFile := fs.String("port-file", "", "把实际端口写到这里")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		return fmt.Errorf("fileserver 需要 -dir")
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(*dir)))
	// /stall.bin 先吐 1 MiB 再挂住不结束，用来制造"部分下载"的任务：
	// 只有真实存在的残留文件与 .aria2 控制文件，才能验证删除任务的连带清理。
	mux.HandleFunc("/stall.bin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", "67108864")
		chunk := make([]byte, 1024*1024)
		for i := range chunk {
			chunk[i] = byte(i % 251)
		}
		for i := 0; i < 4; i++ {
			if _, err := w.Write(chunk); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Minute):
		}
	})
	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		return err
	}
	if *portFile != "" {
		_, portS, _ := net.SplitHostPort(ln.Addr().String())
		if err := os.WriteFile(*portFile, []byte(portS), 0o600); err != nil {
			return err
		}
	}
	fmt.Printf("fileserver listening on %s dir=%s\n", ln.Addr().String(), *dir)
	return http.Serve(ln, mux)
}

type rpcResp struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Detail  string `json:"detail"`
	} `json:"error"`
}

// gatewayCall 通过应用 Unix Socket 发一次 JSON-RPC，模拟已登录管理员的浏览器请求。
func gatewayCall(sock, user string, body []byte) (*rpcResp, error) {
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", sock)
		},
	}}
	req, err := http.NewRequest(http.MethodPost, "http://unix/jsonrpc", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// 网关只信任飞牛统一网关注入的身份头，这里按管理员身份模拟。
	req.Header.Set("X-Trim-Username", user)
	req.Header.Set("X-Trim-Isadmin", "1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var out rpcResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("响应无法解析: %w (%s)", err, string(raw))
	}
	if out.Error != nil {
		return &out, fmt.Errorf("RPC 错误 %d: %s %s", out.Error.Code, out.Error.Message, out.Error.Detail)
	}
	return &out, nil
}

func cmdGateway(args []string) error {
	fs := flag.NewFlagSet("gateway", flag.ContinueOnError)
	sock := fs.String("sock", "/run/aria2/aria2.sock", "应用 Unix Socket")
	user := fs.String("user", "admin", "模拟的登录用户")
	method := fs.String("method", "", "JSON-RPC 方法名")
	params := fs.String("params", "[]", "JSON 数组形式的参数")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *method == "" {
		return fmt.Errorf("gateway 需要 -method")
	}
	body := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":"1","method":%q,"params":%s}`, *method, *params))
	res, err := gatewayCall(*sock, *user, body)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", string(res.Result))
	return nil
}

// cmdWaitFile 轮询等待某个文件出现且大小稳定，用于断言下载与归档搬移。
func cmdWaitFile(args []string) error {
	fs := flag.NewFlagSet("waitfile", flag.ContinueOnError)
	path := fs.String("path", "", "等待出现的文件")
	timeout := fs.Int("timeout", 60, "最长等待秒数")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("waitfile 需要 -path")
	}
	deadline := time.Now().Add(time.Duration(*timeout) * time.Second)
	var last int64 = -1
	stable := 0
	for time.Now().Before(deadline) {
		st, err := os.Stat(*path)
		if err == nil && st.Size() > 0 {
			if st.Size() == last {
				stable++
				if stable >= 3 {
					fmt.Printf("waitfile ok %s size=%d\n", *path, st.Size())
					return nil
				}
			} else {
				stable = 0
				last = st.Size()
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("等待超时: %s", *path)
}
