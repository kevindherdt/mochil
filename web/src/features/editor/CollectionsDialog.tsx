// features/editor/CollectionsDialog.tsx — put the selected books into a collection, or take them out.
//
// Collections are per person, not per library file: this only changes Mochil's own record of who belongs where, and it
// takes effect immediately (there is no pending-edit stage for membership — the collections endpoint is the write).
// A smart collection is a saved rule, so books cannot be pushed into it by hand; it is offered for *filtering* only.
import { useState } from "react";
import { Button, Heading, Inline, Stack, Text, TextField } from "../../ui";
import { setCollectionMembers, type EditorCollection } from "./api/editor";

export function CollectionsDialog({
  libraryId,
  ids,
  collections,
  onClose,
  onDone,
}: {
  libraryId: string;
  ids: string[];
  collections: EditorCollection[];
  onClose: () => void;
  onDone: (message: string) => void;
}) {
  const manual = collections.filter((collection) => !collection.smart);
  const [chosen, setChosen] = useState<string>(manual[0]?.id ?? "");
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [problem, setProblem] = useState<string | undefined>(undefined);

  const usingNew = chosen === "";

  async function save(remove: boolean) {
    setBusy(true);
    setProblem(undefined);
    try {
      if (usingNew && name.trim() === "") {
        setProblem("Pick a collection, or type a name to make one.");
        return;
      }
      await setCollectionMembers(libraryId, usingNew ? { name: name.trim() } : { id: chosen }, ids, remove);
      const label = usingNew ? name.trim() : (manual.find((entry) => entry.id === chosen)?.name ?? "the collection");
      onDone(
        `${ids.length} ${ids.length === 1 ? "book" : "books"} ${remove ? "taken out of" : "put in"} “${label}”.`,
      );
    } catch (error) {
      setProblem(error instanceof Error ? error.message : "That could not be saved.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div
      data-collections-dialog
      onKeyDown={(event) => {
        if (event.key === "Escape") onClose();
      }}
      style={{
        position: "fixed",
        inset: 0,
        zIndex: 40,
        background: "color-mix(in srgb, #000 55%, transparent)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: "var(--space-16)",
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Collections"
        style={{
          width: "min(460px, 100%)",
          background: "var(--color-bg)",
          border: "1px solid var(--color-line)",
          borderRadius: "var(--radius-16)",
          padding: "var(--space-24)",
          boxShadow: "0 24px 60px rgba(0,0,0,0.35)",
        }}
      >
        <Stack gap={16}>
          <Stack gap={4}>
            <Heading size="18">Collections</Heading>
            <Text size="12" muted>
              {ids.length} {ids.length === 1 ? "book" : "books"} selected.
            </Text>
          </Stack>

          {manual.length > 0 && (
            <label style={{ display: "block" }}>
              <Text size="11" muted>
                Collection
              </Text>
              <select
                value={chosen}
                onChange={(event) => setChosen(event.target.value)}
                style={{
                  width: "100%",
                  padding: "6px 8px",
                  border: "1px solid var(--color-line)",
                  borderRadius: "var(--radius-8)",
                  background: "var(--color-bg)",
                  color: "var(--color-text)",
                  font: "inherit",
                  fontSize: "var(--font-size-12)",
                }}
              >
                {manual.map((collection) => (
                  <option key={collection.id} value={collection.id}>
                    {collection.name} ({collection.count})
                  </option>
                ))}
                <option value="">— a new collection —</option>
              </select>
            </label>
          )}

          {(usingNew || manual.length === 0) && (
            <TextField label="Name for the new collection" value={name} onChange={(event) => setName(event.target.value)} />
          )}

          {problem !== undefined && <div style={{ fontSize: "var(--font-size-12)", color: "var(--color-error)" }}>{problem}</div>}

          <Inline gap={8} wrap>
            <Button variant="primary" disabled={busy} onClick={() => void save(false)}>
              Add to collection
            </Button>
            <Button variant="secondary" disabled={busy || usingNew} onClick={() => void save(true)}>
              Remove from it
            </Button>
            <Button variant="secondary" disabled={busy} onClick={onClose}>
              Cancel
            </Button>
          </Inline>
        </Stack>
      </div>
    </div>
  );
}
