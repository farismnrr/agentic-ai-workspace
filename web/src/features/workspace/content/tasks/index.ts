import type { Task } from '../task-types';
import { contactForm } from './contact-form';
import { gallery } from './gallery';
import { prices } from './prices';
import { signIn } from './signin';
import { simple } from './simple';

/** Newest first: the sidebar's Recent list relies on this order. */
export const tasks: Task[] = [
  contactForm,
  signIn,
  prices,
  gallery,
  simple({ id: 'homepage-text', projectId: 'bakery', title: 'Update homepage text', request: 'Change the homepage headline to mention our new cakes.', result: 'Headline now mentions the new cakes', updated: 'Yesterday' }),
  simple({ id: 'footer-link', projectId: 'bakery', title: 'Fix broken link in footer', request: 'The Instagram link in the footer is broken.', result: 'Instagram link now opens your profile', updated: 'Mon' }),
  simple({ id: 'faster', projectId: 'bakery', title: 'Make the site load faster', request: 'My site feels slow. Can you speed it up?', result: 'Home page now opens in 1.2 seconds (was 4.8)', updated: 'Oct 9' }),
  simple({ id: 'checkout-text', projectId: 'shop', title: 'Reword the checkout page', request: 'Make the checkout wording friendlier.', result: 'Checkout wording updated', updated: 'Oct 9' }),
  simple({ id: 'rsvp', projectId: 'wedding', title: 'Add an RSVP form', request: 'Guests should be able to say if they are coming.', result: 'RSVP form added', status: 'queued', updated: 'Today' }),
  simple({ id: 'music', projectId: 'wedding', title: 'Add background music', request: 'Play our song softly on the invitation.', result: 'Music added with a mute button', updated: 'Yesterday' }),
  simple({ id: 'about', projectId: 'portfolio', title: 'Rewrite the About page', request: 'Make my About page shorter and friendlier.', result: 'About page rewritten', updated: 'Oct 9' }),
  simple({ id: 'newsletter', projectId: 'blog', title: 'Add a newsletter sign-up', request: 'Readers should be able to subscribe.', result: 'Sign-up box added', status: 'queued', updated: 'Oct 8' }),
  simple({ id: 'booking', projectId: 'kids', title: 'Show class times in a table', request: 'List class times in a clear table.', result: 'Class times shown in a table', updated: 'Oct 2' }),
];

export const findTask = (projectId: string, taskId: string): Task | undefined =>
  tasks.find((task) => task.projectId === projectId && task.id === taskId);

export const tasksOf = (projectId: string): Task[] => tasks.filter((task) => task.projectId === projectId);
