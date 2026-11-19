package net.kdhmedia.mochil

import java.io.File
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ManifestPrivacyContractTest {
    @Test fun mainManifestKeepsPlaybackPrivateAndPermissionMinimal() {
        val manifest = sequenceOf(
            File("src/main/AndroidManifest.xml"),
            File("app/src/main/AndroidManifest.xml"),
        ).firstOrNull(File::isFile) ?: error("Android manifest was not found")
        val source = manifest.readText()

        assertFalse(source.contains("android.permission.POST_NOTIFICATIONS"))
        assertFalse(source.contains("android.permission.READ_EXTERNAL_STORAGE"))
        assertFalse(source.contains("android.permission.MANAGE_EXTERNAL_STORAGE"))
        assertTrue(source.contains("android.permission.FOREGROUND_SERVICE_MEDIA_PLAYBACK"))
        assertTrue(source.contains("android:permission=\"android.permission.BIND_MEDIA_BROWSER_SERVICE\""))
        assertTrue(source.contains("android:name=\".reader.comic.ComicReaderActivity\"\n            android:exported=\"false\""))
        assertTrue(source.contains("android:allowBackup=\"false\""))
        // Cleartext policy moved from the manifest into the app, because most installs are reached at a
        // private address (home LAN or Tailscale) and the manifest can only allow or forbid all of it.
        // What must stay true: the release config is the one declared, it carries no debug overrides,
        // and the code still refuses plain http for a public host.
        assertTrue(source.contains("android:networkSecurityConfig=\"@xml/network_security_config\""))
        val policy = sequenceOf(
            File("src/main/res/xml/network_security_config.xml"),
            File("app/src/main/res/xml/network_security_config.xml"),
        ).firstOrNull(File::isFile) ?: error("release network security config was not found")
        val policySource = policy.readText()
        assertTrue(policySource.contains("cleartextTrafficPermitted=\"true\""))
        assertFalse(policySource.contains("debug-overrides"))
        val addressRule = sequenceOf(
            File("src/main/java/net/kdhmedia/mochil/auth/ServerAddress.kt"),
            File("app/src/main/java/net/kdhmedia/mochil/auth/ServerAddress.kt"),
        ).firstOrNull(File::isFile) ?: error("ServerAddress.kt was not found")
        val addressSource = addressRule.readText()
        assertTrue(addressSource.contains("isPrivateHost"))
        assertTrue(addressSource.contains("https"))
        assertTrue(
            Regex("androidx\\.core\\.content\\.FileProvider[\\s\\S]{0,300}android:exported=\\\"false\\\"")
                .containsMatchIn(source),
        )
    }

    @Test fun distributableSourcesContainNoPersonalDeploymentHostname() {
        val root = sequenceOf(File("src/main"), File("app/src/main")).firstOrNull(File::isDirectory)
            ?: error("Android sources were not found")
        val personalHost = Regex("(?i)([a-z0-9-]+\\.)*kdhmedia\\.net")
        val textExtensions = setOf("kt", "xml", "json", "properties", "pro")
        val matches = root.walkTopDown().filter { it.isFile && it.extension in textExtensions }.mapNotNull { file ->
            file.readText().takeIf(personalHost::containsMatchIn)?.let { file.path }
        }.toList()

        assertTrue("Personal deployment hostname embedded at: ${matches.joinToString()}", matches.isEmpty())
    }

    @Test fun authenticatedHttpConnectionsNeverAutomaticallyFollowRedirects() {
        val root = sequenceOf(File("src/main/java"), File("app/src/main/java")).firstOrNull(File::isDirectory)
            ?: error("Android sources were not found")
        val unsafe = root.walkTopDown().filter { it.extension == "kt" }.flatMap { file ->
            val lines = file.readLines()
            lines.asSequence().mapIndexedNotNull { index, line ->
                if (!line.contains("setRequestProperty(\"Authorization\", \"Bearer")) return@mapIndexedNotNull null
                val setup = lines.subList((index - 8).coerceAtLeast(0), index + 1).joinToString("\n")
                if (setup.contains("instanceFollowRedirects = false")) null else "${file.path}:${index + 1}"
            }
        }.toList()

        assertTrue("Bearer credentials could follow a redirect at: ${unsafe.joinToString()}", unsafe.isEmpty())
    }
}
