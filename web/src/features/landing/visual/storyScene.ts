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
  const understand = segment(progress, 0.08, 0.34);
  const act = segment(progress, 0.3, 0.66);
  const verify = segment(progress, 0.62, 0.9);
  const settle = segment(progress, 0.9, 1);
  const float = runtime.reducedMotion ? 0 : Math.sin(time * 0.00055) * 3 * (1 - verify);
  const { parts } = runtime;

  parts.root.position.set(runtime.anchorX, float, 0);
  parts.root.scale.setScalar(runtime.rootScale);
  parts.root.rotation.set(-0.025 + understand * 0.02, -0.07 + act * 0.1 - verify * 0.04, 0);

  parts.frame.position.set(0, 0, -42 + settle * 18);
  parts.repo.position.set(-305 - understand * 85 + act * 44, 12, 12 + understand * 170 - act * 72 - verify * 28);
  parts.context.position.set(-220 - understand * 55, -155 + understand * 8, 80 + understand * 210 - act * 95);
  parts.editor.position.set(128 + act * 36 - verify * 24, 42, 18 + understand * 54 + act * 190 - verify * 108);
  parts.terminal.position.set(128 + act * 28, -170 - act * 18, 22 + act * 228 - verify * 72);
  parts.preview.position.set(350 - verify * 62, 78 - verify * 10, 28 + verify * 190);
  parts.status.position.set(315 - settle * 20, -178 + settle * 8, 110 + verify * 220 - settle * 70);

  parts.repo.rotation.y = -understand * 0.08;
  parts.context.rotation.y = -0.05 - understand * 0.08;
  parts.editor.rotation.y = act * 0.035;
  parts.terminal.rotation.x = -act * 0.035;
  parts.preview.rotation.y = verify * 0.055;
  parts.status.scale.setScalar(0.84 + verify * 0.16);

  setOpacity(parts.context, segment(progress, 0.12, 0.22) * (1 - segment(progress, 0.46, 0.58)));
  setOpacity(parts.preview, segment(progress, 0.5, 0.7));
  setOpacity(parts.status, segment(progress, 0.7, 0.86));
  setOpacity(parts.repo, 1 - verify * 0.15);

  runtime.camera.position.set(-18 * act, 4 * understand, 1440 - act * 110 + verify * 42);
}

function renderRuntime(runtime: StoryRuntime, time: number): void {
  const delta = runtime.lastTime === 0 ? 0 : Math.min((time - runtime.lastTime) / 1000, 0.1);
  runtime.lastTime = time;
  const blend = runtime.reducedMotion ? 1 : 1 - Math.exp(-9 * delta);
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
  runtime.frame = window.requestAnimationFrame((nextTime) => {
    animate(runtime, nextTime);
  });
}

function start(runtime: StoryRuntime): void {
  if (runtime.frame !== null || runtime.reducedMotion || !runtime.active) return;
  runtime.lastTime = performance.now();
  runtime.frame = window.requestAnimationFrame((time) => {
    animate(runtime, time);
  });
}

function createRuntime(host: HTMLElement): StoryRuntime {
  const renderer = new CSS3DRenderer();
  renderer.domElement.className = 'agent-story-renderer';
  renderer.domElement.setAttribute('aria-hidden', 'true');
  host.append(renderer.domElement);
  const scene = new THREE.Scene();
  const camera = new THREE.PerspectiveCamera(38, 1, 1, 4000);
  camera.position.z = 1440;
  const parts = createStoryParts();
  scene.add(parts.root);
  return {
    active: true,
    anchorX: 0,
    camera,
    displayedProgress: 0,
    frame: null,
    lastTime: 0,
    parts,
    reducedMotion: window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    renderer,
    rootScale: 1,
    scene,
    targetProgress: 0,
  };
}

function resizeRuntime(runtime: StoryRuntime, width: number, height: number): void {
  if (width <= 0 || height <= 0) return;
  runtime.camera.aspect = width / height;
  runtime.camera.updateProjectionMatrix();
  runtime.renderer.setSize(width, height);
  runtime.anchorX = width >= 1180 ? 235 : width >= 760 ? 120 : 0;
  runtime.rootScale = width >= 1180 ? 0.88 : width >= 760 ? 0.72 : 0.52;
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
    resize: (width, height) => {
      resizeRuntime(runtime, width, height);
    },
  };
}
