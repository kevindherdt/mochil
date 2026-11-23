// lib/http — the one fetch wrapper (plan docs/rewrite-plan.md §5.2). Every feature's api/ module calls this
// instead of fetch() directly, so session-expiry handling exists in exactly one place.
export class HttpError extends Error {
  readonly status: number;
  readonly path: string;

  constructor(status: number, path: string, message: string) {
    super(message);
    this.name = "HttpError";
    this.status = status;
    this.path = path;
  }
}

export interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  signal?: AbortSignal;
}

export interface Envelope<T> {
  data: T;
  /** `X-Next-Cursor` — the server's own paging token for the next window of this exact view. */
  nextCursor: string | null;
  /** `X-Total-Count` — how many items the whole filtered view holds, not how many came back. */
  total: number | null;
}

/**
 * Fetches `path` against the same origin and redirects to /login on session expiry — the one place that
 * handles it, so screens never duplicate that logic (plan §5.2).
 *
 * There is no CSRF token on API requests: verified against the real server (`internal/web/session_auth.go`) —
 * the `csrf` field only guards the server-rendered login/setup HTML forms, and `multi.go`'s session wall has no
 * CSRF check for `/api/*` at all (same-origin session cookie is the whole story). An earlier draft of this
 * function sent an `X-CSRF-Token` header read from a `<meta name="csrf-token">` tag that doesn't exist anywhere
 * in the app — a no-op that implied a security mechanism which was never real. Removed rather than kept as
 * defensive dead code.
 *
 * Session expiry does **not** show up as a 401: the real server 303-redirects every unauthenticated request —
 * API or not — to `/login?next=...` (`session_auth.go`'s session wall, confirmed live against the dev box:
 * `fetch('/api/books')` while signed out comes back `status: 200, redirected: true, url: '.../login?next=...'`,
 * never a 401). `fetch()` follows that redirect itself, so the signal to look for is the *final* URL landing on
 * `/login`, not a status code. The 401 check is kept anyway in case any endpoint is ever changed to answer that
 * way directly, but it is not the path that actually fires today.
 */
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const envelope = await rawRequest(path, options);
  return envelope.data as T;
}

/** `request`, plus the two paging headers the paged endpoints answer with. A separate export rather than a
 *  changed return type: every other caller wants the body and nothing else. */
export async function requestEnvelope<T>(path: string, options: RequestOptions = {}): Promise<Envelope<T>> {
  const envelope = await rawRequest(path, options);
  const totalHeader = envelope.response.headers.get("X-Total-Count");
  const total = totalHeader === null ? null : Number.parseInt(totalHeader, 10);
  return {
    data: envelope.data as T,
    // An empty cursor header is the server saying "no more"; normalise it to null so callers test one thing.
    nextCursor: envelope.response.headers.get("X-Next-Cursor") || null,
    total: Number.isFinite(total) ? total : null,
  };
}

/** One implementation of the session-expiry/error/JSON rules, shared by both exports above. */
async function rawRequest(path: string, options: RequestOptions = {}): Promise<{ data: unknown; response: Response }> {
  const method = options.method ?? "GET";
  const headers: Record<string, string> = { Accept: "application/json" };

  const init: RequestInit = { method, headers, credentials: "same-origin" };
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    init.body = JSON.stringify(options.body);
  }
  if (options.signal !== undefined) {
    init.signal = options.signal;
  }

  const response = await fetch(path, init);
  const landedOnLogin = response.redirected && new URL(response.url).pathname === "/login";

  if (response.status === 401 || landedOnLogin) {
    const next = encodeURIComponent(window.location.pathname + window.location.search);
    window.location.assign(`/login?next=${next}`);
    throw new HttpError(401, path, "session expired");
  }

  if (!response.ok) {
    const text = await response.text().catch(() => "");
    throw new HttpError(response.status, path, text || response.statusText);
  }

  if (response.status === 204) return { data: undefined, response };
  return { data: await response.json(), response };
}
