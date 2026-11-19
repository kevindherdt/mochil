package net.kdhmedia.mochil.wiki

import android.content.Context
import java.io.File
import kotlinx.coroutines.CoroutineScope
import net.kdhmedia.mochil.catalogue.PublicationOpener

internal fun createWikiPdfOpening(context: Context, scope: CoroutineScope, status: (String?) -> Unit, failure: (String) -> Unit): WikiPdfOpening {
    val app = context.applicationContext
    val downloads = WikiPdfDownloads(File(app.filesDir, "publications/wiki-pdf-attachments"))
    val opener = PublicationOpener(app)
    return WikiPdfOpening(scope, downloads::prepare, { file, entry, progressKey ->
        app.startActivity(opener.prepareFile(file, "application/pdf", entry.label, progressKey, "pdf"))
    }, status, failure)
}
