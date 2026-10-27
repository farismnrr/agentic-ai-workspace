import type { Route, SettingsTab } from '../../../app/router';
import type { IconName } from '../../../shared/ui/icon-names';

export type SettingsNavId = SettingsTab | 'archive' | 'extensions';

export interface NavItem {
  id: SettingsNavId;
  label: string;
  icon: IconName;
  href: string;
  adminOnly?: boolean;
}

export interface NavGroup {
  label: string;
  items: NavItem[];
}

export const navGroups: NavGroup[] = [
  {
    label: 'AI',
    items: [
      { id: 'models', label: 'AI models', icon: 'box', href: '/app/settings/models' },
      { id: 'persona', label: 'Persona & rules', icon: 'chat', href: '/app/settings/persona' },
      { id: 'team', label: 'Agent team', icon: 'user', href: '/app/settings/team' },
      { id: 'spaces', label: 'Spaces & tools', icon: 'folder', href: '/app/settings/spaces' },
      { id: 'connected', label: 'Connected apps', icon: 'globe', href: '/app/settings/connected' },
      { id: 'integrations', label: 'Integrations', icon: 'link', href: '/app/settings/integrations' },
      { id: 'activity', label: 'Activity', icon: 'clock', href: '/app/settings/activity' },
    ],
  },
  { label: 'Preferences', items: [{ id: 'notifications', label: 'Notifications', icon: 'bell', href: '/app/settings/notifications' }] },
  { label: 'Workspace', items: [
    { id: 'archive', label: 'Archive', icon: 'box', href: '/app/archive' },
    { id: 'extensions', label: 'Extensions', icon: 'puzzle', href: '/app/extensions' },
    { id: 'database', label: 'Database', icon: 'box', href: '/app/settings/database', adminOnly: true },
  ] },
  {
    label: 'Account',
    items: [
      { id: 'usage', label: 'Usage', icon: 'clock', href: '/app/settings/usage' },
      { id: 'account', label: 'Account', icon: 'user', href: '/app/settings/account' },
    ],
  },
];

export const currentLabel = (tab: SettingsNavId): string => navGroups.flatMap((group) => group.items).find((item) => item.id === tab)?.label ?? 'Settings';

export const isAdminSettingsTab = (tab: SettingsNavId): boolean => tab === 'database';

export const activeSettingsItem = (route: Route): SettingsNavId | undefined => {
  if (route.name === 'settings') return route.tab;
  if (route.name === 'archive') return 'archive';
  if (route.name === 'extensions') return 'extensions';
  return undefined;
};

export const visibleNavGroups = (admin: boolean): NavGroup[] => navGroups
  .map((group) => ({ ...group, items: group.items.filter((item) => admin || !item.adminOnly) }))
  .filter((group) => group.items.length > 0);
