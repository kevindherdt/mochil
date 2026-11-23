// features/settings/api/modules.ts — GET /api/modules (internal/web/modules_http.go). Read-only: the endpoint
// answers GET and 405s anything else, so switching a module on or off is a *settings document* write, not a
// module write — see api/settings.ts, which is exactly how the legacy panel does it too
// (static/modules-settings.js's `write()` merges into `target.modules` and saves the whole document).
import { z } from "zod";
import { request } from "../../../lib/http";

export const moduleSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  /** Library kinds this module owns; absent/empty for a module that isn't tied to a library kind (e.g. Hermes). */
  kinds: z.array(z.string()).nullish(),
  /** The server's own answer — `moduleOn`, so it is what actually runs, not a stored preference. */
  enabled: z.boolean(),
  default_on: z.boolean(),
  /** Set when this module is an *option* inside a card rather than a card of its own (e.g. Hardcover's extras). */
  group: z.string().optional(),
  option: z.boolean().optional(),
});

export type ModuleInfo = z.infer<typeof moduleSchema>;

export const modulesSchema = z.array(moduleSchema);

export async function fetchModules(): Promise<ModuleInfo[]> {
  return modulesSchema.parse(await request<unknown>("/api/modules"));
}

/** A module plus the options that belong inside its card — the shape the Settings screen renders. Port of the
 *  legacy `groupedModules`: an option whose parent isn't in the list falls back to a card of its own, because a
 *  switch that vanishes from Settings is a setting the user can no longer reach. */
export interface ModuleGroup {
  module: ModuleInfo;
  children: ModuleInfo[];
}

export function groupModules(modules: ModuleInfo[]): ModuleGroup[] {
  const childrenByGroup = new Map<string, ModuleInfo[]>();
  for (const module of modules) {
    if (module.option === true && module.group !== undefined && module.group !== module.id) {
      const children = childrenByGroup.get(module.group) ?? [];
      children.push(module);
      childrenByGroup.set(module.group, children);
    }
  }
  const childIds = new Set([...childrenByGroup.values()].flat().map((module) => module.id));
  const groups: ModuleGroup[] = modules
    .filter((module) => !childIds.has(module.id))
    .map((module) => ({ module, children: childrenByGroup.get(module.id) ?? [] }));
  for (const children of childrenByGroup.values()) {
    for (const module of children) {
      if (!modules.some((other) => other.id === module.group)) groups.push({ module, children: [] });
    }
  }
  return groups;
}

/** How many configured libraries a module would govern, for the "3 libraries hidden while off" line. */
export function moduleLibraryCount(module: ModuleInfo, libraries: { kind: string }[]): number {
  const kinds = module.kinds ?? [];
  return libraries.filter((library) => kinds.includes(library.kind)).length;
}
