export const loginCopy = {
  subtitle: 'Sign in to see what your AI team has been up to.',
  caption: 'Your work is right where you left it.',
};

export const registerCopy = {
  subtitle: 'No tech skills needed. Just tell it what you need.',
  caption: 'Everyone starts somewhere.',
};

export const FALLBACK_ERROR = 'Something went wrong on our side. Please try again in a moment.';

export const errorCopy: Record<string, string> = {
  cross_origin: 'This page was opened from an address the app does not recognise. Reload it from the main address.',
  too_many_requests: 'Too many attempts. Please wait a few minutes and try again.',
  invalid_credentials: 'That email and password do not match.',
  email_taken: 'An account with this email already exists. Try signing in instead.',
  invalid_email: 'Enter an email like you@example.com.',
  invalid_name: 'Tell us what to call you.',
  weak_password: 'Use between 8 and 128 characters.',
  wrong_password: 'That is not your current password.',
  invalid_token: 'This reset link has expired or was already used. Ask for a new one.',
};

export const forgotCopy = {
  subtitle: 'Enter your email and we will send you a link to choose a new one.',
  caption: 'It happens to everyone.',
  sent: 'If that address has an account, a reset link is on its way. It works for one hour.',
};

export const resetCopy = {
  subtitle: 'Choose a new password. You will be signed out everywhere else.',
  caption: 'A fresh start.',
  missing: 'This page needs the link from your email. Ask for a new link.',
};

export const consentCopy = {
  caption: 'You decide who gets in.',
  canDo: 'This will let it:',
  revoke: 'You can remove access any time in Settings under Connected apps.',
  expiredTitle: 'This request has expired',
  expiredBody: 'Go back to the app you were connecting and start again.',
  back: 'Go to my workspace',
};
