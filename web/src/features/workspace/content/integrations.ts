export const githubCopy = {
  intro: 'Sign in to GitHub once and the AI can work with your repositories the way you would: download them, save changes and push them.',
  notConfigured: 'GitHub is not set up on this server yet. Ask whoever runs it to add a GitHub App (see the deployment guide).',
  repos: {
    title: 'Choose repositories',
    body: 'The AI can only reach repositories you picked when you installed the app on GitHub. Add or remove them there any time.',
  },
  connected: (login: string): string => `Connected as @${login}`,
  connectedBody: 'Commands the AI runs in your projects are signed in as you. Commits carry your name. It pushes only when you ask.',
  notConnected: { title: 'Not connected', body: 'Nothing here can reach GitHub until you connect.' },
  result: {
    connected: 'GitHub is connected.',
    failed: 'We could not connect GitHub. Please try again.',
  },
  failed: 'We could not load your GitHub connection. Please try again in a moment.',
  startFailed: 'We could not start the GitHub sign-in. Please try again in a moment.',
  disconnect: {
    title: 'Disconnect GitHub?',
    body: 'The AI loses access to your repositories right away. You can connect again later.',
    confirm: 'Disconnect',
    cancel: 'Stay connected',
  },
};
