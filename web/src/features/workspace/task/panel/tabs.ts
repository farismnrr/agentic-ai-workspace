export type PanelTab = 'plan' | 'documents' | 'team' | 'changes' | 'files';

const allTabs: { id: PanelTab; label: string }[] = [
  { id: 'plan', label: 'Plan' },
  { id: 'documents', label: 'Documents' },
  { id: 'team', label: 'Team' },
  { id: 'changes', label: 'Changes' },
  { id: 'files', label: 'Files' },
];

export const panelTabs = allTabs;
