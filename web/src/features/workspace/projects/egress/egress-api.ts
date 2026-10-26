import { ApiError, request } from '../../../../shared/api/http';

export interface Grant {
  id: string;
  projectId: string;
  taskId: string;
  host: string;
  reason: string;
  status: 'pending' | 'approved' | 'denied' | 'revoked' | 'expired';
  requester: string;
  requestedByKind: 'person' | 'agent';
  decidedBy: string;
  expiresAt: string;
  createdAt: string;
}

const base = (projectId: string): string => `/api/projects/${encodeURIComponent(projectId)}/egress-grants`;

/** Endpoint absence means grants are switched off, not that a grant was denied. */
export const listGrants = (projectId: string): Promise<Grant[]> => request('GET', base(projectId) + '/');

async function mutate(projectId: string, url: string, body: unknown): Promise<Grant> {
  const { token } = await request<{ token: string }>('GET', base(projectId) + '/csrf');
  const response = await fetch(base(projectId) + url, {
    method: 'POST',
    credentials: 'same-origin',
    cache: 'no-store',
    headers: { 'Content-Type': 'application/json', 'X-Egress-CSRF': token },
    body: JSON.stringify(body),
  });
  const data: unknown = await response.json();
  if (!response.ok) {
    const code = typeof data === 'object' && data !== null && 'error' in data &&
      typeof data.error === 'string' ? data.error : 'unknown';
    throw new ApiError(response.status, code);
  }
  return data as Grant;
}

export function requestGrant(projectId: string, taskId: string, host: string, reason: string): Promise<Grant> {
  return mutate(projectId, '/', { taskId, host, reason });
}
export function decideGrant(projectId: string, id: string, decision: 'approve' | 'deny' | 'revoke',
  ttlHours = 24): Promise<Grant> {
  return mutate(projectId, `/${encodeURIComponent(id)}/${decision}`, decision === 'approve' ? { ttlHours } : {});
}
