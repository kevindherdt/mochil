package net.kdhmedia.mochil.reader.epub

import android.content.Context
import java.io.File
import org.readium.r2.shared.publication.Publication
import org.readium.r2.shared.util.asset.AssetRetriever
import org.readium.r2.shared.util.getOrElse
import org.readium.r2.shared.util.http.DefaultHttpClient
import org.readium.r2.streamer.PublicationOpener
import org.readium.r2.streamer.parser.DefaultPublicationParser

sealed interface EpubPublicationLoadResult {
    data class Loaded(val publication: Publication) : EpubPublicationLoadResult
    data class Failed(val reason: EpubPublicationFailure) : EpubPublicationLoadResult
}

enum class EpubPublicationFailure(val userMessage: String) {
    Retrieve("This EPUB could not be opened."),
    Parse("This EPUB could not be read."),
}

/** Retrieves and parses one local EPUB without leaking Readium setup into the Activity. */
class EpubPublicationLoader(private val context: Context) {
    suspend fun open(file: File): EpubPublicationLoadResult {
        val httpClient = DefaultHttpClient()
        val retriever = AssetRetriever(context.contentResolver, httpClient)
        val opener = PublicationOpener(DefaultPublicationParser(context, httpClient, retriever, pdfFactory = null))
        val asset = retriever.retrieve(file).getOrElse {
            return EpubPublicationLoadResult.Failed(EpubPublicationFailure.Retrieve)
        }
        val publication = opener.open(asset, allowUserInteraction = true).getOrElse {
            return EpubPublicationLoadResult.Failed(EpubPublicationFailure.Parse)
        }
        return EpubPublicationLoadResult.Loaded(publication)
    }
}
