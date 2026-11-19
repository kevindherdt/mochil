package net.kdhmedia.mochil.reader.epub

import kotlin.math.abs
import org.readium.r2.shared.publication.Locator

private val highlightWhitespace = Regex("\\s+")
private const val EQUIVALENT_POSITION_TOLERANCE = 0.002

internal fun stableHighlightQuery(value: String): String = value.trim().replace(highlightWhitespace, " ")

internal fun SyncedAnnotation.matchesSelection(selected: Locator): Boolean {
    if (deleted || kind != "highlight") return false
    val selectedText = stableHighlightQuery(selected.text.highlight.orEmpty())
    if (selectedText.isBlank() || stableHighlightQuery(quote) != selectedText) return false
    val saved = readiumLocator() ?: return false
    return highlightRangesMatch(
        saved.href.toString(), saved.locations.progression, saved.text.before.orEmpty(), saved.text.after.orEmpty(),
        selected.href.toString(), selected.locations.progression, selected.text.before.orEmpty(), selected.text.after.orEmpty(),
    )
}

internal fun highlightRangesMatch(savedHref: String, savedProgress: Double?, savedBefore: String, savedAfter: String, selectedHref: String, selectedProgress: Double?, selectedBefore: String, selectedAfter: String): Boolean {
    if (savedHref != selectedHref) return false
    if (savedProgress != null && selectedProgress != null && abs(savedProgress - selectedProgress) > 0.01) return false
    val before = stableHighlightQuery(selectedBefore)
    val after = stableHighlightQuery(selectedAfter)
    val referenceBefore = stableHighlightQuery(savedBefore)
    val referenceAfter = stableHighlightQuery(savedAfter)
    return (before.isBlank() || referenceBefore.isBlank() || before.endsWith(referenceBefore.takeLast(80)) || referenceBefore.endsWith(before.takeLast(80))) &&
        (after.isBlank() || referenceAfter.isBlank() || after.startsWith(referenceAfter.take(80)) || referenceAfter.startsWith(after.take(80)))
}

internal fun highlightLocatorScore(selected: Locator, candidate: Locator): Double {
    var score = 0.0
    val wantedBefore = stableHighlightQuery(selected.text.before.orEmpty()).takeLast(96)
    val wantedAfter = stableHighlightQuery(selected.text.after.orEmpty()).take(96)
    val candidateBefore = stableHighlightQuery(candidate.text.before.orEmpty())
    val candidateAfter = stableHighlightQuery(candidate.text.after.orEmpty())
    if (wantedBefore.isNotBlank() && candidateBefore.endsWith(wantedBefore)) score += 20.0
    if (wantedAfter.isNotBlank() && candidateAfter.startsWith(wantedAfter)) score += 20.0
    val selectedProgression = selected.locations.progression
    val candidateProgression = candidate.locations.progression
    if (selectedProgression != null && candidateProgression != null) score -= abs(selectedProgression - candidateProgression)
    return score
}

internal fun readingPositionsEquivalent(local: Locator?, remote: Locator, remoteFraction: Double): Boolean {
    local ?: return false
    return readingPositionValuesEquivalent(
        local.locations.totalProgression, local.href.toString(), local.locations.progression,
        remoteFraction, remote.href.toString(), remote.locations.progression,
    )
}

internal fun readingPositionValuesEquivalent(
    localTotal: Double?, localHref: String, localProgress: Double?,
    remoteTotal: Double, remoteHref: String, remoteProgress: Double?,
): Boolean {
    if (localTotal != null && localTotal.isFinite() && remoteTotal.isFinite() &&
        abs(localTotal - remoteTotal) <= EQUIVALENT_POSITION_TOLERANCE
    ) return true
    if (localHref != remoteHref) return false
    return localProgress != null && remoteProgress != null && localProgress.isFinite() && remoteProgress.isFinite() &&
        abs(localProgress - remoteProgress) <= EQUIVALENT_POSITION_TOLERANCE
}
