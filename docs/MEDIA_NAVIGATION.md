# Media navigation direction

Mochil should not add permanent movie, television, podcast, or music buttons
before those libraries exist. Navigation grows from configured content rather
than advertising empty features.

## Selection model

```text
Dashboard
Library
  ├─ Books        media family (shown only when configured)
  │   ├─ Light Novels
  │   ├─ Manga
  │   └─ Comics   shelves
  ├─ Movies
  ├─ TV
  └─ Audio
      ├─ Music
      └─ Podcasts
```

- The masthead remains unchanged: Dashboard, Library, Settings, and theme.
- When only one media family exists, no media-family control is shown.
- With two or more families, one compact horizontal text switcher appears at
  the start of the Library view: `Books  Movies  TV  Audio`.
- The existing shelf chips remain directly below it and show only shelves in
  the active family.
- Empty families are omitted. The selector never contains disabled promises.
- Phones use the same single-line switcher with horizontal scrolling; it does
  not become a drawer, sidebar, or floating bar.
- Search defaults to the active family. A later explicit “Everywhere” scope can
  search across media without merging incompatible browse layouts.

## Entering Library

Clicking Library should not mean `All shelves`, and should not start a full
catalogue request. Restore the last active media family and shelf when that
choice is still valid. For a new browser, show a lightweight family/shelf
chooser and wait for a selection before requesting items or artwork. If only
one family and one shelf exist, entering that shelf directly remains the
quietest behavior.

`All shelves` is an explicit option inside the active media family, never an
implicit cross-media aggregate. The server should answer the selector from
small precomputed counts and index metadata. It should not process thousands of
books while the user is trying to open Movies, Anime, Music, or another family.

Persist durable navigation preferences such as the last family, shelf, and
view choices. Do not persist ephemeral search text or transient loading state.

## Independent browse and presentation choices

The current Series list, Series covers, and All books modes combine separate
decisions. Treat them as independent axes:

- **Browse by:** `Series` or `All items`.
- **Series index:** `List` or `Covers`.
- **Items inside a series:** `List` or `Covers`.
- **All-items presentation:** `List` or `Covers`.

In a Series list, rows remain collapsed initially; expanding a row uses the
remembered item presentation. In Series covers, opening a series uses that same
remembered item presentation instead of always forcing book covers. All items
likewise uses its own remembered list-or-cover choice. This supports collapsed
and expanded list workflows as well as collapsed and expanded cover workflows.

Keep these controls compact and contextual. The primary control chooses
`Series` versus `All items`; a small view control chooses `List` versus
`Covers` for the content currently on screen. If the series-index and item
preferences need separate editing, expose them in a compact view menu instead
of adding several permanent buttons.

Remember presentation choices per media family and browser/device initially;
they can move to account-level preferences when multi-user settings sync is
introduced. Changing a shelf must not unexpectedly reset them. Media modules
may offer only the combinations they can represent meaningfully.

## Why families and shelves are separate

Books, movies, episodes, podcasts, and music need different browse and playback
models. Shelves answer “where is it stored?” while media families answer “how
do I browse and use it?”. Keeping those levels separate prevents a Seafile
folder list from becoming the product navigation.

The dashboard can remain cross-media: Continue reading, Continue watching, and
Continue listening appear only when populated. Each row uses its media module’s
own progress semantics while sharing account identity and dismissal behavior.

## Implementation boundary

The family switcher should be driven by media capabilities returned by the
catalogue service, not hard-coded DOM buttons. Add it only after the first
non-book media module establishes real data and routes. Until then, the current
library navigation remains quieter by omitting it entirely.
