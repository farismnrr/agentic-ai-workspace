#!/opt/rembg/bin/python
"""Cut the background out of a picture: removebg <in> <out.png> [good|fast].

Works offline: both models are baked into the image. The model is always named, because the library's own
default is a large model with a paid commercial licence.
"""
import sys

from PIL import Image
from rembg import new_session, remove

MODELS = {"good": "isnet-general-use", "fast": "u2netp"}


def main(argv: list[str]) -> int:
    if len(argv) not in (3, 4) or (len(argv) == 4 and argv[3] not in MODELS):
        print("usage: removebg <in> <out.png> [good|fast]", file=sys.stderr)
        return 2

    session = new_session(MODELS[argv[3] if len(argv) == 4 else "good"])
    with Image.open(argv[1]) as source:
        remove(source.convert("RGBA"), session=session).save(argv[2], "PNG")

    return 0


sys.exit(main(sys.argv))
