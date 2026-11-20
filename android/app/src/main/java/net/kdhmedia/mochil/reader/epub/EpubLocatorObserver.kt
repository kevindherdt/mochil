package net.kdhmedia.mochil.reader.epub

import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleOwner
import androidx.lifecycle.repeatOnLifecycle
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.FlowPreview
import kotlinx.coroutines.flow.debounce
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.onEach
import kotlinx.coroutines.launch
import org.readium.r2.navigator.epub.EpubNavigatorFragment
import org.readium.r2.shared.ExperimentalReadiumApi
import org.readium.r2.shared.publication.Locator
import org.readium.r2.shared.publication.Publication

internal data class EpubLocatorCallbacks(
    val onImmediate: (Locator, ReaderSyncStatus) -> Unit,
    val onSettled: (Double, ReadingStatistics) -> Unit,
    val onProgressUpdate: (EpubProgressUpdate) -> Unit,
    val onResourceChanged: suspend () -> Unit,
)

/** Owns lifecycle-bound locator persistence, statistics, and remote progress saves. */
@OptIn(ExperimentalReadiumApi::class, FlowPreview::class)
internal class EpubLocatorObserver(
    private val publication: Publication,
    private val navigator: EpubNavigatorFragment,
    private val publicationKey: String,
    private val locatorStore: EpubLocatorStore,
    private val statisticsStore: ReadingStatisticsStore,
    private val progressCoordinator: EpubProgressCoordinator,
) {
    fun start(
        scope: CoroutineScope,
        lifecycleOwner: LifecycleOwner,
        initialStatus: ReaderSyncStatus,
        callbacks: EpubLocatorCallbacks,
    ) {
        scope.launch {
            var status = initialStatus
            val resumePendingPosition = progressCoordinator.shouldResumePending()
            lifecycleOwner.repeatOnLifecycle(Lifecycle.State.STARTED) {
                navigator.currentLocator.drop(if (resumePendingPosition) 0 else 1).onEach { locator ->
                    locatorStore.save(publicationKey, locator)
                    status = progressCoordinator.activeStatus(status)
                    callbacks.onImmediate(locator, status)
                }.debounce(SAVE_DEBOUNCE_MILLIS).collect { locator ->
                    try {
                        val structural = runCatching { navigator.firstVisibleElementLocator() }.getOrNull() ?: locator
                        val progression = locator.locations.totalProgression?.takeIf(Double::isFinite) ?: Double.NaN
                        val statistics = statisticsStore.record(
                            publicationKey,
                            locator.locations.totalProgression,
                            locator.locations.progression,
                            locator.href.toString(),
                        )
                        callbacks.onSettled(progression, statistics)
                        if (progressCoordinator.canAttemptSave(locator)) {
                            val structuralSpine = publication.readingOrder.indexOfFirst { it.url() == structural.href }
                            val koReaderLocator = captureKoReaderLocator(structural, structuralSpine)
                            val update = progressCoordinator.save(locator, structural, koReaderLocator) { synchronized ->
                                publication.readingOrder.indexOfFirst { it.url() == synchronized.href }
                            }
                            status = update.status
                            callbacks.onProgressUpdate(update)
                        }
                    } catch (cancelled: CancellationException) {
                        throw cancelled
                    } catch (_: Throwable) {
                        val failure = locatorSaveFailure(status)
                        status = failure.status
                        callbacks.onProgressUpdate(failure)
                    }
                }
            }
        }
        scope.launch {
            lifecycleOwner.repeatOnLifecycle(Lifecycle.State.STARTED) {
                navigator.currentLocator.map { it.href }.distinctUntilChanged().collect {
                    callbacks.onResourceChanged()
                }
            }
        }
    }

    /**
     * Sends a position that is still waiting for the server. The debounced save
     * above is cancelled when the reader closes within its window, which would
     * leave the last pages unsynced until this book is opened again.
     */
    suspend fun flush(locator: Locator) {
        if (!progressCoordinator.shouldResumePending() || !progressCoordinator.canAttemptSave(locator)) return
        val structural = runCatching { navigator.firstVisibleElementLocator() }.getOrNull() ?: locator
        val structuralSpine = publication.readingOrder.indexOfFirst { it.url() == structural.href }
        val koReaderLocator = captureKoReaderLocator(structural, structuralSpine)
        progressCoordinator.save(locator, structural, koReaderLocator) { synchronized ->
            publication.readingOrder.indexOfFirst { it.url() == synchronized.href }
        }
    }

    private suspend fun captureKoReaderLocator(locator: Locator, spineIndex: Int): String? {
        val selector = locator.locations.otherLocations["cssSelector"] as? String ?: return null
        val script = KoReaderXPointer.captureScript(spineIndex, selector) ?: return null
        return KoReaderXPointer.decodeJavascriptResult(runCatching { navigator.evaluateJavascript(script) }.getOrNull())
    }

    private companion object {
        const val SAVE_DEBOUNCE_MILLIS = 1_500L
    }
}

internal fun locatorSaveFailure(previous: ReaderSyncStatus) = EpubProgressUpdate(
    status = ReaderSyncStatus(ReaderSyncPhase.DeviceOnly, previous.lastSynchronizedAtMillis),
    message = "This position could not be saved. Reading can continue safely.",
    messageIsError = true,
)
