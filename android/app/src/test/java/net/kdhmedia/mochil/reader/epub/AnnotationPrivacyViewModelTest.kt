package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class AnnotationPrivacyViewModelTest {
    @Test fun expiredSessionNamesRecoveryForEachAction() {
        assertEquals("Your session expired. Sign in again before trying to export annotations.", annotationPrivacyFailure(401, "export"))
        assertEquals("Your session expired. Sign in again before trying to delete annotations.", annotationPrivacyFailure(401, "delete"))
    }

    @Test fun exportAndDeletionFailuresStayActionable() {
        assertEquals("There are no annotations to export yet.", annotationPrivacyFailure(409, "export"))
        assertTrue(annotationPrivacyFailure(null, "delete").contains("Check the connection"))
    }
}
