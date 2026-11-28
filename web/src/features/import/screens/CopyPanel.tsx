// features/import/screens/CopyPanel.tsx — the writing half of Import: review, then copy.
//
// Three steps, because each one answers a question the previous cannot: preview says what would happen to every book,
// plan answers with a transaction that can be polled, and execute does the work. Nothing moves until the second click, and
// the screen never claims a copy happened because a request returned 201 — it waits for the transaction's own state.
import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Button } from "../../../ui/Button";
import { Card } from "../../../ui/Card";
import { Inline } from "../../../ui/Inline";
import { Stack } from "../../../ui/Stack";
import { Text, Heading } from "../../../ui/Text";
import {
  embedMetadata,
  executeCopy,
  fetchTransaction,
  planCopy,
  previewCopy,
  writeSidecars,
  type BatchPreview,
  type CopyEntry,
  type CopyMetadata,
  type CopyWriteResult,
} from "../api/import";

const FINAL_STATES = new Set(["committed", "failed", "rolled_back"]);

/** What a sidecar or embed run did, in one line: the successes are uninteresting, the failures are the whole point. */
function writtenLabel(written: { kind: string; results: CopyWriteResult[] }) {
  const ok = written.results.filter((result) => result.status === "written" || result.status === "embedded").length;
  const failed = written.results.filter((result) => result.status === "failed");
  if (failed.length === 0) return `${ok} ${written.kind === "embedded" ? "embedded" : "written"}, none failed.`;
  return `${ok} written, ${failed.length} failed: ${failed[0]?.target_path ?? ""} — ${failed[0]?.message ?? "unknown error"}`;
}

