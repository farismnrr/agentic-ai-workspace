import { writable, type Readable } from 'svelte/store';
import { describeTaskError } from './task-errors';
import {
  acceptTask, answerApproval, answerQuestion, cancelTask, chatOnExternalTask, confirmTask, retryTask,
  sendMessage, undoStep,
} from './tasks-api';
import type { Thread } from './thread';
import { refresh } from './workspace-store';

/** What the person can do from inside the thread; the page that owns the task provides these. */
export interface ThreadActions {
  allow: (eventId: string, allow: boolean) => void;
  choose: (eventId: string, answers: string[][]) => void;
  undo: (eventId: string) => void;
}

export interface TaskController {
  actions: ThreadActions;
  /** The last problem in plain words, or an empty string. */
  error: Readable<string>;
  accept: () => void;
  confirm: () => void;
  stop: () => void;
  retry: () => void;
  /** Resolves true when the message was accepted. */
  follow: (text: string, external: boolean) => Promise<{ accepted: boolean; linkedTaskId?: string }>;
}

/** Runs the buttons of one task and refreshes the thread and lists afterwards. */
export function createController(taskId: string, thread: Thread): TaskController {
  const error = writable('');

  const act = async <T>(run: () => Promise<T>): Promise<T | undefined> => {
    error.set('');
    let result: T | undefined;
    try {
      result = await run();
    } catch (failure) {
      error.set(describeTaskError(failure));
    } finally {
      await Promise.all([thread.refresh(), refresh()]);
    }
    return result;
  };

  return {
    error,
    actions: {
      allow: (eventId, allow) => void act(() => answerApproval(taskId, eventId, allow)),
      choose: (eventId, answers) => void act(() => answerQuestion(taskId, eventId, answers)),
      undo: (eventId) => void act(() => undoStep(taskId, eventId)),
    },
    accept: () => void act(() => acceptTask(taskId)),
    confirm: () => void act(() => confirmTask(taskId)),
    stop: () => void act(() => cancelTask(taskId)),
    retry: () => void act(() => retryTask(taskId)),
    follow: async (text, external) => {
      const result = await act(async (): Promise<{ linkedTaskId?: string }> => {
        if (external) return chatOnExternalTask(taskId, text);
        await sendMessage(taskId, text);
        return {};
      });
      return result === undefined
        ? { accepted: false }
        : { accepted: true, ...result };
    },
  };
}
