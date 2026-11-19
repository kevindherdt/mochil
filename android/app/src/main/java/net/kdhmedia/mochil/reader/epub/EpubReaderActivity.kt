package net.kdhmedia.mochil.reader.epub

import android.os.Bundle
import android.app.AlertDialog
import android.view.View
import android.view.KeyEvent
import android.view.ActionMode
import android.widget.TextView
import androidx.activity.enableEdgeToEdge
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableDoubleStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.fragment.app.FragmentActivity
import androidx.fragment.app.commit
import androidx.lifecycle.lifecycleScope
import java.io.File
import kotlinx.coroutines.launch
import kotlinx.coroutines.cancel
import kotlinx.coroutines.CancellationException
import org.readium.r2.navigator.epub.EpubNavigatorFactory
import org.readium.r2.navigator.epub.EpubNavigatorFragment
import org.readium.r2.navigator.html.HtmlDecorationTemplates
import org.readium.r2.shared.ExperimentalReadiumApi
import org.readium.r2.shared.publication.Publication
import org.readium.r2.shared.publication.Link
import org.readium.r2.shared.publication.Locator
import net.kdhmedia.mochil.ui.theme.AppPalette
import net.kdhmedia.mochil.ui.theme.AppThemePreferences
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics

@OptIn(ExperimentalReadiumApi::class)
class EpubReaderActivity : FragmentActivity() {
    private var publication: Publication? = null
    private var chromeVisible by mutableStateOf(false)
    private var locatorObserver: EpubLocatorObserver? = null
    // Outlives the activity so a position can still reach the server as the reader closes.
    private val flushScope = kotlinx.coroutines.CoroutineScope(kotlinx.coroutines.SupervisorJob() + kotlinx.coroutines.Dispatchers.Main.immediate)
    private var progression by mutableDoubleStateOf(Double.NaN)
    private var readingStatistics by mutableStateOf(ReadingStatistics())
    private var readerPreferences by mutableStateOf(ReaderPreferences())
    private lateinit var fontRuntime: EpubFontRuntime
    private var navigator: EpubNavigatorFragment? = null
    private val textSharpener = EpubTextSharpener(lifecycleScope)
    private val instantPages = EpubInstantPages(lifecycleScope)
    private val reduceMotion by lazy { net.kdhmedia.mochil.ui.theme.AppThemePreferences(this).load().reduceMotion }
    private val readAloud = EpubReadAloudHost(this, { navigator }, { publication }, { readerPreferences.theme })
    private var chapters by mutableStateOf(emptyList<EpubChapter>())
    private var searchState by mutableStateOf(EpubSearchState())
    private val searchGeneration = EpubSearchGeneration()
    private var bookmarks by mutableStateOf(emptyList<EpubBookmark>())
    private var currentLocator by mutableStateOf<Locator?>(null)
    private lateinit var locatorStore: EpubLocatorStore
    private lateinit var bookmarkStore: EpubBookmarkStore
    private lateinit var readingStatisticsStore: ReadingStatisticsStore
    private lateinit var annotationSession: EpubAnnotationSession
    private lateinit var selectionController: EpubSelectionActionController
    private lateinit var readerSurface: EpubReaderSurface
    private lateinit var publicationKey: String
    private val syncMessageState = mutableStateOf<String?>(null)
    private var syncMessage: String?
        get() = syncMessageState.value
        set(value) {
            syncMessageState.value = value
            if (::readerSurface.isInitialized) readerSurface.announce(value)
        }
    private var syncMessageIsError by mutableStateOf(false)
    private var readingSyncStatus by mutableStateOf(ReaderSyncStatus())
    private lateinit var progressCoordinator: EpubProgressCoordinator
    private val hardwarePreferences by lazy { net.kdhmedia.mochil.reader.ReaderHardwareNavigationPreferences(this) }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(null)
        val file = validatedPublication() ?: return finish()
        val title = intent.getStringExtra(EXTRA_TITLE).orEmpty()
        publicationKey = intent.getStringExtra(EXTRA_PROGRESS_KEY).orEmpty()
        PerformanceDiagnostics.initialize(applicationContext)
        enableEdgeToEdge()
        val preferencesStore = EpubReaderPreferencesStore(this)
        fontRuntime = EpubFontRuntime(CustomFontStore(this))
        readerPreferences = preferencesStore.update(fontRuntime::validated)
        applyReaderSystemBars(readerPreferences.showSystemStatusBar)
        bookmarkStore = EpubBookmarkStore(this)
        readingStatisticsStore = ReadingStatisticsStore(this)
        readingStatistics = readingStatisticsStore.snapshot(publicationKey)
        bookmarks = bookmarkStore.list(publicationKey)
        annotationSession = EpubAnnotationSession(EpubAnnotationController(AnnotationStateStore(this), publicationKey))
        readerSurface = EpubReaderSurface(this)
        selectionController = EpubSelectionActionController(
            scope = lifecycleScope,
            navigator = { navigator },
            publication = { publication },
            annotations = annotationSession,
            callbacks = EpubSelectionCallbacks(
                onDeleteRequested = ::deleteAnnotation,
                onMessage = { message, isError ->
                    syncMessage = message
                    syncMessageIsError = isError
                },
                onShowChrome = ::showChrome,
                onApplyDecorations = ::applyAnnotationDecorations,
                onSynchronize = ::synchronizeAnnotations,
                canReadAloud = { readAloud.available },
                onReadAloud = readAloud::readFromSelection,
            ),
        )

