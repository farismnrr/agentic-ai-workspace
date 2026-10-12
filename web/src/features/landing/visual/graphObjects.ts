import * as THREE from 'three';

const NODE_COUNT = 42;
const PULSE_COUNT = 18;
const TAU = Math.PI * 2;

type GraphMaterial = THREE.LineBasicMaterial | THREE.MeshBasicMaterial | THREE.PointsMaterial;

export interface GraphParts {
  core: THREE.Group;
  lines: THREE.LineSegments;
  nodes: THREE.InstancedMesh;
  particles: THREE.Points;
  positions: THREE.Vector3[];
  rings: THREE.Group;
}

function readColor(variable: string, fallback: string): THREE.Color {
  const value = getComputedStyle(document.documentElement).getPropertyValue(variable).trim();
  return new THREE.Color(value || fallback);
}

function createPositions(): THREE.Vector3[] {
  return Array.from({ length: NODE_COUNT }, (_, index) => {
    if (index < 16) {
      const column = index % 4;
      const row = Math.floor(index / 4);
      return new THREE.Vector3(-3 + column * 0.42, -1.7 + row * 1.08, Math.sin(index * 1.7) * 0.55);
    }
    if (index < 26) {
      const angle = ((index - 16) / 10) * TAU;
      return new THREE.Vector3(Math.cos(angle) * 1.35, Math.sin(angle) * 1.35, Math.sin(angle * 2) * 0.65);
    }
    const local = index - 26;
    const column = local % 4;
    const row = Math.floor(local / 4);
    return new THREE.Vector3(1.75 + column * 0.42, -1.7 + row * 1.08, Math.cos(index * 1.5) * 0.55);
  });
}

function createNodes(positions: THREE.Vector3[]): THREE.InstancedMesh {
  const mesh = new THREE.InstancedMesh(
    new THREE.OctahedronGeometry(0.075, 0),
    new THREE.MeshBasicMaterial({ transparent: true, opacity: 0.92 }),
    positions.length,
  );
  const dummy = new THREE.Object3D();
  positions.forEach((position, index) => {
    dummy.position.copy(position);
    dummy.scale.setScalar(index >= 16 && index < 26 ? 1.45 : 1);
    dummy.updateMatrix();
    mesh.setMatrixAt(index, dummy.matrix);
  });
  mesh.instanceMatrix.needsUpdate = true;
  return mesh;
}

function createConnections(positions: THREE.Vector3[]): THREE.LineSegments {
  const vertices: number[] = [];
  const addEdge = (from: number, to: number): void => {
    vertices.push(...positions[from]!.toArray(), ...positions[to]!.toArray());
  };
  for (let index = 0; index < 16; index += 1) addEdge(index, 16 + (index % 10));
  for (let index = 16; index < 26; index += 1) addEdge(index, 26 + ((index - 16) % 16));
  for (let index = 0; index < 15; index += 1) addEdge(index, index + 1);
  for (let index = 26; index < 41; index += 1) addEdge(index, index + 1);
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute('position', new THREE.Float32BufferAttribute(vertices, 3));
  return new THREE.LineSegments(geometry, new THREE.LineBasicMaterial({ transparent: true, opacity: 0.2 }));
}

function createCore(): { core: THREE.Group; rings: THREE.Group } {
  const core = new THREE.Group();
  core.add(
    new THREE.Mesh(
      new THREE.IcosahedronGeometry(0.72, 1),
      new THREE.MeshBasicMaterial({ transparent: true, opacity: 0.34, wireframe: true }),
    ),
    new THREE.Mesh(
      new THREE.OctahedronGeometry(0.28, 1),
      new THREE.MeshBasicMaterial({ transparent: true, opacity: 0.82 }),
    ),
  );
  const rings = new THREE.Group();
  const geometry = new THREE.TorusGeometry(1.08, 0.012, 6, 96);
  for (let index = 0; index < 3; index += 1) {
    const ring = new THREE.Mesh(geometry, new THREE.MeshBasicMaterial({ transparent: true, opacity: 0.24 }));
    ring.rotation.set(index * 0.7, index * 0.95, index * 0.35);
    rings.add(ring);
  }
  return { core, rings };
}

function createParticles(): THREE.Points {
  const geometry = new THREE.BufferGeometry();
  geometry.setAttribute('position', new THREE.BufferAttribute(new Float32Array(PULSE_COUNT * 3), 3));
  return new THREE.Points(
    geometry,
    new THREE.PointsMaterial({ size: 0.065, transparent: true, opacity: 0.9, sizeAttenuation: true }),
  );
}

export function createGraphParts(): GraphParts {
  const positions = createPositions();
  const nodes = createNodes(positions);
  const lines = createConnections(positions);
  const { core, rings } = createCore();
  return { core, lines, nodes, particles: createParticles(), positions, rings };
}

export function refreshGraphTheme(parts: GraphParts): void {
  const primary = readColor('--color-primary', '#c8ff4d');
  const secondary = readColor('--color-secondary', '#77adf8');
  const accent = readColor('--color-accent', '#d788f2');
  for (let index = 0; index < NODE_COUNT; index += 1) {
    parts.nodes.setColorAt(index, index < 16 ? secondary : index < 26 ? primary : accent);
  }
  if (parts.nodes.instanceColor !== null) parts.nodes.instanceColor.needsUpdate = true;
  (parts.lines.material as THREE.LineBasicMaterial).color.copy(secondary);
  (parts.particles.material as THREE.PointsMaterial).color.copy(primary);
  parts.core.children.forEach((child) => ((child as THREE.Mesh).material as THREE.MeshBasicMaterial).color.copy(primary));
  parts.rings.children.forEach((child) => ((child as THREE.Mesh).material as THREE.MeshBasicMaterial).color.copy(accent));
}

export function updateGraphParticles(parts: GraphParts, elapsed: number): void {
  const attribute = parts.particles.geometry.getAttribute('position') as THREE.BufferAttribute;
  const target = new THREE.Vector3();
  for (let index = 0; index < PULSE_COUNT; index += 1) {
    const from = parts.positions[(index * 5) % parts.positions.length]!;
    const to = index % 2 === 0 ? new THREE.Vector3() : parts.positions[26 + ((index * 3) % 16)]!;
    const progress = (elapsed * (0.22 + (index % 4) * 0.035) + index / PULSE_COUNT) % 1;
    target.lerpVectors(from, to, progress);
    attribute.setXYZ(index, target.x, target.y, target.z);
  }
  attribute.needsUpdate = true;
}

export function disposeGraph(root: THREE.Object3D): void {
  root.traverse((child) => {
    const mesh = child as THREE.Mesh;
    mesh.geometry?.dispose();
    const material = mesh.material as GraphMaterial | GraphMaterial[] | undefined;
    if (Array.isArray(material)) material.forEach((item) => item.dispose());
    else material?.dispose();
  });
}
