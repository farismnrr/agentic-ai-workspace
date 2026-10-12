import { mount } from 'svelte';

import App from './app/App.svelte';
import './app.css';
import { initTheme } from './shared/theme/theme';

const target = document.getElementById('app');

if (target === null) {
  throw new Error('Application mount target was not found');
}

initTheme();
mount(App, { target });
