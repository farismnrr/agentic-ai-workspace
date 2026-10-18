export type PanelTab = 'plan' | 'team' | 'proof' | 'changes' | 'files';

const allTabs: { id: PanelTab; label: string }[] = [
  { id: 'plan', label: 'Plan' },
  { id: 'team', label: 'Team' },
  { id: 'proof', label: 'Proof' },
  { id: 'changes', label: 'Changes' },
  { id: 'files', label: 'Files' },
];

/** The Team tab only exists on a task that has a team. */
export const panelTabsFor = (team: boolean): { id: PanelTab; label: string }[] => allTabs.filter((tab) => team || tab.id !== 'team');
