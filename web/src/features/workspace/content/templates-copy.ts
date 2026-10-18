/** The starting points of a new project. Names match the templates the server knows; '' is an empty project. */
export interface StartingPoint {
  id: string;
  title: string;
  body: string;
}

export const emptyStart: StartingPoint = {
  id: '',
  title: 'Empty project',
  body: 'The AI designs the look from your request. You can change it by hand afterwards.',
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

export const startCopy = {
  label: 'How do you want to start?',
  note: 'A template brings only the look. The AI still builds the logic and login for your project.',
};
