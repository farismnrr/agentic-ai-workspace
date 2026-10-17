// A minimal Chrome DevTools Protocol client over the WebSocket Node ships with, so the capture needs no npm package.
type Params = Record<string, unknown>;
type Listener = (params: Params) => void;

export interface Cdp {
  send: (method: string, params?: Params) => Promise<Params>;
  on: (method: string, listener: Listener) => void;
  close: () => void;
}

interface Reply {
  id?: number;
  method?: string;
  params?: Params;
  result?: Params;
  error?: { message: string };
}

export async function connect(url: string): Promise<Cdp> {
  const socket = new WebSocket(url);
  await new Promise<void>((resolve, reject) => {
    socket.addEventListener('open', () => { resolve(); });
    socket.addEventListener('error', () => { reject(new Error('cannot reach the browser')); });
  });

  let next = 0;
  const pending = new Map<number, { resolve: (value: Params) => void; reject: (error: Error) => void }>();
  const listeners = new Map<string, Listener[]>();

  socket.addEventListener('message', (event) => {
    const reply = JSON.parse(String(event.data)) as Reply;
    if (reply.id !== undefined) {
      const call = pending.get(reply.id);
      pending.delete(reply.id);
      if (reply.error) call?.reject(new Error(reply.error.message));
      else call?.resolve(reply.result ?? {});
    } else if (reply.method) {
      for (const listener of listeners.get(reply.method) ?? []) listener(reply.params ?? {});
    }
  });

  return {
    send: (method, params = {}) =>
      new Promise((resolve, reject) => {
        next += 1;
        pending.set(next, { resolve, reject });
        socket.send(JSON.stringify({ id: next, method, params }));
      }),
    on: (method, listener) => { listeners.set(method, [...(listeners.get(method) ?? []), listener]); },
    close: () => { socket.close(); },
  };
}
