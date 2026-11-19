package net.kdhmedia.mochil.playback

import java.io.InputStream
import net.kdhmedia.mochil.network.readBoundedResponseText
import org.json.JSONObject

// A single position/revision record, not a catalogue or progress-history listing.
internal const val MAX_PLAYBACK_PROGRESS_RESPONSE_BYTES = 64 * 1024

internal fun InputStream.readPlaybackProgressResponse(): JSONObject =
    JSONObject(readBoundedResponseText(MAX_PLAYBACK_PROGRESS_RESPONSE_BYTES))
