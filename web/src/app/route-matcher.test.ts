import { describe, expect, it } from 'vitest';
import { matchRoute } from './route-matcher';
import { activeSettingsItem } from '../features/workspace/settings/settings-nav';

describe('Settings navigation routes', () => {
  it('matches Archive to its selected Settings item', () => {
    const route = matchRoute('/app/archive');
    expect(route).toEqual({ name: 'archive' });
    if (route !== undefined) expect(activeSettingsItem(route)).toBe('archive');
  });

  it('matches Extensions to its selected Settings item', () => {
    const route = matchRoute('/app/extensions');
    expect(route).toEqual({ name: 'extensions' });
    if (route !== undefined) expect(activeSettingsItem(route)).toBe('extensions');
  });

  it('keeps the project id and selects Extensions on project-scoped routes', () => {
    const route = matchRoute('/app/extensions/project-1');
    expect(route).toEqual({ name: 'extensions', projectId: 'project-1' });
    if (route !== undefined) expect(activeSettingsItem(route)).toBe('extensions');
  });
});
