// web/eslint.config.js — enforces the plan's layout laws (docs/rewrite-plan.md §4.2) that a code review alone
// cannot be trusted to catch every time:
//
//   L1  Only shell/ and lib/useVisibleViewport may read viewport size or use matchMedia.
//   L2  Screens own no layout: features/**/screens/** and feature components may not import a *.module.css.
//   L5  Every async view uses <Async>: no bare `isLoading ? … : …` ternary in a screen.
//
// A lint rule nobody has seen fail is not proven to work — see src/features/__lint_fixtures__ for the
// deliberately violating snippets `node scripts/check-lint-fixtures.mjs` checks against this config. That
// directory is globally ignored below so it never fails the app's own `npm run lint`.
import js from "@eslint/js";
import tseslint from "typescript-eslint";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import globals from "globals";

export default tseslint.config(
  { ignores: ["dist", "node_modules", "src/features/__lint_fixtures__/**"] },
  {
    files: ["**/*.{ts,tsx}"],
    extends: [js.configs.recommended, ...tseslint.configs.recommended, reactHooks.configs.flat["recommended-latest"], reactRefresh.configs.vite],
    languageOptions: {
      ecmaVersion: 2023,
      globals: globals.browser,
    },
  },

  // L1: viewport reads belong only in shell/ and lib/useVisibleViewport.ts.
  {
    files: ["**/*.{ts,tsx}"],
    ignores: ["src/shell/**", "src/lib/useVisibleViewport.ts", "**/*.test.{ts,tsx}", "src/lib/__lint-fixtures__/**"],
    rules: {
      "no-restricted-syntax": [
        "error",
        {
          selector: "MemberExpression[object.name='window'][property.name='visualViewport']",
          message: "L1: only shell/ and lib/useVisibleViewport.ts may read window.visualViewport.",
        },
        {
          selector: "MemberExpression[object.name='window'][property.name='innerHeight']",
          message: "L1: only shell/ and lib/useVisibleViewport.ts may read window.innerHeight.",
        },
        {
          selector: "CallExpression[callee.property.name='matchMedia']",
          message: "L1: matchMedia belongs only in shell/ — primitives respond via container queries (L7).",
        },
      ],
    },
  },

  // L2: a screen or feature component composes ui/ primitives; it does not own its own CSS module.
  {
    files: ["src/features/**/screens/**/*.{ts,tsx}", "src/features/**/*.{ts,tsx}"],
    ignores: ["src/features/**/api/**", "**/*.test.{ts,tsx}"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["*.module.css"],
              message: "L2: screens own no layout. Extend or add a primitive in ui/ instead of a screen-owned CSS module.",
            },
          ],
        },
      ],
    },
  },

  // L5: a screen handles its own loading state with <Async> (gallery/ui are exempt: they build the primitive).
  {
    files: ["src/features/**/screens/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-syntax": [
        "error",
        {
          selector: "ConditionalExpression[test.type='MemberExpression'][test.property.name=/^is(Loading|Pending|Fetching)$/]",
          message: "L5: wrap this view in <Async> instead of a bare isLoading ? … : … ternary in a screen.",
        },
      ],
    },
  },
);
