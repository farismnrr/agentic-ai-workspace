import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';

const { chromium } = createRequire(import.meta.url)(process.env.PW as string) as
  typeof import('../../web/node_modules/playwright');
interface BrowserInfo { base: string; email: string; password: string; project: string; task: string }
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

await page.goto(`${info.base}/app/projects/${info.project}`);
const snapshot = await page.evaluate(async (projectId) => {
  const response = await fetch(`/api/projects/${projectId}/git/status`);
  return await response.json() as { available: boolean; files: Array<{ path: string }> };
}, info.project);
check('Git API reports the fixture changes', snapshot.available && snapshot.files.some((file) => file.path === 'src/tracked.txt'), JSON.stringify(snapshot).slice(0, 500));
const head = await page.evaluate(async (projectId) => {
  const response = await fetch(`/api/projects/${projectId}/git/head?path=src%2Ftracked.txt`);
  return await response.json() as { available: boolean; content: string };
}, info.project);
check('Git API returns the committed editor baseline', head.available && head.content.includes('old line'));
await page.getByRole('tab', { name: 'Files' }).click();
const sourceFolder = page.getByText('src', { exact: true });
await sourceFolder.click();
const folderBadge = page.getByRole('tree').locator('.badge[aria-label="Contains Git changes"]');
await folderBadge.first().waitFor({ timeout: 10000 });
const tracked = page.getByRole('tree').getByRole('button', { name: /tracked\.txt/ });
await tracked.waitFor({ timeout: 10000 });
await tracked.locator('.badge').waitFor({ timeout: 10000 });
check('Explorer shows a changed file', (await tracked.innerText()).includes('M'));
check('parent folder rolls up descendant changes',
  await page.getByRole('button', { name: /^src/ }).first().innerText().then((text) => text.includes('•')));
await tracked.click();
await page.locator('.cm-changeGutter').waitFor({ timeout: 10000 });
check('opened editor shows a Git comparison gutter', await page.locator('.cm-changedLineGutter').count() > 0);
await page.screenshot({ path: `${shots}/git-editor-gutter.png` });

await page.goto(`${info.base}/app/projects/${info.project}/tasks/${info.task}`);
await page.getByRole('tab', { name: 'Changes' }).click();
await page.getByText('Current Git changes', { exact: false }).waitFor({ timeout: 10000 });
await page.getByRole('button', { name: 'Show diff' }).first().click();
await page.locator('.cm-merge-b').waitFor({ timeout: 10000 });
check('Changes opens the live HEAD diff', await page.locator('.cm-merge-b').count() > 0);
await page.screenshot({ path: `${shots}/git-changes-diff.png` });

await browser.close();
console.log(failures === 0 ? '\nALL PASSED' : `\n${failures} FAILED`);
process.exit(failures === 0 ? 0 : 1);
