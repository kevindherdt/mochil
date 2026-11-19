import assert from "node:assert/strict";
import test from "node:test";
import {
  AudioRecoveryAction,
  createAudioPlaybackRecovery,
} from "../static/audio-playback-recovery.js";

test("transient audio failures use a bounded retry budget before one fallback", () => {
  const recovery = createAudioPlaybackRecovery();
  recovery.transition("podcasts:episode");
  assert.deepEqual(recovery.onError(true), { type: AudioRecoveryAction.Retry, attempt: 1 });
  assert.deepEqual(recovery.onError(true), { type: AudioRecoveryAction.Retry, attempt: 2 });
  assert.deepEqual(recovery.onError(true), { type: AudioRecoveryAction.Retry, attempt: 3 });
  assert.deepEqual(recovery.onError(true), { type: AudioRecoveryAction.Fallback });
  assert.deepEqual(recovery.onError(true), { type: AudioRecoveryAction.Retry, attempt: 1 });
});

test("stalls and errors share one recovery budget", () => {
  const recovery = createAudioPlaybackRecovery({ retryLimit: 2 });
  recovery.transition("audiobooks:chapter");
  assert.equal(recovery.onStall(false).type, AudioRecoveryAction.Retry);
  assert.equal(recovery.onError(false).type, AudioRecoveryAction.Retry);
  assert.equal(recovery.onStall(false).type, AudioRecoveryAction.Terminal);
});

test("ready playback restores retries without restoring fallback", () => {
  const recovery = createAudioPlaybackRecovery({ retryLimit: 1 });
  recovery.transition("music:track");
  recovery.onError(true);
  assert.equal(recovery.onError(true).type, AudioRecoveryAction.Fallback);
  recovery.ready();
  assert.equal(recovery.onError(true).type, AudioRecoveryAction.Retry);
  assert.equal(recovery.onError(true).type, AudioRecoveryAction.Terminal);
});

test("same-item source replacement cannot reset fallback ownership", () => {
  const recovery = createAudioPlaybackRecovery({ retryLimit: 0 });
  recovery.transition("music:one");
  assert.equal(recovery.onError(true).type, AudioRecoveryAction.Fallback);
  recovery.transition("music:one");
  assert.equal(recovery.onError(true).type, AudioRecoveryAction.Terminal);
  recovery.transition("music:two");
  assert.equal(recovery.onError(true).type, AudioRecoveryAction.Fallback);
});
