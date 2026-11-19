package net.kdhmedia.mochil.reader.epub

import kotlinx.coroutines.CancellationException
import java.io.IOException
import java.net.SocketTimeoutException
import kotlin.math.abs
import net.kdhmedia.mochil.auth.DeviceCredential
import org.readium.r2.shared.publication.Locator
import org.readium.r2.shared.publication.Publication

internal data class EpubProgressUpdate(
    val locator: Locator? = null,
    val status: ReaderSyncStatus,
    val message: String? = null,
    val messageIsError: Boolean = false,
)

internal class EpubProgressCoordinator(
    private val store: EpubProgressStateStore,
    private val publicationKey: String,
    private val credential: DeviceCredential?,
    private val libraryId: String,
    private val bookId: String,
    private val externalTitleId: String,
    private val gateway: ReadingProgressGateway = ReadingProgressClient(),
) {
    private var revision = store.revision(publicationKey)
    private var blocked = false
    private val ownership = ReaderProgressOwnership()
    val canSynchronize = credential != null && (externalTitleId.isNotBlank() || libraryId.isNotBlank() && bookId.isNotBlank())

    fun initialStatus(): ReaderSyncStatus {
        val synchronizedAt = store.lastSynchronizedAt(publicationKey)
        return ReaderSyncStatus(
            if (!store.pending(publicationKey) && canSynchronize && synchronizedAt > 0) ReaderSyncPhase.Synced else ReaderSyncPhase.DeviceOnly,
            synchronizedAt,
        )
    }

    fun shouldResumePending(): Boolean = store.pending(publicationKey) && !blocked
    fun canAttemptSave(locator: Locator): Boolean {
        val fraction = locator.locations.totalProgression?.takeIf(Double::isFinite) ?: return false
        if (credential == null || !canSynchronize || blocked) return false
        return ownership.canWrite(fraction)
    }
    fun activeStatus(previous: ReaderSyncStatus): ReaderSyncStatus = ReaderSyncStatus(if (canSynchronize && !blocked) ReaderSyncPhase.Syncing else ReaderSyncPhase.DeviceOnly, previous.lastSynchronizedAtMillis)

    suspend fun resolveInitial(publication: Publication, local: Locator?, confirmRemote: suspend (ServerReadingPosition) -> Boolean): EpubProgressUpdate {
        val saved = credential ?: return EpubProgressUpdate(store.current(publicationKey), initialStatus())
        if (!canSynchronize) return EpubProgressUpdate(store.current(publicationKey), initialStatus())
        return try {
            val remote = if (externalTitleId.isNotBlank()) gateway.getExternal(saved, externalTitleId) else gateway.get(saved, libraryId, bookId)
            val current = store.current(publicationKey)
            if (current?.toJSON()?.toString() != local?.toJSON()?.toString()) return EpubProgressUpdate(current, initialStatus())
            if (remote == null || remote.revision <= maxOf(revision, store.revision(publicationKey))) return EpubProgressUpdate(current, initialStatus())
            val resolved = RemotePositionResolver.resolve(publication, remote)
            if (resolved == null) {
                blocked = true
                EpubProgressUpdate(store.current(publicationKey), ReaderSyncStatus(ReaderSyncPhase.DeviceOnly, initialStatus().lastSynchronizedAtMillis), "A newer position from another reader needs review. Your local position was kept.")
            } else {
                val initialDecision = decideEpubInitialPosition(
                    localPending = store.pending(publicationKey),
                    sameDevice = remote.deviceId == saved.deviceId,
                    equivalent = readingPositionsEquivalent(local, resolved, remote.fraction),
                )
                val decision = when (initialDecision) {
                    EpubInitialPositionDecision.Ask -> if (confirmRemote(remote)) EpubInitialPositionDecision.UseRemote else EpubInitialPositionDecision.KeepLocal
                    else -> initialDecision
                }
                val remoteRevision = remote.revision
                when (decision) {
                    EpubInitialPositionDecision.UseRemote -> {
                        if (store.adoptRemote(publicationKey, local, resolved, remoteRevision)) {
                            revision = maxOf(revision, remoteRevision)
                            EpubProgressUpdate(store.current(publicationKey), initialStatus())
                        } else {
                            EpubProgressUpdate(store.current(publicationKey), initialStatus())
                        }
                    }
                    EpubInitialPositionDecision.KeepLocalSynchronized -> {
                        // Equivalent positions can resolve to adjacent rendered
                        // pages on different screen sizes. Acknowledge the newer
                        // revision without moving this device or writing it back.
                        val exactLocal = local ?: resolved
                        store.markSynchronized(publicationKey, exactLocal, remoteRevision)
                        revision = maxOf(revision, remoteRevision)
                        EpubProgressUpdate(store.current(publicationKey), initialStatus())
                    }
                    EpubInitialPositionDecision.KeepLocal -> {
                        revision = maxOf(revision, remoteRevision)
                        store.saveRevision(publicationKey, revision)
                        store.markPending(publicationKey)
                        EpubProgressUpdate(store.current(publicationKey), ReaderSyncStatus(ReaderSyncPhase.DeviceOnly, initialStatus().lastSynchronizedAtMillis))
                    }
                    EpubInitialPositionDecision.Ask -> error("The EPUB position decision was not resolved.")
                }
            }
        } catch (cancelled: CancellationException) {
            throw cancelled
        } catch (error: Throwable) {
            EpubProgressUpdate(store.current(publicationKey), ReaderSyncStatus(ReaderSyncPhase.DeviceOnly, initialStatus().lastSynchronizedAtMillis), progressSyncFailureMessage(error), true)
        }
    }

    suspend fun save(exactPage: Locator, structural: Locator, koReaderLocator: String?, spineIndexFor: (Locator) -> Int): EpubProgressUpdate {
        val saved = credential
        val fraction = exactPage.locations.totalProgression?.takeIf(Double::isFinite)?.coerceIn(0.0, 1.0)
        if (saved == null || fraction == null || !canSynchronize || blocked) return EpubProgressUpdate(exactPage, activeStatus(initialStatus()))
        val positions = readerPositionPair(exactPage, structural.copyWithLocations(totalProgression = fraction))
        val spineIndex = spineIndexFor(positions.synchronized).coerceAtLeast(0)
        store.beginSync(publicationKey)
        var acknowledged = false
        val update = try {
            when (val result = if (externalTitleId.isNotBlank()) gateway.putExternal(saved, externalTitleId, positions.synchronized, fraction, revision) else gateway.put(saved, libraryId, bookId, positions.synchronized, fraction, revision, spineIndex, koReaderLocator)) {
                is ReadingSaveResult.Saved -> {
                    revision = maxOf(revision, result.position.revision)
                    store.markSynchronized(publicationKey, positions.local, result.position.revision)
                    acknowledged = true
                    EpubProgressUpdate(exactPage, initialStatus())
                }
                is ReadingSaveResult.Conflict -> {
                    revision = maxOf(revision, result.position.revision)
                    store.saveRevision(publicationKey, result.position.revision)
                    if (readingProgressEquivalent(fraction, result.position.fraction)) {
                        store.markSynchronized(publicationKey, exactPage, result.position.revision)
                        acknowledged = true
                        EpubProgressUpdate(exactPage, initialStatus())
                    } else {
                        // Do not let a passive reader overwrite a newer device. Keep
                        // this page locally and retry only after an actual page move.
                        ownership.waitForMovement(fraction)
                        store.markPending(publicationKey)
                        EpubProgressUpdate(exactPage, ReaderSyncStatus(ReaderSyncPhase.DeviceOnly, initialStatus().lastSynchronizedAtMillis))
                    }
                }
            }
        } catch (cancelled: CancellationException) {
            throw cancelled
        } catch (error: Throwable) {
            EpubProgressUpdate(exactPage, ReaderSyncStatus(ReaderSyncPhase.DeviceOnly, initialStatus().lastSynchronizedAtMillis), progressSyncFailureMessage(error), true)
        } finally {
            store.endSync(publicationKey, acknowledged)
        }
        return if (acknowledged) update.copy(status = initialStatus()) else update
    }
}

