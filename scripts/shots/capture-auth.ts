import { spawn } from 'node:child_process';
import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { connect, type Cdp } from '../../sandbox/capture/cdp.ts';

interface Shot { name: string; path: string; tab?: string; scrollY?: number; settingsKind?: string; databaseOpen?: boolean; caOnly?: boolean; }
const pause = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

function cookieValue(jar: string): string {
  const row = readFileSync(jar, 'utf8').split('\n').find((line) => line.includes('\tsession\t'));
  const value = row?.split('\t').at(-1)?.trim();
  if (!value) throw new Error('registration did not save a session cookie');
  return value;
}

async function pageSocket(port: number): Promise<string> {
  for (let attempt = 0; attempt < 80; attempt += 1) {
    try {
      const pages = await (await fetch(`http://127.0.0.1:${port}/json/list`)).json() as { type: string; webSocketDebuggerUrl: string }[];
      const page = pages.find((target) => target.type === 'page');
      if (page) return page.webSocketDebuggerUrl;
    } catch { /* Chromium is still starting. */ }
    await pause(150);
  }
  throw new Error('Chromium did not expose a page over CDP');
}

async function waitForLoad(cdp: Cdp): Promise<void> {
  await Promise.race([
    new Promise<void>((resolve) => cdp.on('Page.loadEventFired', () => resolve())),
    pause(1500),
  ]);
  await Promise.race([
    cdp.send('Runtime.evaluate', {
      expression: "new Promise((resolve) => document.readyState === 'complete' ? resolve(true) : window.addEventListener('load', () => resolve(true), { once: true }))",
      awaitPromise: true,
    }),
    pause(1000),
  ]);
  await Promise.race([
    cdp.send('Runtime.evaluate', { expression: 'document.fonts.ready', awaitPromise: true }),
    pause(1000),
  ]);
  await pause(300);
  await Promise.race([
    cdp.send('Runtime.evaluate', { expression: 'new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)))', awaitPromise: true }),
    pause(1000),
  ]);
}

async function selectTab(cdp: Cdp, tab: string): Promise<void> {
  const targetTab = tab === 'Database' ? 'Env' : tab;
  for (let attempt = 0; attempt < 20; attempt += 1) {
    const result = await cdp.send('Runtime.evaluate', {
      expression: `(() => { const items = [...document.querySelectorAll('[role="tab"], button, a')]; const target = items.find((item) => item.textContent?.trim() === ${JSON.stringify(targetTab)}); if (!target) return false; target.click(); if (${JSON.stringify(tab)} === 'Database') { const heading = [...document.querySelectorAll('h1,h2,h3,h4')].find((item) => item.textContent?.trim() === 'Database'); heading?.scrollIntoView(); } return true; })()`,
      returnByValue: true,
    });
    if ((result.result as { value?: boolean } | undefined)?.value) {
      await pause(800);
      await cdp.send('Runtime.evaluate', { expression: 'new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)))', awaitPromise: true });
      return;
    }
    await pause(200);
  }
  console.warn(`tab not found: ${tab}; capturing the current page`);
}

