const apiBase = new URL("api/v1/", window.location.href);
let appPromise;
const getApp = () => {
    if (!appPromise) {
        appPromise = import("./vendor/trim-web-app-0.4.2.js").then(({ TrimApp }) => new TrimApp());
    }
    return appPromise;
};
const request = async (path, options) => {
    const response = await fetch(new URL(path, apiBase), {
        credentials: "same-origin",
        ...options,
        headers: { "Content-Type": "application/json", ...(options && options.headers) }
    });
    if (!response.ok) throw new Error("配置更新被服务拒绝");
    return response.status === 204 ? undefined : response.json();
};
const selectedDirectory = result =>
    result && Array.isArray(result.data) && result.data.length === 1 && typeof result.data[0] === "string"
        ? result.data[0] : undefined;
const applyDirectory = async (path, authorize) => {
    const config = await request("config");
    if (authorize) config.authorized_roots = Array.from(new Set([...(Array.isArray(config.authorized_roots) ? config.authorized_roots : []), path]));
    config.download_root = path;
    await request("config", { method: "PUT", body: JSON.stringify(config) });
};
const chooseUserDirectory = async () => {
    const app = await getApp();
    const path = selectedDirectory(await app.pickUserFile({ directory: true, multiple: false, title: "选择并授权下载目录" }));
    if (path) await applyDirectory(path, true);
};
const chooseSharedDirectory = async () => {
    const app = await getApp();
    const path = selectedDirectory(await app.pickSharedFile({ multiple: false, title: "选择共享下载目录" }));
    if (path) await applyDirectory(path, false);
};
// 归档目录自本版起恒定为「下载目录/completed」，由管理器自动派生并创建，
// 因此菜单里不再提供手工指定归档目录的入口；AriaNg 原生「下载完成后归档」
// 开关被打开时会调 window.fnosArchiveDirectory.pick()，fnos 适配里不再
// 提供选择器，直接返回空字符串让 AriaNg 走"用户取消"分支（不保存开启状态），
// 避免弹窗后又被后端归一覆盖成派生值。
const pickArchiveDirectory = async () => "";
window.fnosArchiveDirectory = { pick: pickArchiveDirectory };
const openDownloadDirectory = async () => {
    const app = await getApp();
    const config = await request("config");
    if (!config.download_root) throw new Error("请先选择下载目录");
    await app.openFileManager(config.download_root);
};
const configureNetwork = async () => {
    const config = await request("config");
    const rpc = window.prompt("RPC 监听端口（1024-65535）", String(config.rpc_port));
    if (rpc === null) return;
    const listen = window.prompt("BT TCP/UDP 监听端口（1024-65535）", String(config.listen_port));
    if (listen === null) return;
    const rpcPort = Number(rpc), listenPort = Number(listen);
    if (!Number.isInteger(rpcPort) || !Number.isInteger(listenPort) || rpcPort < 1024 || rpcPort > 65535 || listenPort < 1024 || listenPort > 65535 || rpcPort === listenPort) {
        window.alert("端口必须是 1024-65535 的不同整数。");
        return;
    }
    const timeout = window.prompt("DHT 消息超时（秒）", config.dht_message_timeout || "10");
    if (timeout === null || !/^\d+$/.test(timeout) || Number(timeout) < 1) { window.alert("DHT 超时必须是正整数秒。"); return; }
    config.rpc_port = rpcPort; config.listen_port = listenPort;
    config.dht_message_timeout = timeout;
    config.enable_dht = window.confirm("启用 DHT IPv4？");
    config.enable_dht6 = window.confirm("启用 DHT IPv6？");
    await request("config", { method: "PUT", body: JSON.stringify(config) });
    window.alert("端口已保存，服务正在重启。");
    window.setTimeout(() => window.location.reload(), 1500);
};
const showSettingsPanel = async () => {
    const config = await request("config");
    const html = `<div id="aria2-fnos-settings" style="position:fixed;z-index:10000;top:12%;left:50%;transform:translateX(-50%);background:#fff;padding:20px;border:1px solid #777;min-width:320px"><h4>Aria2 设置</h4><label>RPC 端口 <input id="a-rpc" type="number" min="1024" max="65535" value="${config.rpc_port}"></label><br><label>BT TCP/UDP 端口 <input id="a-listen" type="number" min="1024" max="65535" value="${config.listen_port}"></label><br><label>DHT 超时（秒） <input id="a-timeout" type="number" min="1" value="${config.dht_message_timeout || 10}"></label><br><label><input id="a-dht" type="checkbox" ${config.enable_dht !== false ? "checked" : ""}> 启用 DHT IPv4</label><br><label><input id="a-dht6" type="checkbox" ${config.enable_dht6 ? "checked" : ""}> 启用 DHT IPv6</label><br><button id="a-save">保存并重启</button> <button id="a-cancel">取消</button></div>`;
    const wrap = document.createElement("div"); wrap.innerHTML = html; document.body.appendChild(wrap);
    wrap.querySelector("#a-cancel").onclick = () => wrap.remove();
    wrap.querySelector("#a-save").onclick = async () => {
        const next = {
            ...config,
            rpc_port: Number(wrap.querySelector("#a-rpc").value),
            listen_port: Number(wrap.querySelector("#a-listen").value),
            dht_message_timeout: String(wrap.querySelector("#a-timeout").value),
            enable_dht: wrap.querySelector("#a-dht").checked,
            enable_dht6: wrap.querySelector("#a-dht6").checked
        };
        if (!Number.isInteger(next.rpc_port) || !Number.isInteger(next.listen_port) || next.rpc_port === next.listen_port) {
            alert("端口必须为不同整数"); return;
        }
        await request("config", { method: "PUT", body: JSON.stringify(next) });
        alert("已保存，服务正在重启"); wrap.remove(); setTimeout(() => location.reload(), 1500);
    };
};
// 归档模式自本版起恒定为“下载完成后自动归档到 下载目录/completed”，
// 派生与创建都由管理器处理，因此 fnos 适配层不再提供手工切换入口。
function installDirectoryMenu() {
    const toolbar = document.querySelector(".main-header .navbar-toolbar .navbar-nav");
    if (!toolbar || document.getElementById("fnos-directory-menu")) return;
    const item = document.createElement("li");
    item.id = "fnos-directory-menu";
    item.className = "dropdown";
    const toggle = document.createElement("a");
    toggle.className = "toolbar dropdown-toggle";
    toggle.setAttribute("data-toggle", "dropdown");
    toggle.title = "下载目录";
    toggle.innerHTML = '<i class="fa fa-folder-open" aria-hidden="true"></i>';
    const menu = document.createElement("ul");
    menu.className = "dropdown-menu";
    [["选择并授权目录", chooseUserDirectory], ["选择共享目录", chooseSharedDirectory], ["打开下载目录", openDownloadDirectory], ["配置网络与 DHT", configureNetwork], ["Aria2 设置面板", showSettingsPanel]].forEach(([label, action]) => {
        const li = document.createElement("li");
        const button = document.createElement("a");
        button.className = "pointer-cursor";
        button.textContent = label;
        button.addEventListener("click", () => action().catch(() => window.alert("目录操作未完成。")));
        li.appendChild(button);
        menu.appendChild(li);
    });
    item.append(toggle, menu);
    toolbar.appendChild(item);
}
let attempts = 0;
const retryInstall = () => {
    installDirectoryMenu();
    attempts += 1;
    if (!document.getElementById("fnos-directory-menu") && attempts < 120) {
        window.setTimeout(retryInstall, 500);
    }
};
if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", retryInstall, { once: true });
else retryInstall();
