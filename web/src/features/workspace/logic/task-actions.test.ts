import { beforeEach, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import { ApiError } from '../../../shared/api/http';

const { acceptTask, answerApproval, answerQuestion, cancelTask, chatOnExternalTask, refresh, retryTask, sendMessage, undoStep } = vi.hoisted(() => ({
  acceptTask: vi.fn(),
  answerApproval: vi.fn(),
  answerQuestion: vi.fn(),
  cancelTask: vi.fn(),
  chatOnExternalTask: vi.fn(),
  refresh: vi.fn(),
  retryTask: vi.fn(),
  sendMessage: vi.fn(),
  undoStep: vi.fn(),
}));

vi.mock('./tasks-api', () => ({
  acceptTask, answerApproval, answerQuestion, cancelTask, chatOnExternalTask,
  retryTask, sendMessage, undoStep,
}));
vi.mock('./workspace-store', () => ({ refresh }));

beforeEach(() => {
  vi.clearAllMocks();
  refresh.mockResolvedValue(undefined);
});

it('routes external task messages to the linked chat endpoint', async () => {
  chatOnExternalTask.mockResolvedValue({ linkedTaskId: 'internal-1' });
  const { createController } = await import('./task-actions');
  const thread = { refresh: vi.fn() } as never;

  const result = await createController('external-1', thread).follow('Please explain this result.', true);

  expect(result).toEqual({ accepted: true, linkedTaskId: 'internal-1' });
  expect(chatOnExternalTask).toHaveBeenCalledWith('external-1', 'Please explain this result.');
  expect(sendMessage).not.toHaveBeenCalled();
});

it('keeps a failed external send failed with its error and does not fall through to wrong_state', async () => {
  chatOnExternalTask.mockRejectedValue(new ApiError(409, 'no_model'));
  const { createController } = await import('./task-actions');
  const thread = { refresh: vi.fn() } as never;
  const controller = createController('external-1', thread);

  const result = await controller.follow('Draft with attachment: uploads/report.pdf', true);

  expect(result).toEqual({ accepted: false });
  expect(chatOnExternalTask).toHaveBeenCalledWith('external-1', 'Draft with attachment: uploads/report.pdf');
  expect(sendMessage).not.toHaveBeenCalled();
  expect(get(controller.error)).toContain('No AI model is set up yet');
});

it('keeps internal task follow-ups on the existing messages endpoint', async () => {
  sendMessage.mockResolvedValue(undefined);
  const { createController } = await import('./task-actions');
  const controller = createController('internal-1', { refresh: vi.fn() } as never);

  await expect(controller.follow('Please make this change.', false)).resolves.toEqual({ accepted: true });

  expect(sendMessage).toHaveBeenCalledWith('internal-1', 'Please make this change.');
  expect(chatOnExternalTask).not.toHaveBeenCalled();
});
