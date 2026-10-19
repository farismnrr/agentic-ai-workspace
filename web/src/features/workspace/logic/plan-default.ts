import { loadPreferences, savePreferences } from './preferences-api';
import type { Program } from './subscriptions-api';

/** The provider id the server uses for "this task runs on the person's own plan". */
const planId = (program: Program): string => `plan:${program}`;

/** Whether new tasks start on the person's own plan with this program. */
export async function isPlanDefault(program: Program): Promise<boolean> {
  return (await loadPreferences()).defaultProviderId === planId(program);
}

/** Makes this plan the default for new tasks, or, when turned off, clears it if it is the current default. */
export async function setPlanDefault(program: Program, on: boolean): Promise<void> {
  const prefs = await loadPreferences();
  if (on) {
    await savePreferences({ ...prefs, defaultProviderId: planId(program), defaultModel: 'default' });
  } else if (prefs.defaultProviderId === planId(program)) {
    await savePreferences({ ...prefs, defaultProviderId: '', defaultModel: '' });
  }
}
