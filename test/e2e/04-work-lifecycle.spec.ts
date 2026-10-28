// External work lifecycle: status, actions, and heartbeat-driven updates.
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';

const { chromium } = createRequire(import.meta.url)(process.env.PW as string) as
  typeof import('../../web/node_modules/playwright');
interface BrowserInfo {
  base: string;
  email: string;
  password: string;
  running: string;
  runningWork: string;
  reported: string;
  reportedWork: string;
  idle: string;
  approval: string;
  token: string;
}
const info = JSON.parse(readFileSync(process.argv[2] as string, 'utf8')) as BrowserInfo;
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

const task = async (id: string): Promise<{ projectId: string }> =>
  page.evaluate(async (taskId) => {
    const response = await fetch(`/api/tasks/${taskId}`);
    return await response.json() as { projectId: string };
  }, id);
const running = await task(info.running);
const path = (id: string, project: string): string => `/app/projects/${project}/tasks/${id}`;
await page.goto(info.base + path(info.running, running.projectId));
const progress = await page.evaluate(async ({ token, workId }) => {
  const response = await fetch('/api/coding', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
      Accept: 'application/json, text/event-stream',
      'MCP-Protocol-Version': '2025-11-25',
    },
    body: JSON.stringify({
      jsonrpc: '2.0', id: 1, method: 'tools/call',
      params: { name: 'report_progress', arguments: { work_id: workId, message: 'I am working.' } },
    }),
  });
  return response.status;
}, { token: info.token, workId: info.runningWork });
check('MCP can record live progress', progress === 200, String(progress));
await page.locator('main header').getByText('Working on it', { exact: true }).waitFor({ timeout: 10000 });
const conversation = page.getByRole('log', { name: 'Conversation' });
const header = page.locator('main header');
check('running work shows the header and thread indicator',
  await header.getByText('Working on it', { exact: true }).count() === 1 &&
  await conversation.getByRole('status').filter({ hasText: 'Working on it' }).count() === 1);

const log = await page.evaluate(async ({ token, workId }) => {
  const response = await fetch('/api/coding', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
      Accept: 'application/json, text/event-stream',
      'MCP-Protocol-Version': '2025-11-25',
    },
    body: JSON.stringify({
      jsonrpc: '2.0', id: 1, method: 'tools/call',
      params: { name: 'log_message', arguments: { work_id: workId, role: 'assistant', text: 'Done.' } },
    }),
  });
  return response.status;
}, { token: info.token, workId: info.runningWork });
check('MCP can report the running task result', log === 200, String(log));
await header.getByText('Reported, not confirmed', { exact: true }).waitFor({ timeout: 10000 });
check('the page changes to reported without a reload',
  await conversation.getByText('Working on it', { exact: true }).count() === 0 &&
  await conversation.getByText('Reported, not confirmed', { exact: true }).count() === 1);
check('reported work has Mark complete and Stop',
  await page.getByRole('button', { name: 'Mark complete' }).count() === 1 &&
  await page.getByRole('button', { name: 'Stop work' }).count() === 1);
await page.getByRole('button', { name: 'Mark complete' }).click();
await header.getByText('Done', { exact: true }).waitFor({ timeout: 10000 });
check('confirmation shows Done and no Proven stamp', await page.getByText('Proven', { exact: true }).count() === 0);

const reported = await task(info.reported);
await page.goto(info.base + path(info.reported, reported.projectId));
await page.locator('main header').getByText('Reported, not confirmed', { exact: true }).waitFor({ timeout: 10000 });
check('reported task has no working indicator',
  await page.getByRole('log', { name: 'Conversation' }).getByText('Working on it', { exact: true }).count() === 0);
check('reported task can be stopped', await page.getByRole('button', { name: 'Stop work' }).count() === 1);

const idle = await task(info.idle);
await page.goto(info.base + path(info.idle, idle.projectId));
await page.locator('main header').getByText('No updates for over 30 min', { exact: true }).waitFor({ timeout: 10000 });
check('idle task has no working indicator',
  await page.getByRole('log', { name: 'Conversation' }).getByText('Working on it', { exact: true }).count() === 0);
check('idle task has Mark complete and Stop',
  await page.getByRole('button', { name: 'Mark complete' }).count() === 1 &&
  await page.getByRole('button', { name: 'Stop work' }).count() === 1);

const approval = await task(info.approval);
await page.goto(info.base + path(info.approval, approval.projectId));
const approvalHeader = page.locator('main header');
const approvalConversation = page.getByRole('log', { name: 'Conversation' });
await approvalHeader.getByText('Ready for approval', { exact: true }).waitFor({ timeout: 10000 });
const approvalLabelCount = await approvalConversation.getByText('Ready for approval', { exact: true }).count();
const completeButtonCount = await page.getByRole('button', { name: 'Mark complete' }).count();
const spinnerCount = await approvalConversation.locator('.loading').count();
const workingLabelCount = await approvalConversation.getByText('Working on it', { exact: true }).count();
check('plan approval keeps its label without completion or spinner',
  approvalLabelCount > 0 && completeButtonCount === 0 && spinnerCount === 0 && workingLabelCount === 0,
  `label=${approvalLabelCount}, mark-complete=${completeButtonCount}, spinner=${spinnerCount}, working=${workingLabelCount}`);
await browser.close();
console.log(failures === 0 ? '\nALL PASSED' : `\n${failures} FAILED`);
process.exit(failures === 0 ? 0 : 1);
