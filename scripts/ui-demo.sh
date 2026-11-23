#!/usr/bin/env bash
# Boot a disposable Mochil instance with a small, invented library, for looking at the interface.
#
# The point is a UI pass that is repeatable: the same six books, the same call numbers, the same empty
# states, every time -- so "the grid looks wrong" can be measured instead of remembered. Nothing here
# touches real data: the config, the cache and the books all live under one temporary directory that
# `--stop` removes.
#
#   ./scripts/ui-demo.sh            # start on a free port, print the base URL and a login
#   ./scripts/ui-demo.sh --stop     # stop it and remove the directory
#
# The login is created through the same first-account route the browser uses, so a sweep can sign in.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
work=${UI_DEMO_DIR:-/tmp/mochil-ui-demo}
port=${UI_DEMO_PORT:-18399}
user=${UI_DEMO_USER:-ui}
pass=${UI_DEMO_PASS:-ui-sweep-password-1}

if [[ ${1:-} == --stop ]]; then
  [[ -f $work/pid ]] && kill "$(cat "$work/pid")" 2>/dev/null || true
  rm -rf "$work"
  echo "stopped and removed $work"
  exit 0
fi

[[ -x $work/mochil ]] || { echo "build it first: go build -o $work/mochil ./cmd/mochil" >&2; exit 1; }

# A demo built before the last change to the server or the web build is worse than no demo: it looks like the current
# app, it answers on the same port, and every measurement taken against it describes code that no longer exists. That
# is exactly what happened on 23 Nov 2026 — the sweep ran a two-day-old binary and reported the retired shell's pages.
newest=$(find cmd internal web/src -newer "$work/mochil" -type f \( -name '*.go' -o -name '*.ts' -o -name '*.tsx' -o -name '*.css' \) -print -quit 2>/dev/null)
if [[ -n $newest ]]; then
  echo "the demo binary is older than $newest" >&2
  echo "rebuild it: go build -o $work/mochil ./cmd/mochil" >&2
  exit 1
fi
rm -rf "$work/books" "$work/cache"
mkdir -p "$work/books" "$work/cache"

# The books. Real editions are better than invented ones in a screenshot: a grid of drawn placeholders says nothing
# about what a library looks like. These are Standard Ebooks — public-domain text AND public-domain cover art, so a
# picture of this shelf contains nothing that belongs to anyone. Fetch them once with
# `python3 scripts/demo-library.py /tmp/demo-library/books` (they are megabytes each, so they are never committed).
books_dir=$work/books
if ! compgen -G "$books_dir/*.epub" > /dev/null 2>&1; then
  mkdir -p "$books_dir"
  public_domain=${UI_DEMO_PUBLIC_DOMAIN:-/tmp/demo-library/books}
  if compgen -G "$public_domain/*.epub" > /dev/null 2>&1; then
    # Only the canonical set: a directory that has been used before also holds earlier downloads of the same work under
    # another name, and two Draculas on the shelf is not a demo of anything.
    while IFS= read -r wanted; do
      [[ -f $public_domain/$wanted ]] && cp "$public_domain/$wanted" "$books_dir"/
    done < <(python3 "$(dirname "$0")/demo-library.py" --list)
    echo "using the public-domain editions from $public_domain"
  else
    echo "no public-domain editions found; inventing books instead (run scripts/demo-library.py for the real ones)" >&2
    python3 - "$books_dir" <<'PY_FALLBACK'
import os, sys, zipfile

out = sys.argv[1]
books = [
    ("The Cartographer's Apprentice", "Mira Halvorsen", "A mapmaker's daughter finds a country that is not on any map.", (198, 74, 64)),
    ("Salt", "Emil Prosser", "Two brothers, one lighthouse, and a winter that will not end.", (38, 92, 110)),
    ("A Quiet Kind of Thunder", "Noor Haddad", "A year of letters between two people who never meet.", (120, 88, 140)),
    ("Notes on the Long Way Home: A Journey Through the Northern Reaches and Back Again", "Peter Vandermeer", "Travel notes from a route that no longer exists.", (168, 128, 48)),
    ("Nine Kinds of Silence", "Ada Ferreira", "Nine stories about the things people do not say.", (60, 120, 90)),
    ("The Understory", "Kai Lindqvist", "A forest, a family, and forty years between them.", (150, 70, 110)),
]

