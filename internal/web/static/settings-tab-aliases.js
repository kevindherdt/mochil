// Settings tabs that were folded into another one. A remembered tab (sessionStorage) from before the fold must land
// on the tab that now holds the content, not on nothing.
//  - metadata: the review, import and provider-token sections are part of Libraries.
//  - editor: its menu checkbox and Hermes tools are part of Modules.
//  - reading: the reading defaults were the same stored values the reader's own panel edits, so that panel went.
const folded = new Map([["metadata", "libraries"], ["editor", "modules"], ["reading", "libraries"]]);

export function settingsTabFor(name) {
  return folded.get(name) || name;
}
