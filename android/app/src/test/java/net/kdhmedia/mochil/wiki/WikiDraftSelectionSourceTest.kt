package net.kdhmedia.mochil.wiki

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertTrue
import org.junit.Test

class WikiDraftSelectionSourceTest {
    @Test fun `page selection preserves only a matching startup draft and clears discarded work`() {
        val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/wiki/WikiViewModel.kt"))
        val body = source.substringAfter("fun select(entry: WikiEntry)").substringBefore("fun edit(enabled: Boolean)")
        assertTrue(body.indexOf("draftSaveJob?.cancel()") in 0 until body.indexOf("pageLoadJob?.cancel()"))
        assertTrue(body.contains("val mayRecoverDraft = mutableState.value.page == null"))
        assertTrue(body.contains("retainForInitialSelection(null, libraryId, entry.id)"))
        assertTrue(body.contains("if (!mayRecoverDraft) draftStore.clear()"))
        assertTrue(body.contains("if (mayRecoverDraft && recoveryCandidate == null)"))
        assertTrue(body.contains("recoveryCandidate.recoverFor(libraryId, page)"))
    }
}
