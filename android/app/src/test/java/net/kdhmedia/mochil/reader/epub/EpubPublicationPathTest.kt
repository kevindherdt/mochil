package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import java.io.File

/** Characterizes EpubReaderActivity.validatedPublication before it moves to [validatedEpubPath]. */
class EpubPublicationPathTest {
    @Test fun acceptsAnEpubFileInsideThePublicationsRoot() {
        val root = File.createTempFile("pubs", "").apply { delete(); mkdirs() }
        val file = File(root, "book.epub").apply { writeText("x") }
        assertEquals(file.canonicalFile, validatedEpubPath(file.path, root))
        root.deleteRecursively()
    }

    @Test fun rejectsPathTraversalOutsideTheRoot() {
        val root = File.createTempFile("pubs", "").apply { delete(); mkdirs() }
        val outside = File(root.parentFile, "outside.epub").apply { writeText("x") }
        assertNull(validatedEpubPath(File(root, "../outside.epub").path, root))
        outside.delete()
        root.deleteRecursively()
    }

    @Test fun rejectsNonEpubExtensions() {
        val root = File.createTempFile("pubs", "").apply { delete(); mkdirs() }
        val file = File(root, "book.pdf").apply { writeText("x") }
        assertNull(validatedEpubPath(file.path, root))
        root.deleteRecursively()
    }

    @Test fun rejectsAMissingFile() {
        val root = File.createTempFile("pubs", "").apply { delete(); mkdirs() }
        assertNull(validatedEpubPath(File(root, "missing.epub").path, root))
        root.deleteRecursively()
    }

    @Test fun rejectsANullPath() {
        val root = File.createTempFile("pubs", "").apply { delete(); mkdirs() }
        assertNull(validatedEpubPath(null, root))
        root.deleteRecursively()
    }
}
