import * as THREE from 'three';
import { CSS3DObject } from 'three/addons/renderers/CSS3DRenderer.js';

export interface StoryParts {
  context: CSS3DObject;
  editor: CSS3DObject;
  frame: CSS3DObject;
  preview: CSS3DObject;
  repo: CSS3DObject;
  root: THREE.Group;
  status: CSS3DObject;
  terminal: CSS3DObject;
}

function panel(className: string, title: string, body: string): CSS3DObject {
  const element = document.createElement('section');
  element.className = `agent-workspace-panel ${className}`;
  element.setAttribute('aria-hidden', 'true');
  element.innerHTML = `<header class="agent-panel-header"><span>${title}</span><span class="agent-panel-dot"></span></header>${body}`;
  return new CSS3DObject(element);
}

function makeFrame(): CSS3DObject {
  return panel(
    'agent-shell',
    'masih-awam / workspace',
    `<div class="agent-shell-body">
      <div class="agent-goal"><span>Goal</span><strong>Fix the auth flow and verify it.</strong></div>
      <div class="agent-loop"><span class="is-active">read</span><span>edit</span><span>run</span><span>verify</span></div>
    </div>`,
  );
}

function makeRepo(): CSS3DObject {
  return panel(
    'agent-repo',
    'Repository',
    `<div class="agent-file-list">
      <span class="agent-file is-active">AGENTS.md</span>
      <span class="agent-file">web/</span>
      <span class="agent-file agent-file--nested">src/</span>
      <span class="agent-file agent-file--nested">features/landing/</span>
      <span class="agent-file agent-file--nested is-modified">StoryHero.svelte</span>
      <span class="agent-file agent-file--nested is-modified">storyScene.ts</span>
      <span class="agent-file">internal/</span>
      <span class="agent-file">Makefile</span>
    </div>`,
  );
}

function makeContext(): CSS3DObject {
  return panel(
    'agent-context',
    'Context',
    `<div class="agent-context-body">
      <span class="agent-context-tag">AGENTS.md</span>
      <strong>Read rules before code changes.</strong>
      <p>Reuse existing UI. Keep themes semantic. Verify before done.</p>
    </div>`,
  );
}

function makeEditor(): CSS3DObject {
  return panel(
    'agent-editor',
    'StoryHero.svelte · modified',
    `<div class="agent-code">
      <span><i>41</i><code>const task = await agent.plan(goal);</code></span>
      <span class="is-added"><i>42</i><code>+ await workspace.read(task.context);</code></span>
      <span class="is-added"><i>43</i><code>+ await workspace.apply(task.change);</code></span>
      <span class="is-added"><i>44</i><code>+ const proof = await verify.run();</code></span>
      <span><i>45</i><code>return { task, proof };</code></span>
    </div>`,
  );
}

function makeTerminal(): CSS3DObject {
  return panel(
    'agent-terminal',
    'Terminal',
    `<div class="agent-terminal-body">
      <span><b>$</b> make verify</span>
      <span class="is-success">✓ svelte-check · 0 errors</span>
      <span class="is-success">✓ eslint · passed</span>
      <span class="is-success">✓ go test ./... · passed</span>
    </div>`,
  );
}

function makePreview(): CSS3DObject {
  return panel(
    'agent-preview',
    'Browser preview',
    `<div class="agent-preview-body">
      <span class="agent-preview-url">127.0.0.1:8080</span>
      <div class="agent-preview-card"><small>Masih Awam</small><strong>Agentic workspace</strong><span>ready to inspect</span></div>
    </div>`,
  );
}

function makeStatus(): CSS3DObject {
  return panel(
    'agent-status',
    'Verified',
    `<div class="agent-status-body"><span>✓</span><div><strong>4 checks passed</strong><small>Ready to ship</small></div></div>`,
  );
}

export function createStoryParts(): StoryParts {
  const root = new THREE.Group();
  const parts: StoryParts = {
    context: makeContext(),
    editor: makeEditor(),
    frame: makeFrame(),
    preview: makePreview(),
    repo: makeRepo(),
    root,
    status: makeStatus(),
    terminal: makeTerminal(),
  };
  root.add(parts.frame, parts.repo, parts.context, parts.editor, parts.terminal, parts.preview, parts.status);
  return parts;
}

export function disposeStory(root: THREE.Object3D): void {
  root.traverse((object) => {
    if (object instanceof CSS3DObject) object.element.remove();
  });
  root.clear();
}
