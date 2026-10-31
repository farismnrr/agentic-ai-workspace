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

/** Short, human-facing descriptions. Keep the detailed MCP instructions separate. */
export const toolCopy: Record<string, string> = {
  list_files: 'List project files and folders.',
  find_files: 'Find files by name or pattern.',
  read_file: 'Read a project file.',
  search_code: 'Find text in project code.',
  list_tasks: 'See available project checks.',
  whoami: 'See which account is connected.',
  list_projects: 'See projects available to this app.',
  log_message: 'Save a message to the current task.',
  edit_file: 'Change selected text in a file.',
  write_file: 'Create or replace a file.',
  delete_file: 'Delete a file permanently.',
  move_file: 'Move or rename a file.',
  run_task: 'Run a saved project check.',
  run_command: 'Run a command in the sandbox.',
  marimo_execute: 'Run Python code in a MoLab notebook.',
  screenshot_session: 'Manage the screenshot test login.',
  approve_plan: 'Allow AI to approve plans without asking.',
};

/** Unknown tools still get a short description rather than the full MCP guide. */
export function shortToolDescription(name: string, description: string): string {
  if (toolCopy[name]) return toolCopy[name];
  const firstSentence = description.trim().replace(/\s+/g, ' ').split(/\.\s/)[0];
  if (!firstSentence) return 'Use this workspace tool.';
  return firstSentence.length > 70 ? `${firstSentence.slice(0, 69).trimEnd()}…` : firstSentence;
}

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
