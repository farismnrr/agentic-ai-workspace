// Opens one page of the running app in a headless browser and prints one line, CAPTURE_JSON:<json>, with a
// screenshot (base64 PNG) and what the page says about itself. Usage: node capture.ts '<request json>'
import { spawn } from 'node:child_process';
import { connect, type Cdp } from './cdp.ts';
import { inspectPage } from './inspect.ts';

interface Request {
  url: string;
  width: number;
  height: number;
  y: number;
  overview: boolean;
  mobile: boolean;
}

const DEBUG_PORT = 9222;
const MAX_PAGE = 12000;
const MAX_EDGE = 1568;
const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

async function pageSocket(): Promise<string> {
  for (let i = 0; i < 100; i += 1) {
    try {
      const list = (await (await fetch(`http://127.0.0.1:${String(DEBUG_PORT)}/json/list`)).json()) as { type: string; webSocketDebuggerUrl: string }[];
      const page = list.find((target) => target.type === 'page');
      if (page) return page.webSocketDebuggerUrl;
    } catch { /* the browser is still starting */ }
    await sleep(200);
  }
  throw new Error('the browser did not start');
}

function watch(cdp: Cdp, console: string[], failed: string[]) {
  const urls = new Map<string, string>();
  cdp.on('Network.requestWillBeSent', (p) => { urls.set(String(p.requestId), String((p.request as { url: string }).url)); });
  cdp.on('Network.loadingFailed', (p) => {
    if (!p.canceled && failed.length < 10) failed.push(urls.get(String(p.requestId)) ?? String(p.errorText));
  });
  cdp.on('Runtime.exceptionThrown', (p) => {
    const detail = p.exceptionDetails as { text: string; exception?: { description?: string } };
    if (console.length < 10) console.push(`error: ${detail.exception?.description ?? detail.text}`.slice(0, 200));
  });
  cdp.on('Runtime.consoleAPICalled', (p) => {
    if ((p.type === 'error' || p.type === 'warning') && console.length < 10) {
      const text = (p.args as { value?: unknown; description?: string }[]).map((a) => String(a.value ?? a.description ?? '')).join(' ');
      console.push(`${String(p.type)}: ${text}`.slice(0, 200));
    }
  });
}

async function run(req: Request) {
  const browser = spawn('chromium-headless-shell', [
    '--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage', '--hide-scrollbars', '--force-color-profile=srgb',
    `--remote-debugging-port=${String(DEBUG_PORT)}`, '--user-data-dir=/tmp/chrome', 'about:blank',
  ], { stdio: 'ignore' });
  try {
    const cdp = await connect(await pageSocket());
    const messages: string[] = [], failed: string[] = [];
    watch(cdp, messages, failed);
    await Promise.all(['Page', 'Runtime', 'Network'].map((domain) => cdp.send(`${domain}.enable`)));
    await cdp.send('Emulation.setDeviceMetricsOverride', { width: req.width, height: req.height, deviceScaleFactor: 1, mobile: req.mobile });
    const loaded = new Promise<void>((resolve) => { cdp.on('Page.loadEventFired', () => { resolve(); }); });
    const nav = await cdp.send('Page.navigate', { url: req.url });
    if (nav.errorText) throw new Error(`the page did not open: ${String(nav.errorText)}`);
    await Promise.race([loaded, sleep(20000)]);
    await cdp.send('Runtime.evaluate', { expression: 'document.fonts.ready', awaitPromise: true });
    await sleep(600);
    const inspected = await cdp.send('Runtime.evaluate', { expression: `(${inspectPage.toString()})()`, returnByValue: true });
    const metrics = (inspected.result as { value: { pageHeight: number } }).value;
    const total = Math.min(metrics.pageHeight, MAX_PAGE);
    const clip = req.overview
      ? { x: 0, y: 0, width: req.width, height: total, scale: Math.min(1, MAX_EDGE / Math.max(req.width, total)) }
      : { x: 0, y: Math.min(req.y, Math.max(total - 1, 0)), width: req.width, height: 0, scale: 1 };
    if (!req.overview) clip.height = Math.min(req.height, total - clip.y);
    const shot = await cdp.send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: true, clip });
    console.log('CAPTURE_JSON:' + JSON.stringify({ ok: true, png: shot.data, shot: clip, metrics, console: messages, failed }));
  } finally {
    browser.kill('SIGKILL');
  }
}

// Anything the caller leaves out gets a default, so the browser never sees an undefined parameter.
function parse(text: string): Request {
  const given = JSON.parse(text) as Partial<Request>;
  return {
    url: given.url ?? 'about:blank',
    width: given.width ?? 1280,
    height: given.height ?? 800,
    y: given.y ?? 0,
    overview: given.overview ?? false,
    mobile: given.mobile ?? false,
  };
}

run(parse(process.argv[2] ?? '{}')).then(() => process.exit(0), (error: Error) => {
  console.log('CAPTURE_JSON:' + JSON.stringify({ ok: false, error: error.message }));
  process.exit(0);
});
