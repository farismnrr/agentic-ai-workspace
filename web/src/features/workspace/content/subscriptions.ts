export const subscriptionsCopy = {
  planChoice: 'Your plan',
  keyChoice: 'API key',
  planIntro: 'Sign in with the plan you already pay for. You sign in on the provider’s own page; this app never sees your password.',
  signIn: (name: string): string => `Sign in with ${name}`,
  signOut: 'Sign out',
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

/** What the default model list calls a signed-in plan. */
export const planGroupLabel = 'Your own plan';

export interface ProgramCopy {
  name: string;
  /** What the provider list calls the plan next to its name. */
  badge: string;
  connected: string;
  /** Plain steps shown while signing in. */
  steps: { open: string; finish: string };
}

export const programs: Record<'claude' | 'chatgpt', ProgramCopy> = {
  claude: {
    name: 'Claude',
    badge: 'Claude Pro or Max plan',
    connected: 'Signed in. The AI can use your Claude plan.',
    steps: {
      open: 'Open Claude and sign in. It shows you a code when you are done.',
      finish: 'Paste that code here.',
    },
  },
  chatgpt: {
    name: 'ChatGPT',
    badge: 'ChatGPT Plus or Pro plan',
    connected: 'Signed in. The AI can use your ChatGPT plan.',
    steps: {
      open: 'Open ChatGPT and sign in.',
      finish: 'Type this code on that page. This window updates by itself when you are done.',
    },
  },
};
