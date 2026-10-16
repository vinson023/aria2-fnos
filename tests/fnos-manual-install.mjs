#!/usr/bin/env node
import { spawn } from 'node:child_process';
import { mkdtempSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const CHROME = process.env.ARIA2_CHROME || 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const DEVICE = process.env.ARIA2_DEVICE || '192.168.x.x';
const CDP_PORT = Number(process.env.ARIA2_CDP_PORT || 9565);
const BASE_URL = `http://${DEVICE}:5666`;
const USERNAME = process.env.ARIA2_USERNAME || 'admin';
const PASSWORD = process.env.ARIA2_PASSWORD || 'YOUR_PASSWORD';
const FPK = path.resolve(process.env.ARIA2_FPK || 'aria2.fpk');

class Client {
  constructor(ws) {
    this.ws = ws;
    this.id = 0;
    this.pending = new Map();
    this.events = [];
    ws.addEventListener('message', (ev) => {
      const msg = JSON.parse(ev.data);
      if (msg.method && this.events.length < 1000) this.events.push(msg);
      if (!msg.id) return;
      const p = this.pending.get(msg.id);
      if (!p) return;
      this.pending.delete(msg.id);
      msg.error ? p.reject(new Error(msg.error.message)) : p.resolve(msg.result);
    });
  }

  static async connect(url) {
    const ws = new WebSocket(url);
    await new Promise((resolve, reject) => {
      ws.addEventListener('open', resolve, { once: true });
      ws.addEventListener('error', () => reject(new Error('websocket open failed')), { once: true });
    });
    return new Client(ws);
  }

  send(method, params = {}) {
    const id = ++this.id;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      this.ws.send(JSON.stringify({ id, method, params }));
    });
  }

  async eval(expression) {
    const r = await this.send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
    if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description || r.exceptionDetails.text);
    return r.result?.value;
  }

  async waitFor(expression, label, timeout = 30000) {
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

  async waitEvent(method, timeout = 20000) {
    const start = Date.now();
    while (Date.now() - start < timeout) {
      const found = this.events.find((e) => e.method === method);
      if (found) return found;
      await sleep(200);
    }
    throw new Error('event timeout: ' + method);
  }

  async clickExact(text) {
    return this.eval(`(() => {
      const all = Array.from(document.querySelectorAll('div, span, li, a, button, label'));
      const candidates = all.filter((e) => e.innerText && e.innerText.trim() === ${JSON.stringify(text)});
      if (!candidates.length) return null;
      candidates.sort((a, b) => {
        const ra = a.getBoundingClientRect(); const rb = b.getBoundingClientRect();
        return (ra.width * ra.height) - (rb.width * rb.height);
      });
      const el = candidates[0];
      el.click();
      return { text: el.innerText.trim(), tag: el.tagName, cls: (el.className || '').toString().slice(0, 140) };
    })()`);
  }

  async clickContains(text) {
    return this.eval(`(() => {
      const all = Array.from(document.querySelectorAll('div, span, li, a, button, label'));
      const candidates = all.filter((e) => e.innerText && e.innerText.includes(${JSON.stringify(text)}) && e.innerText.length < 120);
      if (!candidates.length) return null;
      candidates.sort((a, b) => {
        const ra = a.getBoundingClientRect(); const rb = b.getBoundingClientRect();
        return ((ra.width || 9999) * (ra.height || 9999)) - ((rb.width || 9999) * (rb.height || 9999));
      });
      const el = candidates[0];
      el.click();
      return { text: el.innerText.trim().slice(0, 80), tag: el.tagName, cls: (el.className || '').toString().slice(0, 140) };
    })()`);
  }

  async dump(label) {
    const state = await this.eval(`(() => ({
      url: location.href,
      bodyTail: (document.body ? document.body.innerText : '').slice(-2600),
      buttons: Array.from(document.querySelectorAll('button')).filter((e) => (e.innerText || '').trim()).map((e) => ({ text: (e.innerText || '').trim().slice(0, 40), cls: (e.className || '').toString().slice(0, 100), ariaDisabled: e.getAttribute('aria-disabled') })).slice(0, 100),
      inputs: Array.from(document.querySelectorAll('input')).map((e) => ({ type: e.type, id: e.id, name: e.name, accept: e.accept || '', value: (e.value || '').slice(0, 120), cls: (e.className || '').toString().slice(0, 100) })).slice(0, 60),
      radios: Array.from(document.querySelectorAll('input[type="radio"], [role="radio"], .semi-radio')).map((e) => ({ text: (e.innerText || '').trim().slice(0, 80), cls: (e.className || '').toString().slice(0, 100), checked: e.checked || e.getAttribute('aria-checked') })).slice(0, 40),
      dialogs: Array.from(document.querySelectorAll('[role="dialog"], .semi-modal, .semi-modal-content, .semi-modal-body')).map((e) => ({ text: (e.innerText || '').slice(0, 600), cls: (e.className || '').toString().slice(0, 120) })).slice(0, 6)
    }))()`);
    console.log(`STATE[${label}]`, JSON.stringify(state, null, 2));
    await this.shot(`tests/manual-${label}.png`);
    return state;
  }

  async dumpModal(label) {
    const state = await this.eval(`(() => {
      const modal = document.querySelector('.semi-modal');
      if (!modal) return null;
      const elements = Array.from(modal.querySelectorAll('button, input, textarea, select, [role="tab"], [role="radio"], label, div[class*="upload"]')).map((e) => ({
        tag: e.tagName,
        text: (e.innerText || '').trim().slice(0, 60),
        type: e.type || '',
        accept: e.accept || '',
        value: e.type === 'file' ? (e.files && e.files.length ? e.files[0].name : '[empty]') : (e.value || ''),
        cls: (e.className || '').toString().slice(0, 120),
        placeholder: e.placeholder || ''
      }));
      return {
        text: modal.innerText.slice(0, 2400),
        elements: elements.slice(0, 120),
        hasProgress: /%|上传中|解析|校验/.test(modal.innerText)
      };
    })()`);
    console.log(`MODAL[${label}]`, JSON.stringify(state, null, 2));
    await this.shot(`tests/manual-${label}.png`);
    return state;
  }
}

