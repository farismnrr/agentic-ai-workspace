import type { SettingsTab } from '../../../app/router';
import type { IconName } from '../../../shared/ui/icon-names';

export interface NavItem {
  id: SettingsTab;
  label: string;
  icon: IconName;
}

export interface NavGroup {
  label: string;
  items: NavItem[];
}

export const navGroups: NavGroup[] = [
  {
    label: 'AI',
    items: [
      { id: 'models', label: 'AI models', icon: 'box' },
      { id: 'persona', label: 'Persona & rules', icon: 'chat' },
      { id: 'team', label: 'Agent team', icon: 'user' },
      { id: 'spaces', label: 'Spaces & tools', icon: 'folder' },
      { id: 'connected', label: 'Connected apps', icon: 'globe' },
      { id: 'integrations', label: 'Integrations', icon: 'link' },
      { id: 'activity', label: 'Activity', icon: 'clock' },
    ],
  },
  { label: 'Preferences', items: [{ id: 'notifications', label: 'Notifications', icon: 'bell' }] },
  { label: 'Workspace', items: [{ id: 'database', label: 'Database', icon: 'box' }] },
  {
    label: 'Account',
    items: [
      { id: 'usage', label: 'Usage', icon: 'clock' },
      { id: 'account', label: 'Account', icon: 'user' },
    ],
  },
];

export const currentLabel = (tab: SettingsTab): string => navGroups.flatMap((group) => group.items).find((item) => item.id === tab)?.label ?? 'Settings';

export const isAdminSettingsTab = (tab: SettingsTab): boolean => tab === 'database';

export const visibleNavGroups = (admin: boolean): NavGroup[] => navGroups
  .map((group) => ({ ...group, items: group.items.filter((item) => admin || !isAdminSettingsTab(item.id)) }))
  .filter((group) => group.items.length > 0);
