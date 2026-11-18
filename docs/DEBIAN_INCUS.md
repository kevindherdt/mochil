# Debian and Incus installation

This setup runs OPDShelf directly under systemd in a Debian 13 container. A
Seafile-backed library needs no book mount: only the configuration, API token,
and disposable catalogue cache live in the container.

## 1. Create the container

On the Incus host:

```sh
incus launch images:debian/13 opdshelf
incus config device add opdshelf http proxy \
  listen=tcp:0.0.0.0:8080 connect=tcp:127.0.0.1:8080
```

The proxy exposes port 8080 on the Incus host. Bind it to a specific host IP
instead of `0.0.0.0` when appropriate. Put a TLS reverse proxy in front before
sending Basic authentication credentials over an untrusted network.

## 2. Build and install

Copy this source tree into the container, then run inside it:

```sh
apt update
apt install -y ca-certificates golang-go
cd /path/to/opdshelf
go test ./...
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' \
  -o /usr/local/bin/mochil ./cmd/mochil
adduser --system --group --home /var/lib/opdshelf opdshelf
install -d -o root -g opdshelf -m 0750 /etc/opdshelf
install -o root -g root -m 0644 packaging/systemd/opdshelf.service \
  /etc/systemd/system/opdshelf.service
```

The installed service creates `/var/lib/opdshelf` automatically and runs
without Linux capabilities.

If you want confirmed metadata repair for a Calibre shelf, also install the
database CLI packaged with Calibre:

```sh
apt install -y calibre
```

This is optional; browsing Calibre shelves remains read-only and does not need
the package.

## 3. Configure Seafile

Create `/etc/opdshelf/config.json`:

```json
{
  "listen": "127.0.0.1:8080",
  "scan_every": "10m",
  "cache_directory": "/var/lib/opdshelf/catalogs",
  "libraries": [
    {
      "id": "books",
      "name": "Books",
      "kind": "books",
      "source": {
        "type": "seafile",
        "url": "https://seafile.example.com",
        "repository": "REPOSITORY-UUID",
        "root": "/Books",
        "manifest": ".opds/catalog.json",
        "token_file": "/etc/opdshelf/seafile-token"
      },
      "inference": { "folder_layout": "auto" }
    }
  ]
}
```

A relative `manifest` path is resolved under `root`, so this example reads
`/Books/.opds/catalog.json`. A manifest beginning with `/` is resolved from the
repository root instead.

Store the Seafile API token without quotes or a variable name:

```sh
install -o root -g opdshelf -m 0640 /dev/null /etc/opdshelf/seafile-token
editor /etc/opdshelf/seafile-token
chown root:opdshelf /etc/opdshelf/config.json
chmod 0640 /etc/opdshelf/config.json
```

Optional reader authentication belongs in `/etc/opdshelf/environment`:

```text
OPDSHELF_USERNAME=reader
OPDSHELF_PASSWORD=replace-this
```

Protect that file with `root:opdshelf` ownership and mode `0640` as well.

## 4. Optionally generate a richer manifest

This is not required for automatic Seafile crawling. For richer embedded
metadata without server-side book downloads, build `opdshelf-manifest` on the
computer that has the Seafile library synced:

```sh
go build -o opdshelf-manifest ./cmd/opdshelf-manifest
./opdshelf-manifest \
  -library "$HOME/Seafile/Books" \
  -output "$HOME/Seafile/Books/.opds/catalog.json" \
  -folder-layout auto
```

Wait for Seafile to synchronize `.opds/catalog.json` before starting the
server. Regenerate it after adding or changing books; the helper writes it
atomically so OPDShelf does not observe a partially written manifest.

## 5. Start and verify

```sh
systemctl daemon-reload
systemctl enable --now opdshelf
systemctl status opdshelf
curl -fsS http://127.0.0.1:8080/healthz
journalctl -u opdshelf -n 100 --no-pager
```

Open `http://INCUS-HOST:8080/` for the web library and settings, or use
`http://INCUS-HOST:8080/opds` in an OPDS reader. On first start the service
copies `/etc/opdshelf/config.json` to its writable managed configuration at
`/var/lib/opdshelf/config.json`; later library changes can be made in the web
settings. When `public_url` is configured, it must match the reverse proxy URL.

For upgrades, rebuild the binary, replace `/usr/local/bin/opdshelf`, and run
`systemctl restart opdshelf`. Catalogue cache files are disposable.
