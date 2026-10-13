import * as THREE from 'three';
import { CSS3DRenderer } from 'three/addons/renderers/CSS3DRenderer.js';

import { createStoryParts, disposeStory, type StoryParts } from './storyObject';

const BASE_CAMERA_Z = 1430;
const MONITOR_WIDTH = 1180;

interface StoryRuntime {
  active: boolean;
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

function setStage(element: HTMLElement | undefined, opacity: number, y: number, scale: number): void {
  if (element === undefined) return;
  element.style.opacity = clamp01(opacity).toFixed(3);
  element.style.transform = `translate3d(0,${y.toFixed(2)}px,0) scale(${scale.toFixed(4)})`;
}

function poseStages(runtime: StoryRuntime, progress: number): void {
  const routeIn = segment(progress, 0.12, 0.28);
  const workIn = segment(progress, 0.38, 0.54);
  const proofIn = segment(progress, 0.68, 0.84);
  const goal = 1 - routeIn;
  const route = routeIn * (1 - workIn);
  const work = workIn * (1 - proofIn);
  const proof = proofIn;

  setStage(runtime.parts.stages[0], goal, -16 * routeIn, 0.98 + goal * 0.02);
  setStage(runtime.parts.stages[1], route, 18 * (1 - route), 0.985 + route * 0.015);
  setStage(runtime.parts.stages[2], work, 18 * (1 - work), 0.985 + work * 0.015);
  setStage(runtime.parts.stages[3], proof, 18 * (1 - proof), 0.985 + proof * 0.015);
}

function poseParts(runtime: StoryRuntime, progress: number, time: number): void {
  const route = segment(progress, 0.12, 0.32);
  const work = segment(progress, 0.4, 0.68);
  const proof = segment(progress, 0.72, 0.96);
  const float = runtime.reducedMotion ? 0 : Math.sin(time * 0.00055) * 1.4 * (1 - proof);
  const shrink = 1 - progress * 0.31;
  const { monitor, root } = runtime.parts;

  root.position.set(0, -18 - progress * 24 + float, 0);
  root.scale.setScalar(runtime.rootScale * shrink);
  root.rotation.set(-0.014 + route * 0.01 - work * 0.006, -0.012 + work * 0.02 - proof * 0.008, 0);

  monitor.position.z = 10 + route * 20 + work * 30 - proof * 12;
  monitor.element.style.setProperty('--story-progress', `${(progress * 100).toFixed(2)}%`);
  poseStages(runtime, progress);

  runtime.camera.position.set(0, route * 4 - proof * 3, BASE_CAMERA_Z - route * 48 - work * 72 + proof * 34);
}

function renderRuntime(runtime: StoryRuntime, time: number): void {
  const delta = runtime.lastTime === 0 ? 0 : Math.min((time - runtime.lastTime) / 1000, 0.1);
  runtime.lastTime = time;
  const blend = runtime.reducedMotion ? 1 : 1 - Math.exp(-11 * delta);
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
  camera.position.z = BASE_CAMERA_Z;
  const parts = createStoryParts();
  scene.add(parts.root);

  return {
    active: true,
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
  const perspective = (height * 0.5) / Math.tan(THREE.MathUtils.degToRad(runtime.camera.fov * 0.5));
  const targetWidth = Math.min(width * 0.88, 1280);
  runtime.rootScale = Math.min(2, Math.max(0.56, (targetWidth * BASE_CAMERA_Z) / (MONITOR_WIDTH * perspective)));
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
