import { request } from '../../../shared/api/http';

export interface ConsentScope {
  id: string;
  description: string;
}

export interface ConsentRequest {
  clientName: string;
  redirectHost: string;
  scopes: ConsentScope[];
}

const path = (id: string): string => `/api/oauth/requests/${encodeURIComponent(id)}`;

export const loadConsent = (id: string): Promise<ConsentRequest> => request<ConsentRequest>('GET', path(id));

/** Records the person's choice and returns where to send the browser next. */
export async function decide(id: string, approve: boolean): Promise<string> {
  const result = await request<{ redirect: string }>('POST', `${path(id)}/decision`, { approve });
  return result.redirect;
}
