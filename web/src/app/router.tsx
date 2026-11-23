// app/router.tsx — code-based TanStack Router config (plan §5.1 names the full route list for Phase 2; this is
// the shell skeleton Phase 1 needs: the five dock/sidebar places as stub screens inside AppShell, plus the
// gallery as a separate, unwrapped route — it's a dev tool, not a real app screen, so it never shows the dock).
import {
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
  lazyRouteComponent,
  Outlet,
} from "@tanstack/react-router";
import { z } from "zod";
import { ShellLayout } from "../shell/ShellLayout";
import { GalleryPage } from "../gallery/GalleryPage";
import { AboutScreen } from "../features/about/screens/AboutScreen";
import { AppearanceScreen } from "../features/theme/screens/AppearanceScreen";
import { SettingsScreen } from "../features/settings/screens/SettingsScreen";
import { ModulesScreen } from "../features/settings/screens/ModulesScreen";
import { AccountsScreen } from "../features/settings/screens/AccountsScreen";
import { LibrariesScreen } from "../features/settings/screens/LibrariesScreen";
import { PlaybackScreen } from "../features/settings/screens/PlaybackScreen";
import { DiagnosticsScreen } from "../features/settings/screens/DiagnosticsScreen";
import { SyncScreen } from "../features/settings/screens/SyncScreen";
import { MetadataProvidersScreen } from "../features/settings/screens/MetadataProvidersScreen";
import { NarrationScreen } from "../features/settings/screens/NarrationScreen";
import { AssistantScreen } from "../features/settings/screens/AssistantScreen";

import { NotFoundScreen } from "./NotFoundScreen";
import { LoginScreen } from "../features/auth/screens/LoginScreen";
import { LibraryScreen } from "../features/library/screens/LibraryScreen";
import { BookDetailScreen } from "../features/library/screens/BookDetailScreen";
import { VideoDetailScreen } from "../features/library/screens/VideoDetailScreen";
import { ShowDetailScreen } from "../features/library/screens/ShowDetailScreen";
import { AudioDetailScreen } from "../features/library/screens/AudioDetailScreen";
import { DiscoverScreen } from "../features/discover/screens/DiscoverScreen";
import { LyanaScreen } from "../features/lyana/screens/LyanaScreen";
import { NotesScreen } from "../features/notes/screens/NotesScreen";

const rootRoute = createRootRoute({
  component: () => <Outlet />,
});

/** A pathless layout route (plan §3: AppShell is the only place that positions chrome) — every real screen is a
    child of this, so it always renders inside the dock/sidebar shell; /__gallery deliberately is not. */
const shellLayoutRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "shell",
  component: ShellLayout,
  // Unknown paths render the app's own 404 *inside* the shell, so the dock/sidebar (and therefore a way out) is
  // still there — TanStack's default is a bare "Not Found" with nothing around it.
  notFoundComponent: NotFoundScreen,
});

const homeRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "/",
  // Plan §5.1: "/" is always a library shelf, never its own screen — redirecting (rather than duplicating the
  // same stub at two paths) is also what makes Dock/Sidebar's active-place check correct on first load: without
  // this, "/" matched no `place.to`, so Library never highlighted when it was the one thing on screen.
  beforeLoad: () => {
    throw redirect({ to: "/library" });
  },
});

// The Library screen's own state (which shelf, which browse mode, which group drilled into) lives in the URL,
// not in useState: a `useState` reset to its default on every mount, so pressing the phone's back button after
// opening a track landed back on whichever shelf rendered first (Books) instead of the one actually being
// browsed (Kevin: "clicking to play a song and pressing the back button opens the book library instead of the
// music library i was at"). The browser's own history now carries this, so back/forward restores it for free.
const librarySearchSchema = z.object({
  lib: z.string().optional(),
  view: z.string().optional(),
  layout: z.enum(["grid", "list"]).optional(),
  group: z.string().optional(),
  groupName: z.string().optional(),
});

const libraryRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "/library",
  validateSearch: librarySearchSchema,
  component: LibraryScreen,
});

// Detail screens — a card opening nowhere was the reported bug this slice fixes. One route per data source
// (books/videos/shows/audio), matching api/mediaDetail.ts's split, not per shelf kind: a movie and an episode
// are both `/library/video/...`, because both are single rows from `/api/videos`.
const bookDetailRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "/library/book/$libraryId/$bookId",
  component: BookDetailScreen,
});

const videoDetailRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "/library/video/$libraryId/$itemId",
  component: VideoDetailScreen,
});

const showDetailRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "/library/show/$libraryId/$itemId",
  component: ShowDetailScreen,
});

const audioDetailRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "/library/audio/$libraryId/$itemId",
  component: AudioDetailScreen,
});

/** The full-screen player. A route of its own rather than a mode of the detail page: entering it is a navigation
 *  (Back returns to the film, the browser's back button does the same thing, and the URL names what is playing), and
 *  the shell hides its chrome for this path alone. */
