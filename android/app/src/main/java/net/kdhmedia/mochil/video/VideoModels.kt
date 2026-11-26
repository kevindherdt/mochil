package net.kdhmedia.mochil.video

data class VideoSubtitle(val index: Int, val language: String, val title: String, val default: Boolean)

data class VideoItem(
    val id: String,
    val libraryId: String,
    val title: String,
    val show: String,
    val season: Int,
    val episode: Int,
    val durationSeconds: Double,
    val artworkPath: String,
    val streamPath: String,
    val hlsPath: String,
    val resumeSeconds: Double,
    val subtitles: List<VideoSubtitle>,
    val dismissed: Boolean = false,
    val finished: Boolean = false,
    val progressRevision: Long = 0,
    val episodeTitle: String = "",
    val revision: String = "",
    val year: Int = 0,
    val overview: String = "",
    val backdropPath: String = "",
    /** The tallest video stream's height, from the server's probe; null when it recorded none. */
    val videoHeight: Int? = null,
    /** The file plays as it is, so the card can say so instead of offering a guess. */
    val directPlay: Boolean = false,
) {
    val displayTitle: String get() = episodeTitle.ifBlank { title }
    val playable: Boolean get() = streamPath.isNotBlank() || hlsPath.isNotBlank()
    val progressFraction: Float get() = if (durationSeconds > 0) (resumeSeconds / durationSeconds).toFloat().coerceIn(0f, 1f) else 0f
}

data class VideoShow(
    val name: String,
    val sourceName: String,
    val itemId: String,
    val count: Int,
    val seasons: List<Int>,
    val artworkPath: String,
    val watchedCount: Int = 0,
    val inProgressCount: Int = 0,
    val year: Int = 0,
    val overview: String = "",
    val backdropPath: String = "",
) {
    val watchFraction: Float get() = if (count > 0) (watchedCount.toFloat() / count).coerceIn(0f, 1f) else 0f
    val completed: Boolean get() = count > 0 && watchedCount >= count
    val watchLabel: String get() = when {
        completed -> "Played"
        watchedCount > 0 -> "$watchedCount of $count played"
        inProgressCount > 0 -> "In progress"
        else -> "$count episodes"
    }
}

/** What TMDB adds about a whole show or movie: genres, minutes per episode or film, age rating and score. */
data class VideoCastMember(val name: String, val character: String = "", val portraitPath: String = "")
data class VideoTitleDetails(val genres: List<String> = emptyList(), val runtimeMinutes: Int = 0, val rating: Double = 0.0, val certification: String = "", val cast: List<VideoCastMember> = emptyList())

data class VideoState(
    val libraryId: String = "",
    val libraryKind: String = "",
    val items: List<VideoItem> = emptyList(),
    val shows: List<VideoShow> = emptyList(),
    val selectedShow: VideoShow? = null,
    val selectedSeason: Int? = null,
    val continueWatching: List<VideoItem> = emptyList(),
    val query: String = "",
    val searchOpen: Boolean = false,
    val layout: VideoLayout = VideoLayout.Covers,
    val loading: Boolean = false,
    val showLoadingMore: Boolean = false,
    val showNextOffset: Int? = null,
    val showPagingError: String? = null,
    val itemLoadingMore: Boolean = false,
    val itemNextOffset: Int? = null,
    val itemPagingError: String? = null,
    val error: String? = null,
    val details: Map<String, VideoTitleDetails> = emptyMap(),
) {
    private val showingShows: Boolean get() = selectedShow == null && libraryKind in setOf("tv", "anime")
    val loadingMore: Boolean get() = if (showingShows) showLoadingMore else itemLoadingMore
    val nextOffset: Int? get() = if (showingShows) showNextOffset else itemNextOffset
    val pagingError: String? get() = if (showingShows) showPagingError else itemPagingError
}