        // Back peels off the controls before it leaves the book.
        onBackPressedDispatcher.addCallback(this, object : androidx.activity.OnBackPressedCallback(true) {
            override fun handleOnBackPressed() {
                if (chromeVisible) hideChrome() else finish()
            }
        })

        val einkProgress = AppThemePreferences(this).load().palette == AppPalette.EInk
        readerSurface.mount(
            progressContent = {
                if (readerPreferences.hasPersistentFooter && progression.isFinite()) {
                    EpubPersistentProgress(
                        progression = progression,
                        chapterCount = chapters.count { it.depth == 0 },
                        theme = readerPreferences.theme,
                        eink = einkProgress,
                        statistics = readingStatistics,
                        progress = readerPreferences.progress,
                    )
                }
            },
            chromeContent = {
                EpubReaderChrome(
                    visible = chromeVisible,
                    title = title,
                    progression = progression.takeIf(Double::isFinite),
                    readingStatistics = readingStatistics,
                    preferences = readerPreferences,
                    customFonts = fontRuntime.availableFonts(),
                    onPreferences = { transform ->
                        val updated = preferencesStore.update(transform)
                        readerPreferences = updated
                        applyReaderSystemBars(updated.showSystemStatusBar)
                        lifecycleScope.launch {
                            fontRuntime.apply(navigator, updated)
                            textSharpener.apply(navigator, updated.sharpness)
                            navigator?.submitPreferences(updated.toReadium(fontRuntime.family(updated)))
                            applyAnnotationDecorations()
                        }
                    },
                    chapters = chapters,
                    onChapter = { chapter ->
                        navigator?.go(chapter.link, animated = false)
                        hideChrome()
                    },
                    searchState = searchState,
                    onSearch = ::search,
                    onSearchResult = { locator ->
                        navigator?.go(locator, animated = false)
                        searchState = EpubSearchState()
                        hideChrome()
                    },
                    bookmarks = bookmarks,
                    annotations = annotationSession.state.visible,
                    annotationConflicts = annotationSession.state.conflicts.values.toList(),
                    currentBookmarked = currentLocator?.let { current -> bookmarks.any { it.locator.bookmarkIdentifier() == current.bookmarkIdentifier() } } == true,
                    onToggleBookmark = {
                        currentLocator?.let { bookmarks = bookmarkStore.toggle(publicationKey, it) }
                    },
                    onBookmark = { bookmark ->
                        navigator?.go(bookmark.locator, animated = false)
                        hideChrome()
                    },
                    onAnnotation = { annotation ->
                        annotation.readiumLocator()?.let { navigator?.go(it, animated = false) }
                        hideChrome()
                    },
                    onEditAnnotationNote = { annotation, note -> editAnnotationNote(annotation.id, note) },
                    onDeleteAnnotation = { annotation -> deleteAnnotation(annotation.id) },
                    onKeepLocalAnnotation = { conflict -> resolveAnnotationConflict(conflict.submitted.id, keepLocal = true) },
                    onUseRemoteAnnotation = { conflict -> resolveAnnotationConflict(conflict.submitted.id, keepLocal = false) },
                    annotationNotesAvailable = annotationSession.notesAvailable,
                    annotationNotesExporting = annotationSession.notesExporting,
                    onExportAnnotationNotes = ::exportAnnotationsToNotes,
                    readingSyncStatus = readingSyncStatus,
                    syncMessage = syncMessage,
                    syncMessageIsError = syncMessageIsError,
                    onDismiss = ::hideChrome,
                    onClose = ::finish,
                    narrationControls = { readAloud.Controls { navigator?.currentLocator?.value ?: currentLocator } },
                )
            },
        )

