import type { StatusTone } from '../content/status';

export const badgeTone: Record<StatusTone, string> = {
  primary: 'badge-primary badge-soft',
  info: 'badge-info badge-soft',
  warning: 'badge-warning badge-soft',
  error: 'badge-error badge-soft',
  neutral: 'badge-ghost',
};

export const iconTone: Record<StatusTone, string> = {
  primary: 'text-primary',
  info: 'text-info',
  warning: 'text-warning',
  error: 'text-error',
  neutral: 'text-[var(--brand-text-tertiary)]',
};

export const dotTone: Record<StatusTone, string> = {
  primary: 'status-primary',
  info: 'status-info',
  warning: 'status-warning',
  error: 'status-error',
  neutral: 'status-neutral',
};
