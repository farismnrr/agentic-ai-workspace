/** The starting points of a new project. Names match the templates the server knows; '' is an empty project. */
export interface StartingPoint {
  id: string;
  title: string;
  body: string;
}

export const emptyStart: StartingPoint = {
  id: '',
  title: 'Without a template',
  body: 'An empty project. The AI designs the look from your request, and you refine it by hand.',
};

export const templateStarts: StartingPoint[] = [
  { id: 'landing', title: 'Landing page', body: 'One page that sells: features, pricing, questions.' },
  { id: 'saas', title: 'Product website', body: 'Home, pricing, docs, blog and sign-in pages.' },
  { id: 'dashboard', title: 'Dashboard', body: 'The part behind the login: sidebar, tables, charts, settings.' },
  { id: 'docs', title: 'Documentation', body: 'Guides with a menu and search.' },
  { id: 'chat', title: 'AI chat', body: 'A chat screen with conversation history.' },
  { id: 'portfolio', title: 'Portfolio', body: 'Projects, posts and talks of a person or studio.' },
  { id: 'changelog', title: 'Release notes', body: 'What changed, version by version.' },
  { id: 'editor', title: 'Text editor', body: 'A rich text writing screen.' },
  { id: 'calendar', title: 'Calendar', body: 'Day, week and month views.' },
];

export const githubStart: StartingPoint = {
  id: 'github',
  title: 'From a GitHub repository',
  body: 'Bring a repository you already have, with its history. No template.',
};

export const templateSource: StartingPoint = {
  id: 'template',
  title: 'From a template',
  body: 'Start with a ready-made look, or with nothing at all. The AI still builds the logic and login.',
};

export const newProjectCopy = {
  title: 'New project',
  sourceHeading: 'How do you want to start?',
  startHeading: 'Pick a template',
  startLead: 'A template brings only the look. Or choose to start without one.',
  whereHeading: 'Where should it live?',
  project: { title: 'New project', body: 'The files stay in this workspace.' },
  repo: { title: 'New repository', body: 'Also creates a repository on your GitHub account, with a README.' },
  detailsHeading: 'Name it',
  nameHint: 'Just a label for you. It can be changed any time, and other projects may share it.',
  random: 'Suggest a name',
  idLabel: 'Project ID',
  idEdit: 'Edit',
  idPermanent: 'Permanent: it cannot be changed after the project is created.',
  idChecking: 'Checking…',
  idFree: (id: string): string => `Available. Your app will open at /run/${id}/`,
  idTaken: (id: string): string => `“${id}” is already used by another project.`,
  idInvalid: 'Use 6 to 30 lowercase letters, digits or dashes, starting with a letter.',
  useSuggestion: (id: string): string => `Use “${id}”`,
  chosen: 'Starting from',
  change: 'Change',
  back: 'Back',
  cancel: 'Cancel',
  connectHint: 'Connect GitHub to use this.',
  connectLink: 'Connect GitHub',
  create: { project: 'Create project', repo: 'Create repository and project', github: 'Create and import' },
};
