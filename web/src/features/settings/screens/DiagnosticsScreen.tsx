// features/settings/screens/DiagnosticsScreen.tsx — Settings → Diagnostics, the administrator's read-only view.
//
// The server keeps this snapshot itself (internal/web/diagnostics.go): counts, memory, goroutines, the slowest recent
// requests, the recent failures, and any background task that has been failing (internal/safely records those per
// task with a restart count). Nothing here changes anything — the endpoint is GET-only — so the panel's job is to
// present it honestly: a failing background task and a slow request are different problems and are shown separately
// rather than averaged into a single "health" number nothing can act on.
import { Async, Button, Card, Heading, Inline, Page, Stack, Text } from "../../../ui";
import { useSession } from "../../auth/useSession";
import { useDiagnostics } from "../usePlayback";
import { formatUptime, type DiagnosticsRecord } from "../api/diagnostics";

export function DiagnosticsScreen() {
  const session = useSession();
  return (
    <Page>
      <Stack gap={24}>
        <Heading level={1}>Diagnostics</Heading>
        <Async query={session}>{(me) => <DiagnosticsBody isAdministrator={me.role === "administrator"} />}</Async>
      </Stack>
    </Page>
  );
}

function DiagnosticsBody({ isAdministrator }: { isAdministrator: boolean }) {
  const diagnostics = useDiagnostics(isAdministrator);

  if (!isAdministrator) {
    return <Text muted>Only an administrator can see this server's diagnostics.</Text>;
  }

  return (
    <Stack gap={24}>
      <Inline gap={8}>
        <Button variant="secondary" onClick={diagnostics.refetch}>Refresh</Button>
        <Text size="11" muted>Counts are since the server started; nothing here is cached.</Text>
      </Inline>

      <Async query={diagnostics}>
        {(data) => (
          <Stack gap={16}>
            <Card>
              <Stack gap={8}>
                <Heading level={2} size="16">Server</Heading>
                <Fact label="Up" value={formatUptime(data.uptime_seconds)} />
                <Fact label="Requests" value={data.requests.toLocaleString()} />
                <Fact label="Failed requests" value={data.failures.toLocaleString()} />
                <Fact label="Libraries" value={String(data.libraries)} />
                <Fact label="Memory" value={`${data.heap_mb.toFixed(0)} MB heap · ${data.system_mb.toFixed(0)} MB reserved`} />
                <Fact label="Goroutines" value={`${data.goroutines} · ${data.gc_cycles} GC cycles`} />
              </Stack>
            </Card>

            <Card>
              <Stack gap={8}>
                <Heading level={2} size="16">Background tasks</Heading>
                {data.background.length === 0 ? (
                  <Text size="13" muted>Nothing has failed in the background.</Text>
                ) : (
                  data.background.map((failure) => (
                    <Stack key={failure.what} gap={4}>
                      <Text size="13">{failure.what}</Text>
                      <Text size="11" muted>
                        {failure.count} failure{failure.count === 1 ? "" : "s"}
                        {failure.restarts > 0 ? ` · restarted ${failure.restarts}×` : ""} · {new Date(failure.at).toLocaleString()}
                      </Text>
                      <Text size="11" tone="error">{failure.message}</Text>
                    </Stack>
                  ))
                )}
              </Stack>
            </Card>

            <SlowestCard title="Slowest recent requests" records={data.slow} empty="No request took longer than 0.7 s." />
            <SlowestCard title="Recent failures" records={data.errors} empty="No request failed." />
          </Stack>
        )}
      </Async>
    </Stack>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <Inline gap={8} align="center">
      <div style={{ flex: 1, minWidth: 0 }}>
        <Text size="13" muted>{label}</Text>
      </div>
      <Text size="13">{value}</Text>
    </Inline>
  );
}

function SlowestCard({ title, records, empty }: { title: string; records: DiagnosticsRecord[]; empty: string }) {
  return (
    <Card>
      <Stack gap={8}>
        <Heading level={2} size="16">{title}</Heading>
        {records.length === 0 ? (
          <Text size="13" muted>{empty}</Text>
        ) : (
          records.map((record) => (
            <Inline key={`${record.at}-${record.path}`} gap={8} align="center">
              <div style={{ flex: 1, minWidth: 0 }}>
                <Text size="12" as="span">
                  {record.path}
                  <Text as="span" size="11" muted>{` · ${record.method} ${record.status}`}</Text>
                </Text>
              </div>
              <Text size="11" muted>{record.duration_ms} ms</Text>
            </Inline>
          ))
        )}
      </Stack>
    </Card>
  );
}
