// features/settings/screens/PlaybackScreen.tsx — Settings → Playback: the server-wide video defaults.
//
// Both halves are described as what they are, because they are different kinds of setting:
//   * quality is a *ceiling* the player must stay under (GET /api/playback-policy is what a player reads), and the
//     panel shows that endpoint's answer next to the fields so a save can be checked rather than trusted;
//   * languages are only the *starting* track. A choice made per show wins (internal/video's PreferenceStore), so
//     the text says "default", not "always".
import { useState } from "react";
import { Async, Button, Card, Heading, Inline, Page, PillRow, Stack, Text, TextField } from "../../../ui";
import { useSession } from "../../auth/useSession";
import { usePlayback, usePlaybackPolicy, useSavePlayback } from "../usePlayback";
import { PLAYBACK_QUALITIES, type PlaybackSettings } from "../api/playback";
import { messageFromError } from "../api/accounts";

export function PlaybackScreen() {
  const session = useSession();
  return (
    <Page>
      <Stack gap={24}>
        <Heading level={1}>Playback</Heading>
        <Async query={session}>{(me) => <PlaybackBody isAdministrator={me.role === "administrator"} />}</Async>
      </Stack>
    </Page>
  );
}

function PlaybackBody({ isAdministrator }: { isAdministrator: boolean }) {
  const stored = usePlayback();
  const policy = usePlaybackPolicy();
  const saving = useSavePlayback();

  // `undefined` means "follow the server"; the moment anything is edited the draft takes over, and Discard puts it
  // back to following. No effect is needed for that — the value is derived, not synchronised.
  const [draft, setDraft] = useState<PlaybackSettings | undefined>(undefined);
  const [saved, setSaved] = useState(false);

  if (!isAdministrator) {
    return <Text muted>Only an administrator can change playback settings on this server.</Text>;
  }

  return (
    <Stack gap={24}>
      <Text muted>
        These apply to video played from this server. Quality is a ceiling a player stays under; languages choose
        which track a film starts on.
      </Text>

      {saving.error !== null && saving.error !== undefined && (
        <Text size="13" tone="error">{messageFromError(saving.error)}</Text>
      )}
      {saved && <Text size="13">Saved.</Text>}

      <Async query={stored}>
        {(value) => {
          const current = draft ?? value;
          // The draft starts from the *server's* values, never from the defaults: seeding it with
          // PLAYBACK_DEFAULTS meant the first edit silently dropped every field it didn't carry — moving the
          // mobile-quality pill wiped a stored `subtitle_language: "english"` back to "". Caught by saving one field
          // and diffing the document.
          const patch = (next: Partial<PlaybackSettings>) => {
            setSaved(false);
            saving.reset();
            setDraft({ ...current, ...next });
          };
          return (
            <Stack gap={16}>
              <Card>
                <Stack gap={12}>
                  <Heading level={2} size="16">Quality</Heading>
                  <Stack gap={8}>
                    <Text size="13" muted>On Wi-Fi</Text>
                    <PillRow
                      items={PLAYBACK_QUALITIES}
                      activeId={current.wifiQuality}
                      onSelect={(id) => patch({ wifiQuality: id as PlaybackSettings["wifiQuality"] })}
                    />
                  </Stack>
                  <Stack gap={8}>
                    <Text size="13" muted>On mobile data</Text>
                    <PillRow
                      items={PLAYBACK_QUALITIES}
                      activeId={current.mobileQuality}
                      onSelect={(id) => patch({ mobileQuality: id as PlaybackSettings["mobileQuality"] })}
                    />
                  </Stack>
                  <Async query={policy}>
                    {(answer) => (
                      <Text size="11" muted>
                        Players are currently told: Wi-Fi {answer.wifiQuality}, mobile {answer.mobileQuality}.
                      </Text>
                    )}
                  </Async>
                </Stack>
              </Card>

              <Card>
                <Stack gap={12}>
                  <Heading level={2} size="16">Languages</Heading>
                  <TextField
                    label="Audio language"
                    value={current.audioLanguage}
                    placeholder="en"
                    autoComplete="off"
                    autoCapitalize="none"
                    spellCheck={false}
                    onChange={(event) => patch({ audioLanguage: event.target.value })}
                  />
                  <TextField
                    label="Subtitle language"
                    value={current.subtitleLanguage}
                    placeholder="en"
                    autoComplete="off"
                    autoCapitalize="none"
                    spellCheck={false}
                    onChange={(event) => patch({ subtitleLanguage: event.target.value })}
                  />
                  <Text size="11" muted>
                    A language name or code, for example english or en. Subtitles accept off to start with none.
                    Leave empty to let the player choose. A language picked on a particular show overrides these.
                  </Text>
                </Stack>
              </Card>

              <Inline gap={8}>
                <Button
                  variant="primary"
                  disabled={saving.isPending}
                  onClick={() =>
                    saving.save(current, () => {
                      setSaved(true);
                      // Drop the local draft so the form shows what the server stored, not what was typed.
                      setDraft(undefined);
                    })
                  }
                >
                  Save
                </Button>
                <Button
                  variant="secondary"
                  disabled={saving.isPending}
                  onClick={() => {
                    setDraft(value);
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
