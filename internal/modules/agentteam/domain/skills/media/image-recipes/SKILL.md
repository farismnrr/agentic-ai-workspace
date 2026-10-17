---
name: image-recipes
description: Command-line recipes for editing pictures in the sandbox: resize, crop, convert, remove a background, optimise.
---

# Image recipes

All of these run with `run_command` inside the sandbox. They work offline. Always write to a new file; never
overwrite an original in `assets/original/`.

## Look at a picture
- `vipsheader -a assets/original/hero.jpg` shows its size and format.
- `ls -l assets` shows how big each file is on disk.

## Resize and crop (libvips: fast and light on memory)
The output is the second argument and is relative to where you are. Put options in brackets after the name.
- Keep the shape, make it 1600 wide: `vips thumbnail assets/original/hero.jpg "assets/hero.jpg[Q=82,strip]" 1600`
- Crop to a wide shape around the interesting part:
  `vips thumbnail assets/original/hero.jpg "assets/hero-wide.jpg[Q=82,strip]" 1600 --height 900 --crop attention`
- Square card: `vips thumbnail in.jpg "out.jpg[Q=82,strip]" 600 --height 600 --crop attention`

## Convert
- WebP for a photo: `cwebp -q 80 -m 6 assets/hero.jpg -o assets/hero.webp`
- AVIF when the page asks for it: `avifenc --min 20 --max 40 assets/hero.jpg assets/hero.avif`
- PNG smaller without loss: `optipng -o4 -strip all assets/cutout.png`
- PNG smaller with a little loss: `pngquant --quality 65-85 --force --output assets/cutout.png assets/cutout.png`
- SVG to PNG: `rsvg-convert -w 800 logo.svg -o assets/logo.png`

## Remove a background
`removebg <in> <out.png> [good|fast]` works offline. `good` is the default and has the better edges; `fast` is rougher.
The result is always a PNG with a see-through background.
- `removebg assets/original/mug.jpg assets/mug.png`
- `removebg assets/original/mug.jpg assets/mug.png fast`
- Then trim the empty border: `convert assets/mug.png -trim +repage assets/mug.png`
- Put it on a flat colour: `convert assets/mug.png -background '#f6efe6' -alpha remove -alpha off assets/mug-on-cream.jpg`

## Shapes and icons (ImageMagick 6: the command is `convert`, there is no `magick`)
- Round avatar 400px: `convert in.jpg -resize 400x400^ -gravity center -extent 400x400 \( +clone -alpha extract -fill black -colorize 100 -fill white -draw "circle 200,200 200,0" \) -alpha off -compose copy_opacity -composite assets/avatar.png`
- Favicon set from a logo: `convert logo.png -define icon:auto-resize=64,48,32,16 assets/favicon.ico`
- Lighten or darken a photo behind text: `convert in.jpg -brightness-contrast -15x0 out.jpg`

## Vector (SVG)
- Shrink an SVG: `svgo --multipass in.svg -o assets/logo.svg`
- Turn a black-and-white cut-out into an SVG shape: `convert assets/mug.png -background white -alpha remove -threshold 50% /tmp/m.pbm && potrace -s /tmp/m.pbm -o assets/mug.svg`
- Ready-made icons (Lucide, ISC licence) are in `/opt/vector/node_modules/lucide-static/icons/` as one SVG per icon:
  `ls /opt/vector/node_modules/lucide-static/icons | grep -i coffee`, then copy the file into `assets/icons/`.
  Lucide's icons use `currentColor`, so CSS sets their colour. Add "Lucide, ISC licence" to `assets/CREDITS.md`.
- Check how an SVG looks: `rsvg-convert -w 600 assets/logo.svg -o /tmp/preview.png`

## Clean up
- Remove camera and location data: `exiftool -all= -overwrite_original assets/hero.jpg`
- Never keep a picture you cannot name the licence of.
