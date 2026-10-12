import * as THREE from 'three';
import { CSS3DObject } from 'three/addons/renderers/CSS3DRenderer.js';

export interface StoryParts {
  context: CSS3DObject;
  editor: CSS3DObject;
  frame: CSS3DObject;
  orchestrator: CSS3DObject;
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
  element.innerHTML = `<header class="agent-panel-header"><span>${title}</span></header>${body}`;
  return new CSS3DObject(element);
}

function makeFrame(): CSS3DObject {
  return panel(
    'agent-shell',
    'Masih Awam · AI Orchestrator',
    `<div class="agent-shell-body">
      <div class="agent-goal">
        <strong>Fix the sign-in redirect.</strong>
        <small>Trace the callback, change the code, and verify it in the browser.</small>
      </div>
      <div class="agent-progress"><span>Plan ✓</span><span>Route ✓</span><span>Work ●</span><span>Verify ···</span></div>
    </div>`,
  );
}

function makeOrchestrator(): CSS3DObject {
  return panel(
    'agent-orchestrator',
    'Orchestrator',
    `<div class="agent-orchestrator-body">
      <div class="agent-lead"><span>Lead</span><strong>Auth redirect</strong><small>3 lanes</small></div>
      <div class="agent-route-list">
        <div><span>Research</span><small>official docs</small><b>running</b></div>
        <div><span>Code</span><small>callback.ts</small><b>ready</b></div>
        <div><span>Browser</span><small>auth flow</small><b>queued</b></div>
      </div>
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
    'Repository rules',
    `<div class="agent-context-body">
      <strong>Read before changing code.</strong>
      <p>Reuse existing UI. Keep both themes semantic. Verify before done.</p>
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
    orchestrator: makeOrchestrator(),
    preview: makePreview(),
    repo: makeRepo(),
    root,
    status: makeStatus(),
    terminal: makeTerminal(),
  };
  root.add(parts.frame, parts.orchestrator, parts.repo, parts.context, parts.editor, parts.terminal, parts.preview, parts.status);
  return parts;
}

export function disposeStory(root: THREE.Object3D): void {
  root.traverse((object) => {
    if (object instanceof CSS3DObject) object.element.remove();
  });
  root.clear();
}
