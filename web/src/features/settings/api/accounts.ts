// features/settings/api/accounts.ts — the account management endpoints (internal/web/account_http.go).
//
// Every one of these is administrator-only (`administrator.go`'s `administratorOnly` returns true for the whole
// `/api/accounts` prefix), so the screen that uses them must gate itself on the session's role rather than calling
// and showing an error — a non-administrator seeing a list of other people's usernames would already be a leak.
//
// Three different verbs for three different things, which is the server's own design and worth keeping straight:
//   POST   /api/accounts              create      {username, password, role}
//   PATCH  /api/accounts/{id}         act on one  {action, password?}   action = disable | enable | reset_password
//   DELETE /api/accounts/{id}         delete      {confirmation}        must be disabled, and name itself exactly
//
// The action verbs are not a PUT of the account, because none of them is "here is the new account": disabling,
// enabling and resetting a password each end sessions and touch other subsystems on the server.
import { request, HttpError } from "../../../lib/http";
import { fetchAccounts, type AccountsResponse } from "../../auth/api/session";

export type AccountRole = "administrator" | "user";

export const ACCOUNT_ROLES: { id: AccountRole; label: string }[] = [
  { id: "administrator", label: "Administrator" },
  { id: "user", label: "User" },
];

export type AccountAction = "disable" | "enable" | "reset_password";

export async function listAccounts(): Promise<AccountsResponse> {
  return fetchAccounts();
}

/** The server answers failures as `{"error": "..."}` with a 4xx, and lib/http throws the raw body as the
 *  HttpError's message. Those sentences are written for a person ("At least one enabled administrator is
 *  required", "Disable this account before deleting it") and are far better than anything this app could invent,
 *  so they are shown as-is — parsed out of the JSON, with the raw text as a fallback. */
export function messageFromError(error: unknown): string {
  if (error instanceof HttpError) {
    try {
      const parsed = JSON.parse(error.message) as { error?: unknown };
      if (typeof parsed.error === "string" && parsed.error !== "") return parsed.error;
    } catch {
      // Not JSON — fall through to the raw text.
    }
    if (error.message !== "") return error.message;
    return `Request failed (${error.status})`;
  }
  return error instanceof Error && error.message !== "" ? error.message : "Something went wrong";
}

/** Creates an account. The response is the server's own `accounts.Identity`, which has **no json tags** — it comes
 *  back as `{"ID":…,"Username":…,"Role":…}`, capitalised, unlike every other payload here. Rather than model that
 *  quirk (or quietly "fix" it by parsing lowercase fields, which is what the first draft did: the schema rejected
 *  the real response and the screen reported failure *after* the account had already been created), this ignores
 *  the body entirely. Nothing on screen needs it — the list is re-fetched after every mutation. */
export async function createAccount(input: { username: string; password: string; role: AccountRole }): Promise<void> {
  await request<unknown>("/api/accounts", { method: "POST", body: input });
}

export async function actOnAccount(id: string, action: AccountAction, password?: string): Promise<void> {
  await request<unknown>(`/api/accounts/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: password === undefined ? { action } : { action, password },
  });
}

/** Deleting is permanent and the server insists on the account being disabled first, plus its exact username as
 *  the confirmation — a typed confirmation, not a click-through. */
export async function deleteAccount(id: string, confirmation: string): Promise<void> {
  await request<unknown>(`/api/accounts/${encodeURIComponent(id)}`, {
    method: "DELETE",
    body: { confirmation },
  });
}
