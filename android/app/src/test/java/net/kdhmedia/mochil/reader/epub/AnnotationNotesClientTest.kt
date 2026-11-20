package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertTrue
import org.junit.Test

class AnnotationNotesClientTest {
    @Test fun failuresExplainRecoveryWithoutServerDetails() {
        assertTrue(annotationNotesFailureMessage(AnnotationNotesException(401)).contains("Reconnect"))
        assertTrue(annotationNotesFailureMessage(AnnotationNotesException(409)).contains("Resolve"))
        assertTrue(annotationNotesFailureMessage(AnnotationNotesException(413)).contains("too large"))
        assertTrue(annotationNotesFailureMessage(AnnotationNotesException(503)).contains("not enabled"))
        assertTrue(annotationNotesFailureMessage(java.io.IOException("private address")).contains("try again"))
    }
}