export function CopyPanel({
  entries,
  metadata,
  onCopied,
}: {
  entries: CopyEntry[];
  /** Each copyable book's fields by source path: what a sidecar or an embed would write. */
  metadata: Record<string, CopyMetadata>;
  onCopied: () => void;
}) {
  const [preview, setPreview] = useState<BatchPreview | undefined>(undefined);
  const [transactionId, setTransactionId] = useState<string | undefined>(undefined);
  const [refreshed, setRefreshed] = useState(false);
  const [written, setWritten] = useState<{ kind: string; results: CopyWriteResult[] } | undefined>(undefined);

  const review = useMutation({
    mutationFn: () => previewCopy(entries),
    onSuccess: (result) => setPreview(result),
  });
  const run = useMutation({
    mutationFn: async () => {
      const planned = await planCopy(entries);
      setTransactionId(planned.id);
      return executeCopy(planned.id);
    },
  });

  // The transaction's own state is the only proof of a copy. Polling carries it, and a committed one refreshes the list so
  // the rows that moved out stop being offered.
  // The copied books' metadata, by target path: what a sidecar or an embed would write.
  const writtenEntries = entries
    .map((entry) => ({ target: entry.target, metadata: metadata[entry.source] }))
    .filter((entry): entry is { target: string; metadata: CopyMetadata } => entry.metadata !== undefined);

  const write = useMutation({
    mutationFn: () => writeSidecars(writtenEntries),
    onSuccess: (results) => setWritten({ kind: "metadata.opf", results }),
  });
  const embed = useMutation({
    mutationFn: () => embedMetadata(writtenEntries),
    onSuccess: (results) => setWritten({ kind: "embedded", results }),
  });

  const watched = useQuery({
    queryKey: ["import", "transaction", transactionId],
    queryFn: () => fetchTransaction(transactionId ?? ""),
    enabled: Boolean(transactionId),
    refetchInterval: (query) => {
      const state = query.state.data?.state;
      return state && FINAL_STATES.has(state) ? false : 1500;
    },
  });

  const state = watched.data?.state;
  const copied = state === "committed";
  if (copied && !refreshed) {
    // Once per commit: the table behind the panel should stop offering books that just moved. The panel itself stays, because
    // a copy is rarely the last thing a person wants to do with these books.
    queueMicrotask(() => {
      setRefreshed(true);
      onCopied();
    });
  }

  // A label, not a loading view — written as a plain assignment because the lint law forbids the ternary shape in screens
  // outright, and weakening a law to allow one label is how laws stop meaning anything.
  let reviewLabel = `Review ${entries.length}`;
  if (review.isPending) reviewLabel = "Checking…";
  const ready = preview?.entries.filter((entry) => entry.ready).length ?? 0;
  const problems = (preview?.entries ?? []).filter((entry) => !entry.ready);

  return (
    <Card>
      <Stack gap={12}>
        <Heading size="18" level={3}>
          Copy
        </Heading>
        {entries.length === 0 ? (
          <Text muted size="12">
            Every book on this page is already copied, or has no name yet.
          </Text>
        ) : (
          <>
            <Text muted size="12">
              {entries.length} {entries.length === 1 ? "book" : "books"} with a name worked out. Copying moves a verified copy
              into the managed folder; the file in the drop folder is left where it is.
            </Text>

            {preview === undefined ? (
              <Inline gap={8}>
                <Button variant="secondary" onClick={() => review.mutate()} disabled={review.isPending}>
                  {reviewLabel}
                </Button>
                {review.isError ? (
                  <Text size="12" tone="error">
                    The check failed: {review.error instanceof Error ? review.error.message : "unknown error"}
                  </Text>
                ) : null}
              </Inline>
            ) : (
              <Stack gap={8}>
                <Text size="12">
                  {ready} ready, {problems.length} with a problem.
                </Text>
                {problems.length > 0 ? (
                  <Stack gap={4}>
                    {problems.slice(0, 8).map((entry) => (
                      <Text key={entry.source_path} size="12" tone="error">
                        {entry.source_path}: {entry.issues[0]?.message ?? entry.issues[0]?.code ?? "refused"}
                      </Text>
                    ))}
                    {problems.length > 8 ? (
                      <Text size="12" muted>
                        …and {problems.length - 8} more.
                      </Text>
                    ) : null}
                  </Stack>
                ) : null}
                <Inline gap={8}>
                  <Button onClick={() => run.mutate()} disabled={ready === 0 || run.isPending || Boolean(transactionId)}>
                    {run.isPending || transactionId ? "Copying…" : `Copy ${ready} ${ready === 1 ? "book" : "books"}`}
                  </Button>
                  <Button variant="secondary" onClick={() => setPreview(undefined)} disabled={Boolean(transactionId)}>
                    Back
                  </Button>
                </Inline>
                {run.isError ? (
                  <Text size="12" tone="error">
                    The copy could not be queued: {run.error instanceof Error ? run.error.message : "unknown error"}
                  </Text>
                ) : null}
                {transactionId ? (
                  <Text size="12" muted>
                    {state ? `Transaction ${state}.` : "Waiting for the server…"}
                    {watched.data?.error ? ` ${watched.data.error}` : ""}
                  </Text>
                ) : null}
                {copied ? (
                  <Stack gap={8}>
                    <Text size="12">
                      Copied {ready} {ready === 1 ? "book" : "books"} into the managed folder. The files in the drop folder
                      are still there.
                    </Text>
                    <Inline gap={8}>
                      <Button
                        variant="secondary"
                        onClick={() => write.mutate()}
                        disabled={write.isPending}
                        aria-label="Write metadata.opf beside the copied books"
                      >
                        Write metadata.opf
                      </Button>
                      <Button
                        variant="secondary"
                        onClick={() => embed.mutate()}
                        disabled={embed.isPending}
                        aria-label="Embed the metadata into the copied EPUBs"
                      >
                        Embed into the EPUBs
                      </Button>
                    </Inline>
                    {written ? <Text size="12" muted>{writtenLabel(written)}</Text> : null}
                  </Stack>
                ) : null}
              </Stack>
            )}
          </>
        )}
      </Stack>
    </Card>
  );
}
