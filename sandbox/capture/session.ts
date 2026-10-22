import type { Cdp } from './cdp.ts';

interface SessionRequest {
  action: 'login' | 'logout';
  path: string;
  emailSelector?: string;
  passwordSelector?: string;
  submitSelector: string;
  readySelector: string;
  errorSelector?: string;
  email?: string;
  password?: string;
}

const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

async function find(cdp: Cdp, selector: string): Promise<number> {
  const doc = await cdp.send('DOM.getDocument', { depth: 1 });
  const root = doc.root as { nodeId: number };
  const found = await cdp.send('DOM.querySelector', { nodeId: root.nodeId, selector });
  return Number(found.nodeId ?? 0);
}

async function attributes(cdp: Cdp, nodeId: number): Promise<{ tag: string; values: Map<string, string> }> {
	const result = await cdp.send('DOM.describeNode', { nodeId });
	const node = result.node as { nodeName: string; attributes: string[] };
	const values = node.attributes;
	return {
		tag: node.nodeName.toLowerCase(),
		values: new Map(values.reduce<string[][]>((all, value, index) => {
			if (index % 2 === 0) all.push([value, values[index + 1] ?? '']);
			return all;
		}, [])),
	};
}

async function fill(cdp: Cdp, selector: string, value: string, expectedType: string): Promise<boolean> {
  const nodeId = await find(cdp, selector);
  if (!nodeId) return false;
	const node = await attributes(cdp, nodeId);
	const type = (node.values.get('type') ?? '').toLowerCase();
	if (node.tag !== 'input' || (expectedType === 'password' ? type !== 'password' : !['email', 'text'].includes(type))) {
		return false;
	}
  await cdp.send('DOM.focus', { nodeId });
  await cdp.send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'a', code: 'KeyA', modifiers: 2 });
  await cdp.send('Input.dispatchKeyEvent', { type: 'keyUp', key: 'a', code: 'KeyA', modifiers: 2 });
  await cdp.send('Input.insertText', { text: value });
  return true;
}

async function click(cdp: Cdp, selector: string): Promise<boolean> {
  const nodeId = await find(cdp, selector);
  if (!nodeId) return false;
  const node = await attributes(cdp, nodeId);
  const type = (node.values.get('type') ?? '').toLowerCase();
  if (node.tag === 'button' && type !== '' && type !== 'submit') return false;
  if (node.tag === 'input' && type !== 'submit') return false;
  if (node.tag !== 'button' && node.tag !== 'input') return false;
	await cdp.send('DOM.scrollIntoViewIfNeeded', { nodeId });
	const box = await cdp.send('DOM.getBoxModel', { nodeId });
  const model = box.model as { content: number[] } | undefined;
  if (!model || model.content.length < 8) return false;
	const viewportResult = await cdp.send('Page.getLayoutMetrics');
	const viewport = viewportResult.cssVisualViewport as { clientWidth: number; clientHeight: number };
	const x = (model.content[0] + model.content[2] + model.content[4] + model.content[6]) / 4;
	const y = (model.content[1] + model.content[3] + model.content[5] + model.content[7]) / 4;
	if (!Number.isFinite(x) || !Number.isFinite(y) || x < 0 || y < 0 ||
	  x >= viewport.clientWidth || y >= viewport.clientHeight) return false;
  await cdp.send('Input.dispatchMouseEvent', { type: 'mousePressed', x, y, button: 'left', clickCount: 1 });
  await cdp.send('Input.dispatchMouseEvent', { type: 'mouseReleased', x, y, button: 'left', clickCount: 1 });
  return true;
}

async function visible(cdp: Cdp, selector: string): Promise<number> {
  const nodeId = await find(cdp, selector);
  if (!nodeId) return 0;
	const result = await cdp.send('DOM.getBoxModel', { nodeId });
	const model = result.model as { width: number; height: number } | undefined;
	return model && model.width > 0 && model.height > 0 ? nodeId : 0;
}

function isTransientNodeError(error: unknown): boolean {
	const message = error instanceof Error ? error.message : '';
	return /(?:Could not find node with given id|No node with given id|Node with given id does not belong to the document|Document is not available)/i.test(message);
}

async function readiness(cdp: Cdp, readySelector: string, errorSelector?: string): Promise<'ready' | 'failed' | 'pending'> {
	try {
		if (errorSelector && await hasVisibleError(cdp, errorSelector)) return 'failed';
		return await visible(cdp, readySelector) ? 'ready' : 'pending';
	} catch (error) {
		if (isTransientNodeError(error)) return 'pending';
		throw error;
	}
}

async function hasVisibleError(cdp: Cdp, selector: string): Promise<boolean> {
  const nodeId = await visible(cdp, selector);
  if (!nodeId) return false;
  const result = await cdp.send('DOM.getOuterHTML', { nodeId });
  const markup = String(result.outerHTML ?? '');
  return markup.replace(/<[^>]*>/g, '').trim() !== '';
}

export async function runSession(cdp: Cdp, url: string, request: SessionRequest): Promise<string> {
  const base = new URL(url);
  let external = false;
  cdp.on('Page.frameNavigated', (event) => {
    const frame = event.frame as { url?: string };
    if (frame.url && new URL(frame.url).origin !== base.origin) external = true;
  });
  const loaded = new Promise<void>((resolve) => cdp.on('Page.loadEventFired', () => resolve()));
  await cdp.send('Page.navigate', { url });
  await Promise.race([loaded, wait(6000)]);
  await wait(250);
  if (external) return 'external_redirect_blocked';
  if (request.action === 'login') {
    if (!request.email || !request.password || !request.emailSelector || !request.passwordSelector) return 'unsupported';
    if (!await fill(cdp, request.emailSelector, request.email, 'email')) return 'not_found';
    if (!await fill(cdp, request.passwordSelector, request.password, 'password')) return 'not_found';
  }
  if (!await click(cdp, request.submitSelector)) return 'not_found';
  for (let attempt = 0; attempt < 40; attempt += 1) {
    if (external) return 'external_redirect_blocked';
		const state = await readiness(cdp, request.readySelector, request.errorSelector);
		if (state === 'failed') return 'login_failed';
		if (state === 'ready') return 'indicator_found';
    await wait(250);
  }
  return 'indicator_not_found';
}
