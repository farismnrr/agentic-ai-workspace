export const MIN_PASSWORD_LENGTH = 8;

export type StrengthLabel = 'Too short' | 'Weak password' | 'Okay password' | 'Good password' | 'Strong password';

export interface Strength {
  score: number;
  label: StrengthLabel;
}

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function validateName(value: string): string {
  return value.trim() === '' ? 'Tell us what to call you.' : '';
}

export function validateEmail(value: string): string {
  return EMAIL_PATTERN.test(value.trim()) ? '' : 'Enter an email like you@example.com.';
}

export function validatePassword(value: string): string {
  return value.length >= MIN_PASSWORD_LENGTH ? '' : `Use at least ${String(MIN_PASSWORD_LENGTH)} characters.`;
}

export function validateConfirm(password: string, confirm: string): string {
  return password === confirm ? '' : 'Passwords do not match.';
}

const labelsByScore: Record<number, StrengthLabel> = {
  0: 'Too short',
  1: 'Too short',
  2: 'Weak password',
  3: 'Okay password',
  4: 'Good password',
  5: 'Strong password',
};

/** Score 0-5: one point per character class plus length tiers, kept deliberately simple. */
export function scorePassword(value: string): Strength {
  if (value.length < MIN_PASSWORD_LENGTH) {
    return { score: value.length === 0 ? 0 : 1, label: 'Too short' };
  }
  const classes = [/[a-z]/, /[A-Z]/, /\d/, /[^A-Za-z0-9]/].filter((re) => re.test(value)).length;
  const score = Math.min(5, 1 + classes + (value.length >= 12 ? 1 : 0));
  return { score, label: labelsByScore[score] ?? 'Weak password' };
}
