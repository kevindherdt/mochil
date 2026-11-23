// features/settings/screens/NarrationScreen.tsx — Settings → Narration, and the last panel of Phase 2.2.
//
// The runtime's own state leads the panel, because it is the answer to the question people actually arrive with:
// `GET /api/narration/status` separates Enabled (the module switch), Available (the runtime is installed) and Loaded
// (the model is in memory). A saved configuration alone would happily say "narration is configured" on a server where
// nothing can be read aloud.
//
// Editable here: offline mode, worker threads, idle shutdown, and the Hugging Face token (write-only, for the gated
// voice-cloning weights). The python/cache/voice paths are shown but not edited — they are install details with
// working server-side defaults, and a hand-typed path that is wrong fails much later, at synthesis, nowhere near this
// form.
//
// "Should the idle timer be off?" is expressed as 0 = never shut down, the server's own meaning for the field, rather
// than a second switch that would need to agree with the number underneath it.
import { useState } from "react";
import { Async, Button, Card, Heading, Inline, Page, Stack, Text, TextField, Switch } from "../../../ui";
import { useSession } from "../../auth/useSession";
import { useNarration, useNarrationStatus, useSaveNarration } from "../useNarration";
import { describeRuntime, type NarrationSettings } from "../api/narration";
import { messageFromError } from "../api/accounts";

export function NarrationScreen() {
  const session = useSession();
  return (
    <Page>
      <Stack gap={24}>
        <Heading level={1}>Narration</Heading>
        <Async query={session}>{(me) => <NarrationBody isAdministrator={me.role === "administrator"} />}</Async>
      </Stack>
    </Page>
  );
}

function NarrationBody({ isAdministrator }: { isAdministrator: boolean }) {
  const stored = useNarration(isAdministrator);
  const status = useNarrationStatus(isAdministrator);
  const saving = useSaveNarration();

  const [draft, setDraft] = useState<NarrationSettings | undefined>(undefined);
  const [newToken, setNewToken] = useState("");
  const [saved, setSaved] = useState(false);

  if (!isAdministrator) {
    return <Text muted>Only an administrator can change narration settings on this server.</Text>;
  }

  return (
    <Stack gap={24}>
      <Text muted>
        Reads books aloud with a local voice. Nothing leaves this server, apart from downloading the voice models the
        first time they are used.
      </Text>

      <Async query={status}>
        {(runtime) => (
          <Card>
            <Stack gap={4}>
              <Heading level={2} size="16">Runtime</Heading>
              <Text size="13">{describeRuntime(runtime)}</Text>
              <Text size="11" muted>
                {runtime.voices.length === 0
                  ? "No voices available yet."
                  : `${runtime.voices.length} ${runtime.voices.length === 1 ? "voice" : "voices"} available.`}
                {runtime.listen ? " Speech recognition is available too." : ""}
              </Text>
            </Stack>
          </Card>
        )}
      </Async>

      {saving.error !== null && saving.error !== undefined && (
        <Text size="13" tone="error">{messageFromError(saving.error)}</Text>
      )}
      {saved && <Text size="13">Saved.</Text>}

      <Async query={stored}>
        {(value) => {
          const current = draft ?? value;
          const patch = (next: Partial<NarrationSettings>) => {
            setSaved(false);
            saving.reset();
            setDraft({ ...current, ...next });
          };
          return (
            <Stack gap={16}>
              <Card>
                <Stack gap={12}>
                  <Inline gap={12} align="center">
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <Stack gap={4}>
                        <Text size="13">Work without network access</Text>
                        <Text size="11" muted>
                          Never fetch a voice model that isn't already downloaded. Turning this on makes a missing
                          model fail instead of downloading it.
                        </Text>
                      </Stack>
                    </div>
                    <Switch
                      checked={current.offline}
                      onChange={(next) => patch({ offline: next })}
                      label="Work without network access"
                    />
                  </Inline>
                </Stack>
              </Card>

              <Card>
                <Stack gap={12}>
                  <Heading level={2} size="16">Performance</Heading>
                  <TextField
                    label="Worker threads (0 lets the runtime decide)"
                    type="number"
                    min={0}
                    value={String(current.threads)}
                    onChange={(event) => patch({ threads: Number(event.target.value) || 0 })}
                  />
                  <TextField
                    label="Stop after this many seconds idle (0 keeps it loaded)"
                    type="number"
                    min={0}
                    value={String(current.idleSeconds)}
                    onChange={(event) => patch({ idleSeconds: Number(event.target.value) || 0 })}
                  />
                  <Text size="11" muted>
                    Keeping the voice loaded answers the first sentence instantly but holds its memory while the
                    server runs.
                  </Text>
                </Stack>
              </Card>

              <Card>
                <Stack gap={12}>
                  <Heading level={2} size="16">Hugging Face token</Heading>
                  <Text size="13" muted>
                    {current.tokenConfigured
                      ? "A token is saved. Enter a new one to replace it."
                      : "Only needed for the gated voice-cloning models."}
                  </Text>
                  <TextField
                    label="Token"
                    type="password"
                    value={newToken}
                    autoComplete="off"
                    autoCapitalize="none"
                    spellCheck={false}
                    onChange={(event) => setNewToken(event.target.value)}
                  />
                </Stack>
              </Card>

              <Card>
                <Stack gap={8}>
                  <Heading level={2} size="16">Where it runs from</Heading>
                  <Text size="11" muted>Shown for troubleshooting; change these in the server's config file.</Text>
                  <Text size="11" muted>Runtime: {current.python === "" ? "bundled" : current.python}</Text>
                  {current.cache !== "" && <Text size="11" muted>Model cache: {current.cache}</Text>}
                  <Text size="11" muted>Voices: {current.voiceDirectory}</Text>
                </Stack>
              </Card>

              <Inline gap={8}>
                <Button
                  variant="primary"
                  disabled={saving.isPending}
                  onClick={() =>
                    saving.save(
                      {
                        offline: current.offline,
                        threads: current.threads,
                        idleSeconds: current.idleSeconds,
                        newToken,
                      },
                      () => {
                        setSaved(true);
                        setNewToken("");
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
                    setNewToken("");
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
