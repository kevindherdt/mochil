export const popupSelector =
  ".view-menu,.audio-view-menu,.audio-more,.book-more,.document-more,.audio-player-more,.track-menu,.podcast-feed-options";

export function closeOpenMenus(menus, target) {
  menus.forEach((menu) => {
    if (!target || !menu.contains(target)) menu.open = false;
  });
}

export function bindPopupDismissal({ document, podcastPanel, podcastToggle }) {
  document.addEventListener(
    "pointerdown",
    (event) => {
      closeOpenMenus(
        document.querySelectorAll(`${popupSelector}[open]`),
        event.target,
      );
      if (
        !podcastPanel.hidden &&
        !podcastPanel.contains(event.target) &&
        !podcastToggle.contains(event.target)
      ) {
        podcastPanel.hidden = true;
        podcastToggle.setAttribute("aria-expanded", "false");
      }
    },
    true,
  );
  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") return;
    closeOpenMenus(document.querySelectorAll(`${popupSelector}[open]`));
    if (podcastPanel.hidden) return;
    podcastPanel.hidden = true;
    podcastToggle.setAttribute("aria-expanded", "false");
    podcastToggle.focus();
  });
}
