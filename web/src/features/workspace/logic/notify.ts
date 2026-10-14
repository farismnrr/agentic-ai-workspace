import { statusMeta } from '../content/status';
import { tasks } from './workspace-store';
import type { TaskStatus } from './task-types';

export interface NotifyChoices {
  /** A task needs an answer or is ready for review. */
  asks: boolean;
  /** A task failed or stopped. */
  problems: boolean;
}

const key = 'notify-choices';
const defaults: NotifyChoices = { asks: true, problems: true };

/** Notifications are per device, so the choices live in this browser only. */
export function loadChoices(): NotifyChoices {
  try {
    const stored = JSON.parse(window.localStorage.getItem(key) ?? 'null') as Partial<NotifyChoices> | null;
    return { asks: stored?.asks ?? defaults.asks, problems: stored?.problems ?? defaults.problems };
  } catch {
    return defaults;
  }
}

export function saveChoices(choices: NotifyChoices): void {
  try {
    window.localStorage.setItem(key, JSON.stringify(choices));
  } catch {
    // Private windows may refuse storage; the choice then lasts until reload.
  }
}

export const supported = (): boolean => 'Notification' in window;

export const permission = (): NotificationPermission => (supported() ? Notification.permission : 'denied');

export const askPermission = async (): Promise<NotificationPermission> =>
  supported() ? Notification.requestPermission() : 'denied';

function wanted(status: TaskStatus, choices: NotifyChoices): boolean {
  if (status === 'needs-input' || status === 'review') return choices.asks;
  return status === 'failed' && choices.problems;
}

/** Tells the person, while this tab is in the background, when a task starts waiting on them. */
export function startNotifications(): () => void {
  let seen: Map<string, TaskStatus> | undefined;

  return tasks.subscribe((list) => {
    const next = new Map(list.map((task) => [task.id, task.status]));
    const before = seen;
    seen = next;
    if (before === undefined || !document.hidden || permission() !== 'granted') return;
    const choices = loadChoices();
    for (const task of list) {
      if (before.get(task.id) === task.status || !wanted(task.status, choices)) continue;
      const alert = new Notification(task.title, { body: statusMeta[task.status].label, tag: task.id });
      alert.onclick = () => {
        window.focus();
      };
    }
  });
}
