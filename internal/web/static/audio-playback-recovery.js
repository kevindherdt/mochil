export const AudioRecoveryAction = Object.freeze({
  Retry: "retry",
  Fallback: "fallback",
  Terminal: "terminal",
});

/** Owns one bounded recovery budget across native and compatibility playback. */
export function createAudioPlaybackRecovery({ retryLimit = 3 } = {}) {
  let mediaID = "";
  let attempts = 0;
  let fallbackAttempted = false;

  function transition(nextMediaID) {
    if (!nextMediaID || nextMediaID === mediaID) return;
    mediaID = nextMediaID;
    attempts = 0;
    fallbackAttempted = false;
  }

  function failure(canFallback) {
    if (attempts < retryLimit) {
      attempts += 1;
      return { type: AudioRecoveryAction.Retry, attempt: attempts };
    }
    if (canFallback && !fallbackAttempted) {
      fallbackAttempted = true;
      attempts = 0;
      return { type: AudioRecoveryAction.Fallback };
    }
    return { type: AudioRecoveryAction.Terminal };
  }

  return {
    transition,
    onError: failure,
    onStall: failure,
    // A fresh start for the same item, used when the person presses Try again.
    reset() {
      attempts = 0;
      fallbackAttempted = false;
    },
    ready() {
      attempts = 0;
    },
  };
}
