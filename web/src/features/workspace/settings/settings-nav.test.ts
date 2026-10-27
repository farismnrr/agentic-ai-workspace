import { describe, expect, it } from 'vitest';
import { activeSettingsItem, visibleNavGroups } from './settings-nav';

const workspaceItems = (admin: boolean) => visibleNavGroups(admin)
  .find((group) => group.label === 'Workspace')?.items ?? [];

describe('workspace settings navigation', () => {
  it('shows archive and extensions to ordinary users with direct routes', () => {
    expect(workspaceItems(false).map(({ id, href }) => ({ id, href }))).toEqual([
      { id: 'archive', href: '/app/archive' },
      { id: 'extensions', href: '/app/extensions' },
    ]);
  });

  it('keeps Database visible only to administrators', () => {
    expect(workspaceItems(false).some((item) => item.id === 'database')).toBe(false);
    expect(workspaceItems(true).some((item) => item.id === 'database')).toBe(true);
  });

  it('marks the settings section active for archive and extensions routes', () => {
    expect(activeSettingsItem({ name: 'archive' })).toBe('archive');
    expect(activeSettingsItem({ name: 'extensions' })).toBe('extensions');
    expect(activeSettingsItem({ name: 'extensions', projectId: 'project-1' })).toBe('extensions');
  });
});
