export const integrationsCopy = {
  title: 'Integrations',
};

export const githubCopy = {
  name: 'GitHub',
  description: 'Access selected repositories and push changes when asked.',
  notConfigured: 'Ask an admin to configure the GitHub App.',
  connected: (login: string): string => `Connected as @${login}`,
  details: 'Uses your GitHub identity and selected repos. Pushes only when asked. Project sites have no access.',
  repos: {
    button: 'Choose repositories',
  },
  failed: 'We could not connect GitHub. Please try again.',
  loadFailed: 'We could not load your GitHub connection. Please try again in a moment.',
  startFailed: 'We could not start the GitHub sign-in. Please try again in a moment.',
  disconnect: {
    title: 'Disconnect GitHub?',
    body: 'Repository access stops immediately. You can reconnect later.',
    confirm: 'Disconnect',
    cancel: 'Stay connected',
  },
};
