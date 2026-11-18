# Preserve an unreadable audio outbox

Status: source-only after 0.16.80.

Previously encrypted load failures and JSON parsing failures both became an empty list. A subsequent checkpoint could overwrite the unreadable outbox with one new entry. A temporary key-store failure should not erase pending progress.

EncryptedPayloadStore now exposes an opt-in strict read: an absent envelope is empty, but present unreadable/incomplete storage fails. Existing best-effort consumers are unchanged. Audio journal parsing now distinguishes empty from unreadable: replay returns no entries on failure, and mutation/acknowledgement leave storage untouched. A failed durable checkpoint emits the content-free `storage/audio_progress_not_saved` diagnostic; online delivery can still proceed without falsely claiming durability.

The storage name, encryption alias, schema and file contents are not migrated. This does not repair permanently corrupt data or audit every other encrypted consumer. Strict read checking is used within the existing process-wide audio journal transaction lock.

Regression tests cover absent, incomplete and unreadable encrypted envelopes, malformed JSON, and failed journal loads across read/put/acknowledgement with zero persistence calls. Both flavor focused suites passed in 10s. Full suites following the diagnostic addition use `/tmp/bobobooks-journal-strict-full-tests.log`. No APK/deployment or hardware Keystore acceptance is claimed.
