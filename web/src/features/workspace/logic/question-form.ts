import type { Question, TaskEvent } from './task-types';

/** What the person has entered for one question: the options ticked and their own typed answer. */
export interface Draft {
  picked: string[];
  other: string;
}

type QuestionEvent = Extract<TaskEvent, { kind: 'question' }>;

/** The form of a question event; one saved before forms becomes a form of a single question. */
export function questionsOf(event: Pick<QuestionEvent, 'text' | 'options' | 'questions'>): Question[] {
  if (event.questions !== undefined && event.questions.length > 0) return event.questions;
  const options = (event.options ?? []).map((label) => ({ label }));
  return [{ text: event.text, type: options.length > 0 ? 'single' : 'text', options }];
}

export const blankDraft: Draft = { picked: [], other: '' };

export const emptyDrafts = (questions: Question[]): Draft[] => questions.map(() => blankDraft);

/** The replies for one question: a typed answer replaces the pick of a single question and joins a multi one. */
export function repliesFor(question: Question, draft: Draft): string[] {
  const other = draft.other.trim();
  if (question.type === 'text') return other === '' ? [] : [other];
  if (question.type === 'single') return other === '' ? draft.picked.slice(0, 1) : [other];
  return other === '' ? draft.picked : [...draft.picked, other];
}

export const isComplete = (questions: Question[], drafts: Draft[]): boolean =>
  questions.every((question, index) => repliesFor(question, drafts[index] ?? blankDraft).length > 0);

export const toAnswers = (questions: Question[], drafts: Draft[]): string[][] =>
  questions.map((question, index) => repliesFor(question, drafts[index] ?? blankDraft));
