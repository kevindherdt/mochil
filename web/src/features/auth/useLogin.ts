// features/auth/useLogin.ts — the login mutation (plan §5.2). Wraps features/auth/api/login.ts's dedicated
// fetch calls (not lib/http's request() — see that file's header comment for why the real /login endpoint
// doesn't fit a JSON wrapper) in TanStack Query's useMutation so the screen gets isPending/error for free.
import { useCallback, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { fetchLoginCsrf, submitLogin, LoginError } from "./api/login";

export function useLogin(next: string) {
  const [csrf, setCsrf] = useState<string | undefined>(undefined);

  const ensureCsrf = useCallback(async () => {
    if (csrf) return csrf;
    const token = await fetchLoginCsrf(next);
    setCsrf(token);
    return token;
  }, [csrf, next]);

  const mutation = useMutation({
    mutationFn: async (input: { username: string; password: string; remember: boolean }) => {
      const token = await ensureCsrf();
      try {
        await submitLogin({ ...input, csrf: token, next });
      } catch (error) {
        // The server's failure response always carries a fresh single-use token — store it so the very next
        // attempt doesn't fail again just because it reused the one that was already spent.
        if (error instanceof LoginError) setCsrf(error.csrf);
        throw error;
      }
    },
  });

  return { ensureCsrf, login: mutation.mutate, isPending: mutation.isPending, error: mutation.error };
}
