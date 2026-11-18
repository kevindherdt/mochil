package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertNotEquals
import org.junit.Test

class CatalogueIdentityTest {
    @Test fun colonContainingComponentsCannotAlias() {
        assertNotEquals(catalogueItemIdentityKey("a:b", "c"), catalogueItemIdentityKey("a", "b:c"))
    }
}