async function waitJson(url, tries = 60) {
  for (let i = 0; i < tries; i++) {
    try {
      const r = await fetch(url);
      if (r.ok) return r.json();
    } catch {}
    await sleep(250);
  }
  throw new Error('cdp not ready');
}

async function launch() {
  const profile = mkdtempSync(path.join(os.tmpdir(), 'aria2-manual-'));
  const chrome = spawn(CHROME, [
    '--headless=new',
    `--remote-debugging-port=${CDP_PORT}`,
    `--user-data-dir=${profile}`,
    '--remote-allow-origins=*',
    '--disable-gpu',
    '--no-first-run',
    '--window-size=1440,1000',
    'about:blank'
  ], { stdio: ['ignore', 'ignore', 'pipe'] });
  await waitJson(`http://127.0.0.1:${CDP_PORT}/json/version`);
  const resp = await fetch(`http://127.0.0.1:${CDP_PORT}/json/new?about:blank`, { method: 'PUT' });
  return { chrome, target: await resp.json() };
}

async function typeInto(page, selector, text) {
  const info = await page.eval(`(() => {
    const el = document.querySelector(${JSON.stringify(selector)});
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: r.x + r.width / 2, y: r.y + r.height / 2 };
  })()`);
  if (!info) throw new Error('element not found: ' + selector);
  await page.send('Input.dispatchMouseEvent', { type: 'mousePressed', x: info.x, y: info.y, button: 'left', clickCount: 1 });
  await page.send('Input.dispatchMouseEvent', { type: 'mouseReleased', x: info.x, y: info.y, button: 'left', clickCount: 1 });
  await sleep(150);
  await page.send('Input.insertText', { text });
}

async function clickSel(page, selector) {
  const info = await page.eval(`(() => {
    const el = document.querySelector(${JSON.stringify(selector)});
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: r.x + r.width / 2, y: r.y + r.height / 2 };
  })()`);
  if (!info) throw new Error('element not found: ' + selector);
  await page.send('Input.dispatchMouseEvent', { type: 'mousePressed', x: info.x, y: info.y, button: 'left', clickCount: 1 });
  await page.send('Input.dispatchMouseEvent', { type: 'mouseReleased', x: info.x, y: info.y, button: 'left', clickCount: 1 });
}

