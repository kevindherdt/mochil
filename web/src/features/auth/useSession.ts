// features/auth/useSession.ts — the one hook feature screens read for "who is signed in" (plan §5.2: "screens
// only call hooks"). Session *expiry* is handled lower down, inside lib/http's request() (a 303-to-login on the
// real server, not a 401 — see lib/http.ts's comment) — this hook is for the common case: showing the signed-in
// account, e.g. on Settings → About, or deciding whether a panel is for administrators.
//
// It reads GET /api/me, not GET /api/accounts. The list is administrator-only, so a session read through it
// failed with a 403 for every ordinary user — see api/session.ts's comment and internal/web/account_me.go.
// Shaped to match ui/Async's `AsyncQueryLike<Me>` directly (`data`, not `account`) so a screen can pass
// `useSession()` straight into `<Async query={useSession()}>` with no adapter.
import { useQuery } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { fetchMe, type Me } from "./api/session";

export function useSession(): AsyncQueryLike<Me> {
  const query = useQuery({
    queryKey: ["session", "me"],
    queryFn: fetchMe,
    // Who you are does not change while you stay signed in, but an administrator can disable or re-role an
    // account mid-session; a short staleTime keeps that visible without re-asking on every navigation.
    staleTime: 30_000,
  });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}
