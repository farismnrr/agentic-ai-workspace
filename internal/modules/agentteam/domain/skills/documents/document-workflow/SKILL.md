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

## 1b. Reading a document that already exists
The team sends you here when a PDF, Word, Excel or PowerPoint needs to be read or converted. Developer cannot
run shell commands. Use an enabled, owner-configured `run_artifact` extraction/conversion task, or ask for the
extracted text. The commands below are references for named artifact tasks, not Developer shell instructions:
- PDF: `pdftotext -layout file.pdf out.txt` (add `-f 3 -l 9` for pages 3 to 9), then read the text. A scanned PDF has
  no text layer: say so, and render a page with `pdftoppm -png -r 80 -f 1 -l 1 file.pdf page` to see it.
- Word, Excel, PowerPoint: `docpython` with `docx`, `openpyxl` or `pptx`, or `docconvert file.docx pdf .` first.
- Report what you found with the page, sheet or slide it came from, so the next teammate can check it. Do not
  summarise away tables or codes the brief asked for: copy them out exactly (for example into a CSV or a JSON file
  under `data/`) and say where. Never fill a gap from memory.
The sandbox has no network: a document on the web is not here until the person uploads it into the project.

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
Developer has NO `run_command`. Only execute an owner-configured kind=artifact task through `run_artifact`.
If none is configured or enabled, report that exact prerequisite rather than pretending a file was produced.
- `docpython script.py` runs Python with python-pptx, python-docx, reportlab, pypdf, openpyxl and Pillow.
- `docconvert file.docx pdf` converts with LibreOffice (to `pdf`, `docx`, `pptx`); the output lands beside the input.
- `docsite documents` turns the files of a folder into a website with a picture of every page (see step 5).
- `pdftoppm`, `pdftotext`, `pdfinfo` and `qpdf` (poppler) read and render PDFs.
Keep reproducible source under `documents/src/`; a configured artifact task stores the generated files in its
`.artifact-output-*` folder. Report the actual returned path and never claim an artifact you did not generate.
Pictures come from `assets/`; if a needed picture is missing, say so in your report rather than leaving a hole.

## 5. Look at every page
Never report a file you have not seen. A configured artifact preview task can run `docsite` and render page images;
inspect available images with `view_image`. Developer must not invoke `docsite` through a shell.
1. Invoke the configured preview artifact task to run `docsite documents` (the folder of your files). For every Word, PowerPoint, Excel and PDF file in it, it makes
   a PDF with LibreOffice, a picture of each page in `documents/preview/<name>-<ext>/page-N.png`, and
   `documents/index.html`. `pdfinfo` gives a PDF's page count.
2. Open the pictures with `view_image` (PNG only, one by one) and check: text cut off or overflowing, text too small
   to read, overlapping items, empty pages, a table running off the page, a picture stretched, wrong language, a
   placeholder left in, inconsistent fonts or colours.
3. Fix the source and rerun the configured artifact preview task. At most three rounds; then report what is still
   wrong.
LibreOffice may differ a little from Word or PowerPoint, so keep layouts simple and leave room: do not fill a slide to
its edge.

## 6. Open it as a site
The person opens the result from Sites, not from screenshots. After the last `docsite documents`, say in your
report that the DevOps teammate must start the app with the command
`python3 -m http.server 8000 --bind 127.0.0.1 --directory documents` and port 8000. You cannot start it yourself. The page shows
every file with a Download button and all its pages; the links are relative, so it works under the site address.
Run the configured artifact preview task after every source change, and ask DevOps to restart the Site. Do not edit `index.html` or `preview/`
by hand: they are rebuilt each time.

## 7. Report
Give: the file path(s), the number of slides or pages, the outline in one line each, what you checked (pages looked at,
page count), the placeholders and assumptions, and that it is open under Sites. Nothing about scripts or libraries
unless the person asked.

## Quality rules for all formats
- One clear message per slide or section; short sentences; no filler.
- Use real headings, styles and lists, not blank lines and spaces to fake layout.
- Fonts that exist everywhere: Calibri/Arial for Office files (LibreOffice draws them with Carlito or Liberation Sans
  look-alikes); never rely on a font the reader may not have.
- Contrast: dark text on a light background or the reverse; never light grey on white.
- Set the document title and author in its properties.
- Do not put secrets or private data from the project into a file that will be sent out.
