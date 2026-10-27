// Browser end-to-end check of sites.
// It uses a fresh, signed-out browser against a real server with a real container and checks that a
// private site is forbidden without the owner's login, that sign-in leads back to it, and that the app
// cannot touch the app around it. Usage: PW=<playwright dir> node e2e-sites.ts <info.json> <screenshot dir>
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';

const { chromium } = createRequire(import.meta.url)(process.env.PW as string) as
  typeof import('../../web/node_modules/playwright');
interface BrowserInfo { base: string; path: string; email: string; password: string }
const info = JSON.parse(readFileSync(process.argv[2] as string, 'utf8')) as BrowserInfo;
const shots = process.argv[3] as string;
const url: string = info.base + info.path;
const mask = (value: string): string => value.replace(/~[^/]+/, '~<ticket>');
const forged = { 'Sec-Fetch-Mode': 'no-cors', 'Sec-Fetch-Dest': 'script', 'Sec-Fetch-Site': 'same-origin' };
let failures = 0;

function check(name: string, ok: boolean, detail = ''): void {
  if (!ok) failures += 1;
  console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}${detail === '' ? '' : `  - ${detail}`}`);
}

const browser = await chromium.launch();
const owner = await browser.newContext();
const page = await owner.newPage();

let res = await page.goto(url);
check('a signed-out browser gets 403', res?.status() === 403, `status ${res?.status() ?? 'no response'}`);
check('...with the private page', /This page is\s+private/.test(await page.locator('h1').innerText()));
await page.screenshot({ path: `${shots}/forbidden.png` });

await page.getByText('Sign in', { exact: true }).click();
await page.waitForURL(/\/login\?next=/);
check('Sign in leads to login with a way back', page.url().includes('next=%2Frun%2F'), mask(page.url()));
await page.getByLabel(/email/i).fill(info.email);
await page.locator('#password').fill(info.password);
await page.getByRole('button', { name: /sign in/i }).click();
await page.waitForURL(/\/run\/.*~/, { timeout: 15000 });
const ticketed = page.url();
check('after login the owner lands on the ticketed address', /\/~\d+\.[0-9a-f]+\//.test(ticketed), mask(ticketed));

await page.waitForFunction(() => document.getElementById('out')?.textContent !== 'running', null, { timeout: 15000 });
const probe = await page.locator('#out').innerText();
check('the app renders for the owner', (await page.locator('h1').innerText()) === 'hello site');
check('the page cannot read cookies', /cookie=blocked/.test(probe), probe);
check('the page cannot use storage', /storage=blocked/.test(probe));
check('the page cannot call the app API with the login', /api=blocked|api=403/.test(probe));
check('the page cannot register a service worker', /sw=blocked/.test(probe));
await page.screenshot({ path: `${shots}/owner.png` });

const stranger = await browser.newContext();
res = await (await stranger.newPage()).goto(ticketed);
check('another browser with the copied address gets 403', res?.status() === 403, `status ${res?.status() ?? 'no response'}`);
const asCurl = await stranger.request.get(ticketed, { headers: { ...forged, 'User-Agent': 'curl/8.0' } });
check('a forged request from another client gets 403', asCurl.status() === 403, `status ${asCurl.status()}`);
const bare = await stranger.request.get(url, { headers: forged });
check('a forged request without the ticket gets 403', bare.status() === 403, `status ${bare.status()}`);

await browser.close();
console.log(failures === 0 ? '\nALL PASSED' : `\n${failures} FAILED`);
process.exit(failures === 0 ? 0 : 1);
