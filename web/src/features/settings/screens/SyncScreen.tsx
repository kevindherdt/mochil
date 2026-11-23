// features/settings/screens/SyncScreen.tsx — Settings → Reading sync.
//
// The section has two flags that both matter (`enabled` and `provider`), and the panel keeps them visible rather than
// inventing a single "on" that hides which server your devices are actually talking to. Password is write-only: the
// panel shows whether one is saved and only sends a value when a new one is typed (api/secrets.ts says why).
//
// The server's own validation does the rest, and its sentences are better than anything here — "kosync.username and
// kosync.password must be configured together", "kosync.server_url must be an absolute HTTP or HTTPS URL…" — so a
// failed save shows the server's message verbatim.
import { useState } from "react";
import { Async, Button, Card, Heading, Inline, Page, PillRow, Stack, Text, TextField, Switch } from "../../../ui";
import { useSession } from "../../auth/useSession";
import { useSaveSync, useSync } from "../useSync";
import { SYNC_PROVIDERS, type SyncProvider, type SyncSettings } from "../api/sync";
import { messageFromError } from "../api/accounts";

export function SyncScreen() {
  const session = useSession();
  return (
    <Page>
      <Stack gap={24}>
        <Heading level={1}>Reading sync</Heading>
        <Async query={session}>{(me) => <SyncBody isAdministrator={me.role === "administrator"} />}</Async>
      </Stack>
    </Page>
  );
}

function SyncBody({ isAdministrator }: { isAdministrator: boolean }) {
  const stored = useSync(isAdministrator);
  const saving = useSaveSync();

  const [draft, setDraft] = useState<SyncSettings | undefined>(undefined);
  const [newPassword, setNewPassword] = useState("");
  const [saved, setSaved] = useState(false);

  if (!isAdministrator) {
    return <Text muted>Only an administrator can change reading sync on this server.</Text>;
  }

  return (
    <Stack gap={24}>
      <Text muted>
        Keeps reading position in step between KOReader on your devices and this server.
      </Text>

      {saving.error !== null && saving.error !== undefined && (
        <Text size="13" tone="error">{messageFromError(saving.error)}</Text>
      )}
      {saved && <Text size="13">Saved.</Text>}

      <Async query={stored}>
        {(value) => {
          const current = draft ?? value;
          const external = current.provider === "external";
          // There is no separate "enabled" switch in the UI: choosing Off *is* turning it off, and two switches for
          // one decision is exactly the duplication this app keeps removing elsewhere.
          const patch = (next: Partial<SyncSettings>) => {
            setSaved(false);
            saving.reset();
            setDraft({ ...current, ...next });
          };
          return (
            <Stack gap={16}>
              <Card>
                <Stack gap={12}>
                  <Heading level={2} size="16">Where progress lives</Heading>
                  <PillRow
                    items={SYNC_PROVIDERS}
                    activeId={current.provider}
                    onSelect={(id) => patch({ provider: id as SyncProvider, enabled: id !== "disabled" })}
                  />
                  <Text size="11" muted>
                    This server keeps its own sync service, so devices can point here. Choose another server if you
                    already run one; Off stops syncing entirely.
                  </Text>
                  {current.effectiveUsername !== "" && current.provider !== "external" && (
                    <Text size="11" muted>Devices sign in as {current.effectiveUsername}.</Text>
                  )}
                </Stack>
              </Card>

              {external && (
                <Card>
                  <Stack gap={12}>
                    <Heading level={2} size="16">The other server</Heading>
                    <TextField
                      label="Server URL"
                      value={current.serverUrl}
                      placeholder="https://sync.example.com"
                      autoComplete="off"
                      autoCapitalize="none"
                      spellCheck={false}
                      onChange={(event) => patch({ serverUrl: event.target.value })}
                    />
                    <TextField
                      label="Username"
                      value={current.username}
                      autoComplete="off"
                      autoCapitalize="none"
                      spellCheck={false}
                      onChange={(event) => patch({ username: event.target.value })}
                    />
                    <TextField
                      label={current.passwordConfigured ? "Password (saved — type a new one to replace it)" : "Password"}
                      type="password"
                      value={newPassword}
                      autoComplete="new-password"
                      onChange={(event) => setNewPassword(event.target.value)}
                    />
                    <Text size="11" muted>
                      A username and password must be set together; leaving the password blank keeps the saved one.
                    </Text>
                  </Stack>
                </Card>
              )}

              <Card>
                <Stack gap={12}>
                  <Heading level={2} size="16">This server</Heading>
                  <Inline gap={12} align="center">
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <Stack gap={4}>
                        <Text size="13">Allow new devices to register</Text>
                        <Text size="11" muted>
                          Off means an account has to exist here before a device can sync.
                        </Text>
                      </Stack>
                    </div>
                    <Switch
                      checked={current.allowRegistration}
                      onChange={(next) => patch({ allowRegistration: next })}
                      label="Allow new devices to register"
                    />
                  </Inline>
                  <Inline gap={12} align="center">
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <Stack gap={4}>
                        <Text size="13">Follow progress from the web reader</Text>
                        <Text size="11" muted>
                          Reading a book in this browser updates the same position your devices sync.
                        </Text>
                      </Stack>
                    </div>
                    <Switch
                      checked={current.webReaderBridge}
                      onChange={(next) => patch({ webReaderBridge: next })}
                      label="Follow progress from the web reader"
                    />
                  </Inline>
                  {current.database !== "" && <Text size="11" muted>Sync data: {current.database}</Text>}
                </Stack>
              </Card>

              <Inline gap={8}>
                <Button
                  variant="primary"
                  disabled={saving.isPending}
                  onClick={() =>
                    saving.save(
                      {
                        enabled: current.provider !== "disabled",
                        provider: current.provider,
                        serverUrl: current.serverUrl,
                        username: current.username,
                        allowRegistration: current.allowRegistration,
                        webReaderBridge: current.webReaderBridge,
                        newPassword,
                      },
                      () => {
                        setSaved(true);
                        setNewPassword("");
                        setDraft(undefined);
                      },
                    )
                  }
                >
                  Save
                </Button>
                <Button
                  variant="secondary"
                  disabled={saving.isPending}
                  onClick={() => {
                    setDraft(value);
                    setNewPassword("");
                    setSaved(false);
                    saving.reset();
                  }}
                >
                  Discard changes
                </Button>
              </Inline>
            </Stack>
          );
        }}
      </Async>
    </Stack>
  );
}
