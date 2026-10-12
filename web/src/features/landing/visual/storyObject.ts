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
    'Masih Awam',
    `<div class="agent-shell-body">
      <div class="agent-goal"><span>Task</span><strong>Fix the sign-in redirect and verify it.</strong></div>
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
      <span class="agent-file agent-file--nested">features/auth/</span>
      <span class="agent-file agent-file--nested is-modified">callback.ts</span>
      <span class="agent-file agent-file--nested is-modified">session.ts</span>
      <span class="agent-file">tests/</span>
      <span class="agent-file agent-file--nested">auth.spec.ts</span>
    </div>`,
  );
}

function makeContext(): CSS3DObject {
  return panel(
    'agent-context',
    'Context',
    `<div class="agent-context-body">
      <span class="agent-context-tag">AGENTS.md</span>
      <strong>Read the rules before changing code.</strong>
      <p>Reuse existing UI. Keep themes semantic. Verify before done.</p>
    </div>`,
  );
}

function makeEditor(): CSS3DObject {
  return panel(
    'agent-editor',
    'callback.ts · modified',
    `<div class="agent-code">
      <span><i>41</i><code>const next = params.get('next');</code></span>
      <span class="is-added"><i>42</i><code>+ const target = safeRedirect(next);</code></span>
      <span class="is-added"><i>43</i><code>+ await session.persist();</code></span>
      <span class="is-added"><i>44</i><code>+ return redirect(target);</code></span>
      <span><i>45</i><code>}</code></span>
    </div>`,
  );
}

function makeTerminal(): CSS3DObject {
  return panel(
    'agent-terminal',
    'Terminal',
    `<div class="agent-terminal-body">
      <span><b>$</b> make verify</span>
      <span class="is-success">✓ frontend · passed</span>
      <span class="is-success">✓ tests · passed</span>
      <span class="is-success">✓ production build · passed</span>
    </div>`,
  );
}

function makePreview(): CSS3DObject {
  return panel(
    'agent-preview',
    'Browser',
    `<div class="agent-preview-body">
      <span class="agent-preview-url">127.0.0.1:8080/auth/callback</span>
      <div class="agent-preview-card"><small>Redirect result</small><strong>/workspace</strong><span>session restored</span></div>
    </div>`,
  );
}

function makeStatus(): CSS3DObject {
  return panel(
    'agent-status',
    'Verified',
    `<div class="agent-status-body"><span>✓</span><div><strong>4 checks passed</strong><small>No failures</small></div></div>`,
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
