import { errorIllustrations } from './assets/illustrations';

export type ErrorKind = 'not-found' | 'forbidden';

export interface ErrorCopy {
  title: string;
  label: string;
  before: string;
  marked: string;
  body: string;
  illustration: (typeof errorIllustrations)[keyof typeof errorIllustrations];
  home: string;
  second: string;
  secondAnonymous: string;
}

export const errorCopy: Record<ErrorKind, ErrorCopy> = {
  'not-found': {
    title: 'Page not found',
    label: 'Error 404',
    before: 'This page',
    marked: 'wandered off.',
    body: 'The link might be old, or the page was moved. Let’s get you back on track.',
    illustration: errorIllustrations.signpost,
    home: 'Go to Home',
    second: 'Go back',
    secondAnonymous: 'Go back',
  },
  forbidden: {
    title: 'This page is private',
    label: 'Error 403',
    before: 'This page is',
    marked: 'private.',
    body: 'You don’t have access to this page. If you think you should, ask the person who shared it with you.',
    illustration: errorIllustrations.gardenGate,
    home: 'Go to Home',
    second: 'Sign in with another account',
    secondAnonymous: 'Sign in',
  },
};
