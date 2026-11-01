#!/usr/bin/env python3
"""Disposable Docker app-image black-box: tus ZIP chunks, restart, quota, hash.

Runs on a CI host against the actual nonroot, read-only application container
and its project volume, not an in-process Fiber test server.
"""
import base64
import hashlib
import http.client
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
from urllib.parse import urlsplit
from zipfile import ZIP_STORED, ZipFile

MIB = 1024 * 1024
CHUNK = 8 * MIB
LIMIT = 512 * MIB


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def cookie_from_jar(jar):
    for line in Path(jar).read_text().splitlines():
        if line.startswith("#HttpOnly_"):
            line = line[len("#HttpOnly_"):]
        if line.startswith("#") or not line.strip():
            continue
        fields = line.split("\t")
        if len(fields) >= 7 and fields[5] == "session":
            return "session=" + fields[6]
    raise AssertionError("Disposable test cookie was not found")


def request(base, cookie, method, relative, headers=None, data=None, timeout=180):
    parsed = urlsplit(base)
    require(parsed.scheme == "http" and parsed.hostname in ("127.0.0.1", "localhost"),
            "Docker test must only target the disposable localhost container")
    conn = http.client.HTTPConnection(parsed.hostname, parsed.port, timeout=timeout)
    all_headers = dict(headers or {})
    if cookie:
        all_headers["Cookie"] = cookie
    try:
        conn.request(method, relative, body=data, headers=all_headers)
        response = conn.getresponse()
        result = response.status, dict(response.getheaders()), response.read()
    finally:
        conn.close()
    return result


def header(headers, name):
    return next((value for key, value in headers.items() if key.lower() == name.lower()), None)


def make_zip(destination, data_bytes):
    with ZipFile(destination, mode="w", compression=ZIP_STORED, allowZip64=True) as archive:
        # Writes in bounded chunks, never allocating the entire 512MiB input.
        with archive.open("data/payload.bin", mode="w", force_zip64=True) as member:
            block = b"Z" * MIB
            remaining = data_bytes
            while remaining:
                piece = block[: min(remaining, len(block))]
                member.write(piece)
                remaining -= len(piece)
    size = Path(destination).stat().st_size
    require(size <= LIMIT, "Generated ZIP exceeds server compressed 512MiB cap")
    return size


def restart_app(container, base, cookie):
    subprocess.run(["docker", "restart", container], check=True, timeout=90,
                   stdout=subprocess.DEVNULL)
    for _ in range(90):
        try:
            code, _, body = request(base, cookie, "GET", "/api/health", timeout=5)
            if code == 200 and json.loads(body).get("status") == "ok":
                return
        except (OSError, TimeoutError):
            pass
        time.sleep(1)
    raise AssertionError("Disposable Docker app did not become healthy after restart")


def sha256_from_volume(volume, project, relative):
    require(relative.startswith("uploads/") and
            all(part not in ("", ".", "..") for part in relative.split("/")),
            "Unsafe upload path received from server")
    source = "/data/projects/" + project + "/" + relative
    result = subprocess.run(
        ["docker", "run", "--rm", "--network", "none",
         "--mount", "type=volume,source=" + volume + ",target=/data,readonly",
         "--entrypoint", "sha256sum", "busybox:1.36", source],
        capture_output=True, text=True, check=True, timeout=90,
    )
    return result.stdout.split()[0]


