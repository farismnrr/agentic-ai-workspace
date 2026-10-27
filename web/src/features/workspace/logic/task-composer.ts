import { withAttachments } from './attachments';

export interface FollowResult {
  accepted: boolean;
  linkedTaskId?: string;
}

export async function submitTaskFollowUp(
  text: string,
  paths: string[],
  external: boolean,
  send: (message: string, external: boolean) => Promise<FollowResult>,
  clear: () => void,
): Promise<FollowResult> {
  const result = await send(withAttachments(text, paths), external);
  if (result.accepted) clear();
  return result;
}
