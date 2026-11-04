import { createServer, request as httpRequest } from 'node:http';
import { createRequire } from 'node:module';
import { readFileSync } from 'node:fs';
import type { Browser, Locator, Page } from 'playwright-core';

interface Action {
  action: 'goto' | 'click' | 'fill' | 'submit' | 'wait' | 'assert' | 'screenshot';
  path?: string;
  locator_type?: 'css' | 'text' | 'label' | 'placeholder' | 'role' | 'testid';
  locator?: string;
  value?: string;
  check?: 'visible' | 'text' | 'value' | 'url' | 'title';
}

interface Input { prefix: string; actions: Action[]; mobile?: boolean }
interface Step { index: number; action: string; status: 'PASS' | 'FAIL'; detail: string }
interface PlaywrightModule { chromium: { launch(options: object): Promise<Browser> } }

const require = createRequire(import.meta.url);
const { chromium } = require('/opt/browser/node_modules/playwright-core') as PlaywrightModule;
const socketPath = process.env.CAPTURE_SOCKET ?? '';
const maxScreenshot = 5 * 1024 * 1024;
const maxRequests = 128;

function replyError(res: import('node:http').ServerResponse, msg: string): void {
  res.writeHead(502, { 'content-type': 'text/plain; charset=utf-8' }).end(msg);
}

function safePath(pathname: string, prefix: string): boolean {
  try {
    return pathname.startsWith(prefix) && !/%(?:2e|2f|5c)/i.test(pathname) &&
      !decodeURIComponent(pathname).split('/').includes('..');
  } catch {
    return false;
  }
}

function proxyRequest(input: import('node:http').IncomingMessage, response: import('node:http').ServerResponse): void {
  const pathname = new URL(input.url ?? '/', 'http://site').pathname;
  if (!safePath(pathname, '/')) return replyError(response, 'the Site path cannot leave this app');
  const upstream = httpRequest({
    socketPath, path: input.url ?? '/', method: input.method,
    headers: { ...input.headers, host: 'site' },
  }, (up) => {
    response.writeHead(up.statusCode ?? 502, { ...up.headers, connection: 'close' });
    up.pipe(response);
  });
  upstream.on('error', (e: Error) => {
    if (!response.headersSent) replyError(response, e.message.slice(0, 160));
    else response.destroy();
  });
  input.pipe(upstream);
}

async function startProxy(): Promise<{ origin: string; stop: () => Promise<void> }> {
  if (!socketPath) throw new Error('the private Site viewer is unavailable; ask DevOps to check sandbox integration');
  const server = createServer(proxyRequest);
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  const address = server.address();
  if (!address || typeof address === 'string') throw new Error('the local Site bridge did not start');
  return {
    origin: `http://127.0.0.1:${String(address.port)}`,
    stop: () => new Promise((resolve) => server.close(() => resolve())),
  };
}

function locator(page: Page, action: Action): Locator {
  const value = action.locator ?? '';
  switch (action.locator_type) {
    case 'text': return page.getByText(value, { exact: true });
    case 'label': return page.getByLabel(value, { exact: true });
    case 'placeholder': return page.getByPlaceholder(value, { exact: true });
    case 'role': return page.getByRole(value as Parameters<Page['getByRole']>[0]);
    case 'testid': return page.getByTestId(value);
    default: return page.locator(value);
  }
}

