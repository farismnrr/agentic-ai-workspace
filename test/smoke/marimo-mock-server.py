from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import sys

class Mock(BaseHTTPRequestHandler):
    def log_message(self, *args):
        pass

    def answer(self, data, status=200, kind="application/json"):
        blob = data.encode()
        self.send_response(status)
        self.send_header("Content-Type", kind)
        self.send_header("Content-Length", str(len(blob)))
        self.end_headers()
        self.wfile.write(blob)

    def authorized(self):
        return self.headers.get("Authorization") == "Bearer synthetic-marimo-token"

    def do_GET(self):
        if not self.authorized():
            self.answer('{"error":"unauthorized"}', 401)
        elif self.path == "/api/sessions":
            self.answer(json.dumps({"demo": {"path": "notebook.py"}}))
        else:
            self.answer('{"error":"not found"}', 404)

    def do_POST(self):
        if not self.authorized():
            self.answer('{"error":"unauthorized"}', 401)
        elif self.path != "/api/kernel/execute":
            self.answer('{"error":"not found"}', 404)
        elif self.headers.get("Marimo-Session-Id") != "demo":
            self.answer('{"error":"bad session"}', 400)
        else:
            payload = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0"))))
            if payload.get("code") != "1 + 1":
                self.answer('{"error":"unexpected code"}', 400)
                return
            self.answer('event: stdout\ndata: {"data":"MOCK_"}\n\nevent: done\ndata: {"success":true,"output":{"data":"OK"}}\n\n',
                        kind="text/event-stream")

server = ThreadingHTTPServer(("127.0.0.1", 0), Mock)
with open(sys.argv[1], "w") as f:
    f.write(str(server.server_port))
server.serve_forever()
