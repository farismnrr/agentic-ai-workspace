import { request } from '../../../shared/api/http';

/** One thing an extension may do, in plain words; high means it asks for a lot of trust. */
export interface Capability {
  text: string;
  risk: 'low' | 'high';
}

export interface Extension {
  projectId: string;
  name: string;
  version: string;
  description: string;
  /** The name the downloaded package gets. */
  fileName: string;
  downloadUrl: string;
  capabilities: Capability[];
  size: number;
  files: number;
  /** ISO timestamp. */
  deployedAt: string;
}

export const loadExtensions = async (): Promise<Extension[]> =>
  (await request<{ extensions: Extension[] }>('GET', '/api/extensions')).extensions;

export const removeExtension = (projectId: string): Promise<undefined> =>
  request<undefined>('DELETE', `/api/extensions/${encodeURIComponent(projectId)}`);