internal enum class EpubInitialPositionDecision { UseRemote, KeepLocalSynchronized, KeepLocal, Ask }

/** Pending local progress represents offline reading and must not be discarded on the same device. */
internal fun decideEpubInitialPosition(
    localPending: Boolean,
    sameDevice: Boolean,
    equivalent: Boolean,
): EpubInitialPositionDecision = when {
    equivalent -> EpubInitialPositionDecision.KeepLocalSynchronized
    !localPending -> EpubInitialPositionDecision.UseRemote
    sameDevice -> EpubInitialPositionDecision.KeepLocal
    else -> EpubInitialPositionDecision.Ask
}

internal fun readingProgressEquivalent(left: Double, right: Double): Boolean =
    left.isFinite() && right.isFinite() && abs(left - right) <= 0.002

private fun progressSyncFailureMessage(error: Throwable): String = when {
    error is SocketTimeoutException -> "Progress sync timed out. Reading remains saved on this device."
    error is IOException -> "Progress sync could not reach Mochil. Reading remains saved on this device."
    error.message == "The EPUB locator is too large to synchronize." -> "This book's location data is too large to synchronize. Reading remains saved on this device."
    error.message?.startsWith("The server returned ") == true -> "Mochil rejected this reading position. Reading remains saved on this device."
    else -> "Progress sync is temporarily unavailable. Reading remains saved on this device."
}

internal fun readingProgressMoved(previous: Double, current: Double): Boolean =
    previous.isFinite() && current.isFinite() && abs(previous - current) > 0.00001

internal fun readingPositionPromptMessage(position: ServerReadingPosition): String {
    val device = position.deviceName.trim().take(120).ifBlank { "Another device" }
    val percent = (position.fraction.coerceIn(0.0, 1.0) * 100).toInt()
    return "$device saved newer progress at $percent%. This device also has unsynced reading progress. Go to the newer position?"
}

/** Prevents a passive stale reader from claiming write ownership until the user moves. */
internal class ReaderProgressOwnership {
    private var waitingForMovementFrom: Double? = null

    fun waitForMovement(fraction: Double) {
        waitingForMovementFrom = fraction.takeIf(Double::isFinite)
    }

    fun canWrite(fraction: Double): Boolean {
        val waiting = waitingForMovementFrom ?: return true
        if (!readingProgressMoved(waiting, fraction)) return false
        waitingForMovementFrom = null
        return true
    }
}
