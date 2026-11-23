// features/settings/screens/MetadataProvidersScreen.tsx — Settings → Metadata: the two catalogue tokens.
//
// Tokens are write-only in both directions: the panel never shows one, only whether one is saved, and it only sends
// a token that was typed now (api/secrets.ts). There is no remove control because the server has no way to clear a
// saved token — a save without the field keeps it — and offering one would be a button that silently does nothing.
//
// Which provider a library uses is not chosen here: each library derives it from its kind. The panel says that
// plainly, because "I added a token and nothing changed" is otherwise the obvious next question — a token only does
// something once a library of the matching kind exists (publications for Hardcover, video for TMDB).
import { useState } from "react";
import { Async, Button, Card, Heading, Inline, Page, Stack, Text, TextField } from "../../../ui";
import { useSession } from "../../auth/useSession";
import { useProviders, useSaveProviderTokens } from "../useSync";
import { messageFromError } from "../api/accounts";

export function MetadataProvidersScreen() {
  const session = useSession();
  return (
    <Page>
      <Stack gap={24}>
        <Heading level={1}>Metadata</Heading>
        <Async query={session}>{(me) => <ProvidersBody isAdministrator={me.role === "administrator"} />}</Async>
      </Stack>
    </Page>
  );
}

function ProvidersBody({ isAdministrator }: { isAdministrator: boolean }) {
  const providers = useProviders(isAdministrator);
  const saving = useSaveProviderTokens();
  const [hardcover, setHardcover] = useState("");
  const [tmdb, setTmdb] = useState("");
  const [saved, setSaved] = useState(false);

  if (!isAdministrator) {
    return <Text muted>Only an administrator can change metadata providers on this server.</Text>;
  }

  return (
    <Stack gap={24}>
      <Text muted>
        Tokens let Mochil fetch covers, descriptions and cast from online catalogues. They are stored on this server
        and never sent back to a browser.
      </Text>

      {saving.error !== null && saving.error !== undefined && (
        <Text size="13" tone="error">{messageFromError(saving.error)}</Text>
      )}
      {saved && <Text size="13">Saved.</Text>}

      <Async query={providers}>
        {(value) => (
          <Stack gap={16}>
            <Card>
              <Stack gap={12}>
                <Heading level={2} size="16">Hardcover</Heading>
                <Text size="13" muted>
                  {value.hardcoverConfigured
                    ? "A token is saved. Enter a new one to replace it."
                    : "No token yet. Books use Open Library until one is set."}
                </Text>
                <TextField
                  label="Hardcover token"
                  type="password"
                  value={hardcover}
                  autoComplete="off"
                  autoCapitalize="none"
                  spellCheck={false}
                  onChange={(event) => setHardcover(event.target.value)}
                />
              </Stack>
            </Card>

            <Card>
              <Stack gap={12}>
                <Heading level={2} size="16">TMDB</Heading>
                <Text size="13" muted>
                  {value.tmdbConfigured
                    ? "A token is saved. Enter a new one to replace it."
                    : "No token yet. Films, TV and anime have no online catalogue until one is set."}
                </Text>
                <TextField
                  label="TMDB token"
                  type="password"
                  value={tmdb}
                  autoComplete="off"
                  autoCapitalize="none"
                  spellCheck={false}
                  onChange={(event) => setTmdb(event.target.value)}
                />
              </Stack>
            </Card>

            <Text size="11" muted>
              Each library picks its provider from its own kind — Hardcover for books, comics and manga, TMDB for
              films, TV and anime — so a token only takes effect once a library of that kind exists. A saved token
              cannot be cleared here; replace it instead.
            </Text>

            <Inline gap={8}>
              <Button
                variant="primary"
                disabled={saving.isPending || (hardcover.trim() === "" && tmdb.trim() === "")}
                onClick={() =>
                  saving.save({ hardcover, tmdb }, () => {
                    setSaved(true);
                    setHardcover("");
                    setTmdb("");
                  })
                }
              >
                Save
              </Button>
              <Button
                variant="secondary"
                disabled={saving.isPending}
                onClick={() => {
                  setHardcover("");
                  setTmdb("");
                  setSaved(false);
                  saving.reset();
                }}
              >
                Clear the fields
              </Button>
            </Inline>
          </Stack>
        )}
      </Async>
    </Stack>
  );
}
