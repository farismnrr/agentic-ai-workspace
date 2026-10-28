// Browser end-to-end check of a task thread.
// The task's model started a page and took two screenshots of it with the real sandbox and browser; this checks
// that the thread shows the agent's markdown as formatted text and the screenshots as images.
// Usage: PW=<playwright dir> node e2e-thread.ts <info.json> <screenshot dir>
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';

const { chromium } = createRequire(import.meta.url)(process.env.PW as string) as
  typeof import('../../web/node_modules/playwright');
interface BrowserInfo { base: string; path: string; email: string; password: string }
const info = JSON.parse(readFileSync(process.argv[2] as string, 'utf8')) as BrowserInfo;
const shots = process.argv[3] as string;
let failures = 0;

function check(name: string, ok: boolean, detail = ''): void {
  if (!ok) failures += 1;
  console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}${detail === '' ? '' : `  - ${detail}`}`);
}

const browser = await chromium.launch({
  executablePath: '/usr/bin/chromium-headless-shell',
  args: ['--no-sandbox'],
});
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

// The first thing the thread shows is the look question: every option has a picture, the best fit is marked.
const form = page.locator('section[aria-label="Question for you"]');
await form.locator('img').first().waitFor({ timeout: 20000 });
const pictures: number[] = await form.evaluate(async (el) => {
  const images = [...el.querySelectorAll('img')] as HTMLImageElement[];
  await Promise.all(images.map((i) => i.decode().catch(() => undefined)));
  return images.map((i) => i.naturalWidth);
});
check('the look question shows eight pictures', pictures.length === 8 && pictures.every((w) => w === 800), pictures.join(', '));
check('the best fit is marked Recommended, once', (await form.locator('.badge', { hasText: 'Recommended' }).count()) === 1);
check('the send button waits for a choice', await form.getByRole('button', { name: 'Send answers' }).isDisabled());
await form.screenshot({ path: `${shots}/look.png` });
await form.locator('label', { hasText: 'Gaya corporate' }).click();
await form.getByRole('button', { name: 'Send answers' }).click();
await page.getByText('You answered: Gaya corporate').waitFor({ timeout: 10000 });
check('choosing a picture answers the question', true);
await page.locator('section[aria-label="Result"]').waitFor({ timeout: 120000 });

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

// The Changes tab: each changed file once, as a diff against how the task found it.
await page.goto(info.base + info.path);
await page.getByRole('tab', { name: 'Changes' }).click();
await page.getByText('2 files changed').waitFor({ timeout: 15000 });
const readme = page.locator('section[aria-label="Changes in README.md"]');
const notes = page.locator('section[aria-label="Changes in notes.md"]');
check('a file the task made is Added with its lines counted', (await notes.innerText()).includes('Added') && (await notes.innerText()).includes('+3'), (await notes.innerText()).replace(/\n/g, ' '));
await readme.getByRole('button').first().waitFor();
await readme.getByRole('button').first().click();
await readme.locator('.cm-deletedChunk').waitFor({ timeout: 10000 });
check('a changed file is Modified, +1 -1', (await readme.innerText()).includes('Modified') && (await readme.innerText()).includes('+1') && (await readme.innerText()).includes('−1'));
check('unified shows the removed line among the added', (await readme.locator('.cm-deletedChunk').innerText()).includes('two'));
await page.getByRole('button', { name: 'Split' }).click();
await readme.locator('.cm-merge-a').waitFor({ timeout: 10000 });
check('split shows old and new side by side', (await readme.locator('.cm-merge-b').count()) === 1);
await page.screenshot({ path: `${shots}/changes.png` });
await page.getByRole('group', { name: 'How to compare' }).screenshot({ path: `${shots}/toggle.png` });

// The Files tab: the project's own tree, a real editor, saving by hand, and the agent's changes arriving live.
const projectId = String(info.path).split('/')[3] as string;
const api = (method: string, path: string, body?: unknown): Promise<{ status: number; json: Record<string, unknown> }> =>
  page.evaluate(async ([m, p, b]) => {
    const init: RequestInit = { method: m as string, headers: { 'Content-Type': 'application/json' } };
    if (b !== undefined) init.body = JSON.stringify(b);
    const res = await fetch(p as string, init);
    return { status: res.status, json: res.status === 204 ? {} : await res.json() };
  }, [method, `/api/projects/${projectId}${path}`, body] as const);
await page.goto(info.base + info.path);
await page.getByRole('tab', { name: 'Files' }).click();
const tree = page.getByRole('tree', { name: 'Project files' });
await tree.waitFor({ timeout: 15000 });
check('the Files tab shows the project tree, with the file the task changed', (await tree.getByRole('button', { name: /index\.html/ }).count()) === 1);
await tree.getByRole('button', { name: /index\.html/ }).click();
const editor = page.locator('.cm-content');
await editor.waitFor({ timeout: 15000 });
check('the file opens in an editor with its text', (await editor.innerText()).includes('Kopi enak setiap pagi'));
await page.screenshot({ path: `${shots}/files.png` });

await editor.click();
await page.keyboard.press('Control+End');
await page.keyboard.type('\n<!-- by hand -->');
check('typing marks the file Unsaved', (await page.getByText('Unsaved', { exact: true }).count()) === 1);
await page.keyboard.press('Control+s');
await page.getByText('Saved', { exact: true }).waitFor({ timeout: 8000 });
check('Ctrl+S saves it on disk', String((await api('GET', '/file?path=index.html')).json.content).includes('by hand'));

const now = String((await api('GET', '/file?path=index.html')).json.hash);
await api('PUT', '/file', { path: 'index.html', content: 'agent version', baseHash: now });
await page.locator('.cm-content', { hasText: 'agent version' }).waitFor({ timeout: 8000 });
check('a change by the agent shows in the open editor at once', true);
await api('PUT', '/file', { path: 'fresh.txt', content: 'x', baseHash: '' });
// The panel is narrow, so the tree and the file take turns: go back to the tree to see the new file.
await page.getByRole('button', { name: 'Back to files' }).click();
await tree.getByRole('button', { name: /fresh\.txt/ }).waitFor({ timeout: 8000 });
check('a new file shows in the tree at once', true);
await tree.getByRole('button', { name: /index\.html/ }).click();
await page.locator('.cm-content', { hasText: 'agent version' }).waitFor({ timeout: 8000 });

await editor.click();
await page.keyboard.press('Control+End');
await page.keyboard.type(' MY EDIT');
const seen = String((await api('GET', '/file?path=index.html')).json.hash);
await api('PUT', '/file', { path: 'index.html', content: 'agent again', baseHash: seen });
await page.getByRole('alert').getByText('changed on disk').waitFor({ timeout: 8000 });
check('an edit in progress is not overwritten: the person is asked', true);
await page.screenshot({ path: `${shots}/conflict.png` });
await page.getByRole('button', { name: 'Keep mine' }).click();
await page.keyboard.press('Control+s');
await page.getByText('Saved', { exact: true }).waitFor({ timeout: 8000 });
check('Keep mine then Save puts their edit on disk', String((await api('GET', '/file?path=index.html')).json.content).includes('MY EDIT'));

await browser.close();
console.log(failures === 0 ? '\nALL PASSED' : `\n${failures} FAILED`);
process.exit(failures === 0 ? 0 : 1);
