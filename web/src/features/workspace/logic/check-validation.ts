import { splitCommand } from './command';

const NAME_PATTERN = /^[a-z0-9][a-z0-9_-]{0,39}$/;
const MAX_WORDS = 24;
const MAX_WORD_LENGTH = 300;
export const MAX_SECONDS = 600;

export const validateCheckName = (value: string): string =>
  NAME_PATTERN.test(value.trim()) ? '' : 'Use lowercase letters, numbers, - or _ (up to 40), starting with a letter or number.';

export function validateCheckCommand(value: string): string {
  const words = splitCommand(value);
  if (words.length === 0) return 'Enter the command to run, for example go test ./...';
  if (words.length > MAX_WORDS) return `Use at most ${String(MAX_WORDS)} words.`;
  if (words.some((word) => word.length > MAX_WORD_LENGTH)) return `Each word can be at most ${String(MAX_WORD_LENGTH)} characters.`;
  return '';
}

export function validateCheckSeconds(value: string): string {
  if (value.trim() === '') return '';
  const seconds = Number(value);
  return Number.isInteger(seconds) && seconds >= 1 && seconds <= MAX_SECONDS ? '' : `Enter a whole number from 1 to ${String(MAX_SECONDS)}.`;
}
