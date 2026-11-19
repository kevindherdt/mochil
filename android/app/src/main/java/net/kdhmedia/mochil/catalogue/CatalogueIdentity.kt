package net.kdhmedia.mochil.catalogue

/** Stable UI identity for a value scoped to one library. */
internal fun catalogueItemIdentityKey(libraryId: String, itemId: String): String =
    "${libraryId.length}:$libraryId${itemId.length}:$itemId"
