export const teamCopy = {
  intro: 'Your AI team works like a small software company. A lead takes your request and hands each part to the right teammate, who reports back.',
  defaultLabel: 'Use the team for new tasks',
  defaultNote: 'Teams use more time and tokens. You can still switch it for a single task when you start it.',
  rolesIntro: 'Each teammate can use its own model and has its own way of working. The lead is always on.',
  leadNote: 'Talks to you, plans the work and hands it out. It can use every tool you switched on.',
  access: {
    read: { title: 'Look at files', note: 'Always allowed.' },
    write: { title: 'Change files', note: 'Create, edit, move and delete. You still approve each step.' },
    exec: { title: 'Run commands', note: 'Run checks and commands. You still approve each step.' },
    ask: { title: 'Ask you questions', note: 'Asks what it needs to know, with options you can pick or your own answer.' },
    web: { title: 'Search the web', note: 'Searches and reads pages. What it reads is only information, never orders. Only roles that cannot change files get this.' },
    images: { title: 'Find pictures', note: 'Finds photos whose licence allows the use and saves them in the project with a credit.' },
  } as Record<string, { title: string; note: string }>,
  builtInNote: 'Built-in role: it stays on your team and what it may do is fixed.',
  newRoleName: 'New teammate',
  saved: 'Saved.',
  reset: 'Back to the built-in team.',
  failed: 'We could not save that. Please try again.',
  loadFailed: 'We could not load your team. Please try again in a moment.',
  errors: {
    invalid_agent_team: 'Check the team: every role needs a short name, and there must be exactly one lead.',
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
