package net.kdhmedia.mochil.video

import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

class VideoTelevisionDetectionTest {
    @Test fun videoPlayerUsesTheSameDevicePolicyAsMainActivity() {
        val source = File("src/main/java/net/kdhmedia/mochil/video/VideoPlayerActivity.kt").readText()
        assertTrue("Video player must use the shared TV device policy, including Leanback and debug preview", source.contains("val television = isTelevisionDevice(this)"))
    }
}
