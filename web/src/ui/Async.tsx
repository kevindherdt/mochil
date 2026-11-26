// Async — the one way a screen handles loading/error/empty (plan §4.2, law L5). A screen renders
// `<Async query={useBooks()} empty={(books) => books.length === 0}>{(books) => <BookGrid books={books} />}</Async>`
// instead of its own `isLoading ? … : …` ternary (which eslint.config.js's L5 rule forbids in features/**/screens).
import type { ReactNode } from "react";
import { Skeleton } from "./Skeleton";
import { EmptyState } from "./EmptyState";
import { ErrorState } from "./ErrorState";
import { StaleNotice } from "./StaleNotice";
import { Stack } from "./Stack";

/** The subset of TanStack Query's UseQueryResult that <Async> needs — kept narrow so it isn't coupled to one
    query library and is easy to satisfy from a hand-built result in a test. */
export interface AsyncQueryLike<T> {
  data: T | undefined;
  isLoading: boolean;
  isError: boolean;
  error?: unknown;
  refetch?: () => void;
}

export interface AsyncProps<T> {
  query: AsyncQueryLike<T>;
  children: (data: T) => ReactNode;
  /** Shown while `isLoading` — defaults to 3 skeleton lines, which is wrong for a grid; screens with a
      non-list layout should pass a `skeleton` shaped like their real content. */
  skeleton?: ReactNode;
  /** True when `data` is loaded but empty (e.g. `(books) => books.length === 0`). */
  empty?: (data: T) => boolean;
  emptyMessage?: string;
  emptyAction?: { label: string; onClick: () => void };
  /** A failed *refetch* keeps the data on screen with a "couldn't refresh" strip; this makes it swap in the full
      error panel instead. Only a screen whose writes depend on the data being current wants that — a stale playlist
      revision, for instance, makes every further edit fail on the server. Everything else should keep the data. */
  failOnRefetchError?: boolean;
}

const defaultSkeleton = (
  <Stack gap={8}>
    <Skeleton />
    <Skeleton />
    <Skeleton width="60%" />
  </Stack>
);

export function Async<T>({
  query,
  children,
  skeleton,
  empty,
  emptyMessage = "Nothing here yet.",
  emptyAction,
  failOnRefetchError = false,
}: AsyncProps<T>) {
  if (query.isLoading) return <>{skeleton ?? defaultSkeleton}</>;

  if (query.data === undefined) {
    // A load that failed has nothing to show, so it gets the one action that fixes it. A load that has not started
    // yet is waiting, not broken: an `enabled: false` query reports `isLoading === false` with no data, which used to
    // paint a red "No data." and a retry button that could only refetch nothing.
    return query.isError ? <ErrorState onRetry={query.refetch} /> : <>{skeleton ?? defaultSkeleton}</>;
  }

  // Data that failed to *refresh* is still the data the person asked for: TanStack reports `isError` for a failed
  // refetch while keeping the last successful result. Swapping in a full error panel here threw away a list someone
  // was reading (and editing, and scrolled halfway down) over one dropped request, with a Retry that a moment later
  // would have worked anyway. Keep it and say it may be out of date. A screen that cannot act on stale data — the
  // playlist detail writes with an ETag, so a stale revision fails every edit — asks for the panel instead.
  if (query.isError && failOnRefetchError) return <ErrorState onRetry={query.refetch} />;
  const stale = query.isError ? <StaleNotice onRetry={query.refetch} /> : null;

  if (empty?.(query.data)) {
    return (
      <>
        {stale}
        <EmptyState message={emptyMessage} action={emptyAction} />
      </>
    );
  }
  return (
    <>
      {stale}
      {children(query.data)}
    </>
  );
}
