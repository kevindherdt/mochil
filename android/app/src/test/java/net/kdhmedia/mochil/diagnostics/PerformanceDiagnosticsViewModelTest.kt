package net.kdhmedia.mochil.diagnostics

import org.junit.Assert.assertTrue
import org.junit.Test

class PerformanceDiagnosticsViewModelTest {
    @Test fun uploadErrorsExplainTheRecoveryWithoutExposingResponseBodies() {
        assertTrue(diagnosticUploadMessage(DiagnosticUploadException(400)).contains("Clear"))
        assertTrue(diagnosticUploadMessage(DiagnosticUploadException(401)).contains("Reconnect"))
        assertTrue(diagnosticUploadMessage(DiagnosticUploadException(429)).contains("Wait"))
        assertTrue(diagnosticUploadMessage(java.io.IOException()).contains("reach"))
    }
}
