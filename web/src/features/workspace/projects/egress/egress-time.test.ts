import { describe, expect, it } from 'vitest';
import { effectiveStatus, remainingLabel } from './egress-time';

const end = '2026-10-27T00:00:00.000Z';
const at = Date.parse(end);

describe('egress grant expiration clock', () => {
  it('expires both pending requests and approved access at the deadline', () => {
    for (const status of ['pending', 'approved'] as const) {
      expect(effectiveStatus({ status, expiresAt: end }, at - 1)).toBe(status);
      expect(effectiveStatus({ status, expiresAt: end }, at)).toBe('expired');
    }
  });
  it('keeps final decisions and fails closed on invalid dates', () => {
    expect(effectiveStatus({ status: 'denied', expiresAt: end }, at)).toBe('denied');
    expect(effectiveStatus({ status: 'revoked', expiresAt: end }, at)).toBe('revoked');
    expect(effectiveStatus({ status: 'pending', expiresAt: 'invalid' }, at)).toBe('expired');
  });
  it('formats remaining time without negative lifetimes', () => {
    expect(remainingLabel(end, at - 25 * 60 * 60_000)).toBe('1d 1h remaining');
    expect(remainingLabel(end, at - 70 * 60_000)).toBe('1h 10m remaining');
    expect(remainingLabel(end, at - 1)).toBe('1m remaining');
    expect(remainingLabel(end, at)).toBe('Expired');
  });
});
