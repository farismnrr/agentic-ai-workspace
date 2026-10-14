export type PanelTab = 'plan' | 'proof' | 'changes' | 'files';

export const panelTabs: { id: PanelTab; label: string }[] = [
  { id: 'plan', label: 'Plan' },
  { id: 'proof', label: 'Proof' },
  { id: 'changes', label: 'Changes' },
  { id: 'files', label: 'Files' },
];
