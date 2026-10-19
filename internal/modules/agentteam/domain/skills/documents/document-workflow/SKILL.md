---
name: document-workflow
description: How to make a PowerPoint, Word or PDF file from request to proof: outline, build, look at every page, fix, save in documents/.
always: true
---

# Documents playbook

You make files people open in PowerPoint, Word or a PDF reader. The person is not a developer: write for them in plain
words and give them a file that is ready to use, not a script. A document is finished when you have looked at every
page and it looks right.

## 1. Understand the job
Read the brief and the project (`assets/`, any notes or data files it names). You need the topic, the audience, the
language (write in the language of the request; Indonesian stays Indonesian), the length and the tone. If the brief
leaves out something that changes the result, such as the number of slides or who reads it, choose a sensible default
and say so in your report instead of stopping. Use only facts from the brief and the project. Never invent numbers,
quotes, names or sources: when a fact is missing, write a clear placeholder such as `[isi angka penjualan]` and list
it in your report.

## 2. Pick the format
- Slides to present: PowerPoint. Load the `pptx` skill.
- A letter, report, proposal, contract or anything the person will keep editing: Word. Load the `docx` skill.
- A final copy to send, print or fill in: PDF. Load the `pdf` skill. For a Word or PowerPoint file that must also be
  sent as PDF, build the Office file and convert it; do not build the same content twice.
Load a skill with `use_skill`.

## 3. Outline first
Write the outline in your own words before you build: the title, each section or slide with its one message, and what
goes on it (text, table, picture, chart). Keep it short. Build from that outline so the structure is clear.

## 4. Build with the sandbox tools
Everything runs with `run_command`, offline.
- `docpython script.py` runs Python with python-pptx, python-docx, reportlab, pypdf, openpyxl and Pillow.
- `docconvert file.docx pdf` converts with LibreOffice (to `pdf`, `docx`, `pptx`); the output lands beside the input.
- `pdftoppm`, `pdftotext`, `pdfinfo` and `qpdf` (poppler) read and render PDFs.
Keep the build script in `documents/src/` (for example `documents/src/deck.py`) so the file can be rebuilt after a
fix. Write the result to `documents/`, with a file name that says what it is (`proposal-kopi-nusantara.docx`).
Pictures come from `assets/`; if a needed picture is missing, say so in your report rather than leaving a hole.

## 5. Look at every page
Never report a file you have not seen.
1. Make a PDF: `docconvert documents/deck.pptx pdf` (a PDF you built is already one).
2. Render pages: `pdftoppm -png -r 60 documents/deck.pdf documents/preview/page`
3. Open the pictures with `view_image` (PNG only, one by one) and check: text cut off or overflowing, text
   too small to read, overlapping items, empty pages, a table running off the page, a picture stretched, wrong
   language, a placeholder left in, inconsistent fonts or colours.
4. Fix the script, rebuild, render again. At most three rounds; then report what is still wrong.
5. Delete `documents/preview/` when done, and keep the PDF only if the person asked for one or the job is a PDF.
`pdfinfo documents/deck.pdf` gives the page count. LibreOffice may differ a little from Word or PowerPoint, so keep
layouts simple and leave room: do not fill a slide to its edge.

## 6. Report
Give: the file path(s), the number of slides or pages, the outline in one line each, what you checked (pages looked at,
page count), the placeholders and assumptions. Nothing about scripts or libraries unless the person asked.

## Quality rules for all formats
- One clear message per slide or section; short sentences; no filler.
- Use real headings, styles and lists, not blank lines and spaces to fake layout.
- Fonts that exist everywhere: Calibri/Arial for Office files (LibreOffice draws them with Carlito or Liberation Sans
  look-alikes); never rely on a font the reader may not have.
- Contrast: dark text on a light background or the reverse; never light grey on white.
- Set the document title and author in its properties.
- Do not put secrets or private data from the project into a file that will be sent out.
