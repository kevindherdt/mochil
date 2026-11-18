# Same-item audio seek ownership

Status: source-only follow-up after 0.16.80; not deployed.

The audio progress wrapper protected item switches with a generation, but a successful earlier save unconditionally released the current seek claim. A delayed conflict lookup could likewise consume or suppress a newer seek on the same item, including when the remote item had been marked finished.

Each explicit seek now advances a local claim generation. Successful saves release only the claim they sampled. An older conflict lookup may adopt the remote revision, but cannot change the newer seek's ownership or retry its old position under that new claim. A new seek resets timed-save throttling. The local generation is stripped before the HTTP payload is sent; the server protocol is unchanged.

Regression tests delay a successful save and a conflict lookup, seek backward on the same episode, then verify the new explicit position survives, uses the fresh revision and is claimed only once. The conflict case includes remotely finished state. Existing different-item, passive-client, timing-drift and outage tests continue to pass.

Verification: 14 focused audio tests and all 542 frontend tests passed. Full-suite log: `/tmp/bobobooks-same-item-seek-tests.log`. The shell command's final status was 1 because a subsequent package-script lookup found no match; the test runner itself reported 542 passes and zero failures. No APK, runtime deployment or UI layout change was made. Video ownership needs a separate review because its save lifecycle differs.

## Conflict followed by network failure

A shared audio/video regression subsequently reproduced another audio failure: conflict resolution consumed the explicit claim, then the retry failed offline. The next request carried `explicit_seek: false` despite the unacknowledged rewind. The test failed against the prior audio code and passed after retaining the sampled claim until success (provided both item and seek generations still match). Video's extracted writer already passed this scenario. The test also verifies that the first successful retry uses the learned revision and subsequent ordinary playback no longer carries the explicit flag.

This remains source-only alongside the video follow-up. Full verification log: `/tmp/bobobooks-seek-conflict-retry-tests.log`.
