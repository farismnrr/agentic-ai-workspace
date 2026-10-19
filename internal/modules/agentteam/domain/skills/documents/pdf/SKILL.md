---
name: pdf
description: Make, convert, merge, split and read PDF files: reportlab for new documents, LibreOffice for Word and PowerPoint, pypdf and poppler for the rest.
---

# PDF recipes

Run Python with `docpython`. Always check the result with `pdfinfo` and by rendering pages (see `document-workflow`).

## Convert Word, PowerPoint or Excel to PDF
`docconvert documents/laporan.docx pdf` (the PDF is written beside the input; add a folder as the third argument to put
it elsewhere). This is the right way when the person also wants the editable file.

## A new PDF from scratch (reportlab, Platypus)
```python
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.units import cm
from reportlab.lib import colors
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, Table, TableStyle, Image, PageBreak

styles = getSampleStyleSheet()
body = ParagraphStyle("body", parent=styles["BodyText"], fontName="Helvetica", fontSize=11, leading=15)
doc = SimpleDocTemplate("documents/surat.pdf", pagesize=A4, leftMargin=2.5*cm, rightMargin=2.5*cm,
                        topMargin=2.5*cm, bottomMargin=2.5*cm, title="Judul", author="Nama")
story = [Paragraph("Judul", styles["Title"]), Spacer(1, 12), Paragraph("Isi paragraf ...", body)]
tbl = Table([["Item", "Qty"], ["Kopi", "2"]], colWidths=[10*cm, 3*cm], repeatRows=1)
tbl.setStyle(TableStyle([("BACKGROUND", (0,0), (-1,0), colors.HexColor("#1F4E79")),
                         ("TEXTCOLOR", (0,0), (-1,0), colors.white), ("GRID", (0,0), (-1,-1), 0.5, colors.grey),
                         ("ALIGN", (1,1), (-1,-1), "RIGHT")]))
story += [Spacer(1, 12), tbl]
doc.build(story)
```
- Page numbers: pass `onFirstPage=num, onLaterPages=num` to `doc.build`, where
  `def num(c, d): c.setFont("Helvetica", 9); c.drawCentredString(A4[0]/2, 1.2*cm, f"Halaman {d.page}")`.
- Pictures: `Image("assets/logo.png", width=5*cm, height=5*cm*h/w)` to keep the shape (read the size with Pillow).
- Text in a Paragraph is XML-like: write `&amp;` for `&` and `&lt;` for `<`, use `<b>`, `<i>`, `<br/>`.
- The built-in fonts (Helvetica, Times, Courier) lack many symbols. For accents beyond Latin-1 or symbols register a TTF:
  `pdfmetrics.registerFont(TTFont("DejaVu", "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"))` and use `fontName="DejaVu"`.
- Long tables break across pages with `repeatRows=1`; wrap long cell text in `Paragraph` objects so it wraps.

## Merge, split, rotate, extract (pypdf)
```python
from pypdf import PdfReader, PdfWriter
w = PdfWriter()
for path in ["a.pdf", "b.pdf"]:
    for page in PdfReader(path).pages: w.add_page(page)
w.add_metadata({"/Title": "Gabungan"}); w.write("documents/gabungan.pdf")
```
Split: add chosen `reader.pages[i]` to a new writer. Rotate: `page.rotate(90)`. Fill a form: `writer.update_page_form_field_values`.
Password: `reader.decrypt("pw")`; do not remove a password the person did not give you.

## Read a PDF
- Text: `pdftotext -layout file.pdf -` (keeps columns); scanned PDFs have no text and there is no OCR offline: say so.
- Facts: `pdfinfo file.pdf`. Pictures out: `pdfimages -png file.pdf out/img`.
- Page to picture: `pdftoppm -png -r 80 -f 1 -l 3 file.pdf out/page`.
- Shrink: `qpdf --linearize --object-streams=generate in.pdf out.pdf`.
