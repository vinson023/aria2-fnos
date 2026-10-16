#!/usr/bin/env node
import { spawn } from 'node:child_process';
import { mkdtempSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const CHROME = process.env.ARIA2_CHROME || 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const DEVICE = process.env.ARIA2_DEVICE || '192.168.x.x';
const CDP_PORT = Number(process.env.ARIA2_CDP_PORT || 9333);
const BASE_URL = `http://${DEVICE}:5666`;
const USERNAME = process.env.ARIA2_USERNAME || 'admin';
const PASSWORD = process.env.ARIA2_PASSWORD || 'YOUR_PASSWORD';
const MODE = process.argv[2] || 'regression';

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

class CdpClient {
  constructor(ws) {
    this.ws = ws;
    this.nextId = 0;
    this.pending = new Map();
    this.events = [];
    ws.addEventListener('message', (event) => {
      const msg = JSON.parse(event.data);
      if (!msg.id) return;
      const waiter = this.pending.get(msg.id);
      if (!waiter) return;
      this.pending.delete(msg.id);
      if (msg.error) waiter.reject(new Error(`${msg.error.message} (${msg.error.code})`));
      else waiter.resolve(msg.result);
      return;
    });
    ws.addEventListener('message', (event) => {
      const msg = JSON.parse(event.data);
      if (msg.method && this.events.length < 500 && (msg.method.startsWith('Network.') || msg.method.startsWith('Page.frameNavigated'))) {
        this.events.push(msg);
      }
    });
  }

  static async connect(wsUrl) {
    const ws = new WebSocket(wsUrl);
    await new Promise((resolve, reject) => {
      ws.addEventListener('open', resolve, { once: true });
      ws.addEventListener('error', () => reject(new Error('WebSocket open failed')), { once: true });
    });
    return new CdpClient(ws);
  }

  send(method, params = {}) {
    const id = ++this.nextId;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      this.ws.send(JSON.stringify({ id, method, params }));
    });
  }

  async eval(expression) {
    const result = await this.send('Runtime.evaluate', {
      expression,
      returnByValue: true,
      awaitPromise: true
    });
    if (result.exceptionDetails) {
      throw new Error(`eval failed: ${result.exceptionDetails.text} ${result.exceptionDetails.exception?.description || ''}`);
    }
    return result.result?.value;
  }

  async screenshot(file) {
    const shot = await this.send('Page.captureScreenshot', { format: 'png', fromSurface: true });
    const fs = await import('node:fs');
    fs.writeFileSync(file, Buffer.from(shot.data, 'base64'));
    return path.resolve(file);
  }

  async waitFor(expression, label, timeoutMs = 20000) {
    const start = Date.now();
    while (Date.now() - start < timeoutMs) {
      const value = await this.eval(expression);
      if (value) return value;
      await sleep(300);
    }
    throw new Error(`waitFor timeout: ${label}`);
  }
}

async function waitJson(url, tries = 60) {
  for (let i = 0; i < tries; i++) {
    try {
      const response = await fetch(url);
      if (response.ok) return response.json();
    } catch {}
    await sleep(250);
  }
  throw new Error(`CDP not ready at ${url}`);
}

async function launch() {
  const profile = mkdtempSync(path.join(os.tmpdir(), 'aria2-ui-'));
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
  chrome.stderr.on('data', () => {});
  await waitJson(`http://127.0.0.1:${CDP_PORT}/json/version`);
  const targetResponse = await fetch(`http://127.0.0.1:${CDP_PORT}/json/new?about:blank`, { method: 'PUT' });
  const target = await targetResponse.json();
  return { chrome, target, profile };
}

async function typeInto(page, selector, text) {
  const info = await page.eval(`(() => {
    const el = document.querySelector(${JSON.stringify(selector)});
    if (!el) return null;
    const rect = el.getBoundingClientRect();
    return { x: rect.x + rect.width / 2, y: rect.y + rect.height / 2 };
  })()`);
  if (!info) throw new Error(`element not found: ${selector}`);
  await page.send('Input.dispatchMouseEvent', { type: 'mousePressed', x: info.x, y: info.y, button: 'left', clickCount: 1 });
  await page.send('Input.dispatchMouseEvent', { type: 'mouseReleased', x: info.x, y: info.y, button: 'left', clickCount: 1 });
  await sleep(200);
  await page.send('Input.insertText', { text });
}

async function clickElement(page, selector) {
  const info = await page.eval(`(() => {
    const el = document.querySelector(${JSON.stringify(selector)});
    if (!el) return null;
    const rect = el.getBoundingClientRect();
    return { x: rect.x + rect.width / 2, y: rect.y + rect.height / 2 };
  })()`);
  if (!info) throw new Error(`element not found: ${selector}`);
  await page.send('Input.dispatchMouseEvent', { type: 'mousePressed', x: info.x, y: info.y, button: 'left', clickCount: 1 });
  await page.send('Input.dispatchMouseEvent', { type: 'mouseReleased', x: info.x, y: info.y, button: 'left', clickCount: 1 });
}

