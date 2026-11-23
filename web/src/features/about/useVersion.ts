// features/about/useVersion.ts — the hook the About screen reads for "which build is this" (plan §5.2). About
// exists specifically so build ambiguity can be resolved by asking the app, not guessing — this is the data
// layer for that.
import { useQuery } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { fetchVersion, type VersionInfo } from "./api/version";

export function useVersion(): AsyncQueryLike<VersionInfo> {
  const query = useQuery({
    queryKey: ["about", "version"],
    queryFn: fetchVersion,
    // /about is never cached server-side either (about_http.go: "a stale version is worse than no version") —
    // match that here instead of showing a build number from 30s ago while the container is mid-restart.
    staleTime: 0,
  });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}
