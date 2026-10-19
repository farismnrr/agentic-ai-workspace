export const subscriptionsCopy = {
  title: 'Your own AI plans',
  intro: 'Sign in with a plan you already pay for. You sign in on the provider’s own page; this app never sees your password.',
  signIn: 'Sign in',
  signOut: 'Sign out',
  useForNew: 'Use for new tasks',
  loadFailed: 'We could not check this sign-in. Please try again in a moment.',
  errors: {
    badCode: 'That code does not look right. Copy it again from the provider’s page.',
    failed: 'The sign-in did not finish. Please try again.',
    unavailable: 'The sign-in cannot start right now. Please try again in a moment.',
  },
  signOutDialog: {
    title: (name: string): string => `Sign out of ${name}?`,
    body: 'The AI stops using this plan right away. You can sign in again later.',
    confirm: 'Sign out',
    cancel: 'Stay signed in',
  },
};

export interface ProgramCopy {
  name: string;
  description: string;
  connected: string;
  /** Plain steps shown while signing in. */
  steps: { open: string; finish: string };
}

export const programs: Record<'claude' | 'chatgpt', ProgramCopy> = {
  claude: {
    name: 'Claude',
    description: 'Use your own Claude plan (Pro or Max) for the AI’s work.',
    connected: 'Signed in. The AI can use your Claude plan.',
    steps: {
      open: 'Open Claude and sign in. It shows you a code when you are done.',
      finish: 'Paste that code here.',
    },
  },
  chatgpt: {
    name: 'ChatGPT',
    description: 'Use your own ChatGPT plan (Plus or Pro) for the AI’s work.',
    connected: 'Signed in. The AI can use your ChatGPT plan.',
    steps: {
      open: 'Open ChatGPT and sign in.',
      finish: 'Type this code on that page. This window updates by itself when you are done.',
    },
  },
};
