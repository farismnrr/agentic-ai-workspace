export const connectedAppsCopy = {
  intro: 'Apps like ChatGPT that you let into your spaces. Remove access any time.',
  empty: { title: 'No apps connected yet', body: 'Copy a space address under Spaces & tools and add it in your AI app, such as ChatGPT.' },
  failed: 'We could not load your connected apps. Please try again in a moment.',
};

export const spacesCopy = {
  intro: 'These switches control what connected AI apps can do in each space. ' +
    'Tasks you start here still follow Coding permissions.',
  empty: { title: 'No spaces available', body: 'This server has no spaces switched on yet, so there is nothing to connect an AI app to.' },
  failed: 'We could not load your spaces. Please try again in a moment.',
};

export const approvalsCopy = {
  title: 'Skip approvals',
  body: 'Let the AI go ahead without asking you each time. Anything you turned off stays off.',
  failed: 'We could not save that switch. Please try again.',
};

/** What each tool does, in plain words. The server's own wording is written for the AI, so it is not shown. */
export const toolCopy: Record<string, string> = {
  list_files: 'See which files and folders are in the project.',
  find_files: 'Look for files by their name.',
  read_file: 'Open and read a file.',
  search_code: 'Search for a word or phrase across all the files.',
  list_tasks: 'See the checks you set up for this project, like running tests.',
  whoami: 'Tell the AI which account it is working for.',
  edit_file: 'Change part of a file.',
  write_file: 'Create a new file, or replace everything in one.',
  delete_file: 'Delete a file.',
  move_file: 'Move a file to another place, or give it a new name.',
  run_task: 'Run one of your checks, like the tests, and see if it passed.',
  run_command: 'Run commands in a safe, closed-off box so the AI can build and test things. This is also how it shows you the web app it makes.',
};

export const toolGroups = {
  read: { title: 'Look around', note: 'Look at your project without changing anything. On by default.' },
  write: {
    title: 'Change files',
    note: 'Create, edit, move and delete files. Off until you turn it on. Every change can be undone under Activity.',
  },
  exec: {
    title: 'Run things',
    note: 'Run checks and commands, and show you the web app it builds. Off until you turn it on.',
  },
} as const;

export const activityCopy = {
  intro: 'Every step a connected AI app takes in your spaces, newest first. Changes to files can be undone.',
  empty: { title: 'No activity yet', body: 'When an AI app uses your spaces, every step shows up here.' },
  failed: 'We could not load the activity. Please try again in a moment.',
  outcomes: { ok: 'Done', failed: 'Failed', off: 'Was turned off', forbidden: 'Needed permission' },
  undoErrors: {
    undo_refused: 'The file changed after this edit, so it was left as it is.',
    already_undone: 'This change was already undone.',
    not_undoable: 'This change cannot be undone.',
  },
  undoFallback: 'We could not undo that. Please try again in a moment.',
} as const;
