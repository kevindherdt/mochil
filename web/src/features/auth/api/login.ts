// features/auth/api/login.ts — the one deliberate exception to lib/http's "one JSON fetch wrapper" rule: the
// real `/login` endpoint (internal/web/login_page.go, session_auth.go's `login()`) is a classic server-rendered
// HTML form, not a JSON API — it takes `application/x-www-form-urlencoded`, requires a single-use CSRF token
// scraped out of the GET response's HTML (there is no JSON endpoint that hands one out), and answers with
// either a 303 redirect (success) or a fresh copy of the same HTML page with a new token and an error message
// (failure). None of that fits `request<T>()`, so this file talks to `fetch()` directly instead of pretending
// a JSON shape exists where it doesn't.
export class LoginError extends Error {
  /** A fresh CSRF token scraped from the failure response — the server always re-renders a new one on failure
      (single-use tokens), so a retry needs this, not the token the previous attempt sent. */
  readonly csrf: string;

  constructor(message: string, csrf: string) {
    super(message);
    this.name = "LoginError";
    this.csrf = csrf;
  }
}

function extractCsrf(html: string): string | undefined {
  return html.match(/<input type="hidden" name="csrf" value="([^"]*)">/)?.[1];
}

function extractErrorMessage(html: string): string | undefined {
  return html.match(/<p class="error"[^>]*>([^<]*)<\/p>/)?.[1];
}

/** GETs the real login page and scrapes out the CSRF token every real sign-in needs — there is no JSON
    endpoint that hands one out on its own. */
export async function fetchLoginCsrf(next: string): Promise<string> {
  const response = await fetch(`/login?next=${encodeURIComponent(next)}`, { credentials: "same-origin" });
  const html = await response.text();
  const csrf = extractCsrf(html);
  if (!csrf) throw new Error("Could not read a sign-in token from the server's response.");
  return csrf;
}

export interface LoginCredentials {
  username: string;
  password: string;
  remember: boolean;
  csrf: string;
  next: string;
}

/** POSTs the real login form. On success the server 303-redirects to `next` and `fetch` follows it, so this
    resolves with nothing — the caller should navigate to `next` itself (full reload, since the session cookie
    the server just set needs to be the one a fresh page load picks up). On failure, throws `LoginError` with
    the server's own message and a fresh CSRF token for a retry. */
export async function submitLogin(credentials: LoginCredentials): Promise<void> {
  const body = new URLSearchParams({
    csrf: credentials.csrf,
    next: credentials.next,
    username: credentials.username,
    password: credentials.password,
    remember: credentials.remember ? "1" : "",
  });
  const response = await fetch("/login", {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body,
  });

  if (response.ok && new URL(response.url).pathname !== "/login") {
    return;
  }

  const html = await response.text();
  const message = extractErrorMessage(html) ?? "Could not sign in.";
  const csrf = extractCsrf(html) ?? credentials.csrf;
  throw new LoginError(message, csrf);
}
