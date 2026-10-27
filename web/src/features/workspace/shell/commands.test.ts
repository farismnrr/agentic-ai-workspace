import { describe, expect, it } from 'vitest';
import { allCommands } from './commands';

describe('workspace commands', () => {
  it('includes shortcuts for Archive and Extensions', () => {
    const commands = allCommands([], []);

    expect(commands.find((command) => command.id === 'archive')?.href).toBe('/app/archive');
    expect(commands.find((command) => command.id === 'extensions')?.href).toBe('/app/extensions');
  });
});
