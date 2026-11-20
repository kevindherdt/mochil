export function readPrimaryNavigationGeometry() {
  const bounds = selector => {
    const element = document.querySelector(selector)
    const rectangle = element?.getBoundingClientRect()
    if (!element || !rectangle) return null
    const style = getComputedStyle(element)
    const hit = rectangle.width > 0 && rectangle.height > 0
      ? document.elementFromPoint(rectangle.left + rectangle.width / 2, rectangle.top + rectangle.height / 2)
      : null
    return {
      left: rectangle.left,
      right: rectangle.right,
      width: rectangle.width,
      painted: style.display !== 'none' && style.visibility !== 'hidden' && Number(style.opacity) > 0 && element.contains(hit),
    }
  }
  return {
    viewportWidth: innerWidth,
    // The themed menu button stands in for the (hidden) native select.
    switcher: bounds('.library-menu-button') || bounds('#filters'),
    search: bounds('#search-toggle'),
    activity: bounds('#activity-toggle'),
    settings: bounds('[data-view="settings"]'),
  }
}

/** Compact navigation remains ordered and reachable without relying on screenshots. */
export function primaryNavigationVisible({ viewportWidth, switcher, search, activity, settings }) {
  if (viewportWidth > 760) return true
  if (!switcher || !search || !settings) return false
  // The activity button only exists while the server is busy; when it is
  // absent the remaining controls must still be ordered and reachable.
  const activityShown = Boolean(activity && activity.width > 0)
  const shown = activityShown ? [switcher, search, activity, settings] : [switcher, search, settings]
  return shown.every(item => item.painted === true) &&
    switcher.width >= 44 && switcher.left >= 0 &&
    switcher.right <= search.left + 1 &&
    (!activityShown || activity.left >= search.right - 1) &&
    settings.width >= 44 && settings.right <= viewportWidth + 1
}
