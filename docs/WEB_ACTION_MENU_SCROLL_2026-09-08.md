# Web action menus: nested scroll hardening

Status: published with 0.16.86 at 18:13 UTC. See `ANDROID_0_16_86_RELEASE_VERIFICATION.md`.

Impeccable hardening scoped the work to behaviour and lifecycle, preserving the
approved layout, themes and mobile menu presentation.

## Findings and changes

- Window's non-capturing scroll listener did not observe element scroll events.
  Fixed-position desktop menus could remain at their old coordinates while a
  nested content panel moved the invoking item. Capture now observes those events.
- Appending `[open]` to a comma-separated selector restricted only the last
  selector. Closed music/book menus were selected during scroll refreshes even
  though `position` rejected them later. `:is(...)[open]` restricts every type.
- Opening and scrolling now share one animation-frame scheduler. The installer
  returns a cleanup function that cancels scheduled work, removes listeners and
  clears owner/menu positioning classes when disposed.

No claim is made about overall frame-rate improvement. The correction avoids
unnecessary closed-menu processing and handles nested scroll geometry correctly.

## Verification

`npm run test:browser:action-menus` uses the actual module and positioning CSS
with isolated nested-scroll geometry at 1440px and 390px. Desktop checks confirm
the menu follows a 120px content scroll and refresh queries contain only open
menus; mobile checks preserve the non-desktop positioning path. Cleanup and
uncaught browser errors are checked at both widths. This is an interaction test,
not a complete visual or mobile-device acceptance review.

All 564 frontend tests and the module-size gate passed.
Frontend log: `/tmp/bobobooks-action-menu-tests.log`.
