"""docsite [folder]: turns the Word, PowerPoint, Excel and PDF files of a folder into a small website.

It renders every page to a picture (so it opens in any browser, even inside the sandbox that sites run in),
writes folder/index.html with relative links only, and puts the pictures under folder/preview/. Serve the
folder with: python3 -m http.server PORT --bind 127.0.0.1 --directory folder
Run with docpython; LibreOffice (docconvert) and poppler (pdftoppm) must be installed.
"""
import html
import shutil
import subprocess
import sys
from pathlib import Path

KINDS = {".pdf": "PDF", ".docx": "Word", ".pptx": "PowerPoint", ".xlsx": "Excel"}
DPI = "144"  # sharp when the page is shown about 50rem wide on a large screen

STYLE = """
:root{--bg:#f3f4f6;--card:#fff;--text:#1f2937;--muted:#6b7280;--line:#e5e7eb;--accent:#1f4e79}
*{box-sizing:border-box}
body{margin:0;font:16px/1.5 system-ui,-apple-system,"Segoe UI",sans-serif;background:var(--bg);color:var(--text)}
header{position:sticky;top:0;z-index:1;background:var(--card);border-bottom:1px solid var(--line)}
.bar{max-width:64rem;margin:0 auto;padding:.75rem 1rem;display:flex;flex-wrap:wrap;gap:.5rem 1.5rem;align-items:center}
h1{margin:0;font-size:1.125rem}
nav{display:flex;flex-wrap:wrap;gap:.5rem}
nav a{padding:.25rem .75rem;border:1px solid var(--line);border-radius:999px;color:var(--text);text-decoration:none;font-size:.875rem}
nav a:hover,nav a:focus-visible{border-color:var(--accent);color:var(--accent)}
main{max-width:64rem;margin:0 auto;padding:1rem 1rem 3rem}
section{margin:1.5rem 0 3rem;scroll-margin-top:4rem}
.head{display:flex;flex-wrap:wrap;gap:.5rem 1rem;align-items:center;justify-content:space-between;margin-bottom:1rem}
h2{margin:0;font-size:1.25rem;overflow-wrap:anywhere}
.kind{color:var(--muted);font-size:.875rem}
.btn{padding:.5rem 1rem;border-radius:.5rem;background:var(--accent);color:#fff;text-decoration:none;font-weight:600;font-size:.9375rem}
.btn:hover,.btn:focus-visible{filter:brightness(1.15)}
figure{margin:0 auto 1.5rem;width:100%;max-width:52rem}
figure.wide{max-width:64rem}
img{display:block;width:100%;height:auto;background:var(--card);border:1px solid var(--line);border-radius:.375rem;box-shadow:0 1px 3px rgb(0 0 0/.12)}
figcaption{margin-top:.375rem;text-align:center;color:var(--muted);font-size:.8125rem}
:focus-visible{outline:2px solid var(--accent);outline-offset:2px}
"""


def to_pdf(source: Path, out: Path) -> Path:
    if source.suffix.lower() == ".pdf":
        return source
    out.mkdir(parents=True, exist_ok=True)
    subprocess.run(["docconvert", str(source), "pdf", str(out)], check=True, capture_output=True)
    return out / (source.stem + ".pdf")


def render(pdf: Path, pages: Path) -> list[str]:
    shutil.rmtree(pages, ignore_errors=True)
    pages.mkdir(parents=True)
    subprocess.run(["pdftoppm", "-png", "-r", DPI, str(pdf), str(pages / "page")], check=True)
    return sorted(p.name for p in pages.glob("page-*.png"))


def is_landscape(png: Path) -> bool:
    """A PNG's size sits in its header: width then height, 4 bytes each, after the 8-byte signature and chunk head."""
    head = png.read_bytes()[16:24]
    return int.from_bytes(head[:4], "big") > int.from_bytes(head[4:], "big")


def section(root: Path, source: Path) -> str:
    name = html.escape(source.name)
    stem = source.stem + "-" + source.suffix.lower().lstrip(".")  # report.docx and report.pdf stay apart
    pdf = to_pdf(source, root / "preview" / "pdf" / stem)
    names = render(pdf, root / "preview" / stem)
    wide = " wide" if is_landscape(root / "preview" / stem / names[0]) else ""
    images = "\n".join(
        f'<figure class="{wide.strip()}"><img loading="lazy" src="preview/{html.escape(stem)}/{n}" alt="{name}, page {i}">'
        f"<figcaption>Page {i} of {len(names)}</figcaption></figure>"
        for i, n in enumerate(names, 1)
    )
    return (
        f'<section id="{html.escape(stem)}"><div class="head"><div><h2>{name}</h2>'
        f'<div class="kind">{KINDS[source.suffix.lower()]}, {len(names)} pages</div></div>'
        f'<a class="btn" href="{name}" download>Download</a></div>\n{images}</section>'
    )


def main() -> None:
    root = Path(sys.argv[1] if len(sys.argv) > 1 else "documents")
    files = sorted(p for p in root.iterdir() if p.is_file() and p.suffix.lower() in KINDS)
    if not files:
        sys.exit(f"no PDF, Word, PowerPoint or Excel file in {root}")
    body = "\n".join(section(root, f) for f in files)
    links = "".join(f'<a href="#{html.escape(f.stem + "-" + f.suffix.lower().lstrip("."))}">{html.escape(f.name)}</a>' for f in files)
    page = (
        f'<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">'
        f"<title>Documents</title><style>{STYLE}</style><header><div class=\"bar\"><h1>Documents</h1><nav>{links}</nav></div></header>"
        f"<main>{body}</main></html>"
    )
    (root / "index.html").write_text(page, encoding="utf-8")
    print(f"{len(files)} document(s) ready in {root}/index.html")


main()
