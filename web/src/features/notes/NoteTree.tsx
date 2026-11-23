// features/notes/NoteTree.tsx — the vault's folder tree.
//
// Why this exists rather than the flat "one section per folder path" list the first cut rendered: a real Obsidian
// vault is deep (Agent/default/lessons/…, Projects/BoboBooks/Changelog/2026-11/…) and every folder rendered as its
// own always-open section turned 48 notes into a screen nobody could scan — Kevin: "can use the folder and a bit
// better sorting of the files so its actually properly viewable".
//
// Rules this follows, all of them about being scannable on a phone:
//   - folders before notes at every level, both alphabetical (case-insensitive, numeric-aware so "2026-11-07"
//     sorts before "2026-11-10" instead of after "2026-11-1")
//   - every folder starts closed, so the first screen is the vault's shape — a handful of rows — not its contents
//   - a folder row carries its note count, including notes in sub-folders, so a closed folder still says how much
//     is inside it
//   - opening one folder never moves the others
//   - searching flattens to matching notes: a tree that hides its matches behind collapsed folders is worse than
//     no tree at all
import { useState, type ReactNode } from "react";
import { ChevronRight, ChevronDown } from "lucide-react";
import { Stack, Inline, Text } from "../../ui";
import { countNotes, noteName, type VaultTreeNode } from "./tree";
import type { VaultPage } from "./api/vault";

interface NoteTreeProps {
  root: VaultTreeNode;
  onOpen: (page: VaultPage) => void;
  openingId: string | null;
}

export function NoteTree({ root, onOpen, openingId }: NoteTreeProps) {
  const [open, setOpen] = useState<Set<string>>(new Set());
  const toggle = (path: string) =>
    setOpen((current) => {
      const next = new Set(current);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });

  const rows: ReactNode[] = [];
  const walk = (node: VaultTreeNode, depth: number) => {
    for (const folder of node.folders) {
      const expanded = open.has(folder.path);
      rows.push(
        <button
          key={`folder:${folder.path}`}
          type="button"
          aria-expanded={expanded}
          onClick={() => toggle(folder.path)}
          style={{
            all: "unset",
            cursor: "pointer",
            display: "flex",
            alignItems: "center",
            gap: "var(--space-8)",
            padding: "var(--space-12) 0",
            paddingLeft: `calc(var(--space-8) * ${depth})`,
            width: "100%",
            borderBottom: "1px solid var(--color-line)",
            boxSizing: "border-box",
          }}
        >
          {expanded ? <ChevronDown size={15} aria-hidden="true" /> : <ChevronRight size={15} aria-hidden="true" />}
          <Text as="span" size="14">
            {folder.name}
          </Text>
          <span style={{ marginLeft: "auto", display: "inline-flex" }}>
            <Text as="span" size="11" muted>
              {countNotes(folder)}
            </Text>
          </span>
        </button>,
      );
      if (expanded) walk(folder, depth + 1);
    }
    for (const note of node.notes) {
      const opening = openingId === note.id;
      rows.push(
        <button
          key={`note:${note.id}`}
          type="button"
          disabled={opening}
          onClick={() => onOpen(note)}
          style={{
            all: "unset",
            cursor: opening ? "default" : "pointer",
            display: "block",
            padding: "var(--space-12) 0",
            paddingLeft: `calc(var(--space-8) * ${depth} + var(--space-24))`,
            width: "100%",
            borderBottom: "1px solid var(--color-line)",
            boxSizing: "border-box",
          }}
        >
          <Stack gap={4}>
            <Text size="14">{opening ? `${noteName(note)} — opening…` : noteName(note)}</Text>
            <Text size="11" muted>
              {new Date(note.modified).toLocaleDateString()} · {Math.max(1, Math.round(note.size / 1024))} kB
            </Text>
          </Stack>
        </button>,
      );
    }
  };
  walk(root, 0);

  return <div>{rows}</div>;
}

/** Search results: a flat, path-labelled list. Deliberately not a filtered tree — see this file's header. */
export function NoteSearchResults({ notes, onOpen, openingId }: { notes: VaultPage[]; onOpen: (page: VaultPage) => void; openingId: string | null }) {
  return (
    <div>
      {notes.map((note) => (
        <button
          key={note.id}
          type="button"
          disabled={openingId === note.id}
          onClick={() => onOpen(note)}
          style={{
            all: "unset",
            cursor: openingId === note.id ? "default" : "pointer",
            display: "block",
            padding: "var(--space-12) 0",
            width: "100%",
            borderBottom: "1px solid var(--color-line)",
            boxSizing: "border-box",
          }}
        >
          <Stack gap={4}>
            <Text size="14">{openingId === note.id ? `${noteName(note)} — opening…` : noteName(note)}</Text>
            <Inline gap={8}>
              <Text size="11" muted>
                {note.path}
              </Text>
            </Inline>
          </Stack>
        </button>
      ))}
    </div>
  );
}
