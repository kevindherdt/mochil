#!/usr/bin/env python3
"""Build the demo library used for the README screenshots.

The books are Standard Ebooks editions: their text and cover art are public domain, and Standard
Ebooks dedicates its own editorial work and artwork to the public domain, so a screenshot of this
library contains nothing that belongs to anyone. Downloads land outside the repository (the files
are several megabytes each and are not part of Mochil), and the ebooks carry their own colophon.

Usage: scripts/demo-library.py [output-directory]   (default /tmp/demo-library/books)
"""
import pathlib
import sys
import urllib.request
import zipfile

BASE = "https://standardebooks.org/ebooks"

# (author slug, book slug) exactly as standardebooks.org addresses them.
BOOKS = [
    ("jane-austen", "pride-and-prejudice"),
    ("herman-melville", "moby-dick"),
    ("mary-shelley", "frankenstein"),
    ("bram-stoker", "dracula"),
    ("h-g-wells", "the-time-machine"),
    ("h-g-wells", "the-war-of-the-worlds"),
    ("robert-louis-stevenson", "treasure-island"),
    ("charlotte-bronte", "jane-eyre"),
    ("arthur-conan-doyle", "the-hound-of-the-baskervilles"),
    ("oscar-wilde", "the-picture-of-dorian-gray"),
    ("henry-david-thoreau", "walden"),
    ("henry-james", "the-turn-of-the-screw"),
]


def url(author: str, book: str) -> str:
    # ?source=download skips the "Your Download Has Started" interstitial page.
    return f"{BASE}/{author}/{book}/downloads/{author}_{book}.epub?source=download"


def is_epub(path: pathlib.Path) -> bool:
    try:
        with zipfile.ZipFile(path) as z:
            return "mimetype" in z.namelist() and z.read("mimetype") == b"application/epub+zip"
    except (zipfile.BadZipFile, KeyError, OSError):
        return False


def main() -> int:
    target = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else "/tmp/demo-library/books")
    target.mkdir(parents=True, exist_ok=True)
    failed = []
    for author, book in BOOKS:
        dest = target / f"{author}_{book}.epub"
        if dest.exists() and is_epub(dest):
            continue
        try:
            request = urllib.request.Request(url(author, book), headers={"User-Agent": "mochil-docs/1.0"})
            with urllib.request.urlopen(request, timeout=60) as response:
                dest.write_bytes(response.read())
            if not is_epub(dest):
                dest.unlink()
                failed.append(f"{author}/{book}: not an EPUB")
        except Exception as error:  # noqa: BLE001 - report every failure by name
            failed.append(f"{author}/{book}: {error}")
    print(f"{len(list(target.glob('*.epub')))} ebooks in {target}")
    for line in failed:
        print("failed:", line)
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
