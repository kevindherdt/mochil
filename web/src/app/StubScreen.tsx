// A placeholder screen for a Phase 2 route that doesn't exist yet — says so plainly instead of pretending.
import type { ReactNode } from "react";
import { Page, Stack, Heading, Text } from "../ui";

export function StubScreen({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <Page>
      <Stack gap={8}>
        <Heading level={1}>{title}</Heading>
        <Text muted>Not built yet — Phase 2 (docs/rewrite-progress.md).</Text>
        {children}
      </Stack>
    </Page>
  );
}
