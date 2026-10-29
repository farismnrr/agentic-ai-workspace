import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
const { chromium } = createRequire(import.meta.url)(process.env.PW as string) as typeof import('../../web/node_modules/playwright');
const info = JSON.parse(readFileSync(process.argv[2] as string, 'utf8')) as {base:string;email:string;password:string;task:string;project:string};
const browser = await chromium.launch({executablePath:'/usr/bin/chromium-headless-shell',args:['--no-sandbox']});
try {
  const page = await browser.newPage();
  await page.goto(info.base+'/login');
  await page.getByLabel(/email/i).fill(info.email);
  await page.locator('#password').fill(info.password);
  await page.getByRole('button',{name:/sign in/i}).click();
  await page.waitForURL(/\/app/,{timeout:15000});
  await page.goto(`${info.base}/app/projects/${info.project}/tasks/${info.task}`);
  await page.getByText('Auto Done · Unverified',{exact:true}).waitFor({timeout:15000});
  console.log('PASS browser auto-done unverified badge');
} finally { await browser.close(); }
