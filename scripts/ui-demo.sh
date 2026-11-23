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

# Six invented books, written as real EPUBs so the reader, covers and metadata all have something to
# show. Deliberately a mix of one-word and long titles: a grid that only looks right with short titles
# is not tested by short titles.
python3 - "$work/books" <<'PY'
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

def cover(width, height, rgb):
    # A tiny solid-colour PNG is enough to prove covers load and tint.
    r, g, b = rgb
    raw = b"".join(b"\x00" + bytes([r, g, b]) * width for _ in range(height))
    def chunk(kind, data):
        import zlib, struct
        body = kind + data
        return struct.pack(">I", len(data)) + body + struct.pack(">I", zlib.crc32(body) & 0xFFFFFFFF)
    import zlib, struct
    header = struct.pack(">IIBBBBB", width, height, 8, 2, 0, 0, 0)
    return (b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", header) + chunk(b"IDAT", zlib.compress(raw, 9)) + chunk(b"IEND", b""))

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
        "OEBPS/cover.png": cover(120, 180, rgb),
    }
    with zipfile.ZipFile(os.path.join(out, f"{slug}.epub"), "w") as archive:
        archive.writestr("mimetype", files.pop("mimetype"), compress_type=zipfile.ZIP_STORED)
        for name, body in files.items():
            archive.writestr(name, body, compress_type=zipfile.ZIP_DEFLATED)
PY

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
