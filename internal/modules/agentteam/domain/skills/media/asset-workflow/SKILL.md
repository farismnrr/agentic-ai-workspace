---
name: asset-workflow
description: Find photos you may use, save them with their credit, edit and optimise them for a page.
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
- Call `search_images` with two or three different phrasings: what it shows, the setting, the mood.
- Choose what fits the subject, the shape and the size (the larger side of at least the size it will be shown at).
  Prefer scenes and objects to recognisable people: a licence covers the photo, not the person's right to their
  own image. Avoid pictures with text, logos, brand names or watermarks.
- Only pictures from the latest search can be saved, and only those with a licence that allows editing and
  commercial use are offered. Never use a picture whose licence you cannot name. If nothing fits, say so and
  propose a drawn alternative (CSS or SVG); never hotlink a picture from another website.

## 3. Save it with its credit
- Save the original with `download_asset` into `assets/original/` with a plain name: `assets/original/hero.jpg`.
  Originals are never edited or overwritten.
- The answer includes the credit line. Append it to `assets/CREDITS.md` (create the file with a heading
  "# Picture credits" if it is missing), one line per picture, naming the file that uses it.
- CC BY and CC BY-SA need the credit shown to visitors. Say in your report which pictures need a visible
  credit, so the page can show a credits line or page. CC0 and public domain need none, but the line stays in the file.

## 4. Edit and optimise
Load the `image-recipes` skill for the commands. Work from the original into `assets/`:
- Crop to the shape, resize to the size it is shown at (and twice that for sharp phones, no more).
- Remove a background only when the design needs a cut-out.
- Save WebP for photos (JPEG as a fallback only if the page needs one), PNG only for cut-outs and graphics.
- Aim for under 200 KB for a hero and under 100 KB for a card. Strip metadata with `exiftool`.
- Check every result: its size on disk and its pixel size, and that it opens.

## 5. Report
Call `report` with a short summary and a table: file, what it shows, pixel size, file size, licence, credit
shown to visitors (yes or no), a one-line alt text. Do not use the document of your report for anything but
that table.
