import * as THREE from 'three';

import { createStoryParts, disposeStory, refreshStoryTheme, type StoryParts } from './storyObject';

interface StoryRuntime {
  active: boolean;
  camera: THREE.PerspectiveCamera;
  displayedProgress: number;
  parts: StoryParts;
  reducedMotion: boolean;
  renderer: THREE.WebGLRenderer;
  scene: THREE.Scene;
  targetProgress: number;
  timer: THREE.Timer;
}

export interface StorySceneController {
  active: (active: boolean) => void;
  destroy: () => void;
  progress: (progress: number) => void;
  refreshTheme: () => void;
  resize: (width: number, height: number) => void;
}

function clamp01(value: number): number {
  return Math.max(0, Math.min(1, value));
}

function segment(value: number, start: number, end: number): number {
  const normalized = clamp01((value - start) / (end - start));
  return normalized * normalized * (3 - 2 * normalized);
}

function createRuntime(canvas: HTMLCanvasElement): StoryRuntime | null {
  const context = canvas.getContext('webgl2', { alpha: true, antialias: true });
  if (context === null) return null;
  const renderer = new THREE.WebGLRenderer({ canvas, context, alpha: true, antialias: true });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5));
  renderer.setClearColor(0x000000, 0);
  renderer.outputColorSpace = THREE.SRGBColorSpace;

  const scene = new THREE.Scene();
  const camera = new THREE.PerspectiveCamera(34, 1, 0.1, 40);
  camera.position.set(0, 0.05, 8.2);
  const parts = createStoryParts();
  scene.add(parts.root, new THREE.HemisphereLight(0xffffff, 0x555566, 1.5));
  const key = new THREE.DirectionalLight(0xffffff, 2.4);
  key.position.set(3, 4, 6);
  scene.add(key);
  const timer = new THREE.Timer();
  timer.connect(document);

  return {
    active: true,
    camera,
    displayedProgress: 0,
    parts,
    reducedMotion: window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    renderer,
    scene,
    targetProgress: 0,
    timer,
  };
}

function poseCursor(parts: StoryParts, progress: number): void {
  const a = new THREE.Vector3(-1.45, 0.7, 0.42);
  const b = new THREE.Vector3(0.35, 0.48, 1.5);
  const c = new THREE.Vector3(0.55, -0.95, 1.1);
  const d = new THREE.Vector3(1.72, 1.2, 0.5);
  if (progress < 0.4) parts.cursor.position.lerpVectors(a, b, segment(progress, 0.08, 0.4));
  else if (progress < 0.72) parts.cursor.position.lerpVectors(b, c, segment(progress, 0.4, 0.72));
  else parts.cursor.position.lerpVectors(c, d, segment(progress, 0.72, 0.96));
}

function poseParts(parts: StoryParts, progress: number, elapsed: number): void {
  const understand = segment(progress, 0.18, 0.42);
  const act = segment(progress, 0.42, 0.7);
  const verify = segment(progress, 0.7, 0.96);
  const settle = 1 - verify;

  parts.root.rotation.x = -0.08 + understand * 0.11 - verify * 0.03;
  parts.root.rotation.y = -0.22 + progress * 0.4 + Math.sin(elapsed * 0.45) * 0.015;
  parts.root.position.y = Math.sin(elapsed * 0.7) * 0.035 * settle;

  parts.context.position.set(-1.72 - understand * 0.45, 0.2, 0.08 + understand * 1.35 - verify * 1.1);
  parts.editor.position.set(0.48 + act * 0.18, 0.35, 0.12 + understand * 0.75 + act * 0.45 - verify * 0.9);
  parts.terminal.position.set(0.48, -1.05 - act * 0.08, 0.14 + act * 1.3 - verify * 1.05);

  parts.context.scale.set(0.29, 0.76, 1);
  parts.editor.scale.set(0.58 + act * 0.05, 0.68 + act * 0.04, 1);
  parts.terminal.scale.set(0.58 + act * 0.05, 0.18 + act * 0.03, 1);
  parts.status.scale.setScalar(0.001 + verify * 0.17);
  parts.status.position.set(1.82, 1.25, 0.2 + verify * 0.3);
  parts.glow.scale.setScalar(0.92 + understand * 0.08 + verify * 0.12);
  poseCursor(parts, progress);
}

function renderRuntime(runtime: StoryRuntime): void {
  runtime.timer.update();
  const elapsed = runtime.timer.getElapsed();
  const mix = runtime.reducedMotion ? 1 : 0.075;
  runtime.displayedProgress += (runtime.targetProgress - runtime.displayedProgress) * mix;
  poseParts(runtime.parts, runtime.displayedProgress, elapsed);
  runtime.renderer.render(runtime.scene, runtime.camera);
}

function resizeRuntime(runtime: StoryRuntime, width: number, height: number): void {
  if (width <= 0 || height <= 0) return;
  runtime.camera.aspect = width / height;
  runtime.camera.position.z = width < 720 ? 10.6 : 8.2;
  runtime.camera.updateProjectionMatrix();
  runtime.renderer.setSize(width, height, false);
  renderRuntime(runtime);
}

export function createStoryScene(canvas: HTMLCanvasElement): StorySceneController | null {
  const runtime = createRuntime(canvas);
  if (runtime === null) return null;
  const render = (): void => renderRuntime(runtime);
  const setActive = (active: boolean): void => {
    runtime.active = active;
    runtime.renderer.setAnimationLoop(active && !runtime.reducedMotion ? render : null);
    if (active && runtime.reducedMotion) render();
  };
  setActive(true);
  render();

  return {
    active: setActive,
    destroy: () => {
      runtime.renderer.setAnimationLoop(null);
      runtime.timer.dispose();
      disposeStory(runtime.parts.root);
      runtime.renderer.dispose();
    },
    progress: (progress) => {
      runtime.targetProgress = clamp01(progress);
      if (runtime.reducedMotion) render();
    },
    refreshTheme: () => {
      refreshStoryTheme(runtime.parts);
      render();
    },
    resize: (width, height) => resizeRuntime(runtime, width, height),
  };
}