async function performLogin(page) {
  await page.send('Page.enable');
  await page.send('Runtime.enable');
  await page.send('Network.enable');
  await page.send('Page.navigate', { url: `${BASE_URL}/login` });
  await page.waitFor(`document.querySelector('#username') !== null`, 'username input', 15000);
  console.log('LOGIN PAGE READY');

  await typeInto(page, '#username', USERNAME);
  await typeInto(page, '#password', PASSWORD);
  await clickElement(page, 'button[type="submit"]');

  await sleep(6000);
  const afterSubmit = await page.eval(`({
    url: location.href,
    title: document.title,
    hasUsername: !!document.querySelector('#username'),
    body: document.body ? document.body.innerText.slice(0, 500) : ''
  })`);
  console.log('AFTER SUBMIT', JSON.stringify(afterSubmit, null, 2));

  const requests = page.events
    .filter((e) => e.method === 'Network.requestWillBeSent')
    .map((e) => ({ method: e.params.request.method, url: e.params.request.url, postData: e.params.request.postData || '' }));
  console.log('REQUESTS', JSON.stringify(requests, null, 2));

  const responses = page.events
    .filter((e) => e.method === 'Network.responseReceived')
    .map((e) => ({ status: e.params.response.status, url: e.params.response.url }));
  console.log('RESPONSES', JSON.stringify(responses, null, 2));

  const loggedIn = await page.waitFor(
    `document.body && !document.querySelector('#username') && location.href !== ${JSON.stringify(`${BASE_URL}/login`)}`,
    'desktop after login',
    35000
  ).catch(() => false);
  if (!loggedIn) {
    console.error('LOGIN NOT DETECTED');
    process.exitCode = 1;
    throw new Error('login failed');
  }
  console.log('AFTER LOGIN OK', await page.eval(`({ url: location.href, body: document.body.innerText.slice(0, 200) })`));
}

async function openSettings(page) {
  await page.send('Page.navigate', { url: `${BASE_URL}/app/aria2/#!/settings/aria2/bt` });
  await page.waitFor(`document.querySelector('[data-option-key="fnos-tracker-auto-update"]') !== null`, 'tracker option row', 25000);
  const state = await page.eval(`(() => {
    const row = document.querySelector('[data-option-key="fnos-tracker-auto-update"]');
    const select = row.querySelector('select');
    return {
      url: location.href,
      title: document.title,
      value: select ? select.value : null,
      model: select ? angular.element(select).scope().optionValue : null,
      options: select ? Array.from(select.options).map((o) => ({ value: o.value, text: o.text })) : [],
      rowText: row.innerText.slice(0, 200),
      pageLang: document.documentElement.lang || ''
    };
  })()`);
  console.log('SETTINGS STATE', JSON.stringify(state, null, 2));
  return state;
}

async function setTrackerAutoUpdate(page, value) {
  const result = await page.eval(`(() => {
    const row = document.querySelector('[data-option-key="fnos-tracker-auto-update"]');
    if (!row) return { ok: false, reason: 'row not found' };
    const select = row.querySelector('select');
    if (!select) return { ok: false, reason: 'select not found' };
    const before = select.value;
    const targetIndex = Array.from(select.options).findIndex((o) => o.value.endsWith(${JSON.stringify(value)}));
    if (targetIndex < 0) return { ok: false, reason: 'option not found: ' + ${JSON.stringify(value)} };
    select.selectedIndex = targetIndex;
    select.dispatchEvent(new Event('change', { bubbles: true }));
    return { ok: true, before, targetIndex };
  })()`);
  console.log('SET TRACKER', JSON.stringify(result));
  return result;
}

async function readTrackerState(page) {
  return page.eval(`(() => {
    const row = document.querySelector('[data-option-key="fnos-tracker-auto-update"]');
    if (!row) return null;
    const select = row.querySelector('select');
    const icon = row.querySelector('.form-control-icon i');
    return {
      value: select ? select.value : null,
      model: select ? angular.element(select).scope().optionValue : null,
      iconClass: icon ? icon.className : '',
      rowText: row.innerText
    };
  })()`);
}

async function main() {
  const { chrome, target } = await launch();
  try {
    const page = await CdpClient.connect(target.webSocketDebuggerUrl);
    await performLogin(page);

    const state = await openSettings(page);
    const outputDir = path.join(__dirname, 'ui-evidence');
    const fs = await import('node:fs');
    fs.mkdirSync(outputDir, { recursive: true });
    await page.screenshot(path.join(outputDir, 'tracker-before.png'));

    const current = state.model;
    const desired = current === 'false' ? 'true' : 'false';
    await setTrackerAutoUpdate(page, desired);
    await sleep(4000);
    const afterSave = await readTrackerState(page);
    console.log('AFTER SAVE', JSON.stringify(afterSave, null, 2));
    await page.screenshot(path.join(outputDir, `tracker-saved-${desired}.png`));

    await page.send('Page.navigate', { url: `${BASE_URL}/app/aria2/#!/settings/aria2/bt` });
    await page.waitFor(`document.querySelector('[data-option-key="fnos-tracker-auto-update"]') !== null`, 'tracker row after reload', 25000);
    await sleep(1500);
    const afterReload = await readTrackerState(page);
    console.log('AFTER RELOAD', JSON.stringify(afterReload, null, 2));
    console.log('MODEL AFTER RELOAD', afterReload.model);
    await page.screenshot(path.join(outputDir, `tracker-reloaded-${desired}.png`));
  } finally {
    try { chrome.kill(); } catch {}
  }
}

main().catch((error) => {
  console.error('FATAL', error);
  process.exitCode = 1;
});
