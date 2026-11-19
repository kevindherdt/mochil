package net.kdhmedia.mochil.opds

import java.io.ByteArrayInputStream
import javax.xml.parsers.DocumentBuilderFactory
import org.w3c.dom.Element

internal object OpdsSearchParser {
    fun parseTemplate(bytes: ByteArray): String {
        require(bytes.size in 1..OpdsParser.MAX_FEED_BYTES) { "The OPDS search description is empty or too large." }
        val factory = DocumentBuilderFactory.newInstance().apply {
            isNamespaceAware = true
            setFeature("http://apache.org/xml/features/disallow-doctype-decl", true)
            setFeature("http://xml.org/sax/features/external-general-entities", false)
            setFeature("http://xml.org/sax/features/external-parameter-entities", false)
            setAttribute("http://javax.xml.XMLConstants/property/accessExternalDTD", "")
            setAttribute("http://javax.xml.XMLConstants/property/accessExternalSchema", "")
        }
        val root = factory.newDocumentBuilder().parse(ByteArrayInputStream(bytes)).documentElement
        require(root.localName == "OpenSearchDescription") { "The catalogue returned an invalid search description." }
        val urls = root.getElementsByTagNameNS("*", "Url")
        for (index in 0 until urls.length) {
            val element = urls.item(index) as? Element ?: continue
            val type = element.getAttribute("type").lowercase()
            val template = element.getAttribute("template").trim()
            if (type.contains("atom+xml") && template.contains("{searchTerms")) return template.take(4096)
        }
        error("This catalogue does not provide an OPDS search template.")
    }
}
