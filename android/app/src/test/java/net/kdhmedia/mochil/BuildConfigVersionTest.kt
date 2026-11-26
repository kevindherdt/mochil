package net.kdhmedia.mochil

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The app's version is the release's version. It used to be two literals in Gradle that nobody bumped,
 * so every release after 0.1.20 shipped an APK that claimed to be 0.1.20 / 444: phones were told they
 * were up to date, the installer refused to replace a package with an equal versionCode, and a palette
 * removed from the server lived on in the installed app.
 */
class BuildConfigVersionTest {
    private val PREVIEW_SUFFIX = Regex("-preview(\\d+)$")

    /** The repository VERSION, found by walking up: a unit test's working directory is not the module's. */
    private fun repositoryVersion(): String {
        var directory: File? = File(System.getProperty("user.dir")).absoluteFile
        while (directory != null) {
            val candidate = File(directory, "VERSION")
            if (candidate.isFile) return candidate.readText().trim()
            directory = directory.parentFile
        }
        throw AssertionError("VERSION not found above ${System.getProperty("user.dir")}")
    }

    @Test
    fun `the app is labelled with the repository version`() {
        assertEquals(repositoryVersion(), BuildConfig.VERSION_NAME.replace(PREVIEW_SUFFIX, ""))
    }

    /**
     * 444 was the last hard-coded code, so every release up to 0.1.61 carries it. An upgrade needs a
     * code strictly above the installed one, so the derived code must clear that floor.
     */
    @Test
    fun `the version code can install over the releases that were all 444`() {
        assertTrue(
            "versionCode ${BuildConfig.VERSION_CODE} would be refused as a downgrade",
            BuildConfig.VERSION_CODE > 444,
        )
    }

    @Test
    fun `the version code rises with the version name`() {
        // A -PmochilPreview=N build is the release code plus N, and says so in its name.
        val preview = PREVIEW_SUFFIX.find(BuildConfig.VERSION_NAME)?.groupValues?.get(1)?.toInt() ?: 0
        val parts = BuildConfig.VERSION_NAME.replace(PREVIEW_SUFFIX, "").split(".").map { it.toInt() }
        val expected = (parts[0] * 10000 + parts[1] * 100 + parts[2]) * 100 + preview
        assertEquals(expected, BuildConfig.VERSION_CODE)
    }
}
