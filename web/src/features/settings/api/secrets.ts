// features/settings/api/secrets.ts — the one rule for write-only fields across settings panels.
//
// The server never returns a saved secret: GET /api/settings replaces `kosync.password` with
// `password_configured`, `metadata_providers.*_token` with `*_token_configured`, and `narration.token` with
// `narration_token_configured` (internal/settings/service.go's redact*), and the matching preserve* functions put the
// stored value back when a save arrives without it. So a panel must do exactly two things:
//
//   * never render a secret, only whether one is saved; and
//   * only include the field in a save when the reader actually typed a new value — never send the empty string
//     "just to keep it", because "empty means keep" is a rule of the preserve functions, not a promise of the API,
//     and a panel that relies on it is one server change away from wiping a token.
//
// That is what `typedSecret` is for: it returns an object to spread into a patch, or an empty object to send
// nothing at all.
export function typedSecret(field: string, value: string): Record<string, string> {
  const trimmed = value.trim();
  return trimmed === "" ? {} : { [field]: trimmed };
}

/** Reads a `*_configured` marker out of a settings section, which is how the panel knows a secret exists without
 *  ever seeing it. */
export function secretConfigured(section: Record<string, unknown>, marker: string): boolean {
  return section[marker] === true;
}
