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
    $("#shell-context").hidden = !showingLibrary;
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
