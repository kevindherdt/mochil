package net.kdhmedia.mochil.audio

import org.json.JSONArray
import org.json.JSONObject

/** Complete, version-tolerant metadata codec for an offline audio file. */
internal object AudioDownloadMetadata {
    private const val MAX_CHAPTERS = 10_000

    fun encode(track: AudioTrack) = JSONObject().apply {
        put("id", track.id); put("library_id", track.libraryId); put("title", track.title)
        put("artist", track.artist); put("album", track.album); put("duration_seconds", track.durationSeconds)
        put("media_type", track.mediaType); put("replaygain_track_db", track.replayTrackDb)
        put("replaygain_album_db", track.replayAlbumDb); put("replaygain_track_peak", track.replayTrackPeak)
        put("replaygain_album_peak", track.replayAlbumPeak); put("series", track.series)
        put("series_index", track.seriesIndex); put("narrator", track.narrator); put("library_kind", track.libraryKind)
        put("resume_position_seconds", track.resumePositionSeconds); put("finished", track.finished)
        put("feed_id", track.feedId); put("description", track.description)
        put("published_at_millis", track.publishedAtMillis); put("size_bytes", track.sizeBytes)
        put("album_artist", track.albumArtist); put("genre", track.genre); put("track_number", track.trackNumber)
        put("disc_number", track.discNumber); put("container", track.container); put("year", track.year)
        put("bitrate", track.bitrate); put("codec", track.codec); put("sample_rate", track.sampleRate)
        put("bit_depth", track.bitDepth); put("playback_mode", track.playbackMode)
        put("playback_quality", track.playbackQuality); put("stream_path", track.streamPath)
        put("artwork_path", track.artworkPath); put("server_downloaded", track.serverDownloaded)
        put("chapters", JSONArray().apply {
            track.chapters.take(MAX_CHAPTERS).forEach { chapter ->
                put(JSONObject().put("title", chapter.title).put("start_seconds", chapter.startSeconds).put("end_seconds", chapter.endSeconds))
            }
        })
    }

    fun decode(value: JSONObject): AudioTrack = value.run {
        AudioTrack(
            getString("id"), getString("library_id"), optString("title", "Untitled track"),
            optString("artist", "Unknown artist"), optString("album"), optDouble("duration_seconds"),
            optString("media_type", "audio/mpeg"), optionalFiniteDouble("replaygain_track_db"),
            optionalFiniteDouble("replaygain_album_db"), optionalFiniteDouble("replaygain_track_peak"),
            optionalFiniteDouble("replaygain_album_peak"), series = optString("series"),
            seriesIndex = optionalFiniteDouble("series_index") ?: 0.0, narrator = optString("narrator"),
            libraryKind = optString("library_kind"), resumePositionSeconds = optionalFiniteDouble("resume_position_seconds") ?: 0.0,
            finished = optBoolean("finished"), feedId = optString("feed_id"), description = optString("description"),
            publishedAtMillis = optLong("published_at_millis"), sizeBytes = optLong("size_bytes"),
            chapters = optJSONArray("chapters")?.let { values ->
                List(minOf(values.length(), MAX_CHAPTERS)) { index -> values.getJSONObject(index).run {
                    AudioChapter(optString("title", "Chapter ${index + 1}"), optDouble("start_seconds"), optDouble("end_seconds"))
                } }
            }.orEmpty(),
            albumArtist = optString("album_artist"), genre = optString("genre"), trackNumber = optInt("track_number"),
            discNumber = optInt("disc_number"), container = optString("container"),
            year = optInt("year").takeIf { it in 1000..2100 } ?: 0,
            bitrate = optLong("bitrate").takeIf { it in 1_000..1_000_000_000 } ?: 0,
            codec = optString("codec"), sampleRate = optInt("sample_rate").coerceAtLeast(0),
            bitDepth = optInt("bit_depth").coerceAtLeast(0), playbackMode = optString("playback_mode"),
            playbackQuality = optString("playback_quality"), streamPath = optString("stream_path"),
            artworkPath = optString("artwork_path"), serverDownloaded = optBoolean("server_downloaded"),
        )
    }
}
