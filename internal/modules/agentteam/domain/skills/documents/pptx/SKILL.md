---
name: pptx
description: Build a PowerPoint deck with python-pptx: slide layouts, text, pictures, tables, charts, notes and a clean look.
---

# PowerPoint (.pptx) recipes

Run scripts with `docpython`. Always import from `pptx`. Slides are 13.333 x 7.5 inch (16:9) in the steps below.

## Start
```python
from pptx import Presentation
from pptx.util import Inches, Pt
from pptx.dml.color import RGBColor
from pptx.enum.text import PP_ALIGN, MSO_ANCHOR

prs = Presentation()
prs.slide_width, prs.slide_height = Inches(13.333), Inches(7.5)
prs.core_properties.title = "Judul"
prs.core_properties.author = "Nama"
```
The default template has layouts: 0 title, 1 title and content, 5 title only, 6 blank. Prefer layouts 0, 1 and 5 so
every slide has a real title (it shows in the outline and for screen readers). Layout placeholders keep their own
positions for the default 4:3 size; after you change the size, move the placeholders yourself or use blank slides with
your own text boxes, and be consistent across the deck.

## Design
- Choose one palette of at most three colours plus a dark text colour and a light background; set them once in
  constants. One font family for the whole deck.
- Title 32-40 pt, body 20-28 pt, notes and sources never below 14 pt. At most 6 lines of 8 words on a slide.
- One message per slide: the title says it as a sentence ("Penjualan naik 20% di Q3"), not a label ("Penjualan").
- Margins of 0.6 inch; align to the same left edge on every slide; leave empty space.
- A title slide, an agenda only when there are 6+ slides, content slides, a closing slide.

## Text
```python
slide = prs.slides.add_slide(prs.slide_layouts[5])
slide.shapes.title.text = "Penjualan naik 20% di Q3"
box = slide.shapes.add_textbox(Inches(0.8), Inches(1.8), Inches(11.7), Inches(4.5))
tf = box.text_frame; tf.word_wrap = True
for i, line in enumerate(["Poin pertama", "Poin kedua"]):
    p = tf.paragraphs[0] if i == 0 else tf.add_paragraph()
    p.text = line; p.font.size = Pt(24); p.space_after = Pt(10)
```
Do not let text grow past its box: python-pptx does not shrink it. Keep it short, or lower the size.

## Pictures, tables, shapes
- Picture, keeping its shape: give only `width` (or only `height`):
  `slide.shapes.add_picture("assets/foto.jpg", Inches(7), Inches(1.8), width=Inches(5.5))`.
  Resize big photos first (see Media's `vips thumbnail`), a 20 MB deck is not a good gift.
- Table: `t = slide.shapes.add_table(rows, cols, left, top, width, height).table; t.cell(0,0).text = "..."`; set a font
  size on every cell's paragraphs (default is 18 pt), and keep to 6 rows.
- Coloured block: `s = slide.shapes.add_shape(1, l, t, w, h); s.fill.solid(); s.fill.fore_color.rgb = RGBColor(0x1F,0x4E,0x79); s.line.fill.background()`.
- Background colour: `slide.background.fill.solid(); slide.background.fill.fore_color.rgb = RGBColor(...)`.

## Charts (real, editable charts)
```python
from pptx.chart.data import CategoryChartData
from pptx.enum.chart import XL_CHART_TYPE, XL_LEGEND_POSITION
data = CategoryChartData(); data.categories = ["Q1", "Q2", "Q3"]
data.add_series("2026", (120, 150, 180))
gf = slide.shapes.add_chart(XL_CHART_TYPE.COLUMN_CLUSTERED, Inches(0.8), Inches(1.8), Inches(8), Inches(5), data)
chart = gf.chart; chart.has_legend = False; chart.plots[0].has_data_labels = True
chart.has_title = False  # the slide title already says it
```
Use bar/column for comparing, line for time, pie only for 2-4 parts of a whole. Label units.

## Speaker notes
`slide.notes_slide.notes_text_frame.text = "Yang dikatakan di slide ini ..."` when the person will present.

## Save and check
`prs.save("documents/deck.pptx")`, then follow steps 5 and 6 of `document-workflow` (`docsite`, look at the pictures, open it as a site).

## Editing an existing deck
Open it with `Presentation("path.pptx")`, change text by walking `shape.text_frame.paragraphs[*].runs[*]` and set
`run.text` so the formatting stays; never replace the whole `text_frame.text` of a formatted shape. Save to a new file.
