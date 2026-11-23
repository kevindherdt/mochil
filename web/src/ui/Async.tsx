// Async — the one way a screen handles loading/error/empty (plan §4.2, law L5). A screen renders
// `<Async query={useBooks()} empty={(books) => books.length === 0}>{(books) => <BookGrid books={books} />}</Async>`
// instead of its own `isLoading ? … : …` ternary (which eslint.config.js's L5 rule forbids in features/**/screens).
import type { ReactNode } from "react";
import { Skeleton } from "./Skeleton";
import { EmptyState } from "./EmptyState";
import { ErrorState } from "./ErrorState";
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
}

const defaultSkeleton = (
  <Stack gap={8}>
    <Skeleton />
    <Skeleton />
    <Skeleton width="60%" />
  </Stack>
);

export function Async<T>({ query, children, skeleton, empty, emptyMessage = "Nothing here yet.", emptyAction }: AsyncProps<T>) {
  if (query.isLoading) return <>{skeleton ?? defaultSkeleton}</>;
  if (query.isError) return <ErrorState onRetry={query.refetch} />;
  if (query.data === undefined) return <ErrorState message="No data." onRetry={query.refetch} />;
  if (empty?.(query.data)) return <EmptyState message={emptyMessage} action={emptyAction} />;
  return <>{children(query.data)}</>;
}
