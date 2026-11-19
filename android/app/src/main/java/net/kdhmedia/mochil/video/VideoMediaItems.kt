package net.kdhmedia.mochil.video

import android.net.Uri
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.MimeTypes
import net.kdhmedia.mochil.auth.DeviceCredential

/** Converts catalogue playback entries into Media3 items and attached text tracks. */
internal fun VideoPlaybackEntry.toVideoMediaItem(
    credential: DeviceCredential?,
    sourceUrl: String = url,
): MediaItem = MediaItem.Builder()
    .setUri(sourceUrl)
    .setMediaId(videoPlaybackIdentityKey(libraryId, itemId))
    .setSubtitleConfigurations(subtitleConfigurations(credential))
    .build()

private fun VideoPlaybackEntry.subtitleConfigurations(
    credential: DeviceCredential?,
): List<MediaItem.SubtitleConfiguration> {
    credential ?: return emptyList()
    return subtitles.lineSequence().mapNotNull { encoded ->
        val parts = encoded.split('\t')
        val index = parts.getOrNull(0)?.toIntOrNull() ?: return@mapNotNull null
        MediaItem.SubtitleConfiguration.Builder(
            Uri.parse(credential.serverUrl + VideoClient().subtitlePath(libraryId, itemId, index)),
        )
            .setMimeType(MimeTypes.TEXT_VTT)
            .setLanguage(parts.getOrNull(1)?.takeIf(String::isNotBlank))
            .setLabel(parts.getOrNull(2)?.takeIf(String::isNotBlank))
            .setSelectionFlags(if (parts.getOrNull(3).toBoolean()) C.SELECTION_FLAG_DEFAULT else 0)
            .build()
    }.toList()
}
