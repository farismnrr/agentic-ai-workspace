import { describe, expect, it } from 'vitest';
import { sslModeForTLSChoice, tlsChoiceFromSettings } from './database-api';

describe('database TLS choices', () => {
  it('maps disabled settings to No TLS', () => {
    expect(tlsChoiceFromSettings('disable', false)).toBe('none');
    expect(tlsChoiceFromSettings('disable', true)).toBe('none');
  });

  it('maps a saved client certificate to Full', () => {
    expect(tlsChoiceFromSettings('verify-full', true)).toBe('full');
  });

  it('maps other saved modes to CA only', () => {
    expect(tlsChoiceFromSettings('allow', false)).toBe('ca');
    expect(tlsChoiceFromSettings('prefer', false)).toBe('ca');
    expect(tlsChoiceFromSettings('require', false)).toBe('ca');
    expect(tlsChoiceFromSettings('verify-ca', false)).toBe('ca');
    expect(tlsChoiceFromSettings('verify-full', false)).toBe('ca');
  });

  it('uses only disabled or verified TLS modes when saving', () => {
    expect(sslModeForTLSChoice('none')).toBe('disable');
    expect(sslModeForTLSChoice('ca')).toBe('verify-full');
    expect(sslModeForTLSChoice('full')).toBe('verify-full');
  });
});
