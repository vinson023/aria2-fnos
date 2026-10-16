#!/usr/bin/env node
// fnOS 设备端 Aria2 界面验收：语言、左侧菜单、归档开关与归档目录的显示和保存。
// 用法: node tests/fnos-verify-ui.mjs   (可用环境变量 ARIA2_DEVICE / ARIA2_USERNAME / ARIA2_PASSWORD)
import { spawn } from 'node:child_process';
import { mkdtempSync, writeFileSync, mkdirSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const CHROME = process.env.ARIA2_CHROME || 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const DEVICE = process.env.ARIA2_DEVICE || '192.168.x.x';
const CDP_PORT = Number(process.env.ARIA2_CDP_PORT || 9613);
const BASE = `http://${DEVICE}:5666`;
const USER = process.env.ARIA2_USERNAME || 'admin';
const PASS = process.env.ARIA2_PASSWORD || 'YOUR_PASSWORD';
const OUT = 'tests/ui-evidence';
let failures = 0;

class Client {
  constructor(ws) { this.ws = ws; this.id = 0; this.pending = new Map();
    ws.addEventListener('message', (e) => { const m = JSON.parse(e.data); if (!m.id) return; const q = this.pending.get(m.id); if (!q) return; this.pending.delete(m.id); m.error ? q.reject(new Error(m.error.message)) : q.resolve(m.result); }); }
  static async connect(url) { const ws = new WebSocket(url); await new Promise((res, rej) => { ws.addEventListener('open', res, { once: true }); ws.addEventListener('error', rej, { once: true }); }); return new Client(ws); }
  send(method, params = {}) { const id = ++this.id; return new Promise((resolve, reject) => { this.pending.set(id, { resolve, reject }); this.ws.send(JSON.stringify({ id, method, params })); }); }
  async eval(expr) { const r = await this.send('Runtime.evaluate', { expression: expr, returnByValue: true, awaitPromise: true }); if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description || r.exceptionDetails.text); return r.result?.value; }
  async waitFor(expr, label, timeout = 40000) { const t = Date.now(); for (;;) { if (await this.eval(expr)) return; if (Date.now() - t > timeout) throw new Error('timeout: ' + label); await sleep(300); } }
  async shot(name) { const r = await this.send('Page.captureScreenshot', { format: 'png' }); mkdirSync(OUT, { recursive: true }); const f = `${OUT}/${name}.png`; writeFileSync(f, Buffer.from(r.data, 'base64')); console.log('   截图 ->', f); }
  async click(sel) { const box = await this.eval(`(()=>{const el=document.querySelector(${JSON.stringify(sel)}); if(!el) return null; el.scrollIntoView({block:'center'}); const r=el.getBoundingClientRect(); return {x:r.x+r.width/2,y:r.y+r.height/2};})()`); if (!box) throw new Error('no element ' + sel); await sleep(150); await this.send('Input.dispatchMouseEvent', { type: 'mousePressed', x: box.x, y: box.y, button: 'left', clickCount: 1 }); await this.send('Input.dispatchMouseEvent', { type: 'mouseReleased', x: box.x, y: box.y, button: 'left', clickCount: 1 }); }
  ok(m) { console.log('ok   ' + m); }
  bad(m) { console.log('FAIL ' + m); failures++; }
}

