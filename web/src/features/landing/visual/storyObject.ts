import * as THREE from 'three';
import { RoundedBoxGeometry } from 'three/addons/geometries/RoundedBoxGeometry.js';

export interface StoryParts {
  context: THREE.Mesh;
  cursor: THREE.Mesh;
  editor: THREE.Mesh;
  frame: THREE.Mesh;
  glow: THREE.Mesh;
  root: THREE.Group;
  status: THREE.Mesh;
  terminal: THREE.Mesh;
}

type DisposableMaterial = THREE.Material | THREE.Material[];

function cssColor(variable: string, fallback: string): THREE.Color {
  const value = getComputedStyle(document.documentElement).getPropertyValue(variable).trim();
  return new THREE.Color(value || fallback);
}

function plate(width: number, height: number, depth: number, radius: number): THREE.Mesh {
  return new THREE.Mesh(
    new RoundedBoxGeometry(width, height, depth, 5, radius),
    new THREE.MeshStandardMaterial({ roughness: 0.78, metalness: 0.03 }),
  );
}

function makeFrame(): THREE.Mesh {
  const frame = plate(5.1, 3.25, 0.16, 0.2);
  frame.material = new THREE.MeshStandardMaterial({ roughness: 0.52, metalness: 0.08 });
  return frame;
}

function makeCursor(): THREE.Mesh {
  return new THREE.Mesh(
    new THREE.SphereGeometry(0.07, 24, 24),
    new THREE.MeshBasicMaterial(),
  );
}

function makeGlow(): THREE.Mesh {
  return new THREE.Mesh(
    new THREE.PlaneGeometry(7.4, 5.1),
    new THREE.MeshBasicMaterial({ transparent: true, opacity: 0.055, depthWrite: false }),
  );
}

function positionParts(parts: StoryParts): void {
  parts.frame.position.z = -0.12;
  parts.context.position.set(-1.65, 0.22, 0.02);
  parts.context.scale.set(0.29, 0.76, 1);
  parts.editor.position.set(0.55, 0.35, 0.04);
  parts.editor.scale.set(0.58, 0.68, 1);
  parts.terminal.position.set(0.55, -1.05, 0.06);
  parts.terminal.scale.set(0.58, 0.18, 1);
  parts.status.position.set(1.82, 1.25, 0.1);
  parts.status.scale.set(0.17, 0.11, 1);
  parts.cursor.position.set(-0.4, 0.42, 0.18);
  parts.glow.position.z = -0.5;
}

export function createStoryParts(): StoryParts {
  const root = new THREE.Group();
  const parts: StoryParts = {
    context: plate(4.35, 2.65, 0.08, 0.16),
    cursor: makeCursor(),
    editor: plate(4.35, 2.65, 0.08, 0.16),
    frame: makeFrame(),
    glow: makeGlow(),
    root,
    status: plate(4.35, 2.65, 0.08, 0.16),
    terminal: plate(4.35, 2.65, 0.08, 0.16),
  };
  positionParts(parts);
  root.add(parts.glow, parts.frame, parts.context, parts.editor, parts.terminal, parts.status, parts.cursor);
  refreshStoryTheme(parts);
  return parts;
}

function setStandard(mesh: THREE.Mesh, color: THREE.Color, opacity = 1): void {
  const material = mesh.material as THREE.MeshStandardMaterial;
  material.color.copy(color);
  material.transparent = opacity < 1;
  material.opacity = opacity;
  material.needsUpdate = true;
}

export function refreshStoryTheme(parts: StoryParts): void {
  const base100 = cssColor('--color-base-100', '#fffdf7');
  const base200 = cssColor('--color-base-200', '#f6f3eb');
  const base300 = cssColor('--color-base-300', '#ece8de');
  const primary = cssColor('--color-primary', '#657c1a');
  const secondary = cssColor('--color-secondary', '#275391');
  const success = cssColor('--color-success', '#2f7d4a');

  setStandard(parts.frame, base100);
  setStandard(parts.context, base300, 0.98);
  setStandard(parts.editor, base100, 0.98);
  setStandard(parts.terminal, base200, 0.98);
  setStandard(parts.status, success, 0.92);
  (parts.cursor.material as THREE.MeshBasicMaterial).color.copy(primary);
  (parts.glow.material as THREE.MeshBasicMaterial).color.copy(secondary);
}

function disposeMaterial(material: DisposableMaterial): void {
  if (Array.isArray(material)) {
    material.forEach((item) => item.dispose());
    return;
  }
  material.dispose();
}

export function disposeStory(root: THREE.Object3D): void {
  root.traverse((child) => {
    if (!(child instanceof THREE.Mesh)) return;
    child.geometry.dispose();
    disposeMaterial(child.material);
  });
}
