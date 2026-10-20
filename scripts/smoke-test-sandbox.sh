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
    convert -size 120x80 gradient:white-blue -fill red -draw "circle 60,40 60,15" p.jpg
    vips thumbnail p.jpg "q.jpg[Q=80,strip]" 60 && cwebp -q 80 q.jpg -o q.webp > /dev/null 2>&1 && echo pictures-ok
    removebg p.jpg cut.png fast && removebg p.jpg cut2.png && echo removebg-ok
    cp /opt/vector/node_modules/lucide-static/icons/coffee.svg i.svg && svgo --multipass i.svg -o i.min.svg > /dev/null && echo icons-ok
    convert p.jpg -threshold 50% m.pbm && potrace -s m.pbm -o m.svg && grep -q "<svg" m.svg && echo vector-ok
    command -v new-nuxt-app > /dev/null && test -f /usr/local/share/starter/nuxt/nuxt.config.ts && echo starter-ok
    test -f /usr/local/share/templates/saas/app/pages/index.vue && test -f /usr/local/share/templates/dashboard/LICENSE && echo templates-ok
    bun --version | grep -q "^1\." && bun -e "import {Database} from \"bun:sqlite\"; const d=new Database(\":memory:\"); d.exec(\"create table t(a)\"); console.log(\"bun-ok\")"
    printf "require(\"http\").createServer((q,s)=>{s.setHeader(\"content-type\",\"text/html\");s.end(q.url)}).listen(process.env.PORT,\"127.0.0.1\")" > echo.cjs
    PORT=8100 BASE_PATH=/run/a/b node /usr/local/share/serve/site-serve.ts node echo.cjs > /dev/null 2>&1 &
    sleep 2
    curl -s http://127.0.0.1:8100/x | grep -q "^/run/a/b/x$" && echo prefix-ok
    curl -s -H "X-Forwarded-Prefix: /run/a/b/~1.ab" http://127.0.0.1:8100/x | grep -q "^/run/a/b/~1.ab/x$" && echo ticket-prefix-ok
    convert "msl:p.jpg" m.png 2>&1 | grep -q "security policy" && echo policy-ok
    docpython -c "from pptx import Presentation; from docx import Document; from reportlab.pdfgen import canvas; import pypdf, openpyxl; p=Presentation(); p.slides.add_slide(p.slide_layouts[0]); p.save(\"d.pptx\"); d=Document(); d.add_heading(\"Hi\"); d.save(\"d.docx\"); canvas.Canvas(\"d.pdf\").save()" \
      && docconvert d.pptx pdf . > /dev/null 2>&1 && docconvert d.docx pdf . > /dev/null 2>&1 \
      && pdftoppm -png -r 40 -f 1 -l 1 d.pdf page && ls page*.png > /dev/null && mkdir documents && cp d.pptx d.docx d.pdf documents/ && docsite documents > /dev/null && test -f documents/preview/d-pptx/page-1.png && grep -q "preview/d-pptx/page-1.png" documents/index.html && echo docs-ok
    node /usr/local/share/capture/capture.ts "{\"url\":\"http://127.0.0.1:8000/\",\"width\":800,\"height\":600,\"y\":0}" \
      | grep -q "\"ok\":true" && echo shot-ok
  ' 2>&1)"

echo "$out"

for want in "Python 3.13" "v24." "go1." "rustc 1." "openjdk" "Apache Maven" "go-ok" "rust-ok" "java-ok" "python-ok" "node-ok" "shot-ok" "pictures-ok" "icons-ok" "vector-ok" "prefix-ok" "ticket-prefix-ok" "bun-ok" "starter-ok" "templates-ok" "removebg-ok" "policy-ok" "docs-ok"; do
  echo "$out" | grep -q "$want" || { echo "missing: $want" >&2; exit 1; }
done

docker image inspect "$image" --format '{{ index .Config.Labels "io.masihawam.toolchains" }}' | grep -q '"Go"'
echo "sandbox smoke test passed"