// 行定位：按行内文本前缀查找，不依赖具体标签类型（AriaNg 设置行为表格/自定义指令）。
const HELPERS = `window.V = {
  row(name){ const seen=[]; for (const el of document.querySelectorAll('*')) { const t=(el.textContent||'').trim().replace(/\\s+/g,' '); if(!t.startsWith(name)) continue; if (el.querySelector('select,input')) { if (el.children.length<=6) return el; } seen.push(el.tagName); } return null; },
  read(name){ const r=window.V.row(name); if(!r) return null; const s=r.querySelector('select'); const i=r.querySelector('input'); return { control: s?'select':(i?'input':'none'), value: s?(s.options[s.selectedIndex]?s.options[s.selectedIndex].text:null):(i?i.value:null), raw: s?s.value:(i?i.value:null), html: r.outerHTML.replace(/\\s+/g,' ').slice(0,240) }; },
  // 跳过 AriaNg 的隐藏占位项（value 为空且 disabled），它不在真实用户的下拉可见范围内。
  pick(name,want){ const r=window.V.row(name); const s=r&&r.querySelector('select'); if(!s) return 'no-select'; const list=[...s.options].filter(o=>o.value!=='' && !o.disabled); const o=list.find(x=>x.text.trim()===want); if(!o) return 'no-option['+want+'] have='+list.map(x=>x.text+'/'+x.value).join(','); s.value=o.value; s.dispatchEvent(new Event('change',{bubbles:true})); s.dispatchEvent(new Event('blur',{bubbles:true})); return 'picked:'+o.text+'('+o.value+')'; },
  save(){ const b=[...document.querySelectorAll('a,button,span,i')].find(x=>['保存','Save'].includes((x.textContent||'').trim())); if(!b) return 'no-save-button'; b.click(); return 'clicked'; },
  navText(){ return (document.body.innerText||'').replace(/\\s+/g,' '); }
};`;

