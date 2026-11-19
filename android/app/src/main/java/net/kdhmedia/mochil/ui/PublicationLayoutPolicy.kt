package net.kdhmedia.mochil.ui

internal const val PUBLICATION_TWO_PANE_MIN_WIDTH_DP = 840

internal fun usesPublicationTwoPane(widthDp: Int, hasSelection: Boolean): Boolean =
    hasSelection && widthDp >= PUBLICATION_TWO_PANE_MIN_WIDTH_DP

internal fun usesMediaTwoPane(widthDp: Int, hasSelection: Boolean): Boolean =
    usesPublicationTwoPane(widthDp, hasSelection)
