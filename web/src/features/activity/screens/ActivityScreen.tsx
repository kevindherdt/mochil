// features/activity/screens/ActivityScreen.tsx — what the server is doing, and what it failed at.
//
// Phase 2.8 of the rewrite listed "activity" as a screen the legacy app had and `/next` did not; the legacy side is
// `activity-monitor.js` + `activity-panel-template.js`. Rather than port that DOM, this reads the one endpoint both
// sides share (`/api/jobs`) and answers three questions in order: what is running now, what finished recently, and
// what failed.
//
// Deliberately a Settings panel, not a sixth dock place: it is something a person opens when something looks stuck,
// like Diagnostics next to it, not somewhere they browse.
import { useQuery } from "@tanstack/react-query";
import { Page, Stack, Inline, Heading, Text, Card, Async } from "../../../ui";
import { fetchJobs, isActive, jobProgress, type Job } from "../api/jobs";

/** Server time strings are RFC3339; `undefined` for a job that has not started. */
function when(value: string | undefined): string {
  if (value === undefined || value === "") return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleString();
}

function QueueGroup({ queue, jobs }: { queue: string; jobs: Job[] }) {
  return (
    <Card>
      <Stack gap={12}>
        <Inline justify="space-between" align="center">
          <Text size="14">{queue}</Text>
          <Text size="11" muted>
            {jobs.length === 1 ? "1 job" : `${jobs.length} jobs`}
          </Text>
        </Inline>
        <Stack gap={8}>
          {jobs.map((job) => {
            const progress = jobProgress(job);
            return (
              <Stack key={`${job.queue}:${job.key}`} gap={4}>
                <Inline justify="space-between" align="center" gap={8}>
                  <Text size="13">{job.progress_message !== undefined && job.progress_message !== "" ? job.progress_message : job.key}</Text>
                  <Text size="11" muted>
                    {job.status}
                  </Text>
                </Inline>
                {progress !== undefined && (
                  <div
                    role="progressbar"
                    aria-valuenow={Math.round(progress * 100)}
                    aria-valuemin={0}
                    aria-valuemax={100}
                    style={{ height: 4, borderRadius: "var(--radius-pill)", background: "var(--color-line)", overflow: "hidden" }}
                  >
                    <div style={{ width: `${Math.round(progress * 100)}%`, height: "100%", background: "var(--color-accent)" }} />
                  </div>
                )}
                {job.error !== undefined && job.error !== "" && (
                  <Text size="11" tone="error">
                    {job.error}
                  </Text>
                )}
                <Text size="11" muted>
                  {job.started_at !== undefined ? `started ${when(job.started_at)}` : `queued ${when(job.queued_at)}`}
                  {job.finished_at !== undefined ? ` · finished ${when(job.finished_at)}` : ""}
                </Text>
              </Stack>
            );
          })}
        </Stack>
      </Stack>
    </Card>
  );
}

export function ActivityScreen() {
  const query = useQuery({
    queryKey: ["activity", "jobs"],
    queryFn: fetchJobs,
    // A job board is only interesting while something moves: poll fast when work is in flight, and let it settle
    // when nothing is (an idle server should not be asked every two seconds forever).
    refetchInterval: (current) => {
      const jobs = current.state.data ?? [];
      return jobs.some(isActive) ? 2000 : 30_000;
    },
  });

  return (
    <Page>
      <Stack gap={16}>
        <Heading level={1}>Activity</Heading>
        <Text size="13" muted>
          Scans, downloads, transcoding and analysis, as the server reports them. A book or episode that has not
          appeared yet is usually sitting in one of these.
        </Text>
        <Async query={query} empty={(jobs) => jobs.length === 0} emptyMessage="Nothing has run on this server yet.">
          {(jobs) => {
            const active = jobs.filter(isActive);
            const failed = jobs.filter((job) => job.status === "failed" || job.status === "error");
            const finished = jobs.filter((job) => !isActive(job) && !failed.includes(job));
            const groups = (list: Job[]) => {
              const byQueue = new Map<string, Job[]>();
              for (const job of list) {
                const bucket = byQueue.get(job.queue);
                if (bucket === undefined) byQueue.set(job.queue, [job]);
                else bucket.push(job);
              }
              return [...byQueue];
            };
            return (
              <Stack gap={24}>
                {failed.length > 0 && (
                  <Stack gap={8}>
                    <Text size="13" tone="error">
                      Failed
                    </Text>
                    {groups(failed).map(([queue, list]) => (
                      <QueueGroup key={queue} queue={queue} jobs={list} />
                    ))}
                  </Stack>
                )}
                <Stack gap={8}>
                  <Text size="13" muted>
                    Running
                  </Text>
                  {active.length === 0 ? (
                    <Text size="13" muted>
                      Idle — nothing is running right now.
                    </Text>
                  ) : (
                    groups(active).map(([queue, list]) => <QueueGroup key={queue} queue={queue} jobs={list} />)
                  )}
                </Stack>
                {finished.length > 0 && (
                  <Stack gap={8}>
                    <Text size="13" muted>
                      Recently finished
                    </Text>
                    {groups(finished.slice(-12)).map(([queue, list]) => (
                      <QueueGroup key={queue} queue={queue} jobs={list} />
                    ))}
                  </Stack>
                )}
              </Stack>
            );
          }}
        </Async>
      </Stack>
    </Page>
  );
}
