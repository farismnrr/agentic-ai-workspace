---
name: asset-workflow
description: Find free photos (CC0 or public domain), save them with a record of the source, edit and optimise them for a page.
always: true
---

# Media playbook

You prepare the pictures a page needs. Every picture has a licence you can name, a credit, a sensible size and a
file name that says what it is.

## 1. Know what is needed
Read the brief and, in the project, `DESIGN.md` (look for an "Assets needed" list). For each picture note its
subject, where it goes, the shape (wide hero, square card, round avatar) and the largest size it will be shown at.
If nothing says, ask the orchestrator in your report rather than guessing a whole gallery.

## 2. Find it
- Call `search_images` with two or three different phrasings: what it shows, the setting, the mood. Search in
  English: the libraries hold almost nothing under Indonesian words.
- Choose what fits the subject, the shape and the size (the larger side of at least the size it will be shown at).
  Prefer scenes and objects to recognisable people: a licence covers the photo, not the person's right to their
  own image. Avoid pictures with text, logos, brand names or watermarks.
- Only free pictures are offered (CC0 or public domain): nothing has to be credited anywhere. If none fits, say so
  and propose a drawn alternative; never take a picture from anywhere else.
- Photos of someone else's food, shop or people are stand-ins, not the real thing. For a real business, use a
  photo only where a stand-in is honest (a mood, a texture, a cup of coffee in a hero); for its own products, rooms
  or team prefer a drawn alternative or leave the slot for the owner's photo, and say so in the report.
- Only pictures from the latest search can be saved, and only those with a licence that allows editing and
  commercial use are offered. Never use a picture whose licence you cannot name. If nothing fits, say so and
  propose a drawn alternative (CSS or SVG); never hotlink a picture from another website.

## 3. Save it with its credit
- Save the original with `download_asset` into `assets/original/` with a plain name: `assets/original/hero.jpg`.
  Originals are never edited or overwritten.
- The answer includes the credit line. Append it to `assets/CREDITS.md` (create the file with a heading
  "# Picture credits" if it is missing), one line per picture, naming the file that uses it.
- Nothing about a picture goes on the web page: no credit, licence or photographer. The line stays only in
  `assets/CREDITS.md`, as a record of where the file came from.

## 4. Edit and optimise
Load the `image-recipes` skill for the commands. Work from the original into `assets/`:
- Crop to the shape, resize to the size it is shown at (and twice that for sharp phones, no more).
- Tell the person, in plain words, which pictures are stand-ins they should replace with their own.
- Remove a background only when the design needs a cut-out.
- Save WebP for photos (JPEG as a fallback only if the page needs one), PNG only for cut-outs and graphics.
- Aim for under 200 KB for a hero and under 100 KB for a card. Strip metadata with `exiftool`.
- Check every result: its size on disk and its pixel size, and that it opens.

## 5. Report
Call `report` with a short summary and a table: file, what it shows, pixel size, file size, licence, a one-line alt text. Do not use the document of your report for anything but
that table.
