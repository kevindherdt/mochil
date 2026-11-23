// features/settings/useModules.ts — the Modules panel's data: the module list, and one mutation that writes a
// switch. Kept separate from the screen so the same list can feed a future Settings overview (which modules are
// off) without the screen's own layout coming along.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { AsyncQueryLike } from "../../ui";
import { fetchModules, type ModuleInfo } from "./api/modules";
import { saveModuleSwitches } from "./api/settings";

export function useModules(): AsyncQueryLike<ModuleInfo[]> {
  const query = useQuery({
    queryKey: ["settings", "modules"],
    queryFn: fetchModules,
    // Module state is the server's answer about what is *running*; caching it for a minute after a save would
    // show switches that disagree with the process. Short staleTime, and the mutation invalidates explicitly.
    staleTime: 15_000,
  });
  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: () => void query.refetch(),
  };
}

export interface ModuleSwitchState {
  /** Modules whose switch the user has moved and whose save is still in flight or unsaved. */
  pending: ReadonlySet<string>;
  /** Saved values for modules the server has confirmed, keyed by module id. */
  saved: Record<string, boolean>;
  toggle: (moduleId: string, enabled: boolean) => void;
  isSaving: boolean;
  error: unknown;
}

/** One switch moved = one save. Batching was considered and rejected: a module switch is a deliberate, rare,
 *  single action, and an unsaved-changes bar for it would be the kind of extra chrome this app has been told to
 *  stop adding (Kevin's standing "uncluttered" rule). The write is a read-then-write of the settings document, so
 *  two rapid toggles are serialised by the mutation rather than racing. */
export function useSetModule(modules: ModuleInfo[]): ModuleSwitchState {
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: (switches: Record<string, boolean>) => saveModuleSwitches(switches),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["settings", "modules"] });
    },
  });

  const saved: Record<string, boolean> = {};
  for (const module of modules) saved[module.id] = module.enabled;

  return {
    pending: new Set(mutation.isPending && mutation.variables !== undefined ? Object.keys(mutation.variables) : []),
    saved,
    toggle: (moduleId, enabled) => mutation.mutate({ [moduleId]: enabled }),
    isSaving: mutation.isPending,
    error: mutation.error,
  };
}
