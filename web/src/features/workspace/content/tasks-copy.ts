export const taskErrors: Record<string, string> = {
  bad_request: 'Write what you need.',
  no_project: 'Choose a project first.',
  no_model: 'No AI model is set up yet. Add a provider under Settings, AI models.',
  busy: 'The agent is still working on this task.',
  needs_answer: 'Answer the question or approval above first.',
  not_waiting: 'That question was already answered.',
  wrong_state: 'That is not possible right now.',
  undo_refused: 'The file changed after this step, so it was left as it is.',
  not_found: 'We could not find that task.',
  bad_effort: 'That effort level is not available.',
  binary: 'Only pictures can be attached: PNG, JPG, WebP, GIF or SVG.',
  too_large: 'A picture can be 10 MB at most.',
  exists: 'A picture with that name is already in the project. Rename it and try again.',
};

export const contextCopy = {
  label: 'Memory used',
  /** One line, so the tip never grows taller than the header row it opens beside. */
  hover: "How much of the AI's memory this task uses",
  tip: "How much of the AI's memory this task fills. When it is nearly full, the AI summarizes its notes and keeps going.",
};

export const taskFallbackError = 'Something went wrong. Please try again in a moment.';

export const homeCopy = {
  noProjects: 'Create a project first. Each project is one website or thing you are working on.',
  createProject: 'Create a project',
  caughtUp: "You're all caught up.",
};

export const proofCopy = {
  proven: 'Proven to work',
  unproven: 'Not independently checked',
  verified: 'Checked by the app',
};

interface CheckSummaryInput {
  verified: boolean;
  passed: boolean;
}

const plural = (count: number, singular: string): string => `${String(count)} ${singular}${count === 1 ? '' : 's'}`;

export function summarizeChecks(checks: CheckSummaryInput[]): string {
  const verified = checks.filter((check) => check.verified);
  const claimed = checks.filter((check) => !check.verified);
  const failedVerified = verified.filter((check) => !check.passed).length;
  const failedClaimed = claimed.filter((check) => !check.passed).length;

  if (checks.length === 0) return 'No checks were reported.';
  if (failedVerified > 0) return `${plural(failedVerified, 'check')} ran but did not pass.`;
  if (failedClaimed > 0) return `The AI says it ran ${plural(claimed.length, 'check')}; ${plural(failedClaimed, 'check')} did not pass, and nobody confirmed the results.`;
  if (claimed.length > 0 && verified.length > 0) {
    const otherChecks = `${String(claimed.length)} other ${claimed.length === 1 ? 'check' : 'checks'}`;
    return `${plural(verified.length, 'check')} ran and passed; the AI says it ran ${otherChecks}, which nobody confirmed.`;
  }
  if (claimed.length > 0) return `The AI says it ran ${plural(claimed.length, 'check')}; nobody confirmed the results.`;
  return `${plural(verified.length, 'check')} ran and passed.`;
}
