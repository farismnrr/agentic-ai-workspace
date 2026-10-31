export const teamCopy = {
  rolesIntro: 'Set models and permissions for each role. The lead stays on.',
  leadNote: 'Plans and delegates work using enabled tools.',
  access: {
    read: { title: 'Look at files', note: 'Always allowed.' },
    write: { title: 'Change files', note: 'Create, edit, move or delete files.' },
    exec: { title: 'Run commands', note: 'Run project commands and checks.' },
    ask: { title: 'Ask you questions', note: 'When input is needed.' },
    web: { title: 'Search the web', note: 'Reads untrusted pages; only available to non-editing roles.' },
    images: { title: 'Find pictures', note: 'Saves licensed images with credit.' },
  } as Record<string, { title: string; note: string }>,
  builtInNote: 'Built-in role with fixed permissions.',
  newRoleName: 'New teammate',
  saved: 'Saved.',
  reset: 'Back to the built-in team.',
  failed: 'We could not save that. Please try again.',
  loadFailed: 'Could not load your team. Try again.',
  errors: {
    invalid_agent_team: 'Each role needs a name and exactly one lead is required.',
  } as Record<string, string>,
};

/** What a role may do, in words, from its access groups. */
export function accessSummary(access: string[]): string {
  const can = ['look at files'];
  if (access.includes('write')) can.push('change files');
  if (access.includes('exec')) can.push('run commands');
  if (access.includes('ask')) can.push('ask you questions');
  if (access.includes('web')) can.push('search the web');
  if (access.includes('images')) can.push('find pictures');
  return `Can ${can.join(', ').replace(/, ([^,]*)$/, ' and $1')}.`;
}
