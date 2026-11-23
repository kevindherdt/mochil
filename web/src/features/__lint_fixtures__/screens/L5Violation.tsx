// Deliberately violates law L5 (plan §4.2: every async view uses <Async>; no bare isLoading ? … : … ternary in
// a screen). Not imported anywhere; exists for scripts/check-lint-fixtures.mjs to prove the rule fires.
interface Query {
  isLoading: boolean;
  data?: string;
}

export function BadAsyncScreen({ query }: { query: Query }) {
  return query.isLoading ? <span>Loading…</span> : <span>{query.data}</span>;
}
