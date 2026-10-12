import * as THREE from 'three';

import {
  createGraphParts,
  disposeGraph,
  refreshGraphTheme,
  updateGraphParticles,
  type GraphParts,
} from './graphObjects';

interface GraphRuntime {
  active: boolean;
  camera: THREE.PerspectiveCamera;
  clock: THREE.Clock;
  parts: GraphParts;
  pointer: THREE.Vector2;
  reducedMotion: boolean;
  renderer: THREE.WebGLRenderer;
  root: THREE.Group;
  scene: THREE.Scene;
}

export interface AgentGraphController {
  active: (active: boolean) => void;
  destroy: () => void;
  pointer: (x: number, y: number) => void;
  refreshTheme: () => void;
  resize: (width: number, height: number) => void;
}

function createRuntime(canvas: HTMLCanvasElement): GraphRuntime | null {
  const context = canvas.getContext('webgl2', { alpha: true, antialias: true });
  if (context === null) return null;
  const renderer = new THREE.WebGLRenderer({
    canvas,
    context,
    alpha: true,
    antialias: true,
    powerPreference: 'high-performance',
  });
  renderer.setClearColor(0x000000, 0);
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5));

  const scene = new THREE.Scene();
  const camera = new THREE.PerspectiveCamera(42, 1, 0.1, 30);
  const root = new THREE.Group();
  const parts = createGraphParts();
  camera.position.set(0, 0.1, 7.4);
  root.add(parts.lines, parts.nodes, parts.rings, parts.core, parts.particles);
  scene.add(root);
  refreshGraphTheme(parts);

  return {
    active: true,
    camera,
    clock: new THREE.Clock(),
    parts,
    pointer: new THREE.Vector2(),
    reducedMotion: window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    renderer,
    root,
    scene,
  };
}

function renderRuntime(runtime: GraphRuntime): void {
  const elapsed = runtime.clock.getElapsedTime();
  runtime.root.rotation.y += (runtime.pointer.x * 0.18 - runtime.root.rotation.y) * 0.035;
  runtime.root.rotation.x += (-runtime.pointer.y * 0.09 - runtime.root.rotation.x) * 0.035;
  runtime.parts.core.rotation.y = elapsed * 0.28;
  runtime.parts.core.rotation.x = elapsed * 0.17;
  runtime.parts.rings.rotation.z = elapsed * -0.08;
  runtime.parts.rings.rotation.y = elapsed * 0.12;
  if (!runtime.reducedMotion) updateGraphParticles(runtime.parts, elapsed);
  runtime.renderer.render(runtime.scene, runtime.camera);
}

function resizeRuntime(runtime: GraphRuntime, width: number, height: number): void {
  if (width <= 0 || height <= 0) return;
  runtime.camera.aspect = width / height;
  runtime.camera.updateProjectionMatrix();
  runtime.renderer.setSize(width, height, false);
  if (runtime.reducedMotion || !runtime.active) renderRuntime(runtime);
}

function destroyRuntime(runtime: GraphRuntime): void {
  runtime.renderer.setAnimationLoop(null);
  disposeGraph(runtime.root);
  runtime.renderer.dispose();
}

export function createAgentGraph(canvas: HTMLCanvasElement): AgentGraphController | null {
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
    destroy: () => destroyRuntime(runtime),
    pointer: (x, y) => runtime.pointer.set(x, y),
    refreshTheme: () => {
      refreshGraphTheme(runtime.parts);
      if (runtime.reducedMotion || !runtime.active) render();
    },
    resize: (width, height) => resizeRuntime(runtime, width, height),
  };
}
