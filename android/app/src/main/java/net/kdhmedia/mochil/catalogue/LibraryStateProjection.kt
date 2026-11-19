package net.kdhmedia.mochil.catalogue

data class LibraryShellProjection(
    val libraries: List<LibrarySummary> = emptyList(),
    val selected: LibrarySummary? = null,
)

data class LibraryVisibilityProjection(
    val allLibraries: List<LibrarySummary> = emptyList(),
    val visibleLibraryIds: Set<String> = emptySet(),
)

fun LibraryState.shellProjection() = LibraryShellProjection(libraries, selected)

fun LibraryState.visibilityProjection() = LibraryVisibilityProjection(
    allLibraries = allLibraries,
    visibleLibraryIds = libraries.mapTo(linkedSetOf()) { it.id },
)