async function login(page) {
  await page.send('Page.enable');
  await page.send('Runtime.enable');
  await page.send('DOM.enable');
  await page.send('Network.enable');
  await page.send('Page.navigate', { url: `${BASE_URL}/login` });
  await page.waitFor(`document.querySelector('#username') !== null`, 'login form');
  await typeInto(page, '#username', USERNAME);
  await typeInto(page, '#password', PASSWORD);
  await clickSel(page, 'button[type="submit"]');
  await page.waitFor(`document.body && !document.querySelector('#username')`, 'desktop', 40000);
}

async function setFileInput(page, selector, filePath) {
  const doc = await page.send('DOM.getDocument', { depth: -1, pierce: true });
  const q = await page.send('DOM.querySelectorAll', { nodeId: doc.root.nodeId, selector });
  if (!q.nodeIds || !q.nodeIds.length) throw new Error('file input not found: ' + selector);
  const nodeId = q.nodeIds[0];
  const node = await page.send('DOM.describeNode', { nodeId });
  await page.send('DOM.setFileInputFiles', { nodeId, files: [filePath] });
  return node.node;
}

async function setFileViaChooser(page, filePath) {
  await page.send('Page.setInterceptFileChooserDialog', { enabled: true });
  const viaInput = await page.eval(`(() => {
    const input = document.querySelector('input[type="file"][accept=".fpk"]');
    if (!input) return null;
    input.click();
    return { ok: true };
  })()`);
  console.log('INPUT CLICK', JSON.stringify(viaInput));
  let event;
  try {
    event = await page.waitEvent('Page.fileChooserOpened', 8000);
  } catch {
    console.log('WAIT EVENT FAILURE, methods=', JSON.stringify([...new Set(page.events.map((e) => e.method))].slice(0, 60)));
    await page.clickExact('从电脑上传');
    event = await page.waitEvent('Page.fileChooserOpened', 12000);
  }
  const backendNodeId = event.params.backendNodeId;
  await page.send('DOM.setFileInputFiles', { backendNodeId, files: [filePath] });
  return backendNodeId;
}

async function setComputerFile(page, filePath) {
  const selector = 'div.hidden > input[type="file"][accept=".fpk"]:not([webkitdirectory])';
  console.log('CLICK COMPUTER', await page.clickExact('从电脑上传'));
  await sleep(600);
  const node = await setFileInput(page, selector, filePath);
  const before = await page.eval(`(() => {
    const input = document.querySelector(${JSON.stringify(selector)});
    return input ? { len: input.files.length, name: input.files[0] ? input.files[0].name : null, size: input.files[0] ? input.files[0].size : null, value: input.value } : null;
  })()`);
  const dispatch = [];
  for (let i = 0; i < 3; i++) {
    const r = await page.eval(`(() => {
    const input = document.querySelector(${JSON.stringify(selector)});
    if (!input) return false;
    input.dispatchEvent(new Event('change', { bubbles: true }));
    input.dispatchEvent(new Event('input', { bubbles: true }));
    return true;
    })()`);
    dispatch.push(r);
    await sleep(400);
  }
  return { node, before, dispatch };
}

