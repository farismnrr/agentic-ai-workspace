import type { Grant } from './egress-api';

/** A fetched status must still expire while the page is open. */
export function effectiveStatus(grant: Pick<Grant, 'status' | 'expiresAt'>, now: number): Grant['status'] {
  if (grant.status !== 'pending' && grant.status !== 'approved') return grant.status;
  const end = Date.parse(grant.expiresAt);
  return !Number.isFinite(end) || now >= end ? 'expired' : grant.status;
}

export function remainingLabel(expiresAt: string, now: number): string {
  const end = Date.parse(expiresAt);
  if (!Number.isFinite(end) || now >= end) return 'Expired';
  const minutes = Math.ceil((end - now) / 60_000);
  const days = Math.floor(minutes / 1440);
  const hours = Math.floor((minutes % 1440) / 60);
  if (days > 0) return `${String(days)}d ${String(hours)}h remaining`;
  if (hours > 0) return `${String(hours)}h ${String(minutes % 60)}m remaining`;
  return `${String(minutes)}m remaining`;
}
