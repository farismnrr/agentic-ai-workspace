---
name: docx
description: Build a Word document with python-docx: headings, styles, lists, tables, pictures, header, footer, page numbers, table of contents.
---

# Word (.docx) recipes

Run scripts with `docpython`. Import from `docx`.

## Start
```python
from docx import Document
from docx.shared import Pt, Cm, RGBColor
from docx.enum.text import WD_ALIGN_PARAGRAPH
from docx.enum.section import WD_ORIENT

doc = Document()
doc.core_properties.title = "Judul"; doc.core_properties.author = "Nama"
sec = doc.sections[0]
sec.page_width, sec.page_height = Cm(21), Cm(29.7)     # A4; Letter is Cm(21.59), Cm(27.94)
sec.left_margin = sec.right_margin = Cm(2.5); sec.top_margin = sec.bottom_margin = Cm(2.5)

normal = doc.styles["Normal"]; normal.font.name = "Calibri"; normal.font.size = Pt(11)
normal.paragraph_format.space_after = Pt(6); normal.paragraph_format.line_spacing = 1.15
```
Indonesian documents use A4. Set the style once; do not format every paragraph by hand.

## Structure with real styles
- Title: `doc.add_heading("Judul", 0)`. Sections: `doc.add_heading("1. Latar belakang", 1)`, sub-sections level 2.
  Headings give the navigation pane and the table of contents; never fake them with bold text.
- Paragraph: `doc.add_paragraph("Teks")`. Bold/italic parts: `p = doc.add_paragraph(); r = p.add_run("tebal"); r.bold = True`.
- Lists: `doc.add_paragraph("Butir", style="List Bullet")` and `style="List Number"`.
- Page break: `doc.add_page_break()`. Alignment: `p.alignment = WD_ALIGN_PARAGRAPH.CENTER`.
- Quote or note: style `"Intense Quote"` or `"Quote"`.

## Tables
```python
t = doc.add_table(rows=1, cols=3); t.style = "Light Grid Accent 1"
for cell, text in zip(t.rows[0].cells, ["Item", "Qty", "Harga"]):
    cell.text = text
    cell.paragraphs[0].runs[0].bold = True
for row in data:
    cells = t.add_row().cells
    for cell, value in zip(cells, row): cell.text = str(value)
```
Right-align numbers (`cell.paragraphs[0].alignment = WD_ALIGN_PARAGRAPH.RIGHT`). Set widths with `cell.width = Cm(4)` on every
cell of a column. Repeat the header row on each page:
```python
from docx.oxml import OxmlElement; from docx.oxml.ns import qn
trPr = t.rows[0]._tr.get_or_add_trPr(); h = OxmlElement("w:tblHeader"); h.set(qn("w:val"), "true"); trPr.append(h)
```

## Pictures
`doc.add_picture("assets/logo.png", width=Cm(8))` (width only, so the shape stays). Add a caption paragraph under it.

## Header, footer, page numbers
```python
sec.header.paragraphs[0].text = "Nama Perusahaan - Rahasia"
p = sec.footer.paragraphs[0]; p.alignment = WD_ALIGN_PARAGRAPH.CENTER
def field(par, code):
    for kind, text in (("begin", None), (None, code), ("end", None)):
        run = par.add_run()
        if kind:
            el = OxmlElement("w:fldChar"); el.set(qn("w:fldCharType"), kind)
        else:
            el = OxmlElement("w:instrText"); el.set(qn("xml:space"), "preserve"); el.text = text
        run._r.append(el)
p.add_run("Halaman "); field(p, "PAGE"); p.add_run(" dari "); field(p, "NUMPAGES")
```
A different first page: `sec.different_first_page_header_footer = True`.

## Table of contents
Insert a TOC field where it belongs (long documents only): a paragraph with `field(par, 'TOC \\o "1-3" \\h \\z \\u')`.
Word fills it when the person presses F9 or opens it; LibreOffice does not fill it during conversion, so the
rendered PDF shows it empty: say so in the report, or write a short manual contents list instead.

## Landscape section
`s = doc.add_section(); s.orientation = WD_ORIENT.LANDSCAPE; s.page_width, s.page_height = s.page_height, s.page_width`.

## Save and check
`doc.save("documents/laporan.docx")`, then follow steps 5 and 6 of `document-workflow` (look at every page, open it as a site).

## Editing an existing document
`Document("path.docx")`; change `run.text` inside the paragraphs you need so the formatting stays; save to a new
file. Do not touch tracked changes or comments unless asked: python-docx does not understand them.
