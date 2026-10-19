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
DPI = "80"

STYLE = """
body{margin:0;font:16px/1.5 system-ui,sans-serif;background:#f4f4f1;color:#1d1d1b}
header,main{max-width:60rem;margin:0 auto;padding:1rem}
nav a{margin-right:1rem}
section{margin:2rem 0}
h1{font-size:1.5rem}h2{font-size:1.25rem;margin-bottom:.25rem}
.kind{font-size:.85rem;color:#555}
.btn{display:inline-block;margin:.5rem 0 1rem;padding:.5rem 1rem;border-radius:.5rem;background:#1d1d1b;color:#fff;text-decoration:none}
img{display:block;width:100%;height:auto;margin:0 0 1rem;background:#fff;border:1px solid #ccc;border-radius:.25rem}
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


def section(root: Path, source: Path) -> str:
    name = html.escape(source.name)
    stem = source.stem + "-" + source.suffix.lower().lstrip(".")  # report.docx and report.pdf stay apart
    pdf = to_pdf(source, root / "preview" / "pdf" / stem)
    names = render(pdf, root / "preview" / stem)
    images = "\n".join(
        f'<img loading="lazy" src="preview/{html.escape(stem)}/{n}" alt="{name}, page {i}">' for i, n in enumerate(names, 1)
    )
    return (
        f'<section id="{html.escape(stem)}"><h2>{name}</h2>'
        f'<div class="kind">{KINDS[source.suffix.lower()]}, {len(names)} pages</div>'
        f'<a class="btn" href="{name}" download>Download</a>\n{images}</section>'
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
        f"<title>Documents</title><style>{STYLE}</style><header><h1>Documents</h1><nav>{links}</nav></header>"
        f"<main>{body}</main></html>"
    )
    (root / "index.html").write_text(page, encoding="utf-8")
    print(f"{len(files)} document(s) ready in {root}/index.html")


main()
