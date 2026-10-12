import * as THREE from 'three';

import {
  createGraphParts,
  disposeGraph,
  refreshGraphTheme,
  updateGraphParticles,
} from './graphObjects';

export interface AgentGraphController {
  destroy: () => void;
  pointer: (x: number, y: number) => void;
  refreshTheme: () => void;
  resize: (width: number, height: number) => void;
}

export function createAgentGraph(canvas: HTMLCanvasElement): AgentGraphController | null {
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
  camera.position.set(0, 0.1, 7.4);

  const root = new THREE.Group();
  const parts = createGraphParts();
  root.add(parts.lines, parts.nodes, parts.rings, parts.core, parts.particles);
  scene.add(root);
  refreshGraphTheme(parts);

  const pointer = new THREE.Vector2();
  const clock = new THREE.Clock();
  const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  const render = (): void => {
    const elapsed = clock.getElapsedTime();
    root.rotation.y += (pointer.x * 0.18 - root.rotation.y) * 0.035;
    root.rotation.x += (-pointer.y * 0.09 - root.rotation.x) * 0.035;
    parts.core.rotation.y = elapsed * 0.28;
    parts.core.rotation.x = elapsed * 0.17;
    parts.rings.rotation.z = elapsed * -0.08;
    parts.rings.rotation.y = elapsed * 0.12;
    if (!reducedMotion) updateGraphParticles(parts, elapsed);
    renderer.render(scene, camera);
  };

  renderer.setAnimationLoop(reducedMotion ? null : render);
  render();

  return {
    destroy: () => {
      renderer.setAnimationLoop(null);
      disposeGraph(root);
      renderer.dispose();
    },
    pointer: (x, y) => pointer.set(x, y),
    refreshTheme: () => {
      refreshGraphTheme(parts);
      if (reducedMotion) render();
    },
    resize: (width, height) => {
      if (width <= 0 || height <= 0) return;
      camera.aspect = width / height;
      camera.updateProjectionMatrix();
      renderer.setSize(width, height, false);
      if (reducedMotion) render();
    },
  };
}
