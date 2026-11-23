// features/about/screens/AboutScreen.tsx — Settings → About (plan: proves auth, theme, shell and data end to
// end). No layout rules here (law L2 — no CSS module import in a screen); composes ui/ primitives only.
import { Page, Stack, Heading, Text, Card, Async } from "../../../ui";
import { useVersion } from "../useVersion";
import { useSession } from "../../auth/useSession";

export function AboutScreen() {
  const version = useVersion();
  const session = useSession();

  return (
    <Page>
      <Stack gap={24}>
        <Heading level={1}>About</Heading>

        <Card>
          <Async query={version} skeleton={<Text muted>Checking build…</Text>}>
            {(info) => (
              <Stack gap={8}>
                <Text as="span">
                  {info.product} {info.version}
                  {info.commit ? ` (${info.commit})` : ""}
                </Text>
                <Text size="12" muted>
                  Started {info.started} · up {info.uptime}
                  {info.go ? ` · ${info.go}` : ""}
                </Text>
              </Stack>
            )}
          </Async>
        </Card>

        <Card>
          <Async query={session} skeleton={<Text muted>Checking session…</Text>}>
            {(account) => (
              <Text as="span">
                Signed in as <strong>{account.username}</strong> ({account.role})
              </Text>
            )}
          </Async>
        </Card>
      </Stack>
    </Page>
  );
}