function defaults(projectId: string): Shot[] {
  const project = `/app/projects/${projectId}`;
  return [
    { name: 'app-home', path: '/app' },
    { name: 'inbox', path: '/app/inbox' },
    { name: 'tasks', path: '/app/tasks' },
    { name: 'archive', path: '/app/archive' },
    { name: 'sites', path: '/app/sites' },
    { name: 'extensions', path: '/app/extensions' },
    { name: 'projects-new', path: '/app/projects/new' },
    { name: 'projects', path: '/app/projects' },
    { name: 'project-tasks', path: project, tab: 'Tasks' },
    { name: 'project-about', path: project, tab: 'About' },
    { name: 'project-env', path: project, tab: 'Env' },
    { name: 'project-database', path: project, tab: 'Database' },
    { name: 'project-database-open', path: project, tab: 'Database', databaseOpen: true, scrollY: 600 },
    { name: 'project-database-lower', path: project, tab: 'Database', scrollY: 600 },
    { name: 'project-checks', path: project, tab: 'Checks' },
    { name: 'project-files', path: project, tab: 'Files' },
    { name: 'project-internet-access', path: project, tab: 'Internet access' },
    { name: 'settings-models', path: '/app/settings/models' },
    { name: 'settings-persona', path: '/app/settings/persona' },
    { name: 'settings-team', path: '/app/settings/team' },
    { name: 'settings-spaces', path: '/app/settings/spaces' },
    { name: 'settings-connected', path: '/app/settings/connected' },
    { name: 'settings-integrations', path: '/app/settings/integrations' },
    { name: 'settings-activity', path: '/app/settings/activity' },
    { name: 'settings-usage', path: '/app/settings/usage' },
    { name: 'settings-notifications', path: '/app/settings/notifications' },
    { name: 'settings-database', path: '/app/settings/database' },
    { name: 'settings-database-mysql', path: '/app/settings/database', settingsKind: 'MySQL' },
    { name: 'settings-database-mysql-ca-only', path: '/app/settings/database', settingsKind: 'MySQL', caOnly: true },
    { name: 'settings-account', path: '/app/settings/account' },
  ];
}

function customShots(paths: string[]): Shot[] {
  return paths.map((path) => ({
    name: path.replace(/^\/+/, '').replace(/[^a-zA-Z0-9]+/g, '-').replace(/-+$/g, '') || 'home',
    path: path.startsWith('/') ? path : `/${path}`,
  }));
}

async function capture(cdp: Cdp, origin: string, shot: Shot, width: number, height: number, mobile: boolean, out: string): Promise<void> {
  await cdp.send('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile });
  const loaded = waitForLoad(cdp);
  const nav = await cdp.send('Page.navigate', { url: `${origin}${shot.path}` });
  if (nav.errorText) throw new Error(`could not open ${shot.path}: ${String(nav.errorText)}`);
  await loaded;
  if (shot.tab) await selectTab(cdp, shot.tab);
  if (shot.settingsKind) {
    await cdp.send('Runtime.evaluate', { expression: `(() => { const tab = [...document.querySelectorAll('[role="tab"]')].find((item) => item.textContent?.includes(${JSON.stringify(shot.settingsKind)})); tab?.click(); })()` });
    await pause(800);
  }
  if (shot.caOnly) {
    await cdp.send('Runtime.evaluate', { expression: `(() => { const select = document.querySelector('#database-tls'); if (!select) return; select.value = 'ca'; select.dispatchEvent(new Event('change', { bubbles: true })); })()` });
    await pause(800);
  }
  if (shot.databaseOpen) {
    await cdp.send('Runtime.evaluate', { expression: `(() => { const select = document.querySelector('#project-database'); if (!select) return; select.size = 5; select.style.height = 'auto'; select.scrollIntoView({ block: 'center' }); })()` });
  }
  if (shot.scrollY !== undefined) {
    await cdp.send('Runtime.evaluate', { expression: `window.scrollTo(0, ${shot.scrollY})` });
    await pause(350);
  }
  const page = await cdp.send('Runtime.evaluate', { expression: 'location.pathname', returnByValue: true });
  if ((page.result as { value?: string }).value === '/login') throw new Error(`session was not accepted for ${shot.path}`);
  const geometry = await cdp.send('Runtime.evaluate', {
    expression: '({scrollWidth: document.documentElement.scrollWidth, viewport: window.innerWidth})',
    returnByValue: true,
  });
  const bounds = (geometry.result as { value?: { scrollWidth: number; viewport: number } }).value;
  if (bounds && bounds.scrollWidth > bounds.viewport + 1) {
    const offenders = await cdp.send('Runtime.evaluate', {
      expression: '[...document.querySelectorAll("body *")].filter(e => { const r=e.getBoundingClientRect(); return r.width>0 && r.right>innerWidth+1 && r.left < r.right; }).sort((a,b)=>b.getBoundingClientRect().right-a.getBoundingClientRect().right).slice(0,6).map(e=>({tag:e.tagName,class:String(e.className).slice(0,100),right:Math.round(e.getBoundingClientRect().right)}))',
      returnByValue: true,
    });
    throw new Error(`horizontal overflow on ${shot.path} at ${String(width)}px: ${String(bounds.scrollWidth)}px document. Offenders: ${JSON.stringify((offenders.result as { value?: unknown }).value)}`);
  }
  const png = await cdp.send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false });
  const suffix = width === 1280 ? 'desktop' : width === 390 ? 'mobile' : `${String(width)}-desktop`;
  writeFileSync(join(out, `${shot.name}-${suffix}.png`), Buffer.from(String(png.data), 'base64'));
}

