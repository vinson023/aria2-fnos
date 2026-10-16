import { spawn } from 'node:child_process';
import { mkdtempSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const CHROME = process.env.ARIA2_CHROME || 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const DEVICE = process.env.ARIA2_DEVICE || '192.168.x.x';
const CDP_PORT = Number(process.env.ARIA2_CDP_PORT || 9555);
const BASE_URL = `http://${DEVICE}:5666`;
const USERNAME = process.env.ARIA2_USERNAME || 'admin';
const PASSWORD = process.env.ARIA2_PASSWORD || 'YOUR_PASSWORD';

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
  async waitFor(expression, label, timeout = 20000) {
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
  async clickExact(text) {
    return this.eval(`(() => {
      const candidates = Array.from(document.querySelectorAll('div, span, li, a, button')).filter((e) => e.innerText && e.innerText.trim() === ${JSON.stringify(text)});
      if (!candidates.length) return null;
      candidates.sort((a, b) => {
        const ra = a.getBoundingClientRect(); const rb = b.getBoundingClientRect();
        return (ra.width * ra.height) - (rb.width * rb.height);
      });
      const el = candidates[0];
      el.click();
      return { text: el.innerText.trim(), tag: el.tagName, cls: (el.className || '').toString().slice(0, 120) };
    })()`);
  }
  async clickContains(text) {
    return this.eval(`(() => {
      const candidates = Array.from(document.querySelectorAll('div, span, li, a, button')).filter((e) => e.innerText && e.innerText.includes(${JSON.stringify(text)}) && e.innerText.length < 100);
      if (!candidates.length) return null;
      candidates.sort((a, b) => {
        const ra = a.getBoundingClientRect(); const rb = b.getBoundingClientRect();
        return ((ra.width || 9999) * (ra.height || 9999)) - ((rb.width || 9999) * (rb.height || 9999));
      });
      const el = candidates[0];
      el.click();
      return { text: el.innerText.trim().slice(0, 60), tag: el.tagName };
    })()`);
  }
}

async function waitJson(url, tries = 40) {
  for (let i = 0; i < tries; i++) { try { const r = await fetch(url); if (r.ok) return r.json(); } catch {} await sleep(250); }
  throw new Error('cdp not ready');
}

async function launch() {
  const profile = mkdtempSync(path.join(os.tmpdir(), 'aria2-ac2-'));
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

async function clickSel(page, selector) {
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
  await clickSel(page, 'button[type="submit"]');
  await page.waitFor(`document.body && !document.querySelector('#username') && location.href !== ${JSON.stringify(`${BASE_URL}/login`)}`, 'desktop', 35000);
}

async function dumpInstalled(page) {
  const state = await page.eval(`(() => {
    const aria2 = Array.from(document.querySelectorAll('div, li, article')).filter((e) => e.innerText && /Aria2/.test(e.innerText) && e.innerText.length < 800);
    const buttons = Array.from(document.querySelectorAll('button, [role="button"], div, span')).filter((e) => /卸载|打开|设置|更新|升级|启用|停止/.test(e.innerText || '')).slice(0, 80).map((e) => ({ tag: e.tagName, text: (e.innerText || '').trim().slice(0, 24), cls: (e.className || '').toString().slice(0, 90) }));
    return { body: (document.body ? document.body.innerText : '').slice(0, 2000), aria2: aria2.slice(0, 8).map((e) => ({ text: e.innerText.slice(0, 250), cls: (e.className || '').toString().slice(0, 120) })), buttons };
  })()`);
  console.log('INSTALLED STATE', JSON.stringify(state, null, 2));
  await page.shot('tests/appcenter-installed.png');
}

async function main() {
  const { chrome, target } = await launch();
  try {
    const page = await Client.connect(target.webSocketDebuggerUrl);
    await login(page);
    console.log('OPEN APP CENTER', await page.clickExact('应用中心'));
    await page.waitFor(`document.body && document.body.innerText.includes('已安装')`, 'app center ready', 20000);
    console.log('CLICK INSTALLED', await page.clickExact('已安装'));
    await sleep(2500);
    await dumpInstalled(page);
    const cardClick = await page.eval(`(() => {
      const el = Array.from(document.querySelectorAll('div, article, li')).find((e) => /^Aria2\\n/.test(e.innerText || '') && (e.className || '').toString().includes('cursor-pointer'));
      if (!el) return null;
      el.click();
      return { text: el.innerText.slice(0, 80), cls: (el.className || '').toString().slice(0, 120) };
    })()`);
    console.log('CARD CLICK', JSON.stringify(cardClick));
    await sleep(3000);
    const detail = await page.eval(`(() => {
  const buttons = Array.from(document.querySelectorAll('button')).map((e) => ({ text: (e.innerText || '').trim(), aria: e.getAttribute('aria-label'), title: e.title, cls: (e.className || '').toString().slice(0, 100) }));
  const icons = Array.from(document.querySelectorAll('svg, i, img')).map((e) => ({ tag: e.tagName, aria: e.getAttribute('aria-label'), cls: (e.className || '').toString().slice(0, 80), src: e.getAttribute('src') || '' })).slice(0, 100);
  const body = (document.body ? document.body.innerText : '').slice(0, 2200);
  return { body, buttons, icons };
})()`);
console.log('DETAIL', JSON.stringify(detail, null, 2));
    const emptyButtons = await page.eval(`Array.from(document.querySelectorAll('button')).filter((b) => !(b.innerText || '' ).trim()).map((b) => ({ cls: (b.className || '').toString().slice(0, 120), parent: b.parentElement ? (b.parentElement.className || '').toString().slice(0, 120) : '', aria: b.getAttribute('aria-label'), title: b.title, html: b.outerHTML.slice(0, 300) })).slice(0, 30)`);
    console.log('EMPTY BUTTONS', JSON.stringify(emptyButtons, null, 2));
    const menuItems = await page.eval(`Array.from(document.querySelectorAll('[role="menuitem"], [class*="dropdown"] button, [class*="more"] button, [class*="More"] button')).map((b) => ({ tag: b.tagName, text: (b.innerText || '' ).trim(), cls: (b.className || '').toString().slice(0, 100), aria: b.getAttribute('aria-label') })).slice(0, 40)`);
    console.log('MENU ITEMS', JSON.stringify(menuItems, null, 2));
    const splitClick = await page.eval(`(() => { const b = document.querySelector('button[aria-haspopup="true"]'); if (!b) return null; b.click(); return { cls: (b.className || '').toString().slice(0, 120) }; })()`);
    console.log('SPLIT CLICK', JSON.stringify(splitClick));
    await sleep(1500);
    const popup = await page.eval(`(() => ({
      items: Array.from(document.querySelectorAll('[role="menuitem"], [role="menu"] *, .semi-dropdown, .semi-popover, [class*="dropdown"] *, [class*="popover"] *')).filter((e) => (e.innerText || '').trim()).map((e) => ({ tag: e.tagName, text: (e.innerText || '').trim().slice(0, 60), cls: (e.className || '').toString().slice(0, 90) })).slice(0, 60),
      bodyTail: (document.body ? document.body.innerText : '').slice(-1200)
    }))()`);
    console.log('POPUP', JSON.stringify(popup, null, 2));
    await page.shot('tests/appcenter-popup.png');
    console.log('UNINSTALL CLICK', await page.eval(`(() => { const item = Array.from(document.querySelectorAll('li.semi-dropdown-item')).find((e) => e.innerText && e.innerText.trim() === '卸载'); if (!item) return null; item.click(); return item.innerText.trim(); })()`));
    await sleep(2500);
    const confirm = await page.eval(`(() => ({
      bodyTail: (document.body ? document.body.innerText : '').slice(-1600),
      buttons: Array.from(document.querySelectorAll('button')).map((e) => ({ text: (e.innerText || '').trim(), cls: (e.className || '').toString().slice(0, 90), aria: e.getAttribute('aria-label') })).slice(0, 60),
      radios: Array.from(document.querySelectorAll('input[type="radio"], label, [role="radio"]')).map((e) => ({ text: (e.innerText || '').trim().slice(0, 80), cls: (e.className || '').toString().slice(0, 80) })).slice(0, 30)
    }))()`);
    console.log('UNINSTALL CONFIRM', JSON.stringify(confirm, null, 2));
    await page.shot('tests/appcenter-uninstall-confirm.png');
    console.log('OK CLICK', await page.clickExact('确定'));
    await sleep(3500);
    const next = await page.eval(`(() => ({
      bodyTail: (document.body ? document.body.innerText : '').slice(-1800),
      radios: Array.from(document.querySelectorAll('input[type="radio"], [role="radio"], label')).map((e) => ({ text: (e.innerText || '').trim().slice(0, 100), cls: (e.className || '').toString().slice(0, 80), checked: e.checked || e.getAttribute('aria-checked') })).slice(0, 40),
      buttons: Array.from(document.querySelectorAll('button')).filter((e) => (e.innerText || '').trim()).map((e) => ({ text: (e.innerText || '').trim().slice(0, 40), cls: (e.className || '').toString().slice(0, 90) })).slice(0, 60),
      inputs: Array.from(document.querySelectorAll('input')).map((e) => ({ type: e.type, name: e.name, placeholder: e.placeholder })).slice(0, 30)
    }))()`);
    console.log('UNINSTALL NEXT', JSON.stringify(next, null, 2));
    await page.shot('tests/appcenter-uninstall-next.png');
    console.log('CLEAR CLICK', await page.clickContains('清空所有应用配置'));
    await sleep(1200);
    const wizardButtons = await page.eval(`Array.from(document.querySelectorAll('button')).filter((e) => (e.innerText || '').trim() === '卸载' || (e.innerText || '').trim() === '取消').map((e) => ({ text: (e.innerText || '').trim(), cls: (e.className || '').toString().slice(0, 120), inModal: !!e.closest('.semi-modal, [role="dialog"]'), aria: e.getAttribute('aria-disabled') }))`);
    console.log('WIZARD BUTTONS', JSON.stringify(wizardButtons, null, 2));
    const finalClick = await page.eval(`(() => {
      const buttons = Array.from(document.querySelectorAll('button')).filter((e) => (e.innerText || '').trim() === '卸载');
      const target = buttons.find((e) => e.closest('.semi-modal, [role="dialog"]')) || buttons.find((e) => /semi-button-primary/.test(e.className)) || buttons[buttons.length - 1];
      if (!target) return null;
      target.click();
      return { text: target.innerText.trim(), cls: (target.className || '').toString().slice(0, 120) };
    })()`);
    console.log('FINAL UNINSTALL CLICK', JSON.stringify(finalClick));
    await sleep(12000);
    const afterUninstall = await page.eval(`({ bodyTail: (document.body ? document.body.innerText : '').slice(-1600) })`);
    console.log('AFTER UNINSTALL', JSON.stringify(afterUninstall, null, 2));
    await page.shot('tests/appcenter-after-uninstall.png');
    if (process.env.UNINSTALL_ONLY) {
      console.log('UNINSTALL_ONLY done');
      return;
    }
    console.log('MANUAL CLICK', await page.clickExact('手动安装'));
    await sleep(3500);
    const manual = await page.eval(`(() => ({
      bodyTail: (document.body ? document.body.innerText : '').slice(-1800),
      fileInputs: Array.from(document.querySelectorAll('input[type="file"]')).map((e) => ({ name: e.name, accept: e.accept, cls: (e.className || '').toString().slice(0, 120), ids: e.id }) ),
      inputs: Array.from(document.querySelectorAll('input')).map((e) => ({ type: e.type, name: e.name, placeholder: e.placeholder, value: e.value, cls: (e.className || '').toString().slice(0, 80) })).slice(0, 80),
      buttons: Array.from(document.querySelectorAll('button')).filter((e) => (e.innerText || '').trim()).map((e) => ({ text: (e.innerText || '').trim().slice(0, 40), cls: (e.className || '').toString().slice(0, 90) })).slice(0, 80)
    }))()`);
    console.log('MANUAL STATE', JSON.stringify(manual, null, 2));
    await page.shot('tests/appcenter-manual.png');
    await page.shot('tests/appcenter-aria2-card.png');
    const listResp = await fetch(`http://127.0.0.1:${CDP_PORT}/json/list`);
    const list = await listResp.json();
    console.log('TARGETS', list.map((t) => t.url).join('\n'));
  } finally {
    try { chrome.kill(); } catch {}
  }
}

main().catch((e) => { console.error('FATAL', e); process.exitCode = 1; });
