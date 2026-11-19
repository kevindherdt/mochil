package net.kdhmedia.mochil.wiki

/** Page cancellation ends page loading; vault browsing and offline work remain independent. */
internal fun WikiState.withClosedWikiPage(): WikiState = copy(
    page = null,
    selectedPath = "",
    draft = "",
    backlinks = emptyList(),
    loading = false,
    saving = false,
    editing = false,
    error = null,
    draftRecovered = false,
    navigationHeading = "",
    missingLink = null,
    attachmentStatus = null,
)
