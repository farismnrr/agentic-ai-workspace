import { revealVariable, type EnvVariable } from './env-api';

export function createRevealState(getProjectId: () => string) {
  let revealTarget = $state<EnvVariable>();
  let revealPassword = $state('');
  let revealed = $state<{ name: string; value: string }>();
  let revealMessage = $state('');
  let revealing = $state(false);
  let revealTimer: ReturnType<typeof setTimeout> | undefined;
  let revealRequest = 0;

  function clearSecret(): void {
    if (revealTimer !== undefined) clearTimeout(revealTimer);
    revealTimer = undefined;
    revealed = undefined;
  }

  function closeReveal(): void {
    revealRequest += 1;
    revealing = false;
    revealTarget = undefined;
    revealPassword = '';
    revealMessage = '';
    clearSecret();
  }

  function beginReveal(item: EnvVariable): void {
    closeReveal();
    revealTarget = item;
  }

  async function submitReveal(): Promise<void> {
    if (revealTarget === undefined || revealPassword === '' || revealing) return;
    const requestID = ++revealRequest;
    const requestProjectID = getProjectId();
    const { name: selectedName } = revealTarget;
    const password = revealPassword;
    revealPassword = '';
    revealMessage = '';
    clearSecret();
    revealing = true;
    try {
      const result = await revealVariable(requestProjectID, selectedName, password);
      if (requestID !== revealRequest || getProjectId() !== requestProjectID) return;
      revealed = result;
      revealTarget = undefined;
      revealMessage = 'Secret revealed. It will be hidden automatically in 20 seconds.';
      revealTimer = setTimeout(clearSecret, 20_000);
    } catch {
      if (requestID === revealRequest) revealMessage = 'Could not verify your password or reveal this secret.';
    } finally {
      if (requestID === revealRequest) revealing = false;
    }
  }

  return {
    get target() { return revealTarget; },
    get password() { return revealPassword; },
    set password(v: string) { revealPassword = v; },
    get revealed() { return revealed; },
    get message() { return revealMessage; },
    get revealing() { return revealing; },
    closeReveal,
    beginReveal,
    submitReveal,
  };
}
