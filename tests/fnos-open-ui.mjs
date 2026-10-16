#!/usr/bin/env node
import { spawn } from 'node:child_process';
import { mkdtempSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const CHROME = process.env.ARIA2_CHROME || 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const DEVICE = process.env.ARIA2_DEVICE || '192.168.x.x';
const CDP_PORT = Number(process.env.ARIA2_CDP_PORT || 9589);
const BASE_URL = `http://${DEVICE}:5666`;
const USERNAME = process.env.ARIA2_USERNAME || 'admin';
const PASSWORD = process.env.ARIA2_PASSWORD || 'YOUR_PASSWORD';

class Client {
  constructor(ws) { this.ws = ws; this.id = 0; this.pending = new Map();
    ws.addEventListener('message', (ev) => { const msg = JSON.parse(ev.data); if (!msg.id) return; const p = this.pending.get(msg.id); if (!p) return; this.pending.delete(msg.id); msg.error ? p.reject(new Error(msg.error.message)) : p.resolve(msg.result); });
  }
  static async connect(url) { const ws = new WebSocket(url); await new Promise((res, rej) => { ws.addEventListener('open', res, { once: true }); ws.addEventListener('error', rej, { once: true }); }); return new Client(ws); }
  send(method, params = {}) { const id = ++this.id; return new Promise((resolve, reject) => { this.pending.set(id, { resolve, reject }); this.ws.send(JSON.stringify({ id, method, params })); }); }
  async eval(expression) { const r = await this.send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true }); if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description || r.exceptionDetails.text); return r.result?.value; }
  async waitFor(expression, label, timeout = 30000) { const start = Date.now(); while (Date.now() - start < timeout) { if (await this.eval(expression)) return; await sleep(300); } throw new Error('timeout: ' + label); }
  async shot(file) { const r = await this.send('Page.captureScreenshot', { format: 'png' }); writeFileSync(file, Buffer.from(r.data, 'base64')); }
  async click(selector) { const info = await this.eval(`(() => { const el = document.querySelector(${JSON.stringify(selector)}); if (!el) return null; const r = el.getBoundingClientRect(); return { x: r.x + r.width / 2, y: r.y + r.height / 2 }; })()`); if (!info) throw new Error('element not found: ' + selector); await this.send('Input.dispatchMouseEvent', { type: 'mousePressed', x: info.x, y: info.y, button: 'left', clickCount: 1 }); await this.send('Input.dispatchMouseEvent', { type: 'mouseReleased', x: info.x, y: info.y, button: 'left', clickCount: 1 }); }
  async type(selector, text) { await this.click(selector); await sleep(150); await this.send('Input.insertText', { text }); }
}

async function waitJson(url, tries = 60) { for (let i = 0; i < tries; i++) { try { const r = await fetch(url); if (r.ok) return r.json(); } catch {} await sleep(250); } throw new Error('cdp not ready'); }
async function launch() { const profile = mkdtempSync(path.join(os.tmpdir(), 'aria2-open-')); const chrome = spawn(CHROME, ['--headless=new', `--remote-debugging-port=${CDP_PORT}`, `--user-data-dir=${profile}`, '--remote-allow-origins=*', '--disable-gpu', '--no-first-run', '--window-size=1440,1000', 'about:blank'], { stdio: ['ignore', 'ignore', 'pipe'] }); await waitJson(`http://127.0.0.1:${CDP_PORT}/json/version`); const resp = await fetch(`http://127.0.0.1:${CDP_PORT}/json/new?about:blank`, { method: 'PUT' }); return { chrome, target: await resp.json() }; }
async function login(page) { await page.send('Page.enable'); await page.send('Runtime.enable'); await page.send('Network.enable'); await page.send('Page.navigate', { url: `${BASE_URL}/login` }); await page.waitFor(`document.querySelector('#username') !== null`, 'login form'); await page.type('#username', USERNAME); await page.type('#password', PASSWORD); await page.click('button[type="submit"]'); await page.waitFor(`document.body && !document.querySelector('#username')`, 'desktop', 40000); }

async function main() {
  const { chrome, target } = await launch();
  try {
    const page = await Client.connect(target.webSocketDebuggerUrl);
    await login(page);
    const before = await page.eval(`({ url: location.href, scripts: performance.getEntriesByType('resource').map((e) => e.name).filter((n) => n.endsWith('.js')).slice(-20) })`);
    console.log('BEFORE', JSON.stringify(before, null, 2));
    await page.send('Page.navigate', { url: `${BASE_URL}/app/aria2/#!/stopped` });
    await page.waitFor(`document.body && window.angular && (document.body.innerText.includes('已完成 / 已停止') || document.body.innerText.includes('Finished / Stopped'))`, 'stopped page', 30000);
    await sleep(2500);
    const state = await page.eval(`(() => ({
      url: location.href,
      body: (document.body.innerText || '').slice(0, 900),
      lang: document.documentElement.lang || '',
      bundle: performance.getEntriesByType('resource').map((e) => e.name).filter((n) => /aria-ng-[0-9a-f]+\.min\.js/.test(n)),
      navLinks: Array.from(document.querySelectorAll('aside a[href="#!/"]')).map((a) => a.getAttribute('href')).slice(0, 20),
      rpc: document.body.innerText.includes('已连接') || document.body.innerText.includes('Connected')
    }))()`);
    console.log('STATE', JSON.stringify(state, null, 2));
    await page.shot('tests/open-ui-stopped.png');
  } finally {
    try { chrome.kill(); } catch {}
  }
}

main().catch((err) => { console.error('FATAL', err); process.exitCode = 1; });
