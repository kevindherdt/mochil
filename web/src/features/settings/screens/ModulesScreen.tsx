// features/settings/screens/ModulesScreen.tsx — Settings → Modules (plan 2.2, first panel). Ports the legacy
// panel's semantics rather than its markup (static/modules-settings.js):
//   * a module's state is the server's `enabled` (what actually runs), never a client guess;
//   * a module can be one card with options inside it (`group`/`option`), and turning the card off leaves its
//     options' saved choices alone — so their switches show what runs (off) and stay disabled, saying so in their
//     accessible name instead of pretending the choice was lost;
//   * a switch is a settings-document write, and a switched-off module keeps running until the process restarts —
//     hence the restart card, shown only once something has actually been moved.
//
// No layout CSS here (law L2): the screen composes ui/ primitives only; the row shape is inline tokens, the same
// way the other screens that predate a dedicated layout primitive do it.
import { useState } from "react";
import { Page, Stack, Inline, Heading, Text, Card, Button, Switch, Skeleton, ErrorState, EmptyState } from "../../../ui";
import { useModules, useSetModule } from "../useModules";
import { groupModules, moduleLibraryCount, type ModuleInfo } from "../api/modules";
import { restartServer } from "../api/settings";
import { useLibraries } from "../../library/useLibraries";

export function ModulesScreen() {
  const modulesQuery = useModules();
  const librariesQuery = useLibraries();
  // Local echo for switches the user has moved: the list is re-fetched after each save, and until that lands the
  // switch must show where the finger left it rather than snapping back to the old server value.
  const [moved, setMoved] = useState<Record<string, boolean>>({});
  const modules = modulesQuery.data ?? [];
  const switcher = useSetModule(modules);
  const libraries = librariesQuery.data ?? [];

  const switchFor = (module: ModuleInfo) => moved[module.id] ?? module.enabled;

  function toggle(module: ModuleInfo, enabled: boolean) {
    setMoved((previous) => ({ ...previous, [module.id]: enabled }));
    switcher.toggle(module.id, enabled);
  }

  const restartNeeded = Object.keys(moved).length > 0;

  return (
    <Page>
      <Stack gap={24}>
        <Stack gap={8}>
          <Heading level={1}>Modules</Heading>
          <Text muted>
            Turn optional features on or off. A module that is off does not run at all.
          </Text>
        </Stack>

        {modulesQuery.isLoading && <Skeleton />}
        {modulesQuery.isError && <ErrorState onRetry={modulesQuery.refetch} />}
        {modulesQuery.data !== undefined && modules.length === 0 && (
          <EmptyState message="This server reports no modules." />
        )}

        {groupModules(modules).map(({ module, children }) => {
          const enabled = switchFor(module);
          const count = moduleLibraryCount(module, libraries);
          return (
            <Card key={module.id}>
              <Stack gap={12}>
                <Inline gap={12} align="center">
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <Stack gap={4}>
                      <Heading level={2} size="16">
                        {module.name}
                        <Text as="span" size="11" muted>{enabled ? " · Running" : " · Off"}</Text>
                      </Heading>
                      <Text size="13" muted>{module.description}</Text>
                      {count > 0 && (
                        <Text size="11" muted>
                          {count === 1 ? "1 library" : `${count} libraries`}
                          {enabled ? "" : " hidden while off"}
                        </Text>
                      )}
                    </Stack>
                  </div>
                  <Switch
                    checked={enabled}
                    onChange={(next) => toggle(module, next)}
                    label={module.name}
                    disabled={switcher.pending.has(module.id) || switcher.isSaving}
                  />
                </Inline>

                {children.length > 0 && (
                  <Stack gap={8}>
                    {children.map((child) => {
                      const childEnabled = switchFor(child);
                      // The card being off is what disables an option, not the option's own saved value.
                      const inactive = !enabled;
                      return (
                        <Inline key={child.id} gap={12} align="center">
                          <div style={{ flex: 1, minWidth: 0 }}>
                            <Stack gap={4}>
                              <Text size="13">{child.name}</Text>
                              <Text size="11" muted>{child.description}</Text>
                            </Stack>
                          </div>
                          <Switch
                            checked={childEnabled && !inactive}
                            onChange={(next) => toggle(child, next)}
                            label={
                              inactive && childEnabled
                                ? `${child.name}, saved but off until its module is on`
                                : child.name
                            }
                            disabled={inactive || switcher.pending.has(child.id) || switcher.isSaving}
                          />
                        </Inline>
                      );
                    })}
                  </Stack>
                )}
              </Stack>
            </Card>
          );
        })}

        {switcher.error !== null && switcher.error !== undefined && (
          <Text size="13" tone="error">That switch could not be saved. Reload and try again.</Text>
        )}

        {restartNeeded && (
          <Card>
            <Stack gap={12}>
              <Text size="13">
                A module that was switched off is still loaded until Mochil restarts. Restart now to apply the
                change — anything open in another tab will reconnect when it comes back.
              </Text>
              <Inline gap={8}>
                <Button variant="secondary" onClick={() => void restartServer()}>Restart Mochil</Button>
              </Inline>
            </Stack>
          </Card>
        )}

        <Text size="11" muted>
          Books, comics and manga, reading sync and accounts are always on. Turning a module off hides its libraries
          and stops its background work. Nothing is deleted.
        </Text>
      </Stack>
    </Page>
  );
}