# A cover, drawn rather than borrowed: the book's own initial on a colour ground, with a lighter band under it. Real
# cover art belongs to whoever made it, and this demo exists to be screenshotted into a public repository — so the
# covers are ours, and they look like a designed series instead of six empty squares.
GLYPHS = {
    "A": ("01110", "10001", "10001", "11111", "10001", "10001", "10001"),
    "B": ("11110", "10001", "11110", "10001", "10001", "10001", "11110"),
    "C": ("01110", "10001", "10000", "10000", "10000", "10001", "01110"),
    "D": ("11110", "10001", "10001", "10001", "10001", "10001", "11110"),
    "E": ("11111", "10000", "11110", "10000", "10000", "10000", "11111"),
    "F": ("11111", "10000", "11110", "10000", "10000", "10000", "10000"),
    "G": ("01110", "10001", "10000", "10111", "10001", "10001", "01111"),
    "H": ("10001", "10001", "11111", "10001", "10001", "10001", "10001"),
    "I": ("11111", "00100", "00100", "00100", "00100", "00100", "11111"),
    "J": ("00111", "00010", "00010", "00010", "00010", "10010", "01100"),
    "K": ("10001", "10010", "10100", "11000", "10100", "10010", "10001"),
    "L": ("10000", "10000", "10000", "10000", "10000", "10000", "11111"),
    "M": ("10001", "11011", "10101", "10101", "10001", "10001", "10001"),
    "N": ("10001", "11001", "10101", "10011", "10001", "10001", "10001"),
    "O": ("01110", "10001", "10001", "10001", "10001", "10001", "01110"),
    "P": ("11110", "10001", "10001", "11110", "10000", "10000", "10000"),
    "Q": ("01110", "10001", "10001", "10001", "10101", "10010", "01101"),
    "R": ("11110", "10001", "10001", "11110", "10100", "10010", "10001"),
    "S": ("01111", "10000", "01110", "00001", "00001", "10001", "01110"),
    "T": ("11111", "00100", "00100", "00100", "00100", "00100", "00100"),
    "U": ("10001", "10001", "10001", "10001", "10001", "10001", "01110"),
    "V": ("10001", "10001", "10001", "10001", "10001", "01010", "00100"),
    "W": ("10001", "10001", "10001", "10101", "10101", "11011", "10001"),
    "X": ("10001", "01010", "00100", "00100", "00100", "01010", "10001"),
    "Y": ("10001", "01010", "00100", "00100", "00100", "00100", "00100"),
    "Z": ("11111", "00001", "00010", "00100", "01000", "10000", "11111"),
}


