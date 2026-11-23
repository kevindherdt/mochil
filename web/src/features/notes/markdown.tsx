// features/notes/markdown.tsx — a small Obsidian-flavoured markdown renderer that outputs **React elements**.
//
// Why not `marked` + `DOMPurify`: the entry bundle has ~7 kB of headroom under the 170 kB budget, and a renderer
// plus a sanitiser is 20 kB gzipped or more. Building React nodes instead of HTML strings also removes the XSS
// question entirely — there is no `dangerouslySetInnerHTML` here, so a note can never inject markup.
//
// Obsidian's own dialect is what matters here, not CommonMark completeness: `![[image.png]]` and `[[Note]]` are
// the two syntaxes a real vault is full of, and a plain markdown renderer shows them as raw text.
//
// Supported blocks: frontmatter (stripped), ATX headings, fenced code, blockquotes, ordered/unordered/task lists,
// tables, horizontal rules, paragraphs.
// Supported inline: **bold**, *italic*, `code`, ~~strike~~, ==highlight==, [text](url), ![alt](src),
// ![[attachment]], [[Note]], [[Note|alias]], #tag.
import type { ReactNode } from "react";
import { Text } from "../../ui";
import type { VaultEntry, VaultPage } from "./api/vault";

export interface MarkdownContext {
  /** Folder of the note being rendered — image paths in Obsidian are relative to it. */
  noteFolder: string;
  /** Every vault entry, so `![[x.png]]` can be resolved by name the way Obsidian resolves it. */
  entries: VaultEntry[];
  onOpenNote: (page: VaultPage) => void;
}

const IMAGE_EXTENSIONS = /\.(png|jpe?g|gif|webp|svg|avif)$/i;

function entryPath(entry: VaultEntry): string {
  return entry.path;
}

function isNote(entry: VaultEntry): entry is VaultPage {
  return "text_url" in entry && typeof (entry as VaultPage).text_url === "string";
}

/** URL a vault image is served from. `open_url` is the authenticated same-origin route the server mints. */
function assetUrl(entry: VaultEntry): string | null {
  const url = (entry as VaultPage & { open_url?: string }).open_url;
  return typeof url === "string" ? url : null;
}

function basename(path: string): string {
  return path.slice(path.lastIndexOf("/") + 1);
}

