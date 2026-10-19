const adjectives = [
  'amber', 'bright', 'calm', 'clever', 'cosy', 'daring', 'eager', 'gentle', 'golden', 'happy', 'jolly', 'keen', 'lively',
  'lucky', 'mellow', 'nimble', 'proud', 'quiet', 'rapid', 'sunny', 'swift', 'tidy', 'vivid', 'warm', 'witty',
];

const nouns = [
  'otter', 'harbor', 'meadow', 'lantern', 'maple', 'comet', 'orchard', 'pebble', 'river', 'sparrow', 'summit', 'tulip',
  'valley', 'willow', 'canyon', 'breeze', 'beacon', 'garden', 'island', 'forest', 'falcon', 'compass', 'bakery', 'studio',
];

const pick = (items: string[], random: () => number): string => items[Math.floor(random() * items.length)] ?? items[0] ?? '';

const capital = (word: string): string => word.charAt(0).toUpperCase() + word.slice(1);

/** A friendly two-word project name such as "Quiet Harbor", for a person who does not want to think of one. */
export function randomProjectName(random: () => number = Math.random): string {
  return `${capital(pick(adjectives, random))} ${capital(pick(nouns, random))}`;
}
