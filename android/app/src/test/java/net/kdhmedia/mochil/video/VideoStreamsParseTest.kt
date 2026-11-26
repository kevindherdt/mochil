package net.kdhmedia.mochil.video

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The Android videos route forwards to the web's `/api/videos`, whose item already carries the probe (`video_streams[]`,
 * `direct_play`) — internal/video/item.go embeds those in the view. The client used to drop them, which is why a film card
 * had no resolution to print. This pins the parse against the real shape.
 */
class VideoStreamsParseTest {
    private fun item(extra: String) = VideoClient().video(JSONObject("""{"id":"one","library_id":"movies","title":"One"$extra}"""))

    @Test fun `the tallest probed height is kept, and direct play read`() {
        val parsed = item(""","direct_play":true,"video_streams":[{"index":0,"codec":"hevc","width":1280,"height":720},{"index":1,"codec":"h264","height":1080}]""")
        // The first stream with a height is the one the demuxer plays, as on the web.
        assertEquals(720, parsed.videoHeight)
        assertTrue(parsed.directPlay)
    }

    @Test fun `a file with no probe has no height and does not claim direct play`() {
        val parsed = item(""","container":"mkv"""")
        assertNull(parsed.videoHeight)
        assertFalse(parsed.directPlay)
    }

    @Test fun `a stream without a height does not count as one`() {
        val parsed = item(""","video_streams":[{"index":0,"codec":"hevc"},{"index":1,"height":0}]""")
        assertNull(parsed.videoHeight)
    }
}
