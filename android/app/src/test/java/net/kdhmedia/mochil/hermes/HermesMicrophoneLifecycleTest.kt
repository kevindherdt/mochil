package net.kdhmedia.mochil.hermes

import java.lang.reflect.Modifier
import org.junit.Assert.assertTrue
import org.junit.Test

class HermesMicrophoneLifecycleTest {
    @Test
    fun startAndStopAreSerializedOnTheMicrophoneInstance() {
        val microphone = HermesMicrophone::class.java
        val start = microphone.getDeclaredMethod("start")
        val stop = microphone.getDeclaredMethod("stop")

        assertTrue("start must be synchronized", Modifier.isSynchronized(start.modifiers))
        assertTrue("stop must be synchronized", Modifier.isSynchronized(stop.modifiers))
    }
}
