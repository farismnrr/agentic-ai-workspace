export const connectedAppsCopy = {
  intro: 'Control which AI apps can access your spaces.',
  empty: { title: 'No apps connected yet', body: 'Use a space address to connect an AI app.' },
  failed: 'Could not load connected apps. Try again.',
};

export const spacesCopy = {
  intro: 'Control tools used by tasks and connected AI apps.',
  empty: { title: 'No spaces available', body: 'No spaces are enabled on this server.' },
  failed: 'Could not load spaces. Try again.',
};

export const approvalsCopy = {
  title: 'Skip tool approvals',
  body: 'AI can run enabled tools without asking. Disabled tools stay blocked.',
  failed: 'Could not save this setting. Try again.',
};

/** Short descriptions only for tools whose effects need explanation. */
export const toolCopy: Record<string, string> = {
  list_files: '',
  find_files: '',
  read_file: '',
  search_code: '',
  list_tasks: 'Shows available project checks.',
  whoami: 'Shows the connected account.',
  edit_file: '',
  write_file: 'Can replace existing file contents.',
  delete_file: 'Some deletions cannot be undone.',
  move_file: '',
  run_task: 'Runs a saved project check.',
  run_command: 'Runs commands in an isolated workspace.',
  approve_plan: 'Lets connected AI approve plans without your review.',
};

export const toolGroups = {
  read: { title: 'Read files' },
  write: { title: 'Change files' },
  exec: { title: 'Run commands' },
} as const;

export const activityCopy = {
  intro: 'Recent connected-app actions. Some file changes can be undone.',
  empty: { title: 'No activity yet', body: 'Connected-app actions will appear here.' },
  failed: 'Could not load activity. Try again.',
  outcomes: { ok: 'Done', failed: 'Failed', off: 'Was turned off', forbidden: 'Needed permission' },
  undoErrors: {
    undo_refused: 'The file changed after this edit, so it was left as it is.',
    already_undone: 'This change was already undone.',
    not_undoable: 'This change cannot be undone.',
  },
  undoFallback: 'Could not undo this change. Try again.',
} as const;
