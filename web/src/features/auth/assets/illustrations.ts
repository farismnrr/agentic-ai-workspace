import type { IllustrationAsset } from '../../../shared/ui/Illustration.svelte';
import coatHook from './coat-hook.webp';
import seedling from './seedling.webp';

export const authIllustrations = {
  coatHook: { src: coatHook, width: 502, height: 758, alt: '' },
  seedling: { src: seedling, width: 301, height: 667, alt: '' },
} satisfies Record<string, IllustrationAsset>;
