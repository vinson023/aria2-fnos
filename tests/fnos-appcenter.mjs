import { spawn } from 'node:child_process';
import { mkdtempSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const CHROME = process.env.ARIA2_CHROME || 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const DEVICE = process.env.ARIA2_DEVICE || '192.168.x.x';
const CDP_PORT = Number(process.env.ARIA2_CDP_PORT || 9444);
const BASE_URL = `http://${DEVICE}:5666`;
const USERNAME = process.env.ARIA2_USERNAME || 'admin';
const PASSWORD = process.env.ARIA2_PASSWORD || 'YOUR_PASSWORD';
const TARGET = process.argv[2] || '/app/trim.app-center';

class Client {
  constructor(ws) {
    this.ws = ws;
    this.id = 0;
    this.pending = new Map();
    ws.addEventListener('message', (ev) => {
      const msg = JSON.parse(ev.data);
      if (!msg.id) return;
      const p = this.pending.get(msg.id);
      if (!p) return;
      this.pending.delete(msg.id);
      msg.error ? p.reject(new Error(msg.error.message)) : p.resolve(msg.result);
    });
  }
  static async connect(url) {
    const ws = new WebSocket(url);
    await new Promise((res, rej) => { ws.addEventListener('open', res, { once: true }); ws.addEventListener('error', rej, { once: true }); });
    return new Client(ws);
  }
  send(method, params = {}) {
    const id = ++this.id;
    return new Promise((resolve, reject) => { this.pending.set(id, { resolve, reject }); this.ws.send(JSON.stringify({ id, method, params })); });
  }
  async eval(expression) {
    const r = await this.send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
    if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description || r.exceptionDetails.text);
    return r.result?.value;
  }
  async waitFor(expression, label, timeout = 15000) {
    const start = Date.now();
    while (Date.now() - start < timeout) {
      if (await this.eval(expression)) return;
      await sleep(300);
    }
    throw new Error('timeout: ' + label);
  }
  async shot(file) {
    const r = await this.send('Page.captureScreenshot', { format: 'png' });
    const fs = await import('node:fs');
    fs.writeFileSync(file, Buffer.from(r.data, 'base64'));
  }
}

async function waitJson(url, tries = 40) {
  for (let i = 0; i < tries; i++) { try { const r = await fetch(url); if (r.ok) return r.json(); } catch {} await sleep(250); }
  throw new Error('cdp not ready');
}

async function launch() {
  const profile = mkdtempSync(path.join(os.tmpdir(), 'aria2-ac-'));
  const chrome = spawn(CHROME, ['--headless=new', `--remote-debugging-port=${CDP_PORT}`, `--user-data-dir=${profile}`, '--remote-allow-origins=*', '--disable-gpu', '--no-first-run', '--window-size=1440,1000', 'about:blank'], { stdio: ['ignore', 'ignore', 'pipe'] });
  await waitJson(`http://127.0.0.1:${CDP_PORT}/json/version`);
  const resp = await fetch(`http://127.0.0.1:${CDP_PORT}/json/new?about:blank`, { method: 'PUT' });
  return { chrome, target: await resp.json() };
}

async function typeInto(page, selector, text) {
  const info = await page.eval(`(() => { const el = document.querySelector(${JSON.stringify(selector)}); if (!el) return null; const r = el.getBoundingClientRect(); return { x: r.x + r.width/2, y: r.y + r.height/2 }; })()`);
  await page.send('Input.dispatchMouseEvent', { type: 'mousePressed', x: info.x, y: info.y, button: 'left', clickCount: 1 });
  await page.send('Input.dispatchMouseEvent', { type: 'mouseReleased', x: info.x, y: info.y, button: 'left', clickCount: 1 });
  await sleep(150);
  await page.send('Input.insertText', { text });
}

async function click(page, selector) {
  const info = await page.eval(`(() => { const el = document.querySelector(${JSON.stringify(selector)}); if (!el) return null; const r = el.getBoundingClientRect(); return { x: r.x + r.width/2, y: r.y + r.height/2 }; })()`);
  await page.send('Input.dispatchMouseEvent', { type: 'mousePressed', x: info.x, y: info.y, button: 'left', clickCount: 1 });
  await page.send('Input.dispatchMouseEvent', { type: 'mouseReleased', x: info.x, y: info.y, button: 'left', clickCount: 1 });
}

async function login(page) {
  await page.send('Page.enable');
  await page.send('Runtime.enable');
  await page.send('Page.navigate', { url: `${BASE_URL}/login` });
  await page.waitFor(`document.querySelector('#username') !== null`, 'login form');
  await typeInto(page, '#username', USERNAME);
  await typeInto(page, '#password', PASSWORD);
  await click(page, 'button[type="submit"]');
  await page.waitFor(`document.body && !document.querySelector('#username') && location.href !== ${JSON.stringify(`${BASE_URL}/login`)}`, 'desktop', 35000);
}

async function main() {
  const { chrome, target } = await launch();
  try {
    const page = await Client.connect(target.webSocketDebuggerUrl);
    await login(page);
    const clicked = await page.eval(`(() => {
      const els = Array.from(document.querySelectorAll('div, span, button'));
      const el = els.find((e) => e.innerText && e.innerText.trim() === '应用中心' && e.children.length <= 2);
      if (!el) return null;
      el.click();
      return { tag: el.tagName, cls: (el.className || '').toString().slice(0, 160), parent: el.parentElement ? (el.parentElement.className || '').toString().slice(0, 160) : '' };
    })()`);
    console.log('CLICKED', JSON.stringify(clicked));
    await sleep(7000);
    const state = await page.eval(`({ url: location.href, title: document.title, body: (document.body ? document.body.innerText : '').slice(0, 2500) })`);
    console.log('STATE', JSON.stringify(state, null, 2));
    const listResp = await fetch(`http://127.0.0.1:${CDP_PORT}/json/list`);
    const list = await listResp.json();
    console.log('TARGETS', JSON.stringify(list.map((t) => ({ url: t.url, title: t.title, type: t.type })), null, 2));
    await page.shot('tests/appcenter-open.png');
  } finally {
    try { chrome.kill(); } catch {}
  }
}

main().catch((e) => { console.error('FATAL', e); process.exitCode = 1; });