// Browser check of the New project dialog with a connected GitHub, run by scripts/e2e-projects.sh. GitHub's answers
// are faked in the page, so this checks the screen, not GitHub.
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

const repos = Array.from({ length: 60 }, (_, i) => ({
  name: `ann/${i % 7 === 0 ? 'shop' : 'repo'}-${String(i).padStart(2, '0')}`, description: '', private: i % 5 === 0,
}));

const browser = await chromium.launch();
const page = await (await browser.newContext({ viewport: { width: 1280, height: 900 } })).newPage();
let created: unknown;

await page.route('**/api/integrations/github', (route) =>
  route.fulfill({ json: { configured: true, connected: true, login: 'ann', name: 'Ann', reposUrl: '' } }));
await page.route('**/api/integrations/github/repos', (route) =>
  route.request().method() === 'POST'
    ? route.fulfill({ status: 201, json: { name: 'ann/new-shop', private: true } })
    : route.fulfill({ json: { repos } }));
await page.route('**/api/projects/*/import', (route) => {
  created = route.request().postDataJSON();
  return route.fulfill({ status: 204, body: '' });
});

await page.goto(`${info.base}/login`);
await page.getByLabel(/email/i).fill(info.email);
await page.locator('#password').fill(info.password);
await page.getByRole('button', { name: /sign in/i }).click();
await page.waitForURL(/\/app/, { timeout: 15000 });
await page.goto(`${info.base}/app/projects`);
await page.getByRole('button', { name: /new project/i }).first().click();

const dialog = page.locator('dialog[open]');
check('three ways to start are offered', (await dialog.locator('.join button').allInnerTexts()).join('|') === 'Empty project|From GitHub|New repository');

await dialog.getByRole('button', { name: 'From GitHub' }).click();
const list = dialog.locator('#project-repo-list');
await list.waitFor();
const box = await list.boundingBox();
check('the list is short and scrolls', (box?.height ?? 999) < 220 && (await list.evaluate((el) => el.scrollHeight > el.clientHeight)), `${box?.height}px high`);

await dialog.getByPlaceholder('Search your repositories').fill('shop-0');
const shown = await list.locator('li[role=option]').count();
check('typing narrows the list', shown > 0 && shown < 10, `${shown} left`);
await dialog.getByPlaceholder('Search your repositories').fill('zzz');
check('no match says so', (await list.innerText()).includes('No repository matches'));
await dialog.getByPlaceholder('Search your repositories').fill('shop-14');
await list.locator('li[role=option] button').first().click();
await page.waitForTimeout(300);
check('picking fills the project name', (await dialog.locator('#project-name').inputValue()) === 'shop-14');
await dialog.screenshot({ path: `${shots}/new-project-github.png` });

await dialog.getByRole('button', { name: 'Create and import' }).click();
await page.waitForTimeout(800);
check('the chosen repository is imported', JSON.stringify(created) === '{"repo":"ann/shop-14"}', JSON.stringify(created));

await page.goto(`${info.base}/app/projects`);
await page.getByRole('button', { name: /new project/i }).first().click();
const second = page.locator('dialog[open]');
await second.getByRole('button', { name: 'New repository' }).click();
await second.locator('#project-name').fill('Warung Kopi');
await second.getByRole('button', { name: 'New repository' }).click();
check('the repository name follows the project name', (await second.locator('#project-repo-name').inputValue()) === 'warung-kopi');
await second.locator('#project-repo-name').fill('kopi-site');
await second.getByRole('button', { name: 'Create repository and project' }).click();
await page.waitForTimeout(800);
check('a new repository is made and imported', JSON.stringify(created) === '{"repo":"ann/new-shop"}', JSON.stringify(created));

await browser.close();
console.log(failures === 0 ? '\nALL PASSED' : `\n${failures} FAILED`);
process.exit(failures === 0 ? 0 : 1);
