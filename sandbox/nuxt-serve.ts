// Runs a Nuxt (or any Nitro) server behind the sites proxy, which removes the address prefix from every request
// while the server wants to live under it. The real server gets the prefix as its base URL and listens on the next
// port; this process listens on the site's port, puts the prefix back on each request and takes it off the
// redirects that come back. A private site is opened under /run/<name>/~<ticket>/, and the sandboxed page loads
// its scripts and styles with no login, so every link in a text answer must carry that ticket: the proxy sends the
// address it was opened at in X-Forwarded-Prefix and the base in the text is replaced with it. Usage: PORT=3000 BASE_PATH=/run/shop/<uuid> nuxt-serve node .output/server/index.mjs
import { spawn } from 'node:child_process';
import { request, createServer } from 'node:http';
import { connect } from 'node:net';

const [program, ...args] = process.argv.slice(2);
const port = Number(process.env.PORT ?? 3000);
const innerPort = port + 1;
const base = (process.env.BASE_PATH ?? '').replace(/\/$/, '');

if (program === undefined) {
  console.error('usage: nuxt-serve <command that starts the server>');
  process.exit(2);
}

const child = spawn(program, args, {
  stdio: 'inherit',
  env: { ...process.env, PORT: String(innerPort), HOST: '127.0.0.1', NUXT_APP_BASE_URL: `${base}/` },
});
child.on('exit', (code) => process.exit(code ?? 1));
for (const signal of ['SIGTERM', 'SIGINT'] as const) process.on(signal, () => child.kill(signal));

// Text a browser reads and may follow links in; anything else (images, fonts) has no prefix inside.
const isText = (type: unknown): boolean => typeof type === 'string' && /^(text\/|application\/(javascript|json))/.test(type);

const openedAt = (header: string | string[] | undefined): string => {
  const prefix = (Array.isArray(header) ? header[0] : header) ?? base;
  return prefix.startsWith(`${base}/~`) || prefix === base ? prefix.replace(/\/$/, '') : base;
};

const server = createServer((req, res) => {
  const prefix = openedAt(req.headers['x-forwarded-prefix']);
  const upstream = request(
    { host: '127.0.0.1', port: innerPort, method: req.method, path: base + (req.url ?? '/'), headers: req.headers },
    (answer) => {
      const headers = { ...answer.headers };
      const location = headers.location;
      if (typeof location === 'string' && location.startsWith(`${base}/`)) headers.location = location.slice(base.length);
      if (prefix === base || !isText(headers['content-type'])) {
        res.writeHead(answer.statusCode ?? 502, headers);
        answer.pipe(res);
        return;
      }
      const chunks: Buffer[] = [];
      answer.on('data', (chunk: Buffer) => chunks.push(chunk));
      answer.on('end', () => {
        const text = Buffer.concat(chunks).toString('utf8').replaceAll(`${base}/`, `${prefix}/`);
        delete headers['content-length'];
        res.writeHead(answer.statusCode ?? 502, headers).end(text);
      });
    },
  );
  upstream.on('error', () => {
    res.writeHead(502).end('The app is still starting.');
  });
  req.pipe(upstream);
});

// Listen only once the real server answers, so the site counts as started when it really is.
const waitThenListen = (): void => {
  const probe = connect(innerPort, '127.0.0.1');
  probe.on('connect', () => {
    probe.destroy();
    server.listen(port, '127.0.0.1');
  });
  probe.on('error', () => setTimeout(waitThenListen, 200));
};
waitThenListen();