/** Resolves an image reference the way Obsidian does: relative to the note, then by bare filename, then by suffix. */
function resolveImage(src: string, context: MarkdownContext): string | null {
  const cleaned = src.replace(/^<|>$/g, "").trim();
  if (/^(https?:)?\/\//i.test(cleaned) || cleaned.startsWith("data:")) return cleaned;
  const decoded = decodeURIComponent(cleaned.replace(/^\.\//, ""));

  // A path that already points at the vault's own document route needs no lookup.
  if (decoded.startsWith("/documents/")) return decoded;

  const relative = context.noteFolder === "" ? decoded : `${context.noteFolder}/${decoded}`;
  const candidates = [relative, decoded];
  for (const candidate of candidates) {
    const exact = context.entries.find((entry) => entryPath(entry) === candidate);
    if (exact !== undefined) return assetUrl(exact);
  }
  // Obsidian's fallback: a bare `![[photo.png]]` matches that file anywhere in the vault.
  const wanted = basename(decoded).toLowerCase();
  const byName = context.entries.find((entry) => basename(entryPath(entry)).toLowerCase() === wanted);
  return byName === undefined ? null : assetUrl(byName);
}

function resolveNote(target: string, context: MarkdownContext): VaultPage | null {
  const cleaned = target.split("#")[0]?.split("|")[0]?.trim() ?? "";
  if (cleaned === "") return null;
  const withExtension = /\.md$/i.test(cleaned) ? cleaned : `${cleaned}.md`;
  const relative = context.noteFolder === "" ? withExtension : `${context.noteFolder}/${withExtension}`;
  const notes = context.entries.filter(isNote);
  const exact = notes.find((note) => note.path === relative || note.path === withExtension);
  if (exact !== undefined) return exact;
  const wanted = basename(withExtension).toLowerCase();
  return notes.find((note) => basename(note.path).toLowerCase() === wanted) ?? null;
}

/** `---\nkey: value\n---` at the very top is Obsidian's frontmatter, not prose. */
function stripFrontmatter(source: string): string {
  return source.replace(/^---\r?\n[\s\S]*?\r?\n---[ \t]*\r?\n?/, "");
}

const INLINE_SOURCE = [
    "(?<code>`[^`\\n]+`)",
    "(?<embed>!\\[\\[([^\\]|]+)(\\|[^\\]]*)?\\]\\])",
    "(?<image>!\\[([^\\]]*)\\]\\(([^)\\s]+)(\\s+\"[^\"]*\")?\\))",
    "(?<link>\\[([^\\]]+)\\]\\(([^)\\s]+)(\\s+\"[^\"]*\")?\\))",
    "(?<wikilink>\\[\\[([^\\]|]+)(\\|[^\\]]*)?\\]\\])",
    "(?<bold>\\*\\*[^*\\n]+\\*\\*|__[^_\\n]+__)",
    "(?<strike>~~[^~\\n]+~~)",
    "(?<highlight>==[^=\\n]+==)",
    "(?<italic>\\*[^*\\n]+\\*|_[^_\\n]+_)",
    "(?<tag>(?:^|\\s)#[\\w/-]{2,})",
  ].join("|");

function renderInline(source: string, context: MarkdownContext, keyPrefix: string): ReactNode[] {
  const nodes: ReactNode[] = [];
  let last = 0;
  let index = 0;
  // A fresh regex per call: a shared module-level /g regex keeps its lastIndex, and `renderInline` recurses (bold
  // inside a list item, emphasis inside a link), so one shared instance made the outer scan resume from the
  // inner call's position — an infinite "render everything again" loop that hung the tab on a 6 kB note.
  const pattern = new RegExp(INLINE_SOURCE, "g");
  for (let match = pattern.exec(source); match !== null; match = pattern.exec(source)) {
    if (match.index > last) nodes.push(source.slice(last, match.index));
    const groups = match.groups ?? {};
    const key = `${keyPrefix}-i${index++}`;

    if (groups.code !== undefined) {
      nodes.push(
        <code
          key={key}
          style={{
            fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
            fontSize: "0.92em",
            background: "var(--color-raised)",
            padding: "1px 5px",
            borderRadius: "var(--radius-4)",
          }}
        >
          {groups.code.slice(1, -1)}
        </code>,
      );
    } else if (groups.embed !== undefined) {
      // Positional `match[n]` indices break the moment an alternative gains a group, so every branch re-parses
      // its own named capture instead. This is also what made wikilinks render as dangling on the first pass.
      const embed = /^!\[\[([^\]|]+)(?:\|([^\]]*))?\]\]$/.exec(groups.embed);
      const target = embed?.[1]?.trim() ?? "";
      const alias = embed?.[2]?.trim() ?? "";
      const url = IMAGE_EXTENSIONS.test(target) ? resolveImage(target, context) : null;
      if (url !== null) {
        // In Obsidian a pipe on an image embed is a size, not alt text: `![[shot.png|220]]` or `|220x140`.
        const size = /^(\d+)(?:x(\d+))?$/.exec(alias.trim());
        const width = size === null ? undefined : Number(size[1]);
        const height = size?.[2] === undefined ? undefined : Number(size[2]);
        nodes.push(<Image key={key} src={url} alt={filenameOf(target)} width={width} height={height} />);
      } else if (IMAGE_EXTENSIONS.test(target)) {
        // An image embed whose file is not in the vault (deleted, or never synced). Naming it beats rendering
        // nothing at all, and it is a different failure from a dangling note link, so it looks different.
        nodes.push(<MissingAttachment key={key} name={target} />);
      } else {
        // A non-image embed (`![[Some note]]`) is a transclusion — shown as a link to that note rather than
        // silently rendered as a broken image.
        const note = resolveNote(target, context);
        nodes.push(<WikiLink key={key} page={note} label={alias || target} context={context} />);
      }
    } else if (groups.image !== undefined) {
      const image = /^!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)$/.exec(groups.image);
      const alt = image?.[1] ?? "";
      const url = resolveImage(image?.[2] ?? "", context);
      nodes.push(url === null ? <span key={key}>{alt}</span> : <Image key={key} src={url} alt={alt} />);
    } else if (groups.link !== undefined) {
      const link = /^\[([^\]]+)\]\(([^)\s]+)(?:\s+"[^"]*")?\)$/.exec(groups.link);
      const label = link?.[1] ?? "";
      const href = link?.[2] ?? "";
      const external = /^(https?:)?\/\//i.test(href);
      nodes.push(
        <a
          key={key}
          href={href}
          {...(external ? { target: "_blank", rel: "noreferrer noopener" } : {})}
          style={{ color: "var(--color-accent)", textDecoration: "underline" }}
        >
          {label}
        </a>,
      );
    } else if (groups.wikilink !== undefined) {
      const wikilink = /^\[\[([^\]|]+)(?:\|([^\]]*))?\]\]$/.exec(groups.wikilink);
      const target = wikilink?.[1]?.trim() ?? "";
      const alias = wikilink?.[2]?.trim();
      nodes.push(<WikiLink key={key} page={resolveNote(target, context)} label={alias ?? target} context={context} />);
    } else if (groups.bold !== undefined) {
      nodes.push(
        <strong key={key}>{renderInline(groups.bold.slice(2, -2), context, `${key}b`)}</strong>,
      );
    } else if (groups.strike !== undefined) {
      nodes.push(<s key={key}>{renderInline(groups.strike.slice(2, -2), context, `${key}s`)}</s>);
    } else if (groups.highlight !== undefined) {
      nodes.push(
        <mark
          key={key}
          style={{ background: "var(--color-accent)", color: "var(--color-accentInk)", borderRadius: "var(--radius-4)", padding: "0 3px" }}
        >
          {groups.highlight.slice(2, -2)}
        </mark>,
      );
    } else if (groups.italic !== undefined) {
      nodes.push(<em key={key}>{renderInline(groups.italic.slice(1, -1), context, `${key}i`)}</em>);
    } else if (groups.tag !== undefined) {
      const tag = groups.tag.trim();
      nodes.push(
        <span
          key={key}
          style={{
            display: "inline-block",
            padding: "1px 8px",
            borderRadius: "var(--radius-pill)",
            background: "var(--color-raised)",
            color: "var(--color-muted)",
            fontSize: "0.85em",
          }}
        >
          {tag}
        </span>,
      );
    }
    last = match.index + match[0].length;
  }
  if (last < source.length) nodes.push(source.slice(last));
  return nodes;
}

