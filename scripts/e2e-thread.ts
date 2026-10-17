// Browser end-to-end check of a task thread, run by scripts/e2e-thread.sh (needs Playwright installed globally).
// The task's model started a page and took two screenshots of it with the real sandbox and browser; this checks
// that the thread shows the agent's markdown as formatted text and the screenshots as images.
// Usage: PW=<playwright dir> node e2e-thread.ts <info.json> <screenshot dir>
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';

const { chromium } = createRequire(import.meta.url)(process.env.PW as string);
const info = JSON.parse(readFileSync(process.argv[2] as string, 'utf8'));
const shots = process.argv[3] as string;
let failures = 0;

function check(name: string, ok: boolean, detail = ''): void {
  if (!ok) failures += 1;
  console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}${detail === '' ? '' : `  - ${detail}`}`);
}

const browser = await chromium.launch();
const page = await (await browser.newContext({ viewport: { width: 1280, height: 900 } })).newPage();

await page.goto(`${info.base}/login`);
await page.getByLabel(/email/i).fill(info.email);
await page.locator('#password').fill(info.password);
await page.getByRole('button', { name: /sign in/i }).click();
await page.waitForURL(/\/app/, { timeout: 15000 });

// A signed-in person who opens the sign-in or register page goes straight on, and Back does not return to it.
const landed = async (url: string): Promise<string> => {
  await page.goto(info.base + url);
  await page.waitForFunction(() => !location.pathname.startsWith('/login') && !location.pathname.startsWith('/register'), null, { timeout: 8000 }).catch(() => undefined);
  return new URL(page.url()).pathname + new URL(page.url()).search;
};
check('signed in, /login goes to the dashboard', (await landed('/login')) === '/app', page.url());
check('...and /register too', (await landed('/register')) === '/app', page.url());
check('...to where ?next= points', (await landed('/login?next=%2Fapp%2Finbox')) === '/app/inbox', page.url());
check('...but never to another site', (await landed('/login?next=//evil.example')) === '/app', page.url());
await page.goto(info.base + info.path);
await page.goto(info.base + '/login');
await page.waitForURL(/\/app$/);
await page.goBack();
check('Back does not return to the sign-in page', !page.url().includes('/login'), page.url());
const stranger = await (await browser.newContext()).newPage();
await stranger.goto(`${info.base}/login`);
check('signed out, /login still shows the form', (await stranger.locator('h1').innerText()).startsWith('Welcome'), await stranger.locator('h1').innerText());
await stranger.close();
await page.goto(info.base + info.path);

const figures = page.locator('figure[aria-label="Screenshot of your app"]');
await figures.first().waitFor({ timeout: 20000 });
check('the thread shows two screenshot cards', (await figures.count()) === 2, `${await figures.count()} cards`);

const sizes: number[] = await page.evaluate(async () => {
  const images = [...document.querySelectorAll('figure[aria-label="Screenshot of your app"] img')] as HTMLImageElement[];
  await Promise.all(images.map((i) => i.decode().catch(() => undefined)));
  return images.map((i) => i.naturalWidth);
});
check('the screenshots load as images (desktop 1280, phone 390 wide)', sizes[0] === 1280 && sizes[1] === 390, sizes.join(', '));
check('each card says what it shows', (await figures.nth(1).innerText()).includes('phone screen'));

const message = page.locator('p', { hasText: 'Ini isi proyek sekarang.' });
const bubble = message.locator('xpath=ancestor::div[contains(@class,"min-w-0")][1]');
check('the message is shown', (await message.count()) === 1);
check('bold text is bold', (await bubble.locator('strong', { hasText: 'Folder di paling atas:' }).count()) === 1);
check('a bullet list has three items', (await bubble.locator('ul li').count()) === 3);
check('the nested item is indented', (await bubble.locator('li', { hasText: 'static/' }).getAttribute('class'))?.includes('ml-4') === true);
check('inline code is code', (await bubble.locator('code', { hasText: 'erp-tahu/' }).count()) === 1);
check('a table has its header and a bold cell', (await bubble.locator('table th', { hasText: 'File' }).count()) === 1 && (await bubble.locator('td strong', { hasText: 'program' }).count()) === 1);
check('a code block is a code block', (await bubble.locator('pre code', { hasText: 'make run' }).count()) === 1);
check('an address becomes a safe link', (await bubble.locator('a[href="https://example.com"][rel~="noopener"]').count()) === 1);

const raw = await bubble.innerText();
check('no markdown marks are left in it', !/\*\*|```|\|---|`/.test(raw), raw.slice(0, 120).replace(/\n/g, ' / '));

await page.screenshot({ path: `${shots}/thread.png`, fullPage: true });
const receipt = page.locator('section[aria-label="Result"]');
const closing = (await receipt.count()) === 0 ? 'no result card' : await receipt.first().innerText();
check('the result card shows its bold text as bold', (await receipt.locator('strong', { hasText: 'teks pucat' }).count()) === 1, closing.replace(/\n/g, ' / '));
check('...and no asterisks', !closing.includes('*'));
check('the page does not scroll sideways', await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth));

await bubble.screenshot({ path: `${shots}/message.png` });
await figures.first().screenshot({ path: `${shots}/shot-card.png` });

await browser.close();
console.log(failures === 0 ? '\nALL PASSED' : `\n${failures} FAILED`);
process.exit(failures === 0 ? 0 : 1);