def upload_one(base, cookie, project, container, volume, source, label, restart):
    url = "/api/projects/" + project + "/uploads/resumable"
    size = os.path.getsize(source)
    filename = (label + ".zip").encode()
    metadata = "filename " + base64.b64encode(filename).decode("ascii")
    code, headers, body = request(base, cookie, "POST", url,
        {"Tus-Resumable": "1.0.0", "Upload-Length": str(size),
         "Upload-Metadata": metadata})
    require(code == 201, "POST " + label + ": " + str(code) + " " + body[:250].decode(errors="replace"))
    location = header(headers, "Location")
    require(location and location.startswith(url + "/"), "Server returned invalid tus Location")
    code, head, _ = request(base, cookie, "HEAD", location, {"Tus-Resumable": "1.0.0"})
    require(code == 200 and header(head, "Upload-Offset") == "0", "Initial HEAD offset incorrect")

    # Do not expose private session state to a request without this account.
    code, _, _ = request(base, "", "HEAD", location, {"Tus-Resumable": "1.0.0"})
    require(code == 401, "Private upload session exposed to anonymous caller")

    offset, restarted = 0, False
    total_hash = hashlib.sha256()
    with open(source, "rb") as file:
        while offset < size:
            end = min(offset + CHUNK, size)
            file.seek(offset)
            chunk = file.read(end - offset)
            require(len(chunk) == end - offset, "Short source ZIP read")
            total_hash.update(chunk)
            code, response, body = request(base, cookie, "PATCH", location, {
                "Tus-Resumable": "1.0.0",
                "Content-Type": "application/offset+octet-stream",
                "Upload-Offset": str(offset)}, chunk)
            require(code == 204 and header(response, "Upload-Offset") == str(end),
                    "PATCH " + label + " at " + str(offset) + ": " + str(code) +
                    ", headers=" + repr(response) + ", body=" + repr(body[:250]))
            # A stale/replayed PATCH cannot increment committed offset.
            if offset == 0:
                code, replay_headers, _ = request(base, cookie, "PATCH", location, {
                    "Tus-Resumable": "1.0.0",
                    "Content-Type": "application/offset+octet-stream",
                    "Upload-Offset": "0"}, chunk[:32])
                require(code == 409 and header(replay_headers, "Upload-Offset") == str(end),
                        "Replay PATCH changed committed offset")
            offset = end
            if restart and not restarted and offset >= 2 * CHUNK and offset < size:
                restart_app(container, base, cookie)
                code, after, _ = request(base, cookie, "HEAD", location,
                                          {"Tus-Resumable": "1.0.0"})
                require(code == 200 and header(after, "Upload-Offset") == str(offset),
                        "Restart lost durable upload offset")
                restarted = True
            if offset % (64 * MIB) == 0 or offset == size:
                print("ZIP " + label + ": " + str(offset // MIB) + "/" +
                      str(size // MIB) + " MiB acknowledged", flush=True)

    # Final validation runs asynchronously; HEAD must become ready and reveal a
    # path only after archive safety scanning is complete.
    final_path = None
    for _ in range(180):
        code, head, _ = request(base, cookie, "HEAD", location,
                                    {"Tus-Resumable": "1.0.0"})
        require(code == 200 and header(head, "Upload-Offset") == str(size),
                "Post-upload HEAD offset incorrect")
        if header(head, "Upload-State") == "ready":
            final_path = header(head, "Upload-Path")
            break
        time.sleep(1)
    require(final_path, "ZIP " + label + " never finished validation")
    disk_hash = sha256_from_volume(volume, project, final_path)
    require(disk_hash == total_hash.hexdigest(), "Published Docker volume file checksum mismatch")
    print("PASS Docker ZIP " + label + ": restart=" + str(restarted) +
          ", path=" + final_path + ", SHA256 verified", flush=True)


def main():
    if len(sys.argv) != 6:
        raise SystemExit("Usage: docker_zip_upload.py APP_BASE COOKIE_JAR PROJECT_ID APP_CONTAINER VOLUME")
    base, jar, project, container, volume = sys.argv[1:]
    require(project and all(ch.isalnum() or ch in "-_" for ch in project),
            "Invalid test project ID")
    cookie = cookie_from_jar(jar)
    with tempfile.TemporaryDirectory(prefix="docker-zip-upload-") as directory:
        oversized = "/api/projects/" + project + "/uploads/resumable"
        code, _, _ = request(base, cookie, "POST", oversized, {
            "Tus-Resumable": "1.0.0",
            "Upload-Length": str(LIMIT + 1),
            "Upload-Metadata": "filename " + base64.b64encode(b"too-large.zip").decode()})
        require(code == 413, "512 MiB + 1 compressed-byte limit was not enforced")
        print("PASS oversized ZIP rejected at creation before transfer", flush=True)

        small = os.path.join(directory, "real-50mib.zip")
        make_zip(small, 50 * MIB)
        upload_one(base, cookie, project, container, volume, small, "50mib", True)

        large = os.path.join(directory, "real-near-512mib.zip")
        make_zip(large, LIMIT - 4096)
        upload_one(base, cookie, project, container, volume, large, "near-512mib", False)

    print("PASS Docker app ZIP resumable black-box, checksum and crash recovery", flush=True)


if __name__ == "__main__":
    main()