def cover(width, height, rgb, letter):
    r, g, b = rgb
    # The band is the same hue lifted, so the cover reads as designed rather than as a flat fill.
    band = (min(255, r + 60), min(255, g + 60), min(255, b + 60))
    ink = (245, 242, 236)
    rows = []
    scale = max(4, width // 12)
    glyph = GLYPHS.get(letter.upper(), GLYPHS["T"])
    glyph_w, glyph_h = 5 * scale, 7 * scale
    left, top = (width - glyph_w) // 2, (height - glyph_h) // 2 - height // 12
    for y in range(height):
        row = bytearray()
        for x in range(width):
            colour = band if y > height - height // 4 else (r, g, b)
            gx, gy = (x - left) // scale, (y - top) // scale
            if 0 <= gx < 5 and 0 <= gy < 7 and glyph[gy][gx] == "1":
                colour = ink
            row.extend(colour)
        rows.append(b"\x00" + bytes(row))

    def chunk(kind, data):
        import zlib, struct
        body = kind + data
        return struct.pack(">I", len(data)) + body + struct.pack(">I", zlib.crc32(body) & 0xFFFFFFFF)

    import zlib, struct
    header = struct.pack(">IIBBBBB", width, height, 8, 2, 0, 0, 0)
    return b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", header) + chunk(b"IDAT", zlib.compress(b"".join(rows), 9)) + chunk(b"IEND", b"")


for index, (title, author, blurb, rgb) in enumerate(books, start=1):
    slug = f"book-{index}"
    chapter = "".join(
        f"<p>{'Paragraph %d of %s. ' % (n, title)}{blurb}</p>" for n in range(1, 13)
    )
    files = {
        "mimetype": "application/epub+zip",
        "META-INF/container.xml": '<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>',
        "OEBPS/content.opf": f'''<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="id">urn:uuid:ui-sweep-{index:04d}</dc:identifier>
    <dc:title>{title}</dc:title><dc:creator>{author}</dc:creator><dc:language>en</dc:language>
    <dc:description>{blurb}</dc:description>
    <meta property="dcterms:modified">2026-01-0{index}T00:00:00Z</meta>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ch1" href="chapter-1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter-2.xhtml" media-type="application/xhtml+xml"/>
    <item id="cover" href="cover.png" media-type="image/png" properties="cover-image"/>
  </manifest>
  <spine><itemref idref="ch1"/><itemref idref="ch2"/></spine>
</package>''',
        "OEBPS/nav.xhtml": '<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops"><head><title>Contents</title></head><body><nav epub:type="toc"><ol><li><a href="chapter-1.xhtml">One</a></li><li><a href="chapter-2.xhtml">Two</a></li></ol></nav></body></html>',
        "OEBPS/chapter-1.xhtml": f'<html xmlns="http://www.w3.org/1999/xhtml"><head><title>One</title></head><body><h1>{title}</h1><h2>{author}</h2>{chapter}</body></html>',
        "OEBPS/chapter-2.xhtml": f'<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Two</title></head><body><h1>Two</h1>{chapter}</body></html>',
        "OEBPS/cover.png": cover(300, 450, rgb, title.strip()[0]),
    }
    with zipfile.ZipFile(os.path.join(out, f"{slug}.epub"), "w") as archive:
        archive.writestr("mimetype", files.pop("mimetype"), compress_type=zipfile.ZIP_STORED)
        for name, body in files.items():
            archive.writestr(name, body, compress_type=zipfile.ZIP_DEFLATED)
PY_FALLBACK
  fi
fi

cat > "$work/config.json" <<JSON
{
  "listen": "127.0.0.1:$port",
  "cache_directory": "$work/cache",
  "public_url": "http://127.0.0.1:$port",
  "libraries": [
    { "id": "books", "name": "Library", "kind": "books", "source": { "type": "filesystem", "path": "$work/books" } },
    { "id": "comics", "name": "Comics", "kind": "comics", "source": { "type": "filesystem", "path": "$work/books" } }
  ]
}
JSON

# A demo from an earlier run that is still holding the port looks exactly like a working one: the new server fails to
# bind, the old one keeps answering, and every measurement describes code from days ago. On 23 Nov 2026 that is how a
# screenshot run captured the retired interface.
if (exec 3<>/dev/tcp/127.0.0.1/$port) 2>/dev/null; then
  exec 3<&- 3>&-
  echo "port $port is already in use — something else is serving a demo" >&2
  echo "find it: ss -ltnp | grep $port    then stop it, or run with UI_DEMO_PORT=<other>" >&2
  exit 1
fi
MOCHIL_DATA_DIR="$work/cache" "$work/mochil" -config "$work/config.json" > "$work/server.log" 2>&1 &
echo $! > "$work/pid"

for _ in $(seq 1 80); do
  curl -fsS "http://127.0.0.1:$port/healthz" >/dev/null 2>&1 && break
  sleep 0.25
done
curl -fsS "http://127.0.0.1:$port/healthz" >/dev/null || { echo "the demo server did not start; see $work/server.log" >&2; exit 1; }

# The first account is created by the same form the browser fills in: the page mints a one-time token
# and the POST has to carry it, so a script cannot skip the page.
csrf=$(curl -fsS "http://127.0.0.1:$port/setup" | sed -n 's/.*name="csrf" value="\([^"]*\)".*/\1/p' | head -1)
if [[ -n $csrf ]]; then
  curl -fsS -X POST "http://127.0.0.1:$port/setup" \
    --data-urlencode "csrf=$csrf" --data-urlencode "username=$user" \
    --data-urlencode "password=$pass" --data-urlencode "confirm=$pass" -o /dev/null || true
fi

echo "base url:  http://127.0.0.1:$port"
echo "login:     $user / $pass"
echo "artifacts: $work"
echo "stop:      $(basename "$0") --stop"
