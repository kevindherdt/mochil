package net.kdhmedia.mochil.opds

import java.io.ByteArrayInputStream
import javax.xml.parsers.DocumentBuilderFactory
import org.w3c.dom.Element

object OpdsParser {
    fun parse(bytes: ByteArray): OpdsFeed {
        require(bytes.size in 1..MAX_FEED_BYTES) { "The OPDS response is empty or too large." }
        val factory = DocumentBuilderFactory.newInstance().apply {
            isNamespaceAware = true
            setFeature("http://apache.org/xml/features/disallow-doctype-decl", true)
            setFeature("http://xml.org/sax/features/external-general-entities", false)
            setFeature("http://xml.org/sax/features/external-parameter-entities", false)
            setAttribute("http://javax.xml.XMLConstants/property/accessExternalDTD", "")
            setAttribute("http://javax.xml.XMLConstants/property/accessExternalSchema", "")
        }
        val root = factory.newDocumentBuilder().parse(ByteArrayInputStream(bytes)).documentElement
        require(root.localName == "feed") { "This address did not return an OPDS Atom feed." }
        val entries = root.children("entry").take(MAX_ENTRIES).map { entry ->
            OpdsEntry(
                id = entry.text("id").ifBlank { entry.text("title") }.take(MAX_TEXT),
                title = entry.text("title").ifBlank { "Untitled" }.take(MAX_TEXT),
                authors = entry.children("author").map { it.text("name").take(MAX_TEXT) }.filter(String::isNotBlank).take(MAX_AUTHORS),
                links = entry.links(),
            )
        }
        return OpdsFeed(
            id = root.text("id").take(MAX_TEXT),
            title = root.text("title").ifBlank { "OPDS catalogue" }.take(MAX_TEXT),
            entries = entries,
            navigation = root.links(),
        )
    }

    private fun Element.links() = children("link").take(MAX_LINKS).mapNotNull { link ->
        val href = link.getAttribute("href").trim().take(MAX_URL)
        if (href.isBlank()) null else OpdsLink(link.getAttribute("rel").trim().take(MAX_RELATION), href, link.getAttribute("type").trim().take(MAX_MEDIA_TYPE))
    }
    private fun Element.text(name: String): String = children(name).firstOrNull()?.textContent?.trim().orEmpty()
    private fun Element.children(name: String): List<Element> = buildList {
        val values = childNodes
        for (index in 0 until values.length) (values.item(index) as? Element)?.takeIf { it.localName == name }?.let(::add)
    }

    const val MAX_FEED_BYTES = 4 * 1024 * 1024
    private const val MAX_ENTRIES = 500
    private const val MAX_LINKS = 100
    private const val MAX_AUTHORS = 20
    private const val MAX_TEXT = 500
    private const val MAX_URL = 4096
    private const val MAX_RELATION = 500
    private const val MAX_MEDIA_TYPE = 160
}
