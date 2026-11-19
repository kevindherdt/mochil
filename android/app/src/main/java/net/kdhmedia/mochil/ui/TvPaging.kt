package net.kdhmedia.mochil.ui

internal data class TvPagingState(
    val hasNextPage: Boolean,
    val loading: Boolean,
    val onLoadNextPage: () -> Unit,
    val error: String? = null,
    val onRetry: () -> Unit = onLoadNextPage,
)

internal fun shouldLoadTvPage(
    totalItems: Int,
    lastVisibleIndex: Int,
    hasNextPage: Boolean,
    loading: Boolean,
    prefetchDistance: Int = 6,
): Boolean = hasNextPage && !loading && totalItems > 0 && lastVisibleIndex >= totalItems - 1 - prefetchDistance
