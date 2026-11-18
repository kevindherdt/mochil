# Wiki folder refresh evidence

Pending the next batch after published 0.16.78; no APK or deployment for this fix.

The repository previously treated any successful tree response after folder creation as completion, even when the folder was absent. Page creation already required a matching indexed page. Folder creation now requires a folder entry with the exact requested path before reporting a complete result.

A missing folder enters the existing acknowledged-write/failed-refresh recovery barrier. Repeated creation is blocked before network IO while stale trees persist; a page with the same path does not clear that barrier. This does not repeat or roll back the completed filesystem operation.

Regression coverage checks successful indexed folder creation, refresh cancellation, missing-folder acknowledgement, repeated request prevention, and wrong-kind tree evidence. Both full Android debug unit suites and module checks passed. Existing UI and source media are unchanged. Physical-device acceptance remains outstanding.

## Move and trash consistency

The same audit found trash accepted a refreshed tree still containing the removed page, while move checked only the destination. Trash now requires absence of the original ID; move requires both destination presence and original-ID absence before fetching the new page. Same-path moves are rejected by the server writer, so they do not require a no-op exception here.

Tests cover stale-tree acknowledgement, duplicate-write barriers, blocked saves after trash acknowledgement, and successful move/trash refresh outcomes. Both complete Android debug suites and module checks passed after these additions (`/tmp/bobobooks-wiki-structure-evidence.log`, 13 seconds). All changes remain pending the next batch; no APK or live configuration change.
