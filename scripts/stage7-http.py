#!/usr/bin/env python3
"""Black-box HTTP approval lifecycle on a throwaway container."""
import http.cookiejar
import json
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path

base, volume, phase, state_file = sys.argv[1:5]
jar = http.cookiejar.CookieJar()
client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
password = "stage7-test-only-dummy-not-production"

def call(method, path, payload=None, opener=client, csrf=""):
    data = None if payload is None else json.dumps(payload).encode()
    headers = {"Content-Type": "application/json"}
    if csrf:
        headers["X-Egress-CSRF"] = csrf
    req = urllib.request.Request(base + path, data=data, headers=headers, method=method)
    try:
        with opener.open(req, timeout=20) as response:
            code, body = response.status, response.read().decode()
    except urllib.error.HTTPError as exc:
        code, body = exc.code, exc.read().decode()
    try:
        result = json.loads(body)
    except ValueError:
        result = body
    return code, result

def checked(method, path, wanted, payload=None, csrf=""):
    status, body = call(method, path, payload, csrf=csrf)
    if status != wanted:
        raise AssertionError(f"{method} {path}: expected {wanted}, got {status}: {body}")
    print(f"PASS {method} {path}: HTTP {wanted}", flush=True)
    return body

def fixture(project, user):
    sql = """
import os, sqlite3, time
db = sqlite3.connect('/data/workspace.db', timeout=20)
t = int(time.time())
for name in ('stage7-task-a', 'stage7-task-b'):
    db.execute('INSERT INTO tasks (id, project_id, user_id, title, status, mode, provider_id, model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)',
        (name, os.environ['STAGE7_PROJECT'], os.environ['STAGE7_USER'], 'Dummy task', 'review', 'risky', 'dummy', 'dummy', t, t))
db.commit()
rows = {r[0] for r in db.execute("SELECT name FROM schema_migrations WHERE name LIKE '05%'")}
assert {'052_egress_grants.sql','053_egress_audit.sql','054_egress_actor_attribution.sql'} <= rows, rows
print('PASS migration journal 052-054 in disposable database', flush=True)
"""
    subprocess.run(["docker", "run", "--rm", "--network", "none",
        "--user", "65532:65532",
        "-e", "STAGE7_PROJECT=" + project, "-e", "STAGE7_USER=" + user,
        "--mount", "type=volume,source=" + volume + ",target=/data",
        "--entrypoint", "python3", "python:3.13-alpine", "-c", sql], check=True)

def csrf_token(path):
    return checked("GET", path + "/csrf", 200)["token"]

if phase == "setup":
    checked("POST", "/api/auth/register", 201, {
        "name": "Stage 7 Dummy", "email": "stage7@example.invalid", "password": password})
    user = checked("GET", "/api/auth/me", 200)["id"]
    project = checked("POST", "/api/projects", 201, {"name": "Stage 7 isolated"})["id"]
    path = "/api/projects/" + project + "/egress-grants"
    fixture(project, user)
    csrf = csrf_token(path)
    status, _ = call("GET", path + "/", opener=urllib.request.build_opener())
    if status != 401:
        raise AssertionError(f"anonymous approval access returned {status}, expected 401")
    print("PASS anonymous approval access denied", flush=True)
    for name in ("a", "b"):
        grant = checked("POST", path + "/", 201, {
            "taskId": "stage7-task-" + name, "host": "example.com",
            "reason": "test only, no production access"}, csrf=csrf)
        if grant["status"] != "pending":
            raise AssertionError("new request was not pending")
        approved = checked("POST", path + "/" + grant["id"] + "/approve",
            200, {"ttlHours": 1}, csrf=csrf)
        if approved["status"] != "approved":
            raise AssertionError("approval not persisted")
        if name == "a":
            revoked = checked("POST", path + "/" + grant["id"] + "/revoke",
                200, {}, csrf=csrf)
            if revoked["status"] != "revoked":
                raise AssertionError("revocation not persisted")
            checked("POST", path + "/" + grant["id"] + "/approve", 409, {}, csrf=csrf)
        else:
            Path(state_file).write_text(json.dumps({
                "project": project, "grant": grant["id"]}), encoding="utf-8")
    print("PASS HTTP request -> approval -> revoke -> replay", flush=True)
elif phase == "restart":
    state = json.loads(Path(state_file).read_text(encoding="utf-8"))
    checked("POST", "/api/auth/login", 200, {
        "email": "stage7@example.invalid", "password": password})
    path = "/api/projects/" + state["project"] + "/egress-grants"
    rows = checked("GET", path + "/", 200)
    restored = [g for g in rows if g["id"] == state["grant"]]
    if len(restored) != 1 or restored[0]["status"] != "approved":
        raise AssertionError("approved grant not restored after app restart")
    print("PASS approved grant restored after app restart", flush=True)
    csrf = csrf_token(path)
    revoked = checked("POST", path + "/" + state["grant"] + "/revoke",
        200, {}, csrf=csrf)
    if revoked["status"] != "revoked":
        raise AssertionError("revoke after restart failed")
    checked("POST", path + "/" + state["grant"] + "/approve", 409, {}, csrf=csrf)
    print("PASS restart -> revoke -> deny replay", flush=True)
else:
    raise SystemExit("unknown phase")
