// features/auth/api/session.ts — the two "who is signed in" endpoints, which are not the same request.
//
//   GET /api/me        the caller's own identity — available to every signed-in account.
//   GET /api/accounts  every account on the server — administrator-only (administrator.go), used by Settings.
//
// Reading the session from /api/accounts was wrong for exactly one reason, and it showed up the moment a second,
// ordinary account signed in: /api/me did not exist, so "who am I" came back 403 and every screen built on it
// (About, the Accounts panel's own role check) showed an error instead of the person's name. /api/me was added to
// the server rather than worked around — see internal/web/account_me.go — and it is the only thing the session
// hook calls now. The list below is still validated against contracts/api/accounts.json, the real Go fixture.
import { z } from "zod";
import { request } from "../../../lib/http";

const meSchema = z.object({
  id: z.string(),
  username: z.string(),
  role: z.string(),
});

export type Me = z.infer<typeof meSchema>;

export async function fetchMe(): Promise<Me> {
  return meSchema.parse(await request<unknown>("/api/me"));
}

const accountSchema = z.object({
  id: z.string(),
  username: z.string(),
  role: z.string(),
  disabled: z.boolean(),
});

const accountsResponseSchema = z.object({
  current_account_id: z.string(),
  registration_enabled: z.boolean(),
  environment_recovery_enabled: z.boolean(),
  items: z.array(accountSchema),
});

export type Account = z.infer<typeof accountSchema>;
export type AccountsResponse = z.infer<typeof accountsResponseSchema>;

export async function fetchAccounts(): Promise<AccountsResponse> {
  const raw = await request<unknown>("/api/accounts");
  return accountsResponseSchema.parse(raw);
}