async function apply(page: Page, base: string, action: Action): Promise<string> {
  if (action.action === 'goto') {
    const path = action.path ?? '';
    if (path.startsWith('/') || path.includes('..') || path.includes('\\') || path.includes('%')) {
      throw new Error('use a relative page path inside this app');
    }
    await page.goto(new URL(path, base).href, { waitUntil: 'domcontentloaded', timeout: 10000 });
    return `opened ${path || 'the app home page'}`;
  }
  if (action.action === 'screenshot') return 'screenshot captured after all actions';
  if (action.action === 'wait' && !action.locator) {
    await page.waitForTimeout(1000);
    return 'waited';
  }
  if (action.check === 'url') {
    if (!page.url().includes(action.value ?? '')) throw new Error('expected URL was not matched');
    return 'expected URL matched';
  }
  if (action.check === 'title') {
    const title = await page.title();
    if (!title.includes(action.value ?? '')) throw new Error('expected title was not matched');
    return 'expected title matched';
  }
  const target = locator(page, action);
  if (action.action === 'click') { await target.click({ timeout: 5000 }); return 'clicked the selected control'; }
  if (action.action === 'submit') { await target.press('Enter', { timeout: 5000 }); return 'submitted form'; }
  if (action.action === 'wait') { await target.waitFor({ state: 'visible', timeout: 5000 }); return 'waited for element to appear'; }
  if (action.action === 'fill') { await target.fill(action.value ?? '', { timeout: 5000 }); return 'filled the selected field'; }
  if (action.check === 'value') {
    await target.waitFor({ state: 'visible', timeout: 5000 });
    const val = await target.inputValue({ timeout: 5000 });
    if (!val.includes(action.value ?? '')) throw new Error('expected value was not found');
    return 'expected value is present';
  }
  if (action.check === 'visible') { await target.waitFor({ state: 'visible', timeout: 5000 }); return 'selected element is visible'; }
  await target.waitFor({ state: 'visible', timeout: 5000 });
  const text = await target.innerText({ timeout: 5000 });
  if (!text.includes(action.value ?? '')) throw new Error('expected text was not found in the selected element');
  return 'expected text is visible';
}

async function run(input: Input): Promise<object> {
  if (!input.prefix.startsWith('/') || !input.prefix.endsWith('/') || input.actions.length < 1 || input.actions.length > 12) {
    throw new Error('the browser request is outside its limits');
  }
  const proxy = await startProxy();
  const base = new URL(input.prefix, proxy.origin).href;
  let browser: Browser | undefined;
  let screenshot = '';
  const steps: Step[] = [];
  try {
    browser = await chromium.launch({
      executablePath: '/usr/bin/chromium-headless-shell', headless: true,
      args: ['--no-sandbox', '--disable-dev-shm-usage', '--hide-scrollbars'], timeout: 10000,
    });
    const viewport = input.mobile ? { width: 375, height: 667 } : { width: 1280, height: 800 };
    const context = await browser.newContext({ viewport, isMobile: Boolean(input.mobile) });
    let requests = 0;
    await context.route('**/*', async (route) => {
      const url = new URL(route.request().url());
      requests += 1;
      if (requests > maxRequests || url.origin !== proxy.origin || !safePath(url.pathname, input.prefix)) {
        await route.abort('blockedbyclient');
        return;
      }
      await route.continue();
    });
    const page = await context.newPage();
    for (const [index, action] of input.actions.entries()) {
      try {
        const detail = await apply(page, base, action);
        steps.push({ index: index + 1, action: action.action, status: 'PASS', detail });
      } catch (error) {
        const detail = error instanceof Error ? error.message.slice(0, 240) : 'browser action failed';
        steps.push({ index: index + 1, action: action.action, status: 'FAIL', detail });
        break;
      }
    }
    try {
      const image = await page.screenshot({ type: 'png', fullPage: false, timeout: 5000 });
      if (image.length > maxScreenshot) throw new Error('screenshot exceeds 5 MB');
      screenshot = image.toString('base64');
    } catch (error) {
      if (steps.length < input.actions.length) return { steps, png: '' };
      throw error;
    }
    await context.close();
    return { steps, png: screenshot };
  } finally {
    if (browser) await browser.close().catch(() => undefined);
    await proxy.stop();
  }
}

let input: Input;
try {
  input = JSON.parse(readFileSync(0, 'utf8')) as Input;
  const result = await run(input);
  process.stdout.write(`BROWSER_JSON:${JSON.stringify(result)}\n`);
} catch (error) {
  const message = error instanceof Error ? error.message.slice(0, 240) : 'browser test failed';
  process.stdout.write(`BROWSER_JSON:${JSON.stringify({ error: message, steps: [], png: '' })}\n`);
  process.exitCode = 1;
}