function filenameOf(path: string): string {
  return basename(path);
}

function Image({ src, alt, width, height }: { src: string; alt: string; width?: number | undefined; height?: number | undefined }) {
  return (
    <img
      src={src}
      alt={alt}
      loading="lazy"
      style={{
        display: "block",
        maxWidth: "100%",
        ...(width === undefined ? {} : { width: `${width}px` }),
        ...(height === undefined ? {} : { height: `${height}px` }),
        height: "auto",
        borderRadius: "var(--radius-8)",
        border: "1px solid var(--color-line)",
        margin: "var(--space-8) 0",
      }}
    />
  );
}

function MissingAttachment({ name }: { name: string }) {
  return (
    <span
      title="This file is not in the vault — it was deleted or never synced here."
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: "var(--space-8)",
        padding: "var(--space-8) var(--space-12)",
        margin: "var(--space-8) 0",
        borderRadius: "var(--radius-8)",
        border: "1px dashed var(--color-line)",
        color: "var(--color-muted)",
        fontSize: "12px",
      }}
    >
      <span aria-hidden="true">🖼</span>
      {name} · not in this vault
    </span>
  );
}

function WikiLink({ page, label, context }: { page: VaultPage | null; label: string; context: MarkdownContext }) {
  if (page === null) {
    // Unresolved wikilink: shown, not hidden — Obsidian users expect to see that a link is dangling.
    return (
      <span style={{ color: "var(--color-muted)", textDecoration: "underline dotted" }} title="No note with that name in this vault">
        {label}
      </span>
    );
  }
  return (
    <button
      type="button"
      onClick={() => context.onOpenNote(page)}
      style={{
        all: "unset",
        cursor: "pointer",
        color: "var(--color-accent)",
        textDecoration: "underline",
      }}
    >
      {label}
    </button>
  );
}

