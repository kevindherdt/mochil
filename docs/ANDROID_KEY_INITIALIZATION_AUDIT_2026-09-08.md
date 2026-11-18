# Shared encrypted-store key initialization

Status: source fix after live 0.16.79; no new APK/deployment.

`EncryptedPayloadStore.secretKey` previously performed an unsynchronized check followed by generation. Distinct store instances sharing an alias could both observe a missing key during first use and generate competing replacements. That creates a risk of encrypting one payload with a key subsequently replaced by the other initialization. This was found by code inspection, not attributed to a reported user data-loss incident.

The check/create sequence now uses one process-wide initialization lock through a small generic helper. Existing keys are returned unchanged. Encryption, decryption and preference commits remain outside the lock; no format, alias, credential or persistence migration occurs. Source search confirms this is the sole Android key-generation implementation in the app. This addresses overlapping instances in the same app process, not multi-process keystore coordination.

Both distribution flavors passed targeted tests (14s Gradle execution). A concurrent 32-request test verifies single creation and identical returned identity; existing-key and failed-creation/retry tests preserve behavior. These exercise the locking contract with a fake key store, not AndroidKeyStore hardware/provider behavior. Module gate passed. No personal key material was read or changed by tests.

The lint review that led here found mostly Kotlin convenience/style recommendations, dependency notices and conservative storage/logging suggestions. These are not evidence that replacing all calls or upgrading every dependency would improve reliability. Existing synchronous encrypted preference commits deliberately return durability success; do not replace them with asynchronous edits solely to silence style warnings.
