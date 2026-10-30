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
await page.addInitScript(() => {
  const NativeEventSource = window.EventSource;
  const streams: Array<{ reconnect: () => void; send: (name: string, data: unknown) => void }> = [];
  Object.assign(window, { __taskStreams: streams });

  class ThreadStream extends EventTarget {
    private native: EventSource;
    private readonly url: string;
    onerror: ((event: Event) => void) | null = null;
    onopen: ((event: Event) => void) | null = null;
    private state = EventSource.CONNECTING;

    constructor(url: string) {
      super();
      this.url = url;
      this.native = this.open();
      if (url.includes('/api/tasks/') && url.endsWith('/stream')) {
        streams.push({
          reconnect: () => {
            this.native.close();
            this.state = EventSource.CONNECTING;
            this.onerror?.(new Event('error'));
            this.native = this.open();
          },
          send: (name, data) => this.dispatchEvent(new MessageEvent(name, { data: JSON.stringify(data) })),
        });
      }
    }

    private open(): EventSource {
      const source = new NativeEventSource(this.url);
      source.onopen = (event): void => {
        this.state = EventSource.OPEN;
        this.onopen?.(event);
      };
      source.addEventListener('update', (event) => {
        const update = event as MessageEvent<string>;
        this.dispatchEvent(new MessageEvent('update', { data: update.data, lastEventId: update.lastEventId }));
      });
      source.addEventListener('changed', (event) => this.dispatchEvent(event));
      source.onerror = (event): void => this.onerror?.(event);
      return source;
    }

    close(): void {
      this.native.close();
      this.state = EventSource.CLOSED;
    }

    get readyState(): number { return this.state; }
  }

  window.EventSource = ThreadStream as unknown as typeof EventSource;
});
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

await page.evaluate(() => {
  const streams = (window as Window & { __taskStreams: Array<{ send: (name: string, data: unknown) => void }> }).__taskStreams;
  streams[0]?.send('delta', { text: 'A persisted reply is being written.' });
});
await conversation.getByText('A persisted reply is being written.', { exact: true }).waitFor();
await page.evaluate(async (taskId) => {
  const [taskResponse, eventResponse] = await Promise.all([
    fetch(`/api/tasks/${taskId}`), fetch(`/api/tasks/${taskId}/events?after=0`),
  ]);
  const task = await taskResponse.json() as Record<string, unknown>;
  const events = await eventResponse.json() as Array<{ seq: number; version: number }>;
  const seq = Math.max(0, ...events.map((event) => event.seq)) + 1;
  const version = Math.max(0, ...events.map((event) => event.version)) + 1;
  const message = {
    id: `browser-message-${seq}`, seq, version, time: new Date().toISOString(),
    kind: 'message', text: 'A persisted reply is being written.',
  };
  const streams = (window as Window & { __taskStreams: Array<{ send: (name: string, data: unknown) => void }> }).__taskStreams;
  streams[0]?.send('update', { task, events: [message] });
}, info.running);
await page.waitForTimeout(100);
check('persisted reply replaces its live draft while work is still running',
  await conversation.getByText('A persisted reply is being written.', { exact: true }).count() === 1,
  `rendered=${await conversation.getByText('A persisted reply is being written.', { exact: true }).count()}`);
await page.evaluate(() => {
  const streams = (window as Window & {
    __taskStreams: Array<{ reconnect: () => void }>;
  }).__taskStreams;
  streams[0]?.reconnect();
});
await page.waitForTimeout(300);
check('reconnect reconciles the persisted reply without a stale draft',
  await conversation.getByText('A persisted reply is being written.', { exact: true }).count() === 1,
  `rendered=${await conversation.getByText('A persisted reply is being written.', { exact: true }).count()}`);

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
await page.getByRole('button', { name: 'Stop work' }).click();
await page.locator('main header').getByText('Done', { exact: true }).waitFor({ timeout: 10000 });
const manuallyStopped = await page.evaluate(async (taskId) => {
  const response = await fetch(`/api/tasks/${taskId}`);
  return await response.json() as { status: string; work: { state: string } };
}, info.reported);
check('manual Stop is Done, not Needs Attention, while retaining stopped provenance',
  manuallyStopped.status === 'done' && manuallyStopped.work.state === 'stopped' &&
  await page.locator('main header').getByText('Needs attention', { exact: true }).count() === 0);

const idle = await task(info.idle);
await page.goto(info.base + path(info.idle, idle.projectId));
await page.locator('main header').getByText('Inactive — awaiting automatic closure', { exact: true }).waitFor({ timeout: 10000 });
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
for (const width of [390, 820, 1024, 1440]) {
  await page.setViewportSize({ width, height: 900 });
  await page.goto(info.base + path(info.approval, approval.projectId));
  await page.locator('main header').getByText('Ready for approval', { exact: true }).waitFor({ timeout: 10000 });
  const overflow = await page.evaluate(() => ({ document: document.documentElement.scrollWidth, viewport: window.innerWidth }));
  check(`approval task ${String(width)}px has no horizontal overflow`, overflow.document <= overflow.viewport + 1, JSON.stringify(overflow));
  await page.getByRole('tab', { name: 'Plan', exact: true }).click();
  check(`plan panel visible at ${String(width)}px`, await page.getByRole('tabpanel', { name: 'plan' }).isVisible());
  await page.screenshot({ path: `${shots}/approval-plan-${String(width)}.png` });
  for (const tab of ['Documents', 'Team', 'Changes', 'Files']) {
    await page.getByRole('tab', { name: tab, exact: true }).click();
    check(`${tab} panel visible at ${String(width)}px`, await page.getByRole('tabpanel', { name: tab.toLowerCase() }).isVisible());
  }
}
await browser.close();
console.log(failures === 0 ? '\nALL PASSED' : `\n${failures} FAILED`);
process.exit(failures === 0 ? 0 : 1);
