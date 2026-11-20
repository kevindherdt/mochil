package net.kdhmedia.mochil.catalogue

/** A failed copy must not leave a newly created partial document looking like saved media. */
internal fun <T> writeNewDownloadDocument(create: () -> T, write: (T) -> Unit, discard: (T) -> Unit): T {
    val target = create()
    try {
        write(target)
        return target
    } catch (failure: Exception) {
        try {
            discard(target)
        } catch (cleanup: Exception) {
            failure.addSuppressed(cleanup)
        }
        throw failure
    }
}
