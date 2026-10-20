// A hook for the vendor's program (Claude Code PreToolUse): it reads the call from standard input and refuses a shell
// command that starts a server. Only the DevOps teammate starts the app, with start_site; any other teammate's server
// would die with its container and nobody could open it. The server passes the pattern and the words to say in
// GUARD_SERVER_PATTERN and GUARD_SERVER_MESSAGE, so the rule is the same here as in the app's own loop.
import { readFileSync } from 'node:fs';

interface Call {
  tool_input?: { command?: unknown };
}

const pattern = process.env.GUARD_SERVER_PATTERN;
if (!pattern) process.exit(0);

let command = '';
try {
  const call = JSON.parse(readFileSync(0, 'utf8')) as Call;
  command = typeof call.tool_input?.command === 'string' ? call.tool_input.command : '';
} catch {
  process.exit(0);
}

// The same exception as the server's: a command that starts a server and kills it leaves nothing behind, unless it
// detaches the server, which kills nothing.
const starts = new RegExp(pattern, 'i').test(command);
const excused = command.includes('kill') && !/\b(nohup|setsid|disown)\b/i.test(command);
if (starts && !excused) {
  process.stderr.write((process.env.GUARD_SERVER_MESSAGE ?? 'Do not start a server with a shell command.') + '\n');
  process.exit(2);
}
