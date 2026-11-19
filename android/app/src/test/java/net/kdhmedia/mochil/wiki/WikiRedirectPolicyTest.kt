package net.kdhmedia.mochil.wiki

import java.net.URI
import net.kdhmedia.mochil.network.isSafeAndroidApiRedirect
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class WikiRedirectPolicyTest {
    @Test fun permitsOnlyCanonicalRedirectsInsideTheSameAndroidApi() {
        val request = URI("http://books.example/api/android/v1/documents?library=wiki")
        assertTrue(isSafeAndroidApiRedirect(request, URI("https://books.example/api/android/v1/documents?library=wiki")))
        assertFalse(isSafeAndroidApiRedirect(request, URI("https://books.example/login")))
        assertFalse(isSafeAndroidApiRedirect(request, URI("https://other.example/api/android/v1/documents")))
    }
}