async function main() {
  const profile = mkdtempSync(path.join(os.tmpdir(), 'aria2-ui-'));
  const chrome = spawn(CHROME, ['--headless=new', `--remote-debugging-port=${CDP_PORT}`, `--user-data-dir=${profile}`, '--remote-allow-origins=*', '--disable-gpu', '--no-first-run', '--window-size=1500,1050', 'about:blank'], { stdio: 'ignore' });
  for (let i = 0; i < 120; i++) { try { if ((await fetch(`http://127.0.0.1:${CDP_PORT}/json/version`)).ok) break; } catch {} await sleep(250); }
  const target = await (await fetch(`http://127.0.0.1:${CDP_PORT}/json/new?about:blank`, { method: 'PUT' })).json();
  const page = await Client.connect(target.webSocketDebuggerUrl);
  await page.send('Page.enable'); await page.send('Runtime.enable');
  try {
    console.log('=== 1. 登录飞牛');
    await page.send('Page.navigate', { url: `${BASE}/login` });
    await page.waitFor(`document.querySelector('#username')!==null`, '登录页');
    await page.eval(`(()=>{const s=(e,v)=>{Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(e,v);e.dispatchEvent(new Event('input',{bubbles:true}));};s(document.querySelector('#username'),${JSON.stringify(USER)});s(document.querySelector('#password'),${JSON.stringify(PASS)});document.querySelector('button[type=submit]').click();return 1;})()`);
    await page.waitFor(`!document.querySelector('#username')`, '桌面', 45000);
    page.ok('登录成功');

    console.log('=== 2. 打开应用并检查语言');
    await page.send('Page.navigate', { url: `${BASE}/app/aria2/#!/downloading` });
    await page.waitFor(`window.angular && document.body.innerText.length>60`, 'AriaNg 启动', 60000);
    await page.eval(HELPERS); await sleep(3500);
    const home = await page.eval(`({url:location.href, text:V.navText().slice(0,300), bundle:performance.getEntriesByType('resource').map(e=>e.name).filter(n=>/aria-ng-[0-9a-f]+\\.min\\.js/.test(n)).map(n=>n.split('/').pop()), connected:/已连接/.test(V.navText())})`);
    console.log('   入口:', home.url, '| bundle:', (home.bundle || []).join(','));
    console.log('   文本:', home.text);
    const zhHits = ['正在下载', '正在等待', '已完成', '系统设置', '已连接'].filter((k) => home.text.includes(k));
    if (zhHits.length >= 4 && home.connected) page.ok('界面为简体中文（' + zhHits.join('、') + '）'); else page.bad('中文不完整：' + zhHits.join('、'));
    if (/Downloading|Waiting|Finished \/ Stopped|Settings/.test(home.text)) page.bad('存在英文界面文案');
    await page.shot('01-downloading');

    console.log('=== 3. 左侧菜单可点击');
    const clicked = await page.eval(`(()=>{const a=[...document.querySelectorAll('a')].find(x=>(x.textContent||'').includes('已完成 / 已停止')); if(!a) return 'no-link'; a.click(); return a.getAttribute('href');})()`);
    await sleep(2500);
    const after = await page.eval(`({hash:location.hash, text:V.navText().slice(0,200)})`);
    console.log('   href:', clicked, '-> hash:', after.hash);
    if (String(clicked).startsWith('#!/') && after.hash.includes('stopped')) page.ok('左侧菜单点击后正常跳转'); else page.bad(`菜单点击异常 href=${clicked} hash=${after.hash}`);
    await page.shot('02-stopped');

    console.log('=== 4. Aria2 基本设置：归档开关与归档目录');
    await page.eval(HELPERS);
    await page.send('Page.navigate', { url: `${BASE}/app/aria2/#!/settings/aria2/basic` });
    await page.waitFor(`(typeof V!=='undefined') && V.row('下载完成后归档')!==null`, '基本设置行', 45000);
    await page.eval(HELPERS); await sleep(2500);
    const m1 = await page.eval(`({mode:V.read('下载完成后归档'), dir:V.read('归档目录'), labels:[...document.querySelectorAll('*')].filter(e=>e.children.length===0&&/归档/.test(e.textContent||'')).map(e=>e.textContent.trim()).slice(0,6)})`);
    console.log('   下载完成后归档:', JSON.stringify(m1.mode));
    console.log('   归档目录      :', JSON.stringify(m1.dir));
    if (m1.mode && m1.mode.value) page.ok('归档开关回显当前值：' + m1.mode.value); else page.bad('归档开关无回显');
    if (m1.dir && m1.dir.value) page.ok('归档目录回显：' + m1.dir.value); else page.bad('归档目录为空');
    await page.shot('03-settings-basic');

    console.log('=== 5. 保存往返（否 -> 刷新 -> 是 -> 刷新）');
    console.log('   ', await page.eval(`V.pick('下载完成后归档','否')`));
    console.log('   保存:', await page.eval(`V.save()`));
    await sleep(5000);
    await page.send('Page.navigate', { url: `${BASE}/app/aria2/#!/settings/aria2/basic` });
    await page.waitFor(`(typeof V!=='undefined') && V.row('下载完成后归档')!==null`, '重载 1', 45000);
    await page.eval(HELPERS); await sleep(2500);
    const off = await page.eval(`({mode:V.read('下载完成后归档'), dir:V.read('归档目录')})`);
    console.log('   关闭后:', JSON.stringify(off.mode), JSON.stringify(off.dir));
    if (off.mode && off.mode.value && off.mode.value.includes('否')) page.ok('“否”保存成功'); else page.bad('“否”未保存：' + JSON.stringify(off.mode));
    await page.shot('04-archive-off');
    console.log('   ', await page.eval(`V.pick('下载完成后归档','是')`));
    console.log('   保存:', await page.eval(`V.save()`));
    await sleep(5000);
    await page.send('Page.navigate', { url: `${BASE}/app/aria2/#!/settings/aria2/basic` });
    await page.waitFor(`(typeof V!=='undefined') && V.row('下载完成后归档')!==null`, '重载 2', 45000);
    await page.eval(HELPERS); await sleep(2500);
    const back = await page.eval(`({mode:V.read('下载完成后归档'), dir:V.read('归档目录')})`);
    console.log('   复原后:', JSON.stringify(back.mode), JSON.stringify(back.dir));
    if (back.mode && back.mode.value && back.mode.value.includes('是') && back.dir && back.dir.value) page.ok('已恢复“是”，归档目录仍为 ' + back.dir.value); else page.bad('恢复失败：' + JSON.stringify(back));
    await page.shot('05-archive-on');
    console.log(`=== 结果：${failures} 项失败`);
  } catch (err) {
    console.error('FATAL', err.message); failures++;
    try { await page.shot('99-fatal'); } catch {}
  } finally { try { chrome.kill(); } catch {} }
  process.exitCode = failures > 0 ? 1 : 0;
}
main();
