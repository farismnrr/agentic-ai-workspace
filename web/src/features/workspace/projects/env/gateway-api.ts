import { request } from '../../../../shared/api/http';

export interface GatewayChoice { id: string; name: string; baseUrl: string }
export interface GatewayView { selected: string; options: GatewayChoice[] }

const path = (id: string): string => `/api/projects/${encodeURIComponent(id)}/gateway`;

export const getGateway = (id: string): Promise<GatewayView> => request('GET', path(id));
export const setGateway = (id: string, providerId: string): Promise<undefined> => request('PUT', path(id), { providerId });
export const clearGateway = (id: string): Promise<undefined> => request('DELETE', path(id));
