import { request } from '../../../shared/api/http';

export interface ChatMessage {
  id: string;
  chatId: string;
  role: 'user' | 'assistant';
  content: string;
  createdAt: string;
}

export interface Chat {
  id: string;
  projectId: string;
  userId: string;
  title: string;
  createdAt: string;
  updatedAt: string;
}

export interface ChatConversation {
  chat: Chat;
  messages: ChatMessage[];
}

export interface SendResult {
  userMessage: ChatMessage;
  assistantMessage: ChatMessage;
}

export const loadChat = (projectId: string): Promise<ChatConversation> =>
  request<ChatConversation>('GET', `/api/projects/${encodeURIComponent(projectId)}/chat`);

export const sendChatMessage = (projectId: string, prompt: string): Promise<SendResult> =>
  request<SendResult>('POST', `/api/projects/${encodeURIComponent(projectId)}/chat`, { prompt });

export const clearChat = (projectId: string): Promise<unknown> =>
  request('DELETE', `/api/projects/${encodeURIComponent(projectId)}/chat`);

export const escalateChatToTask = (projectId: string, prompt?: string, mode?: string): Promise<{ taskId: string }> =>
  request<{ taskId: string }>('POST', `/api/projects/${encodeURIComponent(projectId)}/chat/escalate`, { prompt, mode });
