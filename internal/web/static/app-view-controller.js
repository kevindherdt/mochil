export function createAppViewController({
  $,
  mediaNavigator,
  searchController,
  searchToggle,
  hasLibraries,
  openSettings,
  loadLibrary,
  loadDashboard,
  onViewChanged = () => {},
  updateLayout,
  requestFrame = requestAnimationFrame,
}) {
  let activeView = "dashboard";
  const buttons = { settings: $('[data-view="settings"]') };

  function show(view) {
    activeView = view;
    const showingLibrary = view === "library";
    if (!showingLibrary) {
      mediaNavigator.hide();
      searchController.setOpen(false);
    }
    searchToggle.hidden = !showingLibrary;
    $("#search-panel").hidden = !showingLibrary;
    // Only reveal the masthead context row when something is actually in it: an empty row costs 44px of
    // a phone's screen and, on the library view, used to be where the chips overflowed.
    const context = $("#shell-context");
    context.hidden = !showingLibrary || (context.children?.length ?? 0) === 0;
    for (const [name, button] of Object.entries(buttons)) {
      const active = view === name;
      button.classList.toggle("active", active);
      button.setAttribute("aria-pressed", active);
    }
    const showingDashboard = view === "dashboard";
    $("#dashboard-view").hidden = view !== "dashboard";
    $("#library-view").hidden = !showingLibrary;
    $("#settings-view").hidden = view !== "settings";

    if (view === "settings") openSettings();
    else if (showingLibrary && hasLibraries()) loadLibrary();
    else if (view === "dashboard") loadDashboard();
    onViewChanged(view);
    requestFrame(updateLayout);
  }

  function bind() {
    for (const [view, button] of Object.entries(buttons)) {
      button.onclick = () =>
        show(
          button.getAttribute("aria-pressed") === "true" ? "dashboard" : view,
        );
    }
  }

  return {
    bind,
    show,
    current: () => activeView,
    sourceButton: buttons.sources,
  };
}
