export type PanelTab = 'plan' | 'proof' | 'changes' | 'preview' | 'files';

export const panelTabs: { id: PanelTab; label: string }[] = [
  { id: 'plan', label: 'Plan' },
  { id: 'proof', label: 'Proof' },
  { id: 'changes', label: 'Changes' },
  { id: 'preview', label: 'Preview' },
  { id: 'files', label: 'Files' },
];
