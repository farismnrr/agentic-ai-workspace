from http.server import BaseHTTPRequestHandler, HTTPServer
from http.cookies import SimpleCookie
from urllib.parse import parse_qs
import json

EMAIL = "agent@example.test"
PASSWORD = "fixture-password"
CSRF = "csrf-fixture-value"


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        path = self.path.split("?", 1)[0]
        if path == "/login-form":
            self.page("""<form method="post" action="/api/form-login">
              <input type="hidden" name="csrf" value="csrf-fixture-value">
              <input id="email" type="email" name="email">
              <input id="password" type="password" name="password">
              <button type="submit">Sign in</button></form><p id="login-error"></p>""")
        elif path == "/login-form-offscreen":
            self.page("""<form method="post" action="/api/form-login">
              <input type="hidden" name="csrf" value="csrf-fixture-value">
              <input id="email" type="email" name="email">
              <input id="password" type="password" name="password">
              <div style="height:1800px"></div><button type="submit">Sign in</button></form>""")
        elif path == "/login-form-hidden":
            self.page("""<form method="post" action="/api/form-login">
              <input type="hidden" name="csrf" value="csrf-fixture-value">
              <input id="email" type="email" name="email">
              <input id="password" type="password" name="password">
              <button type="submit" style="display:none">Sign in</button></form>""")
        elif path == "/login-json":
            self.page("""<form id="login" onsubmit="submitLogin(event)">
              <input type="hidden" id="csrf" value="csrf-fixture-value">
              <input id="email" type="email"><input id="password" type="password">
              <button type="submit">Sign in</button></form><p id="login-error"></p>
              <script>async function submitLogin(event) {
                event.preventDefault();
                const response = await fetch('/api/json-login', {method:'POST',
                  headers:{'Content-Type':'application/json'},
                  body:JSON.stringify({email:email.value,password:password.value,csrf:csrf.value})});
                if (response.ok) location.assign('/private-one');
                else document.querySelector('#login-error').textContent = 'Email or password is incorrect';
              }</script>""")
        elif path in ("/private-one", "/private-two"):
            if self.session():
                name = path.rsplit("-", 1)[1]
                self.page(f'<h1 data-ready="{name}">Protected {name}</h1>')
            else:
                self.page('<h1 data-ready="anonymous">Sign in required</h1>', 401)
        elif path == "/logout":
            self.page("""<form method="post" action="/api/logout">
              <input type="hidden" name="csrf" value="csrf-fixture-value">
              <button type="submit">Sign out</button></form>""")
        elif path == "/logged-out":
            self.page('<h1 data-ready="logged-out">Signed out</h1>')
        elif path == "/expire":
            self.send_response(200)
            self.send_header("Set-Cookie", "agent_session=; Path=/; Max-Age=0; HttpOnly")
            self.end_headers()
            self.wfile.write(b"expired")
        elif path == "/external":
            self.send_response(302)
            self.send_header("Location", "https://example.com/")
            self.end_headers()
        else:
            self.page("<h1>Not found</h1>", 404)

    def do_POST(self):
        body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        path = self.path.split("?", 1)[0]
        if path == "/api/json-login":
            values = json.loads(body or b"{}")
        else:
            values = {key: items[0] for key, items in parse_qs(body.decode()).items()}
        valid = values.get("csrf") == CSRF and values.get("email") == EMAIL and values.get("password") == PASSWORD
        if path in ("/api/form-login", "/api/json-login"):
            if not valid:
                if path == "/api/json-login":
                    self.send_json(401, {"error": "invalid"})
                else:
                    self.page('<p id="login-error">Email or password is incorrect</p>', 401)
                return
            if path == "/api/json-login":
                self.send_response(200)
                self.send_header("Set-Cookie", "agent_session=fixture-ok; Path=/; HttpOnly; SameSite=Lax")
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(b'{"ok":true}')
            else:
                self.send_response(303)
                self.send_header("Set-Cookie", "agent_session=fixture-ok; Path=/; HttpOnly; SameSite=Lax")
                self.send_header("Location", "/private-one")
                self.end_headers()
        elif path == "/api/logout" and self.session() and values.get("csrf") == CSRF:
            self.send_response(303)
            self.send_header("Set-Cookie", "agent_session=; Path=/; Max-Age=0; HttpOnly")
            self.send_header("Location", "/logged-out")
            self.end_headers()
        else:
            self.page("<h1>Not found</h1>", 404)

    def session(self):
        cookie = SimpleCookie(self.headers.get("Cookie", ""))
        return cookie.get("agent_session") is not None and cookie["agent_session"].value == "fixture-ok"

    def page(self, body, status=200):
        self.send_response(status)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.end_headers()
        self.wfile.write(f"<!doctype html><html><body>{body}</body></html>".encode())

    def send_json(self, status, value):
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(value).encode())

    def log_message(self, *_):
        return


HTTPServer(("127.0.0.1", 8000), Handler).serve_forever()