        lifecycleScope.launch {
            try {
                openPublication(file, publicationKey, readerSurface.containerId)
            } catch (cancelled: CancellationException) {
                throw cancelled
            } catch (_: Throwable) {
                showError("This EPUB encountered a reader error. Your saved position was kept.")
            }
        }
    }

    override fun onWindowFocusChanged(hasFocus: Boolean) {
        super.onWindowFocusChanged(hasFocus)
        if (hasFocus) applyReaderSystemBars(readerPreferences.showSystemStatusBar)
    }

    private fun applyReaderSystemBars(showStatusBar: Boolean) {
        val controller = WindowCompat.getInsetsController(window, window.decorView)
        controller.systemBarsBehavior = WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
        controller.hide(WindowInsetsCompat.Type.navigationBars())
        if (showStatusBar) controller.show(WindowInsetsCompat.Type.statusBars())
        else controller.hide(WindowInsetsCompat.Type.statusBars())
    }

    override fun onKeyDown(keyCode: Int, event: KeyEvent): Boolean {
        if (chromeVisible && net.kdhmedia.mochil.reader.isFocusKey(keyCode)) return super.onKeyDown(keyCode, event)
        if (!chromeVisible && net.kdhmedia.mochil.reader.opensReaderControls(keyCode)) {
            if (event.repeatCount == 0) {
                showChrome()
                readerSurface.focusChrome()
            }
            return true
        }
        val turn = net.kdhmedia.mochil.reader.readerPageTurn(keyCode, hardwarePreferences.volumeButtonsEnabled()) ?: return super.onKeyDown(keyCode, event)
        if (event.repeatCount == 0) lifecycleScope.launch {
            if (turn == net.kdhmedia.mochil.reader.ReaderPageTurn.Next) navigator?.goForward(animated = false)
            else navigator?.goBackward(animated = false)
        }
        return true
    }

    override fun onKeyUp(keyCode: Int, event: KeyEvent): Boolean =
        if (net.kdhmedia.mochil.reader.readerPageTurn(keyCode, hardwarePreferences.volumeButtonsEnabled()) != null) true else super.onKeyUp(keyCode, event)

    private suspend fun openPublication(file: File, publicationKey: String, containerId: Int) {
        annotationSession.load()
        val opened = when (val loaded = EpubPublicationLoader(this).open(file)) {
            is EpubPublicationLoadResult.Loaded -> loaded.publication
            is EpubPublicationLoadResult.Failed -> {
                showError(loaded.reason.userMessage)
                return
            }
        }
        publication = opened
        chapters = flattenChapters(opened.tableOfContents)
        locatorStore = EpubLocatorStore(this)
        val credential = DeviceCredentialStore(this).load()
        val libraryId = intent.getStringExtra(EXTRA_LIBRARY_ID).orEmpty()
        val bookId = intent.getStringExtra(EXTRA_BOOK_ID).orEmpty()
        annotationSession.bindRemote(credential, libraryId, bookId)
        val externalTitleId = intent.getStringExtra(EXTRA_EXTERNAL_TITLE_ID).orEmpty()
        progressCoordinator = EpubProgressCoordinator(locatorStore, publicationKey, credential, libraryId, bookId, externalTitleId)
        readingSyncStatus = progressCoordinator.initialStatus()
        readAloud.setUp(credential, libraryId, bookId, publicationKey, intent.getStringExtra(EXTRA_TITLE).orEmpty())
        var initialLocator = locatorStore.load(publicationKey)
        if (initialLocator != null && opened.readingOrder.none { it.url() == initialLocator.href }) {
            initialLocator = null
            locatorStore.clear(publicationKey)
        }
        val initialProgress = progressCoordinator.resolveInitial(opened, initialLocator, { confirmRemotePosition(it) })
        initialLocator = initialProgress.locator
        applyProgressUpdate(initialProgress)
        progression = initialLocator?.locations?.totalProgression ?: Double.NaN
        supportFragmentManager.fragmentFactory = EpubNavigatorFactory(opened).createFragmentFactory(
            initialLocator = initialLocator,
            initialPreferences = readerPreferences.toReadium(fontRuntime.family(readerPreferences)),
            configuration = EpubNavigatorFragment.Configuration(
                decorationTemplates = HtmlDecorationTemplates.defaultTemplates(
                    alpha = 1.0,
                    experimentalPositioning = true,
                ),
            ),
        )
        supportFragmentManager.commit {
            replace(containerId, EpubNavigatorFragment::class.java, Bundle(), NAVIGATOR_TAG)
        }
        supportFragmentManager.executePendingTransactions()
        val epubNavigator = supportFragmentManager.findFragmentByTag(NAVIGATOR_TAG) as EpubNavigatorFragment
        navigator = epubNavigator
        currentLocator = initialLocator
        fontRuntime.apply(navigator, readerPreferences)
        textSharpener.apply(epubNavigator, readerPreferences.sharpness)
        annotationSession.repairStableLocators(opened)
        synchronizeAnnotations()
        applyAnnotationDecorations()
        epubNavigator.addInputListener(EpubTapZones({ navigator }, { readerPreferences.tapZones }, { !reduceMotion }, ::showChrome))
        if (reduceMotion) instantPages.apply(epubNavigator)
        locatorObserver = EpubLocatorObserver(opened, epubNavigator, publicationKey, locatorStore, readingStatisticsStore, progressCoordinator)
        locatorObserver?.start(
            scope = lifecycleScope,
            lifecycleOwner = this,
            initialStatus = readingSyncStatus,
            callbacks = EpubLocatorCallbacks(
                onImmediate = { locator, status ->
                    currentLocator = locator
                    readingSyncStatus = status
                },
                onSettled = { updatedProgression, statistics ->
                    progression = updatedProgression
                    readingStatistics = statistics
                },
                onProgressUpdate = ::applyProgressUpdate,
                onResourceChanged = {
                    awaitNavigatorFrame(epubNavigator.publicationView)
                    awaitNavigatorFrame(epubNavigator.publicationView)
                    fontRuntime.apply(navigator, readerPreferences)
                    applyAnnotationDecorations()
                },
            ),
        )
    }

    private fun applyProgressUpdate(update: EpubProgressUpdate) {
        readingSyncStatus = update.status
        syncMessage = update.message
        syncMessageIsError = update.messageIsError
    }

    override fun onActionModeStarted(mode: ActionMode) {
        super.onActionModeStarted(mode)
        selectionController.onStarted(mode, readerSurface.chromeHasFocus())
    }

    override fun onActionModeFinished(mode: ActionMode) {
        super.onActionModeFinished(mode)
        selectionController.onFinished()
    }

    private fun deleteAnnotation(id: String) {
        val annotation = annotationSession.state.items[id] ?: return
        val excerpt = annotation.quote.ifBlank { annotation.note }.trim().take(100)
        AlertDialog.Builder(this)
            .setTitle("Remove highlight?")
            .setMessage(if (excerpt.isBlank()) "This removes the highlight from synchronized devices." else "“$excerpt”\n\nThis removes the highlight from synchronized devices.")
            .setNegativeButton("Cancel", null)
            .setPositiveButton("Remove") { _, _ -> lifecycleScope.launch {
            val changed = annotationSession.delete(id)
            if (!changed) return@launch
            applyAnnotationDecorations()
            synchronizeAnnotations()
            } }
            .show()
    }

    private fun editAnnotationNote(id: String, note: String) {
        lifecycleScope.launch {
            val changed = annotationSession.editNote(id, note)
            if (!changed) return@launch
            synchronizeAnnotations()
        }
    }

    private fun resolveAnnotationConflict(id: String, keepLocal: Boolean) {
        lifecycleScope.launch {
            val result = annotationSession.resolveConflict(id, keepLocal)
            if (!result.changed) return@launch
            applyAnnotationDecorations()
            syncMessage = null
            syncMessageIsError = false
            if (result.synchronize) synchronizeAnnotations()
        }
    }

    private suspend fun applyAnnotationDecorations(): Boolean {
        val epubNavigator = navigator ?: return false
        val decorations = annotationSession.decorations(readerPreferences.theme)
        return runCatching {
            epubNavigator.applyDecorations(decorations, ANNOTATION_GROUP)
            true
        }.getOrElse {
            syncMessage = "The highlight was saved, but could not be drawn. Reopen the book to try again."
            syncMessageIsError = true
            false
        }
    }

    private fun synchronizeAnnotations() {
        val opened = publication ?: return
        if (!annotationSession.notesAvailable) return
        lifecycleScope.launch {
            try {
                val synchronized = annotationSession.synchronize(opened) ?: return@launch
                applyAnnotationDecorations()
                syncMessage = if (synchronized.conflicts.isEmpty()) null else "Some annotations changed on another device and need review."
                syncMessageIsError = false
            } catch (cancelled: CancellationException) {
                throw cancelled
            } catch (_: Throwable) {
                syncMessage = "Annotation sync is temporarily unavailable. Changes remain saved on this device."
                syncMessageIsError = true
            }
        }
    }

    private fun exportAnnotationsToNotes() {
        val opened = publication ?: return
        if (!annotationSession.notesAvailable || annotationSession.notesExporting) return
        lifecycleScope.launch {
            try {
                val export = annotationSession.exportToNotes(opened)
                if (export != null) {
                    applyAnnotationDecorations()
                    syncMessage = "Created Notes page “${export.page.take(120)}”."
                    syncMessageIsError = false
                }
            } catch (cancelled: CancellationException) {
                throw cancelled
            } catch (error: Throwable) {
                syncMessage = annotationNotesFailureMessage(error)
                syncMessageIsError = true
            }
        }
    }

    private fun validatedPublication(): File? {
        val root = File(filesDir, "publications")
        return validatedEpubPath(intent.getStringExtra(EXTRA_PATH), root)
    }

    private fun showError(message: String) {
        setContentView(TextView(this).apply {
            text = message
            gravity = android.view.Gravity.CENTER
            setPadding(32, 32, 32, 32)
        })
    }

    private fun showChrome() {
        chromeVisible = true
        readerSurface.showChrome()
    }

    private fun hideChrome() {
        chromeVisible = false
        readerSurface.hideChrome()
    }

    private fun flattenChapters(links: List<Link>, depth: Int = 0): List<EpubChapter> = buildList {
        if (depth > 16) return@buildList
        links.forEach { link ->
            link.title?.takeIf(String::isNotBlank)?.let { add(EpubChapter(it, depth, link)) }
            addAll(flattenChapters(link.children, depth + 1))
        }
    }

    private fun search(query: String) {
        if (query.isBlank()) {
            // Closing the sheet abandons whatever is still scanning: its result must not reopen it a few seconds later.
            searchGeneration.start()
            searchState = EpubSearchState()
            return
        }
        val opened = publication ?: return
        val token = searchGeneration.start()
        searchState = EpubSearchState(searching = true, query = query.trim())
        lifecycleScope.launch {
            runCatching { EpubSearch.find(opened, query) }
                .onSuccess { found -> if (searchGeneration.isCurrent(token)) searchState = EpubSearchState(query = query.trim(), results = found) }
                .onFailure { if (searchGeneration.isCurrent(token)) searchState = EpubSearchState(query = query.trim(), error = "Search could not be completed.") }
        }
    }

    override fun onPause() {
        if (::locatorStore.isInitialized) navigator?.currentLocator?.value?.let { latest ->
            currentLocator = latest
            locatorStore.saveNow(publicationKey, latest)
            locatorObserver?.let { observer ->
                flushScope.launch { kotlinx.coroutines.withTimeoutOrNull(FLUSH_TIMEOUT_MILLIS) { runCatching { observer.flush(latest) } } }
            }
        }
        super.onPause()
    }

    override fun onDestroy() {
        super.onDestroy()
        publication?.close()
        // No flushScope.cancel() here: onPause launches the position flush on this scope, so cancelling it as the
        // activity goes away threw away the one save that matters — the reader closed before the debounce fired, the
        // local position was already written by saveNow, and the server kept the old spot for the next device. The
        // flush is bounded by FLUSH_TIMEOUT_MILLIS and swallows its own failures, so letting it finish costs nothing.
    }

    companion object {
        const val EXTRA_PATH = "publication_path"
        const val EXTRA_TITLE = "publication_title"
        const val EXTRA_PROGRESS_KEY = "publication_progress_key"
        const val EXTRA_LIBRARY_ID = "library_id"
        const val EXTRA_EXTERNAL_TITLE_ID = "external_title_id"
        const val EXTRA_BOOK_ID = "book_id"
        private const val FLUSH_TIMEOUT_MILLIS = 8_000L
        private const val NAVIGATOR_TAG = "epub_navigator"
        private const val ANNOTATION_GROUP = "mochil-annotations"
    }
}
