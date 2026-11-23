// features/lyana/CompanionSheet.tsx — the settings icon Kevin asked for on the AI page.
//
// "there is no settings icon for the ai page to control the voice, hands free talking ... no way to change the
// avatar, also the name lyana shouldnt be hardcoded".
//
// Three groups, in the order a person thinks about them: what it is called, what it looks like, how it speaks.
// Everything here is backed by a real endpoint (see api/companion.ts) except the name, which is a per-browser
// preference and says so plainly rather than implying it is server-wide.
import { useState } from "react";
import { Stack, Inline, Heading, Text, Button, TextField, PillRow, Switch, Card } from "../../ui";
import { messageFromError } from "../settings/api/accounts";
import { selectAvatar, setDefaultVoice } from "./api/companion";
import { setCompanionName, setHandsFree, useAvatar, useCompanionName, useHandsFree, useNarration } from "./useCompanion";

/** Folder names that say nothing about which avatar it is — this server names several models "runtime" because that
 *  is the directory they were unpacked into, and eight buttons reading "runtime" would be worse than useless. */
const GENERIC_AVATAR_NAMES = new Set(["runtime", "live2d", "model", "models", "assets", "default"]);

function avatarLabel(model: { id: string; name?: string | undefined }): string {
  const name = (model.name ?? "").trim();
  if (name !== "" && !GENERIC_AVATAR_NAMES.has(name.toLowerCase())) return name;
  const parts = model.id.split("/").filter((part) => part !== "" && !GENERIC_AVATAR_NAMES.has(part.toLowerCase()));
  const chosen = parts[parts.length - 1] ?? model.id;
  return chosen.replace(/[_-]+/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

export function CompanionSheet({ onClose }: { onClose: () => void }) {
  const name = useCompanionName();
  const handsFree = useHandsFree();
  const narration = useNarration();
  const avatar = useAvatar();

  const [draftName, setDraftName] = useState(name);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const voices = narration.data?.voices ?? [];
  const models = avatar.data?.models ?? [];
  const selectedModel = avatar.data?.selected ?? "";

  async function run(action: () => Promise<void>) {
    setSaving(true);
    setError(null);
    try {
      await action();
    } catch (caught) {
      setError(messageFromError(caught));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card>
      <Stack gap={16}>
        <Inline justify="space-between" align="center">
          <Heading size="16">Assistant settings</Heading>
          <Button variant="secondary" onClick={onClose}>
            Done
          </Button>
        </Inline>

        <Stack gap={8}>
          <TextField
            label="Name"
            value={draftName}
            placeholder="AI"
            onChange={(event) => setDraftName(event.target.value)}
            onBlur={() => setCompanionName(draftName)}
          />
          <Text size="11" muted>
            What the assistant calls itself here, and what the page says. Saved in this browser — the server has no
            assistant section to store it in yet.
          </Text>
        </Stack>

        <Stack gap={8}>
          <Text size="13">Avatar</Text>
          {models.length === 0 ? (
            <Text size="12" muted>
              No avatar models are installed on this server, so there is nothing to choose from yet.
            </Text>
          ) : (
            <PillRow
              items={models.map((model) => ({ id: model.id, label: avatarLabel(model) }))}
              activeId={selectedModel}
              onSelect={(id) => void run(async () => {
                await selectAvatar(id);
                avatar.refetch();
              })}
            />
          )}
        </Stack>

        <Stack gap={8}>
          <Text size="13">Voice</Text>
          {voices.length === 0 ? (
            <Text size="12" muted>
              Narration is not set up on this server, so there is no voice to choose.
            </Text>
          ) : (
            <PillRow
              items={voices.map((voice) => ({ id: voice.id, label: voice.name }))}
              activeId={narration.data?.voice ?? ""}
              onSelect={(id) => void run(async () => {
                await setDefaultVoice(id);
                narration.refetch();
              })}
            />
          )}
          {narration.data !== undefined && narration.data.enabled && !narration.data.available && (
            <Text size="11" muted>
              This server has voices configured but the narration runtime is not running, so reading aloud falls back to
              the device's own voice — changing the choice here will not be heard until it is.
            </Text>
          )}
          {narration.data !== undefined && !narration.data.manage && (
            <Text size="11" muted>
              Only an administrator can change the default voice on this server.
            </Text>
          )}
        </Stack>

        <Stack gap={8}>
          <Inline justify="space-between" align="center" gap={12}>
            <Stack gap={4}>
              <Text size="13">Hands-free</Text>
              <Text size="11" muted>
                Send a spoken message as soon as it is transcribed, without pressing send.
              </Text>
            </Stack>
            <Switch checked={handsFree} onChange={setHandsFree} label="Hands-free" />
          </Inline>
          {narration.data !== undefined && !narration.data.listen && (
            <Text size="11" muted>
              Speech recognition is not set up on this server, so the microphone will not appear.
            </Text>
          )}
        </Stack>

        {error !== null && (
          <Text size="13" tone="error">
            {error}
          </Text>
        )}
        {saving && (
          <Text size="11" muted>
            Saving…
          </Text>
        )}
      </Stack>
    </Card>
  );
}
