package net.kdhmedia.mochil.update

import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Android's installer answers "App not installed" for a file that is not an APK, a downgrade and a
 * different signing key alike. This was hit for real: the server was advertising build 438 while
 * still serving an older file, and the only symptom was that message. These cases pin the checks
 * that turn each cause into something the person holding the phone can act on.
 */
class UpdateRejectionTest {

    private val ours = "net.kdhmedia.mochil".toByteArray()
    private fun apk(version: Long = 439, pkg: String = "net.kdhmedia.mochil", sig: ByteArray? = ours) =
        DownloadedApk(pkg, version, sig)

    @Test
    fun aNewerBuildOfTheSameAppIsAccepted() {
        assertNull(updateRejection("net.kdhmedia.mochil", 438, ours, apk(439)))
    }

    @Test
    fun theSameBuildIsRefused() {
        val message = updateRejection("net.kdhmedia.mochil", 438, ours, apk(438))
        assertNotNull(message)
        assertTrue("should mention both builds: $message", message!!.contains("438") && message.contains("not newer"))
    }

    @Test
    fun anOlderBuildIsRefusedAndNamed() {
        // The real failure: the server keeps advertising its own build number while the file it hands
        // out is whatever was mounted months ago.
        val message = updateRejection("net.kdhmedia.mochil", 438, ours, apk(428))
        assertNotNull(message)
        assertTrue("should name the offered build: $message", message!!.contains("428"))
        assertTrue("should say the file is out of date: $message", message.contains("out of date"))
    }

    @Test
    fun aFileThatIsNotAnApkIsRefused() {
        val message = updateRejection("net.kdhmedia.mochil", 438, ours, null)
        assertNotNull(message)
        assertTrue("should blame the server's file: $message", message!!.contains("not an Android app"))
    }

    @Test
    fun anotherAppsPackageIsRefused() {
        val message = updateRejection("net.kdhmedia.mochil", 438, ours, apk(439, pkg = "com.example.other"))
        assertNotNull(message)
        assertTrue("should name the stranger: $message", message!!.contains("com.example.other"))
    }

    @Test
    fun aDifferentSigningKeyIsRefused() {
        val message = updateRejection("net.kdhmedia.mochil", 438, ours, apk(439, sig = "other-key".toByteArray()))
        assertNotNull(message)
        assertTrue("should mention signing: $message", message!!.contains("different key"))
    }

    @Test
    fun anUnknownSignatureFallsBackToVersionChecksOnly() {
        // getPackageArchiveInfo cannot always report certificates; a missing signature must not block
        // a genuine update, because Android will still refuse it at install time if the key is wrong.
        assertNull(updateRejection("net.kdhmedia.mochil", 438, null, apk(439, sig = null)))
    }
}
