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
  const shrink = 1 - progress * 0.28;
  const focusX = runtime.anchorX * route * (1 - verify * 0.6);
  const { parts } = runtime;

  parts.root.position.set(focusX, -92 + route * 84 + execute * 20 + float, 0);
  parts.root.scale.setScalar(runtime.rootScale * shrink);
  parts.root.rotation.set(-0.018 + route * 0.014, -0.025 + execute * 0.048 - verify * 0.02, 0);

  parts.frame.position.set(0, 0, -58 + verify * 14);
  parts.orchestrator.position.set(-30 - execute * 210, 18 + execute * 34, 72 + route * 240 - execute * 118 - verify * 36);
  parts.repo.position.set(-405 - route * 62 + execute * 36, 30, 22 + route * 158 - execute * 72 - verify * 28);
  parts.context.position.set(-300 - route * 36, -148, 82 + route * 188 - execute * 116);
  parts.editor.position.set(120 + execute * 34 - verify * 18, 44, 26 + route * 46 + execute * 220 - verify * 94);
  parts.terminal.position.set(120 + execute * 24, -166, 24 + execute * 226 - verify * 78);
  parts.preview.position.set(448 - verify * 120, 58, 30 + execute * 48 + verify * 212);
  parts.status.position.set(390 - verify * 120, -174, 96 + verify * 242);

  parts.orchestrator.rotation.y = -route * 0.03 + execute * 0.05;
  parts.repo.rotation.y = -route * 0.065;
  parts.context.rotation.y = -0.035 - route * 0.05;
  parts.editor.rotation.y = execute * 0.028;
  parts.terminal.rotation.x = -execute * 0.028;
  parts.preview.rotation.y = verify * 0.045;
  parts.status.scale.setScalar(0.88 + verify * 0.12);

  setOpacity(parts.orchestrator, 0.82 + route * 0.18 - execute * 0.44 - verify * 0.16);
  setOpacity(parts.repo, 0.94 - route * 0.5 + execute * 0.12 - verify * 0.2);
  setOpacity(parts.context, route * (1 - execute));
  setOpacity(parts.editor, 0.94 + execute * 0.06 - verify * 0.16);
  setOpacity(parts.terminal, 0.82 + execute * 0.18 - verify * 0.1);
  setOpacity(parts.preview, 0.6 + execute * 0.12 + verify * 0.28);
  setOpacity(parts.status, verify);

  runtime.camera.position.set(-18 * execute, 3 * route, 1360 - execute * 90 + verify * 20);
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
  runtime.anchorX = width >= 1180 ? 300 : width >= 760 ? 120 : 0;
  runtime.rootScale = width >= 1180 ? 1.75 : width >= 760 ? 1.18 : 0.72;
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
