import * as THREE from 'three';
import { CSS3DRenderer } from 'three/addons/renderers/CSS3DRenderer.js';

import { createStoryParts, disposeStory, type StoryParts } from './storyObject';

interface StoryRuntime {
  active: boolean;
  anchorX: number;
  camera: THREE.PerspectiveCamera;
  displayedProgress: number;
  frame: number | null;
  lastTime: number;
  parts: StoryParts;
  reducedMotion: boolean;
  renderer: CSS3DRenderer;
  rootScale: number;
  scene: THREE.Scene;
  targetProgress: number;
}

export interface StorySceneController {
  active: (active: boolean) => void;
  destroy: () => void;
  progress: (progress: number) => void;
  resize: (width: number, height: number) => void;
}

function clamp01(value: number): number {
  return Math.max(0, Math.min(1, value));
}

function segment(value: number, start: number, end: number): number {
  const normalized = clamp01((value - start) / (end - start));
  return normalized * normalized * (3 - 2 * normalized);
}

function setOpacity(object: THREE.Object3D & { element?: HTMLElement }, value: number): void {
  if (object.element !== undefined) object.element.style.opacity = clamp01(value).toFixed(3);
}

function poseParts(runtime: StoryRuntime, progress: number, time: number): void {
  const route = segment(progress, 0.08, 0.33);
  const execute = segment(progress, 0.38, 0.66);
  const verify = segment(progress, 0.7, 0.96);
  const float = runtime.reducedMotion ? 0 : Math.sin(time * 0.0005) * 2 * (1 - verify);
  const { parts } = runtime;

  parts.root.position.set(runtime.anchorX * route, -130 + route * 110 + execute * 18 + float, 0);
  parts.root.scale.setScalar(runtime.rootScale * (0.96 - route * 0.08));
  parts.root.rotation.set(-0.025 + route * 0.018, -0.035 + execute * 0.06 - verify * 0.025, 0);

  parts.frame.position.set(0, 0, -54 + verify * 14);
  parts.orchestrator.position.set(-18 - execute * 175, 24 + execute * 34, 50 + route * 230 - execute * 130 - verify * 35);
  parts.repo.position.set(-312 - route * 58 + execute * 34, 36, 16 + route * 150 - execute * 76 - verify * 24);
  parts.context.position.set(-225 - route * 38, -150, 70 + route * 190 - execute * 120);
  parts.editor.position.set(132 + execute * 28 - verify * 20, 48, 20 + route * 42 + execute * 215 - verify * 95);
  parts.terminal.position.set(132 + execute * 20, -170, 18 + execute * 228 - verify * 78);
  parts.preview.position.set(348 - verify * 55, 72, 24 + execute * 46 + verify * 208);
  parts.status.position.set(308, -178, 90 + verify * 240);

  parts.orchestrator.rotation.y = -route * 0.035 + execute * 0.055;
  parts.repo.rotation.y = -route * 0.075;
  parts.context.rotation.y = -0.04 - route * 0.055;
  parts.editor.rotation.y = execute * 0.03;
  parts.terminal.rotation.x = -execute * 0.03;
  parts.preview.rotation.y = verify * 0.05;
  parts.status.scale.setScalar(0.86 + verify * 0.14);

  setOpacity(parts.orchestrator, 0.22 + route * 0.78 - execute * 0.3 - verify * 0.16);
  setOpacity(parts.repo, 0.48 + route * 0.52 - execute * 0.22 - verify * 0.2);
  setOpacity(parts.context, route * (1 - execute));
  setOpacity(parts.editor, 0.34 + execute * 0.66 - verify * 0.12);
  setOpacity(parts.terminal, 0.12 + execute * 0.88 - verify * 0.08);
  setOpacity(parts.preview, 0.08 + execute * 0.24 + verify * 0.68);
  setOpacity(parts.status, verify);

  runtime.camera.position.set(-14 * execute, 3 * route, 1360 - execute * 95 + verify * 24);
}

function renderRuntime(runtime: StoryRuntime, time: number): void {
  const delta = runtime.lastTime === 0 ? 0 : Math.min((time - runtime.lastTime) / 1000, 0.1);
  runtime.lastTime = time;
  const blend = runtime.reducedMotion ? 1 : 1 - Math.exp(-10 * delta);
  runtime.displayedProgress += (runtime.targetProgress - runtime.displayedProgress) * blend;
  poseParts(runtime, runtime.displayedProgress, time);
  runtime.renderer.render(runtime.scene, runtime.camera);
}

function animate(runtime: StoryRuntime, time: number): void {
  if (!runtime.active || runtime.reducedMotion) {
    runtime.frame = null;
    return;
  }
  renderRuntime(runtime, time);
  runtime.frame = window.requestAnimationFrame((nextTime) => { animate(runtime, nextTime); });
}

function start(runtime: StoryRuntime): void {
  if (runtime.frame !== null || runtime.reducedMotion || !runtime.active) return;
  runtime.lastTime = performance.now();
  runtime.frame = window.requestAnimationFrame((time) => { animate(runtime, time); });
}

function createRuntime(host: HTMLElement): StoryRuntime {
  const renderer = new CSS3DRenderer();
  renderer.domElement.className = 'agent-story-renderer';
  renderer.domElement.setAttribute('aria-hidden', 'true');
  host.append(renderer.domElement);
  const scene = new THREE.Scene();
  const camera = new THREE.PerspectiveCamera(38, 1, 1, 4000);
  camera.position.z = 1360;
  const parts = createStoryParts();
  scene.add(parts.root);
  return { active: true, anchorX: 0, camera, displayedProgress: 0, frame: null, lastTime: 0, parts, reducedMotion: window.matchMedia('(prefers-reduced-motion: reduce)').matches, renderer, rootScale: 1, scene, targetProgress: 0 };
}

function resizeRuntime(runtime: StoryRuntime, width: number, height: number): void {
  if (width <= 0 || height <= 0) return;
  runtime.camera.aspect = width / height;
  runtime.camera.updateProjectionMatrix();
  runtime.renderer.setSize(width, height);
  runtime.anchorX = width >= 1180 ? 230 : width >= 760 ? 105 : 0;
  runtime.rootScale = width >= 1180 ? 0.9 : width >= 760 ? 0.73 : 0.49;
  renderRuntime(runtime, performance.now());
}

export function createStoryScene(host: HTMLElement): StorySceneController {
  const runtime = createRuntime(host);
  start(runtime);
  renderRuntime(runtime, performance.now());
  return {
    active: (active) => {
      runtime.active = active;
      if (!active && runtime.frame !== null) window.cancelAnimationFrame(runtime.frame);
      if (!active) runtime.frame = null;
      else start(runtime);
    },
    destroy: () => {
      if (runtime.frame !== null) window.cancelAnimationFrame(runtime.frame);
      disposeStory(runtime.parts.root);
      runtime.renderer.domElement.remove();
    },
    progress: (progress) => {
      runtime.targetProgress = clamp01(progress);
      if (runtime.reducedMotion) {
        runtime.displayedProgress = runtime.targetProgress;
        renderRuntime(runtime, performance.now());
      }
    },
    resize: (width, height) => { resizeRuntime(runtime, width, height); },
  };
}
