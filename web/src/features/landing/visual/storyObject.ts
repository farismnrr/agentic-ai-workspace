import * as THREE from 'three';
import { CSS3DObject } from 'three/addons/renderers/CSS3DRenderer.js';

export interface StoryParts {
  monitor: CSS3DObject;
  root: THREE.Group;
  stages: readonly HTMLElement[];
}

function goalMarkup(): string {
  return `
    <section class="agent-stage agent-stage-goal" data-story-stage>
      <div class="agent-goal-card">
        <span class="agent-kicker">Task</span>
        <strong>Fix the sign-in redirect.</strong>
        <p>Trace the callback, update the code, then prove the redirect works.</p>
        <div class="agent-goal-meta"><span>Repository attached</span><span>Rules loaded</span></div>
      </div>
    </section>`;
}

function routeMarkup(): string {
  return `
    <section class="agent-stage agent-stage-route agent-orchestrator" data-story-stage>
      <div class="agent-route-request"><span>Goal</span><strong>Fix sign-in redirect</strong></div>
      <div class="agent-route-line agent-route-line-a"></div>
      <div class="agent-route-line agent-route-line-b"></div>
      <div class="agent-route-line agent-route-line-c"></div>
      <div class="agent-route-card agent-route-research"><b>Research</b><small>official docs + repo rules</small><em>done</em></div>
      <div class="agent-route-card agent-route-code"><b>Code</b><small>callback.ts + session.ts</small><em>running</em></div>
      <div class="agent-route-card agent-route-browser"><b>Browser</b><small>auth callback flow</small><em>queued</em></div>
    </section>`;
}

function workMarkup(): string {
  return `
    <section class="agent-stage agent-stage-work" data-story-stage>
      <aside class="agent-work-files">
        <span class="agent-kicker">Repository</span>
        <b>AGENTS.md</b><span>web/</span><span>src/</span><span>features/auth/</span>
        <strong>callback.ts <i>M</i></strong><strong>session.ts <i>M</i></strong><span>tests/auth.spec.ts</span>
      </aside>
      <div class="agent-editor">
        <div class="agent-work-tabs"><b>callback.ts</b><span>session.ts</span></div>
        <div class="agent-code-line"><i>41</i><code>const next = params.get('next');</code></div>
        <div class="agent-code-line is-added"><i>42</i><code>+ const target = safeRedirect(next);</code></div>
        <div class="agent-code-line is-added"><i>43</i><code>+ await session.persist();</code></div>
        <div class="agent-code-line is-added"><i>44</i><code>+ return redirect(target);</code></div>
      </div>
      <div class="agent-work-terminal">
        <span class="agent-kicker">Terminal</span>
        <code>$ make verify</code><b>✓ frontend passed</b><b>✓ tests passed</b><b>✓ build passed</b>
      </div>
    </section>`;
}

function proofMarkup(): string {
  return `
    <section class="agent-stage agent-stage-proof" data-story-stage>
      <div class="agent-proof-heading"><span class="agent-proof-check">✓</span><div><span>Task complete</span><strong>Sign-in redirect verified</strong></div></div>
      <div class="agent-proof-grid">
        <div><span>Tests</span><b>12 passed</b><small>1.2s</small></div>
        <div><span>Lint</span><b>No issues</b><small>0.8s</small></div>
        <div><span>Build</span><b>Success</b><small>4.2s</small></div>
        <div><span>Browser</span><b>/workspace</b><small>session restored</small></div>
      </div>
      <div class="agent-status"><span>Verified</span><strong>4 checks passed</strong><small>No failures · 3 files changed</small></div>
    </section>`;
}

function monitorMarkup(): string {
  return `
    <div class="agent-monitor-frame">
      <div class="agent-monitor-camera"></div>
      <div class="agent-monitor-screen">
        <header class="agent-screen-bar">
          <span class="agent-screen-brand"><i></i>Masih Awam</span>
          <span class="agent-screen-task">Fix the sign-in redirect</span>
          <span class="agent-screen-state">live run</span>
        </header>
        <div class="agent-screen-body">
          ${goalMarkup()}${routeMarkup()}${workMarkup()}${proofMarkup()}
        </div>
        <div class="agent-screen-progress"><span></span></div>
      </div>
    </div>
    <div class="agent-monitor-neck"></div>
    <div class="agent-monitor-base"></div>`;
}

function makeMonitor(): CSS3DObject {
  const element = document.createElement('div');
  element.className = 'agent-workspace-panel agent-shell agent-monitor';
  element.setAttribute('aria-hidden', 'true');
  element.innerHTML = monitorMarkup();
  return new CSS3DObject(element);
}

export function createStoryParts(): StoryParts {
  const root = new THREE.Group();
  const monitor = makeMonitor();
  const stages = Array.from(monitor.element.querySelectorAll<HTMLElement>('[data-story-stage]'));
  root.add(monitor);
  return { monitor, root, stages };
}

export function disposeStory(root: THREE.Object3D): void {
  root.traverse((object) => {
    if (object instanceof CSS3DObject) object.element.remove();
  });
  root.clear();
}
