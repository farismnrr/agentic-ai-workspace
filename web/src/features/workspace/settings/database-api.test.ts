import { describe, expect, it } from 'vitest';
import { certificateErrors, databaseDefaultPorts, databaseKindLabels, databaseNameLabels, tlsChoiceFromSettings, tlsForChoice, type CertificateState } from './database-api';

const emptyCertificates: CertificateState = {
  caSet: false, clientCertSet: false, clientKeySet: false,
  caCert: '', clientCert: '', clientKey: '',
  caError: '', clientCertError: '', clientKeyError: '',
};

describe('shared database settings', () => {
  it('provides the kind labels, field labels and default ports', () => {
    expect(databaseKindLabels).toEqual({ postgres: 'PostgreSQL', mysql: 'MySQL', mongodb: 'MongoDB' });
    expect(databaseDefaultPorts).toEqual({ postgres: 5432, mysql: 3306, mongodb: 27017 });
    expect(databaseNameLabels.mongodb).toBe('Auth database (usually admin)');
    expect(databaseNameLabels.mysql).toBe('Database name');
  });

  it('maps the saved TLS mode to the visible three choices', () => {
    expect(tlsChoiceFromSettings('none', false)).toBe('none');
    expect(tlsChoiceFromSettings('ca', false)).toBe('ca');
    expect(tlsChoiceFromSettings('full', false)).toBe('full');
    expect(tlsChoiceFromSettings('verify-full', true)).toBe('full');
    expect(tlsForChoice('full')).toBe('full');
  });

  it('requires the certificate files selected by the TLS choice', () => {
    expect(certificateErrors('none', emptyCertificates)).toEqual({ caError: '', clientCertError: '', clientKeyError: '' });
    expect(certificateErrors('ca', emptyCertificates).caError).toBe('Choose a CA certificate.');
    expect(certificateErrors('full', emptyCertificates)).toEqual({
      caError: 'Choose a CA certificate.', clientCertError: 'Choose a client certificate.', clientKeyError: 'Choose a client key.',
    });
    expect(certificateErrors('full', { ...emptyCertificates, caSet: true, clientCertSet: true, clientKeySet: true })).toEqual({
      caError: '', clientCertError: '', clientKeyError: '',
    });
    expect(certificateErrors('ca', { ...emptyCertificates, caSet: true, caError: 'Choose a file that is 16 KB or smaller.' }).caError)
      .toBe('Choose a file that is 16 KB or smaller.');
  });
});
