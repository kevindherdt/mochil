package net.kdhmedia.mochil.reader.epub

import org.json.JSONObject
import org.readium.r2.shared.publication.Locator
import org.readium.r2.shared.publication.Publication

object RemotePositionResolver {
    suspend fun resolve(publication: Publication, position: ServerReadingPosition): Locator? {
        position.locators["readium"]?.let { encoded ->
            runCatching { Locator.fromJSON(JSONObject(encoded)) }.getOrNull()
                ?.takeIf { locator -> publication.readingOrder.any { it.url() == locator.href } }
                ?.let { return it }
        }
        resolveKoReaderLocator(publication, position)?.let { return it }
        val anchor = position.anchor
        if (anchor != null && anchor.exact.isNotBlank()) {
            EpubSearch.find(publication, anchor.exact, maximumResults = 50)
                .maxByOrNull { candidate -> score(publication, candidate, anchor) }
                ?.let { return it }
        }
        return approximate(publication, position)
    }

    private fun resolveKoReaderLocator(publication: Publication, position: ServerReadingPosition): Locator? {
        val parsed = KoReaderXPointer.parse(position.koReaderLocator) ?: return null
        val resource = publication.readingOrder.getOrNull(parsed.spineIndex) ?: return null
        val base = publication.locatorFromLink(resource) ?: return null
        val anchor = position.anchor
        val locations = base.locations.otherLocations + ("cssSelector" to parsed.cssSelector)
        val structural = base.copyWithLocations(
            progression = anchor?.progress?.takeIf(Double::isFinite)?.coerceIn(0.0, 1.0),
            totalProgression = position.fraction.takeIf(Double::isFinite)?.coerceIn(0.0, 1.0),
            otherLocations = locations,
        )
        if (anchor == null || anchor.exact.isBlank()) return structural
        return structural.copy(text = Locator.Text(anchor.prefix, anchor.exact, anchor.suffix))
    }

    private fun approximate(publication: Publication, position: ServerReadingPosition): Locator? {
        val resources = publication.readingOrder
        if (resources.isEmpty() || !position.fraction.isFinite()) return null
        val bounded = position.fraction.coerceIn(0.0, 1.0)
        val scaled = bounded * resources.size
        val index = position.anchor?.spineIndex?.takeIf { it in resources.indices }
            ?: scaled.toInt().coerceIn(resources.indices)
        val progression = if (position.anchor?.spineIndex == index) position.anchor.progress.coerceIn(0.0, 1.0)
            else (scaled - index).coerceIn(0.0, 1.0)
        return publication.locatorFromLink(resources[index])?.copyWithLocations(
            progression = progression,
            totalProgression = bounded,
        )
    }

    private fun score(publication: Publication, candidate: Locator, anchor: ServerTextAnchor): Int {
        var score = 0
        val candidateSpine = publication.readingOrder.indexOfFirst { it.url() == candidate.href }
        if (candidateSpine == anchor.spineIndex) score += 100
        if (candidate.text.highlight == anchor.exact) score += 80
        if (anchor.prefix.isNotBlank() && candidate.text.before.orEmpty().endsWith(anchor.prefix.takeLast(80))) score += 30
        if (anchor.suffix.isNotBlank() && candidate.text.after.orEmpty().startsWith(anchor.suffix.take(80))) score += 30
        return score
    }
}
