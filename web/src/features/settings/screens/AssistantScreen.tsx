// features/settings/screens/AssistantScreen.tsx — where the assistant's endpoint is set.
//
// Mochil's chat needs one thing to exist: an address to talk to. That address was only settable in `config.json`, which
// is fine for the person who wrote the server and unreasonable for everyone else — most people will point this at
// something quite different from what the author uses, or not use it at all.
//
// Nothing here assumes Hermes. A Hermes gateway gives the chat, the conversation list, memory and library cards; any
// OpenAI-compatible endpoint gives the chat. The screen says so, and says which of the two the address looks like.
import { useEffect, useState } from "react";
import { Page, Stack, Heading, Text, TextField, Button, Inline } from "../../../ui";
import { fetchAssistantSettings, saveAssistantSettings, testAssistant, type AssistantSettings } from "../api/assistant";

type Test = { ok: boolean; message: string } | null;

export function AssistantScreen() {
  const [settings, setSettings] = useState<AssistantSettings | null>(null);
  const [url, setUrl] = useState("");
  const [model, setModel] = useState("");
  const [name, setName] = useState("");
  const [key, setKey] = useState("");
  const [saving, setSaving] = useState(false);
  const [result, setResult] = useState<Test>(null);
  const [problem, setProblem] = useState("");

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        const loaded = await fetchAssistantSettings();
        if (cancelled) return;
        setSettings(loaded);
        setUrl(loaded.url);
        setModel(loaded.model);
        setName(loaded.name);
      } catch (error) {
        if (!cancelled) setProblem(error instanceof Error ? error.message : "Settings could not be read.");
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  const save = async () => {
    if (settings === null) return;
    setSaving(true);
    setProblem("");
    setResult(null);
    try {
      await saveAssistantSettings(settings, { url, model, name, key });
      const reloaded = await fetchAssistantSettings();
      setSettings(reloaded);
      setKey("");
      setResult({ ok: true, message: "Saved. The chat uses this endpoint from now on." });
    } catch (error) {
      setProblem(error instanceof Error ? error.message : "Could not save.");
    } finally {
      setSaving(false);
    }
  };

  const test = async () => {
    setResult(null);
    const answer = await testAssistant();
    setResult({ ok: answer.ok, message: answer.message });
  };

  const looksLikeHermes = /\/p\/[^/]+/.test(url);

  return (
    <Page>
      <Stack gap={24}>
        <Stack gap={4}>
          <Heading level={1}>Assistant</Heading>
          <Text size="13" muted>
            The chat in Mochil talks to a model endpoint you choose. Leave this empty and Mochil works exactly the same
            without it — the library, the reader and the editor never use it.
          </Text>
        </Stack>

        {settings === null && problem === "" ? (
          <Text size="13" muted>
            Reading the current settings…
          </Text>
        ) : (
          <Stack gap={16}>
            <TextField
              label="Address"
              value={url}
              onChange={(event) => setUrl(event.target.value)}
              placeholder="http://your-server:8642/v1"
              autoComplete="off"
              spellCheck={false}
            />
            <TextField
              label="Model"
              value={model}
              onChange={(event) => setModel(event.target.value)}
              placeholder="whatever the endpoint calls its model"
              autoComplete="off"
              spellCheck={false}
            />
            <TextField
              label="API key"
              type="password"
              value={key}
              onChange={(event) => setKey(event.target.value)}
              placeholder={settings?.keyConfigured === true ? "A key is saved — leave this empty to keep it" : "Paste the key for that endpoint"}
              autoComplete="off"
              spellCheck={false}
            />
            <TextField
              label="Name (optional)"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="What the chat calls the assistant"
              autoComplete="off"
            />

            <Text size="12" muted>
              {looksLikeHermes
                ? "This looks like a Hermes profile address: chat, conversation history, memory and library cards."
                : "A plain OpenAI-compatible address gives the chat and its replies. Conversation history and library cards need a Hermes gateway."}
            </Text>

            <Inline gap={8} wrap>
              <Button onClick={() => void save()} disabled={saving}>
                {saving ? "Saving…" : "Save"}
              </Button>
              <Button variant="secondary" onClick={() => void test()} disabled={settings === null}>
                Test connection
              </Button>
            </Inline>

            {result !== null &&
              (result.ok ? (
                <Text size="13">{result.message}</Text>
              ) : (
                <Text size="13" tone="error">
                  {result.message}
                </Text>
              ))}
            {problem !== "" && (
              <Text size="13" tone="error">
                {problem}
              </Text>
            )}
          </Stack>
        )}
      </Stack>
    </Page>
  );
}
