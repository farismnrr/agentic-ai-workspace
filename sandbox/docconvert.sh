#!/bin/sh
# docconvert <file> [pdf|docx|pptx|png] [outdir]: converts an office file with LibreOffice, offline. The
# profile lives in the writable cache because the rest of the image is read-only.
set -eu

in="${1:?usage: docconvert <file> [pdf|docx|pptx|png] [outdir]}"
to="${2:-pdf}"
out="${3:-$(dirname "$in")}"
profile="${XDG_CACHE_HOME:-/cache}/libreoffice"
[ -w "$(dirname "$profile")" ] || profile="/tmp/libreoffice"
mkdir -p "$profile"

exec soffice --headless --norestore --nolockcheck "-env:UserInstallation=file://${profile}" \
  --convert-to "$to" --outdir "$out" "$in"