async function run(): Promise<void> {
  const [origin, jar, out, projectId, ...paths] = process.argv.slice(2);
  if (!origin || !jar || !out || !projectId) throw new Error('usage: capture-auth.ts <origin> <cookie-jar> <output-dir> <project-id> [path...]');
  const profile = mkdtempSync(join(tmpdir(), 'workspace-shots-chrome-'));
  const home = process.env.HOME || '';
  const chromeBin = process.env.CHROMIUM_BIN || [
    '/usr/bin/chromium-headless-shell',
    '/usr/bin/chromium',
    '/usr/bin/google-chrome',
    join(home, '.local/bin/google-chrome'),
    join(home, '.cache/ms-playwright/chromium-1243/chrome-linux64/chrome'),
  ].find((p) => existsSync(p)) || 'chromium';
  const browser = spawn(chromeBin, [
    '--headless=new', '--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage', '--hide-scrollbars', '--force-color-profile=srgb',
    '--remote-debugging-port=0', `--user-data-dir=${profile}`, 'about:blank',
  ], { stdio: 'ignore' });
  let cdp: Cdp | undefined;
  try {
    let port = 0;
    for (let attempt = 0; attempt < 80 && port === 0; attempt += 1) {
      try { port = Number(readFileSync(join(profile, 'DevToolsActivePort'), 'utf8').split('\n')[0]); }
      catch { await pause(150); }
    }
    if (!port) throw new Error('Chromium did not write DevToolsActivePort');
    cdp = await connect(await pageSocket(port));
    await Promise.all(['Page.enable', 'Runtime.enable', 'Network.enable'].map((method) => cdp?.send(method)));
    await cdp.send('Page.addScriptToEvaluateOnNewDocument', { source: 'const style = document.createElement("style"); style.textContent = "*{animation:none!important;transition:none!important}"; document.documentElement.append(style);' });
    const cookie = await cdp.send('Network.setCookie', { name: 'session', value: cookieValue(jar), url: origin, path: '/', httpOnly: true, sameSite: 'Lax' });
    if (cookie.success === false) throw new Error('Chromium rejected the session cookie');
    const widths = (process.env.SHOTS_WIDTHS ?? '1280,390').split(',').map(Number);
    if (widths.some((width) => !Number.isInteger(width) || width < 320 || width > 2560)) throw new Error('Invalid SHOTS_WIDTHS');
    for (const shot of paths.length ? customShots(paths) : defaults(projectId)) {
      for (const width of widths) {
        const mobile = width < 600;
        await capture(cdp, origin, shot, width, mobile ? 844 : 900, mobile, out);
        const suffix = width === 1280 ? 'desktop' : width === 390 ? 'mobile' : `${String(width)}-desktop`;
        console.log(join(out, `${shot.name}-${suffix}.png`));
      }
    }
  } finally {
    cdp?.close();
    browser.kill('SIGKILL');
    rmSync(profile, { recursive: true, force: true });
  }
}

run().catch((error: Error) => { console.error(error.message); process.exitCode = 1; });
