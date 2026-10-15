export const integrationsCopy = {
  title: 'Integrations',
  intro: 'Connect other services so the AI can work with them for you.',
};

export const githubCopy = {
  name: 'GitHub',
  description: 'Let the AI download your repositories, save changes and push them.',
  notConfigured: 'Not set up on this server yet. Ask whoever runs it to add a GitHub App.',
  connected: (login: string): string => `Connected as @${login}`,
  connectedShort: 'The AI works on your repositories as you.',
  details: 'Commands the AI runs in your projects are signed in as you, and commits carry your name. It pushes only when you ask. Web apps it builds never get access.',
  repos: {
    button: 'Choose repositories',
    body: 'The AI can only reach repositories you picked when you installed the app on GitHub. Add or remove them there any time.',
  },
  failed: 'We could not connect GitHub. Please try again.',
  loadFailed: 'We could not load your GitHub connection. Please try again in a moment.',
  startFailed: 'We could not start the GitHub sign-in. Please try again in a moment.',
  disconnect: {
    title: 'Disconnect GitHub?',
    body: 'The AI loses access to your repositories right away. You can connect again later.',
    confirm: 'Disconnect',
    cancel: 'Stay connected',
  },
};
