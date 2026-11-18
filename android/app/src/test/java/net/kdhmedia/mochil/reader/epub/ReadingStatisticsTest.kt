package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class ReadingStatisticsTest {
    @Test fun durationLabelsStayCompact() {
        assertNull(readingDurationLabel(null))
        assertNull(readingDurationLabel(0))
        assertEquals("1 min", readingDurationLabel(10_000))
        assertEquals("1 hr 5 min", readingDurationLabel(65 * 60_000L))
        assertEquals("12 sec", positionDurationLabel(12_000))
    }

    // A burst of fast page turns must not make the reader look faster than they read.
    @Test fun aSkimBurstNeverMovesTheEstimate() {
        val steady = List(8) { 6_000.0 }  // one hour per book, the reader's real pace
        var window = steady
        // Ten pages flicked through in three seconds: each turn implies two minutes for the whole book.
        repeat(10) {
            val sample = paceSample(elapsedMillis = 300, progressDelta = 0.0025, window = window)
            assertNull("a skim must not become a pace sample", sample)
            window = withPaceSample(window, sample)
        }
        assertEquals(6_000.0, medianPace(window)!!, 0.001)
        // Half the book left at that pace is still fifty minutes: one hour per book, never two minutes.
        assertEquals(3_000_000L, remainingMillis(window, 0.5)!!)
    }

    @Test fun anImpossibleFirstSampleIsRejected() {
        // Nobody reads a whole book in two minutes, however fast they turn pages.
        assertNull(paceSample(elapsedMillis = 3_000, progressDelta = 0.025, window = emptyList()))
        // A plausible first turn is kept.
        assertEquals(6_000.0, paceSample(elapsedMillis = 30_000, progressDelta = 0.005, window = emptyList())!!, 0.001)
    }

    @Test fun aPaceTenTimesOffTheUsualOneIsIgnored() {
        val window = List(5) { 6_000.0 }
        assertNull(paceSample(elapsedMillis = 900_000, progressDelta = 0.005, window = window))
        assertEquals(6_000.0, paceSample(elapsedMillis = 30_000, progressDelta = 0.005, window = window)!!, 0.001)
    }

    @Test fun theMedianIgnoresOneWildSample() {
        assertEquals(60.0, medianPace(listOf(1.0, 60.0, 60.0))!!, 0.001)
        assertEquals(60.0, medianPace(listOf(60.0, 60.0, 60.0, 6000.0))!!, 0.001)
        assertNull(medianPace(emptyList()))
    }

    @Test fun theRemainingTimeNeedsAConfidentPace() {
        assertNull("one sample is not a pace", remainingMillis(listOf(6_000.0), 0.5))
        assertNull("no position, no estimate", remainingMillis(List(4) { 6_000.0 }, null))
        assertEquals(0L, remainingMillis(List(4) { 6_000.0 }, 1.0)!!)
    }

    @Test fun readingSpeedReadsAsPagesPerMinute() {
        assertEquals("About 1.4 pages a minute", readingSpeedLabel(42_000))
        assertEquals("About 12 pages a minute", readingSpeedLabel(5_000))
        assertEquals("About 1 page every 2 min", readingSpeedLabel(120_000))
        assertNull(readingSpeedLabel(null))
    }

    @Test fun aStoredPaceFromTheOlderShapeIsKept() {
        // Written before the windows existed: one running average per measure, plus its sample count.
        val legacy = """{"total":60000,"turns":9,"average":42000.0,"book_rate":6000.0,"book_samples":7,
            "chapter_rate":300.0,"chapter_samples":4,"last_at":1700000000000,"last_total":0.5}"""
        val state = decodeStatistics(legacy)
        assertEquals(listOf(6000.0, 6000.0, 6000.0), state.bookWindow)
        assertEquals(listOf(300.0, 300.0, 300.0), state.chapterWindow)
        assertEquals(listOf(42000.0, 42000.0, 42000.0), state.positionWindow)
        assertEquals(0.5, state.lastTotalProgression!!, 0.0001)
        // And a round trip keeps everything.
        val again = decodeStatistics(encodeStatistics(state).toString())
        assertEquals(state.bookWindow, again.bookWindow)
        assertEquals(state.lastAtMillis, again.lastAtMillis)
    }
}