async function main() {
  const { chrome, target } = await launch();
  try {
    const page = await Client.connect(target.webSocketDebuggerUrl);
    await login(page);
    await sleep(1500);
    console.log('APP CENTER', await page.clickExact('应用中心'));
    await page.waitFor(`document.body && document.body.innerText.includes('手动安装')`, 'app center', 25000);
    await sleep(1000);
    await page.shot('tests/manual-appcenter.png');
    console.log('MANUAL CLICK', await page.clickExact('手动安装'));
    await sleep(4000);
    await page.dump('manual-open');
    const loadedJs = await page.eval(`performance.getEntriesByType('resource').map((e) => e.name).filter((n) => n.endsWith('.js'))`);
    console.log('LOADED JS', JSON.stringify(loadedJs, null, 2));

    const chooserDom = await page.eval(`(() => {
      const modal = document.querySelector('.semi-modal');
      const root = modal || document;
      const html = root.innerHTML;
      const buttons = Array.from(root.querySelectorAll('button')).map((b) => {
        const parent = b.parentElement ? (b.parentElement.outerHTML || '').slice(0, 1000) : '';
        const input = parent.match(/<input[^>]+>/g);
        return { text: (b.innerText || '').trim(), parentCls: b.parentElement ? (b.parentElement.className || '').toString() : '', parentHtml: parent, input };
      });
      const inputs = Array.from(document.querySelectorAll('input[type="file"]')).map((i) => ({
        outer: i.outerHTML,
        inModal: !!i.closest('.semi-modal'),
        rect: (() => { const r = i.getBoundingClientRect(); return { x: r.x, y: r.y, w: r.width, h: r.height, display: getComputedStyle(i).display, visibility: getComputedStyle(i).visibility }; })(),
        parentCls: (i.parentElement ? i.parentElement.className : '').toString(),
        parentHtml: (i.parentElement ? i.parentElement.outerHTML : '').slice(0, 900)
      }));
      return { buttons, inputs, htmlStart: html.slice(0, 5000), modalPresent: !!modal };
    })()`);
    console.log('CHOOSER DOM', JSON.stringify(chooserDom, null, 2).slice(0, 12000));

    console.log('STORAGE CLICK', await page.clickContains('存储空间'));
    await sleep(2000);
    await page.dump('storage-selected');

    const upload = await setComputerFile(page, FPK);
    console.log('COMPUTER UPLOAD', JSON.stringify(upload));
    await sleep(6000);
    const uploadRequests = page.events
      .filter((e) => e.method === 'Network.requestWillBeSent' && /upload|download|app-center/i.test(e.params.request.url))
      .map((e) => ({ url: e.params.request.url, method: e.params.request.method, postData: (e.params.request.postData || '').slice(0, 400) }));
    console.log('UPLOAD REQUESTS', JSON.stringify(uploadRequests.slice(0, 40), null, 2));
    await page.dumpModal('upload-triggered');

    console.log('AGREE CLICK', await page.clickExact('同意'));
    await sleep(4000);
    await page.dumpModal('after-agree');

    console.log('NEXT CLICK', await page.clickExact('下一步'));
    await sleep(4000);
    await page.dumpModal('wizard-next-1');

    const installButtons = await page.eval(`(() => {
      const modal = document.querySelector('.semi-modal');
      if (!modal) return [];
      return Array.from(modal.querySelectorAll('button, [role="button"], div, span')).filter((e) => {
        const t = (e.innerText || '').trim();
        return t && /安装|完成|确定|下一步/.test(t) && t.length < 24;
      }).map((e) => ({ tag: e.tagName, text: (e.innerText || '').trim(), cls: (e.className || '').toString().slice(0, 120) }));
    })()`);
    console.log('INSTALL CANDIDATES', JSON.stringify(installButtons, null, 2));

    const installClick = await page.eval(`(() => {
      const modal = document.querySelector('.semi-modal');
      if (!modal) return null;
      const buttons = Array.from(modal.querySelectorAll('button')).filter((e) => (e.innerText || '').trim() === '安装');
      const target = buttons.find((e) => /ml-2/.test(e.className || '')) || buttons[buttons.length - 1];
      if (!target) return null;
      target.click();
      return { text: target.innerText.trim(), cls: (target.className || '').toString().slice(0, 120) };
    })()`);
    console.log('INSTALL CLICK', JSON.stringify(installClick));
    await sleep(25000);
    await page.dumpModal('after-install');
    const installedState = await page.eval(`(() => ({
      body: (document.body ? document.body.innerText : '').slice(0, 4000),
      hasError: /错误|失败|无法安装/.test(document.body ? document.body.innerText : '')
    }))()`);
    console.log('INSTALLED STATE', JSON.stringify(installedState, null, 2).slice(0, 8000));
    await page.shot('tests/manual-after-install.png');

    const next = await page.eval(`(() => {
      const modal = document.querySelector('.semi-modal');
      if (!modal) return [];
      return Array.from(modal.querySelectorAll('button, [role="button"], div, span')).filter((e) => {
        const t = (e.innerText || '').trim();
        return t && /下一步|开始安装|安装|解析|上传|继续/.test(t) && t.length < 24;
      }).map((e) => ({ tag: e.tagName, text: (e.innerText || '').trim(), cls: (e.className || '').toString().slice(0, 120) }));
    })()`);
    console.log('NEXT CANDIDATES', JSON.stringify(next, null, 2));
  } finally {
    try { chrome.kill(); } catch {}
  }
}

main().catch((err) => {
  console.error('FATAL', err);
  process.exitCode = 1;
});
