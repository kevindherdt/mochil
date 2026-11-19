package net.kdhmedia.mochil.video

import org.junit.Assert.*
import org.junit.Test

class VideoProgressSubmissionsTest {
    @Test fun `newest same-item update retains unacknowledged seek intent`() {
        val submissions = VideoProgressSubmissions()
        val seek = submissions.submit("one", true, 100)
        val passive = submissions.submit("one", false, 200)
        assertFalse(submissions.isCurrent(seek))
        assertTrue(submissions.isCurrent(passive))
        assertTrue(passive.explicitSeek)
        submissions.acknowledge(seek)
        assertTrue(submissions.submit("one", false, 300).explicitSeek)
        val latest = submissions.submit("one", false, 400)
        submissions.acknowledge(latest)
        assertFalse(submissions.submit("one", false, 500).explicitSeek)
    }

    @Test fun `seek intent expires and remains item scoped`() {
        val submissions = VideoProgressSubmissions()
        submissions.submit("one", true, 100)
        assertFalse(submissions.submit("two", false, 101).explicitSeek)
        assertFalse(submissions.submit("one", false, 600_101).explicitSeek)
        submissions.submit("one", true, 900)
        assertFalse(submissions.submit("one", false, 899).explicitSeek)
    }

    @Test fun `bounded ownership invalidates evicted tickets`() {
        val submissions = VideoProgressSubmissions(2)
        val first = submissions.submit("one", false, 0)
        val second = submissions.submit("two", false, 0)
        submissions.submit("three", false, 0)
        assertFalse(submissions.isCurrent(first))
        assertTrue(submissions.isCurrent(second))
    }

    @Test fun `replay cannot supersede live updates and queued seek expires`() {
        val submissions = VideoProgressSubmissions()
        val pending = PendingVideoProgress("a".repeat(64), "anime", "one", 10.0, 100.0, 1, false)
        val live = submissions.submit("one", true, 100)
        assertNull(submissions.replay("one", true, 200, 100, pending))
        assertTrue(submissions.isCurrent(live))
        val replay = submissions.replay("two", true, 200, 100, pending)!!
        assertTrue(replay.seekActive(600_100))
        assertFalse(replay.seekActive(600_101))
        assertFalse(replay.seekActive(99))
        submissions.submit("two", false, 300)
        assertFalse(submissions.isCurrent(replay))
    }
}
