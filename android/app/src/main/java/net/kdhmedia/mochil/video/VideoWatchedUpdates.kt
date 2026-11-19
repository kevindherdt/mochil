package net.kdhmedia.mochil.video

/** Apply a completed mutation to its media identity, never to a newly opened show. */
internal fun VideoState.withConfirmedWatched(item: VideoItem, watched: Boolean): VideoState {
    if (libraryId != item.libraryId) return this
    val currentItem = items.firstOrNull { it.libraryId == item.libraryId && it.id == item.id }
    val delta = videoWatchedDelta(currentItem?.finished ?: item.finished, watched)
    fun VideoShow.updated(): VideoShow {
        val matches = sourceName == item.show || (sourceName == "Other videos" && item.show.isBlank())
        return if (matches) copy(watchedCount = (watchedCount + delta).coerceIn(0, count)) else this
    }
    return copy(
        items = items.map { value ->
            if (value.libraryId == item.libraryId && value.id == item.id) value.withWatched(watched) else value
        },
        selectedShow = selectedShow?.updated(),
        shows = shows.map { it.updated() },
        continueWatching = continueWatching.filterNot { it.libraryId == item.libraryId && it.id == item.id },
        error = null,
    )
}
