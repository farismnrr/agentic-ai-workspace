export const connectedAppsCopy = {
  intro: 'Apps like ChatGPT that you let into your spaces. Remove access any time.',
  empty: 'Nothing is connected yet. Copy a space address under Spaces & tools and add it in your AI app.',
  failed: 'We could not load your connected apps. Please try again in a moment.',
};

export const spacesCopy = {
  intro: 'A space is a set of tools your AI apps can use. Give an app the address of the space you want it to reach.',
  empty: 'No spaces are switched on for this server yet.',
  failed: 'We could not load your spaces. Please try again in a moment.',
};

export const toolGroups = {
  read: { title: 'Look around', note: 'Read files, search and list tasks. On by default.' },
  write: {
    title: 'Change files',
    note: 'Create, edit, move and delete files. Off until you turn it on. Every change can be undone under Activity.',
  },
  exec: { title: 'Run tasks', note: 'Run the tasks you set up in .workspace/tasks.json, such as tests. Off until you turn it on.' },
} as const;

export const activityCopy = {
  intro: 'Every step an AI app takes in your spaces, newest first. Changes to files can be undone.',
  empty: 'Nothing has happened yet. When an AI app uses your spaces, every step shows up here.',
  failed: 'We could not load the activity. Please try again in a moment.',
  outcomes: { ok: 'Done', failed: 'Failed', off: 'Was turned off', forbidden: 'Needed permission' },
  undoErrors: {
    undo_refused: 'The file changed after this edit, so it was left as it is.',
    already_undone: 'This change was already undone.',
    not_undoable: 'This change cannot be undone.',
  },
  undoFallback: 'We could not undo that. Please try again in a moment.',
} as const;
