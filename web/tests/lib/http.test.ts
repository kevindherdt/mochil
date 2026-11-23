// lib/http.test.ts — proves the session-expiry path against the REAL server's observed behaviour, not an
// assumption: a live `fetch('/api/books')` against the dev box while signed out came back
// `{ status: 200, redirected: true, url: '.../login?next=...', contentType: 'text/html' }`, never a 401
// (`internal/web/session_auth.go`'s session wall 303-redirects every unauthenticated request, API or not).
// These tests mock fetch to reproduce exactly that shape, so a future change that reverts to assuming a 401
// fails loudly here instead of only in production.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { HttpError, request, requestEnvelope } from "../../src/lib/http";

function jsonResponse(body: unknown, init: ResponseInit = {}): Response {
  return new Response(JSON.stringify(body), { status: 200, ...init, headers: { "Content-Type": "application/json" } });
}

/** Reproduces the real server's unauthenticated response: a 200 on the *final* URL after `fetch` already
    followed the 303 to /login, with an HTML body — not a 401. */
function loginRedirectResponse(origin: string, next: string): Response {
  const response = new Response("<html>login page</html>", {
    status: 200,
    headers: { "Content-Type": "text/html; charset=utf-8" },
  });
  Object.defineProperty(response, "redirected", { value: true });
  Object.defineProperty(response, "url", { value: `${origin}/login?next=${next}` });
  return response;
}

describe("request", () => {
  let assignSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    window.history.pushState({}, "", "/library?foo=bar");
    assignSpy = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign: assignSpy });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("returns parsed JSON on success", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({ ok: true })));
    await expect(request("/api/books")).resolves.toEqual({ ok: true });
  });

  it("redirects to /login on the real server's 303-to-login shape, not a 401", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(loginRedirectResponse(window.location.origin, "%2Fapi%2Fbooks")),
    );
    await expect(request("/api/books")).rejects.toThrow(HttpError);
    expect(assignSpy).toHaveBeenCalledWith("/login?next=%2Flibrary%3Ffoo%3Dbar");
  });

  it("also redirects on a literal 401, in case any endpoint is ever changed to answer that way", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 401 })));
    await expect(request("/api/books")).rejects.toThrow(HttpError);
    expect(assignSpy).toHaveBeenCalled();
  });

  it("throws HttpError with the response body on a real error, without redirecting", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("library not found", { status: 404 })));
    await expect(request("/api/libraries/missing")).rejects.toMatchObject({ status: 404 });
    expect(assignSpy).not.toHaveBeenCalled();
  });

  it("sends no CSRF header — the real server has none for /api/* (session cookie only)", async () => {
    const fetchSpy = vi.fn().mockResolvedValue(jsonResponse({ ok: true }));
    vi.stubGlobal("fetch", fetchSpy);
    await request("/api/libraries/1", { method: "POST", body: { name: "x" } });
    const headers = (fetchSpy.mock.calls[0]?.[1] as RequestInit).headers as Record<string, string>;
    expect(headers["X-CSRF-Token"]).toBeUndefined();
  });
});

/** A JSON response carrying paging headers — `jsonResponse` deliberately pins Content-Type, so headers are merged
    here instead of being replaced. */
function pagedResponse(body: unknown, headers: Record<string, string>): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json", ...headers },
  });
}

describe("requestEnvelope", () => {
  beforeEach(() => {
    vi.stubGlobal("window", { location: { pathname: "/next/library", search: "", assign: vi.fn() } });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("reports the paging cursor and total the server sent", async () => {
    const fetchSpy = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(pagedResponse([{ id: "a" }], { "X-Next-Cursor": "120:abc", "X-Total-Count": "326" }));
    const page = await requestEnvelope<{ id: string }[]>("/api/books?library=books");
    expect(page.data).toEqual([{ id: "a" }]);
    expect(page.nextCursor).toBe("120:abc");
    expect(page.total).toBe(326);
    expect(fetchSpy).toHaveBeenCalledOnce();
  });

  it("treats a missing or empty cursor as the end of the shelf, not as a cursor", async () => {
    // The server's last page answers with no cursor header at all; some proxies send it empty. Both mean "done",
    // and a caller that tested `!== undefined` would loop forever on the empty string.
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(pagedResponse([], { "X-Total-Count": "0" }));
    expect((await requestEnvelope("/api/books")).nextCursor).toBeNull();

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(pagedResponse([], { "X-Next-Cursor": "" }));
    expect((await requestEnvelope("/api/books")).nextCursor).toBeNull();
  });

  it("reports total as null when the header is absent or not a number", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(jsonResponse([]));
    expect((await requestEnvelope("/api/books")).total).toBeNull();

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(pagedResponse([], { "X-Total-Count": "many" }));
    expect((await requestEnvelope("/api/books")).total).toBeNull();
  });
});
