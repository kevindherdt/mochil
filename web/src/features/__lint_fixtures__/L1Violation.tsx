// Deliberately violates law L1 (plan docs/rewrite-plan.md §4.2: only shell/ and lib/useVisibleViewport.ts may
// read viewport size). Not imported anywhere — exists solely so scripts/check-lint-fixtures.mjs can prove the
// eslint rule actually fires on a real violation, not just on paper.
export function badHeight(): number {
  return window.innerHeight;
}
