// features/auth/screens/LoginScreen.tsx — a real React login screen (plan: proves the POST path lib/http's
// request() doesn't cover). The legacy server-rendered /login still works today; this is the Phase 1 proof,
// not yet the replacement (that's a cutover decision, not a Phase 1 one).
import { useState } from "react";
import { Page, Stack, Heading, Text, TextField, Button } from "../../../ui";
import { useLogin } from "../useLogin";

export function LoginScreen() {
  const [params] = useState(() => new URLSearchParams(window.location.search));
  const next = params.get("next") || "/";
  const { login, isPending, error } = useLogin(next);
  const [remember, setRemember] = useState(true);

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    login(
      { username: String(form.get("username") ?? ""), password: String(form.get("password") ?? ""), remember },
      { onSuccess: () => window.location.assign(next) },
    );
  }

  return (
    <Page>
      <Stack gap={24}>
        <Heading level={1}>Sign in</Heading>
        <Text muted>Sign in to open your books and media.</Text>

        <form onSubmit={handleSubmit}>
          <Stack gap={16}>
            <TextField label="Username" name="username" required autoComplete="username" />
            <TextField label="Password" name="password" type="password" required autoComplete="current-password" />
            <label>
              <input type="checkbox" checked={remember} onChange={(event) => setRemember(event.target.checked)} />
              {" "}Keep me signed in on this device
            </label>
            {error && <Text size="12" tone="error">{error.message}</Text>}
            <Button type="submit" variant="primary" disabled={isPending}>
              {isPending ? "Signing in…" : "Sign in"}
            </Button>
          </Stack>
        </form>
      </Stack>
    </Page>
  );
}
