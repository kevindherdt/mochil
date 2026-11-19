package net.kdhmedia.mochil.reader.epub

/** Keeps page-perfect local restoration separate from renderer-neutral sync. */
internal data class ReaderPositionPair<T>(val local: T, val synchronized: T)

internal fun <T> readerPositionPair(exactPage: T, structural: T): ReaderPositionPair<T> =
    ReaderPositionPair(local = exactPage, synchronized = structural)
