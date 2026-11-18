package net.kdhmedia.mochil.audio

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MusicRequestGenerationTest {
    @Test fun advancingInvalidatesEveryEarlierContentRequest() {
        val requests = MusicRequestGeneration()
        val initial = requests.capture()
        val library = requests.advance()
        assertFalse(requests.isCurrent(initial))
        assertTrue(requests.isCurrent(library))

        val grouping = requests.advance()
        assertFalse(requests.isCurrent(library))
        assertTrue(requests.isCurrent(grouping))
    }

    @Test fun concurrentWorkInTheSameVisibleContextSharesGeneration() {
        val requests = MusicRequestGeneration()
        requests.advance()
        val navigation = requests.capture()
        val catalogue = requests.capture()
        assertTrue(requests.isCurrent(navigation))
        assertTrue(requests.isCurrent(catalogue))
    }
}
