# Web navigation bucket consistency

Status: web/server published at 17:43 UTC; native TV delivery remains pending.
See `NAVIGATION_SCAN_DEPLOYMENT_2026-09-08.md` for deployment evidence.

Impeccable hardening preserved the existing interface and scoped this change
to navigation-label correctness. Music and the media section navigator now use
one focused `catalog-title-bucket.js` helper, checked against the same JSON
contract as Go and Android TV.

The old web code stripped any whitespace after an article; the server strips a
literal space before trimming the remainder. It also allowed multi-character
uppercase expansions through its lexical A–Z check (for example ß -> SS).
The helper now follows the shared cases and accepts only one ASCII bucket letter.

Seventeen shared cases pass across Go, web and Android. All 564 frontend tests,
both full Android debug suites and module gate passed. Logs:
`/tmp/bobobooks-web-bucket-contract.log` and
`/tmp/bobobooks-crossplatform-bucket-tests.log`.

No layout, colour, control placement or scroll geometry was changed. This is
not a rendered visual acceptance result; physical TV polish remains pending.
No new APK was required for the web/server deployment.

## Browser interaction verification

`npm run test:browser:navigation` passed at 1440px and 390px using the actual
module graph and synthetic 600px rows. It checks module loading, letter jumps,
one active aria-pressed marker, normal scroll-driven selection, unavailable
letters, navigator hiding and uncaught page errors. The fixture explicitly
removes heading margins so scroll coordinates do not depend on browser defaults.
This verifies behaviour, not the full production layout's visual quality.
