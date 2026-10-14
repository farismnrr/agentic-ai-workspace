export const taskErrors: Record<string, string> = {
  bad_request: 'Write what you need in a few words.',
  no_project: 'Choose a project first.',
  no_model: 'No AI model is set up yet. Add a provider under Settings, AI models.',
  busy: 'The agent is still working on this task.',
  needs_answer: 'Answer the question or approval above first.',
  not_waiting: 'That question was already answered.',
  wrong_state: 'That is not possible right now.',
  undo_refused: 'The file changed after this step, so it was left as it is.',
  not_found: 'We could not find that task.',
};

export const taskFallbackError = 'Something went wrong. Please try again in a moment.';

export const homeCopy = {
  noProjects: 'Create a project first. Each project is one website or thing you are working on.',
  createProject: 'Create a project',
  caughtUp: "You're all caught up.",
};

export const proofCopy = {
  empty: { title: 'No proof yet', body: "It shows up once the work is finished and checked, with what changed and what was tested." },
  proven: 'Proven to work',
  unproven: 'Not independently checked',
  verified: 'Checked by the app',
  claimed: 'Only said by the AI',
};
