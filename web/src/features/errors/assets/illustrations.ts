import type { IllustrationAsset } from '../../../shared/ui/Illustration.svelte';
import gardenGate from './garden-gate.webp';
import signpost from './signpost.webp';

export const errorIllustrations = {
  signpost: { src: signpost, width: 387, height: 616, alt: '' },
  gardenGate: { src: gardenGate, width: 576, height: 525, alt: '' },
} satisfies Record<string, IllustrationAsset>;
