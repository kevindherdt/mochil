// features/settings/useAccounts.ts — the Accounts panel's data and mutations. One query, three mutations, one
// shared invalidation: every account change (create, disable/enable/reset, delete) can change the *current*
// account's row too, so `session/accounts` is invalidated alongside it — the same key the shell's session hook
// reads, which is what keeps About from showing a stale role.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import type { AccountsResponse } from "../auth/api/session";
import { actOnAccount, createAccount, deleteAccount, listAccounts, type AccountAction, type AccountRole } from "./api/accounts";

const KEY = ["session", "accounts"] as const;

export function useAccounts(enabled: boolean): AsyncQueryLike<AccountsResponse> {
  const query = useQuery({
    queryKey: KEY,
    queryFn: listAccounts,
    // Administrator-only endpoint: a non-administrator must not call it at all, not call it and cope with a 403.
    enabled,
  });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

interface Mutations {
  create: (input: { username: string; password: string; role: AccountRole }, onDone?: () => void) => void;
  act: (id: string, action: AccountAction, password?: string) => void;
  remove: (id: string, confirmation: string) => void;
  isPending: boolean;
  error: unknown;
  reset: () => void;
}

export function useAccountMutations(): Mutations {
  const queryClient = useQueryClient();
  // Both keys: the list is what changed, and the *caller's own* identity may have changed with it (a self
  // role change, or being disabled from another tab), which is what `useSession` reads.
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["session", "accounts"] });
    void queryClient.invalidateQueries({ queryKey: ["session", "me"] });
  };

  const create = useMutation({ mutationFn: createAccount, onSuccess: invalidate });
  const act = useMutation({
    mutationFn: (input: { id: string; action: AccountAction; password?: string }) =>
      actOnAccount(input.id, input.action, input.password),
    onSuccess: invalidate,
  });
  const remove = useMutation({
    mutationFn: (input: { id: string; confirmation: string }) => deleteAccount(input.id, input.confirmation),
    onSuccess: invalidate,
  });

  const errors = [create.error, act.error, remove.error].filter((error) => error !== null && error !== undefined);

  return {
    create: (input, onDone) => create.mutate(input, { onSuccess: () => onDone?.() }),
    act: (id, action, password) => act.mutate(password === undefined ? { id, action } : { id, action, password }),
    remove: (id, confirmation) => remove.mutate({ id, confirmation }),
    isPending: create.isPending || act.isPending || remove.isPending,
    error: errors[0] ?? null,
    reset: () => {
      create.reset();
      act.reset();
      remove.reset();
    },
  };
}
