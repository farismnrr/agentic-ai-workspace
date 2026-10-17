#!/bin/sh
# Starts the toolbox image the way the server does (no network, read-only root, no
# capabilities) and proves every language runs and can compile something, and that the browser takes a screenshot.
set -eu

image="${1:?usage: smoke-test-sandbox.sh <image>}"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

out="$(docker run --rm --network none --read-only --cap-drop ALL --security-opt no-new-privileges \
  --tmpfs /tmp:rw,exec,nosuid,nodev,size=512m --user "$(id -u):$(id -g)" -v "$work:/workspace" \
  -e HOME=/tmp -e GOCACHE=/tmp/go -e CARGO_HOME=/tmp/cargo --entrypoint "" "$image" sh -ec '
    python3 -V; node -v; go version; rustc -V; java -version 2>&1 | head -n1; mvn -v | head -n1; bash --version | head -n1
    cd /tmp
    printf "package main\nfunc main(){println(\"go-ok\")}\n" > m.go && go run m.go
    echo "fn main(){println!(\"rust-ok\");}" > r.rs && rustc r.rs -o r && ./r
    printf "class H{public static void main(String[] a){System.out.println(\"java-ok\");}}" > H.java && java H.java
    python3 -c "print(\"python-ok\")"
    node -e "console.log(\"node-ok\")"
    mkdir s && echo "<title>t</title><h1>hi</h1>" > s/index.html
    (cd s && python3 -m http.server 8000 --bind 127.0.0.1 > /dev/null 2>&1 &)
    sleep 1
    node /usr/local/share/capture/capture.ts "{\"url\":\"http://127.0.0.1:8000/\",\"width\":800,\"height\":600,\"y\":0}" \
      | grep -q "\"ok\":true" && echo shot-ok
  ' 2>&1)"

echo "$out"

for want in "Python 3.13" "v24." "go1." "rustc 1." "openjdk" "Apache Maven" "go-ok" "rust-ok" "java-ok" "python-ok" "node-ok" "shot-ok"; do
  echo "$out" | grep -q "$want" || { echo "missing: $want" >&2; exit 1; }
done

docker image inspect "$image" --format '{{ index .Config.Labels "io.masihawam.toolchains" }}' | grep -q '"Go"'
echo "sandbox smoke test passed"