/** The player is the app's heaviest leaf — hls.js, the player chrome, the settings sheet and a dozen icons that nothing
 *  else uses. Loading it lazily keeps all of that out of the bundle every other screen pays for, which is the split the
 *  size gate was asking for (scripts/check-bundle-budget.mjs). Pressing Play is where a short wait is least noticed. */
const VideoPlayerScreen = lazyRouteComponent(
  () => import("../features/library/screens/VideoPlayerScreen"),
  "VideoPlayerScreen",
);

const playRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "/play/$libraryId/$itemId",
  // `?restart=1` is "from the start" — the detail page offers it next to Resume when there is a position to resume.
  validateSearch: z.object({ restart: z.string().optional() }),
  component: VideoPlayerScreen,
});

/** The metadata editor. Lazy: it is a heavy table opened deliberately, not a browsing surface. */
const editorRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "/editor",
  component: lazyRouteComponent(() => import("../features/editor/screens/EditorScreen"), "EditorScreen"),
});

const discoverRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "/discover",
  component: DiscoverScreen,
});

const lyanaRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "/lyana",
  component: LyanaScreen,
});

const notesRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "/notes",
  component: NotesScreen,
});

/** `/settings` is a layout route (just `<Outlet/>`) with its own index child — without this, visiting
    `/settings/about` rendered the parent's own stub text and never reached `<AboutScreen>` at all: a route with
    a `component` that doesn't render `<Outlet/>` can't have its children show up, ever. Caught by actually
    navigating to the real URL on the dev box rather than trusting the route tree compiled. */
const settingsRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "/settings",
  component: () => <Outlet />,
});

const settingsIndexRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/",
  component: SettingsScreen,
});

const settingsAboutRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/about",
  component: AboutScreen,
});

const settingsAppearanceRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/appearance",
  component: AppearanceScreen,
});

const settingsModulesRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/modules",
  component: ModulesScreen,
});

const settingsAccountsRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/accounts",
  component: AccountsScreen,
});

const settingsLibrariesRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/libraries",
  component: LibrariesScreen,
});

const settingsPlaybackRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/playback",
  component: PlaybackScreen,
});

const settingsDiagnosticsRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/diagnostics",
  component: DiagnosticsScreen,
});

const settingsSyncRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/sync",
  component: SyncScreen,
});

const settingsMetadataRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/metadata",
  component: MetadataProvidersScreen,
});

const settingsNarrationRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/narration",
  component: NarrationScreen,
});

const settingsAssistantRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/assistant",
  component: AssistantScreen,
});

/** Lazy like the video player: the review screen carries its own zod schemas and is opened rarely, so it has no
 *  business in the entry chunk — the budget is 170 kB gz and the entry was 0.2 kB from it. */
const MetadataReviewScreen = lazyRouteComponent(
  () => import("../features/settings/screens/MetadataReviewScreen"),
  "MetadataReviewScreen",
);

const ActivityScreen = lazyRouteComponent(
  () => import("../features/activity/screens/ActivityScreen"),
  "ActivityScreen",
);

const settingsMetadataReviewRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/metadata/review",
  component: MetadataReviewScreen,
});

const settingsActivityRoute = createRoute({
  getParentRoute: () => settingsRoute,
  path: "/activity",
  component: ActivityScreen,
});

/** Any path the router does not know, rendered inside the shell (`$` is TanStack's splat). Needed in addition to
 *  `notFoundComponent`: a URL that matches no child at all never reaches the shell route's own boundary. */
const catchAllRoute = createRoute({
  getParentRoute: () => shellLayoutRoute,
  path: "$",
  component: NotFoundScreen,
});

const galleryRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/__gallery",
  component: GalleryPage,
});

/** Deliberately outside shellLayoutRoute (plan: a signed-out visitor has no dock/sidebar to show) — a sibling
    of /__gallery, not a child of the authenticated shell. */
const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/login",
  component: LoginScreen,
});

const routeTree = rootRoute.addChildren([
  shellLayoutRoute.addChildren([
    homeRoute,
    libraryRoute,
    bookDetailRoute,
    videoDetailRoute,
    showDetailRoute,
    audioDetailRoute,
    playRoute,
    editorRoute,
    discoverRoute,
    lyanaRoute,
    notesRoute,
    catchAllRoute,
    settingsRoute.addChildren([
      settingsIndexRoute,
      settingsAboutRoute,
      settingsAppearanceRoute,
      settingsModulesRoute,
      settingsAccountsRoute,
      settingsLibrariesRoute,
      settingsPlaybackRoute,
      settingsMetadataRoute,
      settingsSyncRoute,
      settingsNarrationRoute,
      settingsAssistantRoute,
      settingsActivityRoute,
      settingsMetadataReviewRoute,
      settingsDiagnosticsRoute,
    ]),
  ]),
  galleryRoute,
  loginRoute,
]);

export const router = createRouter({
  routeTree,
  // Vite's base only rewrites asset URLs, not the pathname the router sees —
  // without this, every real navigation to e.g. /next/__gallery 404s inside the router even though the HTML
  // and its assets loaded fine (verified: curl'd the page and both assets as 200, the app still rendered "Not
  // Found" until this was added, which is what proved the bug was the router's basepath, not the server).
  basepath: "/",
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
