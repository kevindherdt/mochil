// formatTime — "m:ss" for a playback position/duration, shared by MiniPlayerBar and PlaybackControls so the two
// players can never disagree about what "1:32" means.
export function formatTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "0:00";
  const minutes = Math.floor(seconds / 60);
  const rest = Math.floor(seconds % 60);
  return `${minutes}:${rest.toString().padStart(2, "0")}`;
}
