// app/NotFoundScreen.tsx — the 404 the rewrite did not have.
//
// Phase 2.8 lists "404" among the screens the legacy app has and `/next` did not. Before this an unknown path under
// /next rendered TanStack Router's bare default ("Not Found"), with no dock, no way back and no explanation.
//
// Two deliberate choices:
//   - it renders *inside* the shell, so the dock/sidebar are still there and the person can leave in one tap;
//   - it echoes the path they asked for. A stale bookmark is the usual reason to land here (the legacy app used
//     `/#/…` hash URLs), and seeing the path is how someone recognises which bookmark it is.
import { useRouterState } from "@tanstack/react-router";
import { Page, Stack, Heading, Text } from "../ui";
import { LinkButton } from "../ui/LinkButton";

export function NotFoundScreen() {
  const path = useRouterState({ select: (state) => state.location.pathname });
  return (
    <Page>
      <Stack gap={16}>
        <Heading level={1}>Not found</Heading>
        <Text muted>Nothing lives at {path}.</Text>
        <Text size="13" muted>
          If this came from a bookmark, the old app kept its pages behind a `#` — open the library and navigate from
          there instead.
        </Text>
        <div>
          <LinkButton to="/library">Go to the library</LinkButton>
        </div>
      </Stack>
    </Page>
  );
}