const tableRow = /^\s*\|(.+)\|\s*$/;
const tableDivider = /^\s*\|?[\s:-]*-[\s|:-]*\|?\s*$/;

/** Renders a note. A component rather than a bare function so this module exports components only (the
 *  react-refresh rule) and callers write `<Markdown source={text} context={ctx} />`. */
export function Markdown({ source, context }: { source: string; context: MarkdownContext }) {
  const lines = stripFrontmatter(source).split(/\r?\n/);
  const blocks: ReactNode[] = [];
  let paragraph: string[] = [];

  const flush = () => {
    if (paragraph.length === 0) return;
    const key = `p${blocks.length}`;
    blocks.push(
      <p key={key} style={{ margin: "0 0 var(--space-12)" }}>
        {renderInline(paragraph.join(" "), context, key)}
      </p>,
    );
    paragraph = [];
  };

  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i] ?? "";

    if (line.trim() === "") {
      flush();
      continue;
    }

    const fence = line.match(/^\s*```(\w*)/);
    if (fence !== null) {
      flush();
      const body: string[] = [];
      i += 1;
      while (i < lines.length && !/^\s*```/.test(lines[i] ?? "")) {
        body.push(lines[i] ?? "");
        i += 1;
      }
      blocks.push(
        <pre
          key={`code${blocks.length}`}
          style={{
            margin: "0 0 var(--space-12)",
            padding: "var(--space-12)",
            background: "var(--color-raised)",
            border: "1px solid var(--color-line)",
            borderRadius: "var(--radius-8)",
            overflowX: "auto",
            fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
            fontSize: "13px",
            lineHeight: 1.5,
          }}
        >
          <code>{body.join("\n")}</code>
        </pre>,
      );
      continue;
    }

    const heading = line.match(/^(#{1,6})\s+(.*)$/);
    if (heading !== null) {
      flush();
      const level = heading[1]?.length ?? 1;
      const key = `h${blocks.length}`;
      // Obsidian's editor headings are visually distinct without being huge; these follow the app's own scale.
      const size = level === 1 ? "26" : level === 2 ? "22" : level === 3 ? "18" : "16";
      blocks.push(
        <div key={key} style={{ margin: level <= 2 ? "var(--space-16) 0 var(--space-8)" : "var(--space-12) 0 var(--space-4)" }}>
          <Text as="div" size={size}>
            <strong>{renderInline(heading[2] ?? "", context, key)}</strong>
          </Text>
        </div>,
      );
      continue;
    }

    if (/^\s*([-*_])\s*\1\s*\1[\s-*_]*$/.test(line)) {
      flush();
      blocks.push(
        <hr key={`hr${blocks.length}`} style={{ border: "none", borderTop: "1px solid var(--color-line)", margin: "var(--space-16) 0" }} />,
      );
      continue;
    }

    if (/^\s*>/.test(line)) {
      flush();
      const quoted: string[] = [];
      while (i < lines.length && /^\s*>/.test(lines[i] ?? "")) {
        quoted.push((lines[i] ?? "").replace(/^\s*>\s?/, ""));
        i += 1;
      }
      i -= 1;
      const key = `q${blocks.length}`;
      blocks.push(
        <blockquote
          key={key}
          style={{
            margin: "0 0 var(--space-12)",
            padding: "var(--space-4) var(--space-12)",
            borderLeft: "3px solid var(--color-accent)",
            color: "var(--color-muted)",
          }}
        >
          {renderInline(quoted.join(" "), context, key)}
        </blockquote>,
      );
      continue;
    }

    if (tableRow.test(line) && tableDivider.test(lines[i + 1] ?? "")) {
      flush();
      const header = (line.match(tableRow)?.[1] ?? "").split("|").map((cell) => cell.trim());
      const rows: string[][] = [];
      i += 2;
      while (i < lines.length && tableRow.test(lines[i] ?? "")) {
        rows.push(((lines[i] ?? "").match(tableRow)?.[1] ?? "").split("|").map((cell) => cell.trim()));
        i += 1;
      }
      i -= 1;
      blocks.push(
        <div key={`t${blocks.length}`} style={{ overflowX: "auto", marginBottom: "var(--space-12)" }}>
          <table style={{ borderCollapse: "collapse", width: "100%" }}>
            <thead>
              <tr>
                {header.map((cell, column) => (
                  <th
                    key={`h${column}`}
                    style={{ textAlign: "left", padding: "var(--space-8)", borderBottom: "1px solid var(--color-line)" }}
                  >
                    {renderInline(cell, context, `th${column}`)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, rowIndex) => (
                <tr key={`r${rowIndex}`}>
                  {row.map((cell, column) => (
                    <td
                      key={`c${column}`}
                      style={{ padding: "var(--space-8)", borderBottom: "1px solid var(--color-line)" }}
                    >
                      {renderInline(cell, context, `td${rowIndex}-${column}`)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>,
      );
      continue;
    }

    const unordered = line.match(/^(\s*)[-*+]\s+(.*)$/);
    const ordered = line.match(/^(\s*)\d+[.)]\s+(.*)$/);
    if (unordered !== null || ordered !== null) {
      flush();
      const bulleted = unordered !== null;
      const items: string[] = [];
      while (i < lines.length) {
        const candidate = lines[i] ?? "";
        const match = bulleted ? candidate.match(/^(\s*)[-*+]\s+(.*)$/) : candidate.match(/^(\s*)\d+[.)]\s+(.*)$/);
        if (match === null) break;
        items.push(match[2] ?? "");
        i += 1;
      }
      i -= 1;
      const key = `l${blocks.length}`;
      const ListTag = bulleted ? "ul" : "ol";
      blocks.push(
        <ListTag key={key} style={{ margin: "0 0 var(--space-12)", paddingLeft: "var(--space-24)" }}>
          {items.map((item, itemIndex) => {
            const task = item.match(/^\[([ xX])\]\s*(.*)$/);
            if (task !== null) {
              return (
                <li key={`${key}-${itemIndex}`} style={{ marginBottom: "var(--space-4)", listStyle: "none" }}>
                  <label style={{ display: "flex", gap: "var(--space-8)", alignItems: "baseline" }}>
                    <input type="checkbox" checked={task[1] !== " "} readOnly aria-label="Task" />
                    <span style={{ textDecoration: task[1] === " " ? "none" : "line-through", opacity: task[1] === " " ? 1 : 0.7 }}>
                      {renderInline(task[2] ?? "", context, `${key}-${itemIndex}`)}
                    </span>
                  </label>
                </li>
              );
            }
            return (
              <li key={`${key}-${itemIndex}`} style={{ marginBottom: "var(--space-4)" }}>
                {renderInline(item, context, `${key}-${itemIndex}`)}
              </li>
            );
          })}
        </ListTag>,
      );
      continue;
    }

    paragraph.push(line.trim());
  }

  flush();
  return <div>{blocks}</div>;
}
