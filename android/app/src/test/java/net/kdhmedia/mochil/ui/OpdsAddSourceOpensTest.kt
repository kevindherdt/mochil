package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.opds.OpdsViewModel
import org.junit.Assert.assertNotNull
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * Opening "Add a source -> OPDS catalogues" crashed the app on a phone. The screen creates its
 * view model while composing, and that constructor opens the Android keystore and the download
 * store; anything they throw happens inside composition, where there is no loader, so the process
 * dies instead of showing a message.
 *
 * This test exists to make that path fail here first, with a stack trace, rather than on a device
 * with nothing but "Mochil closed because this app has a bug".
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class OpdsAddSourceOpensTest {
    @Test
    fun theViewModelCanBeConstructedFromTheApplicationAlone() {
        // This is what composition does on the real phone: the default factory builds the view model
        // from the Application, so a constructor that throws takes the whole screen down.
        val application = androidx.test.core.app.ApplicationProvider.getApplicationContext<android.app.Application>()
        assertNotNull(OpdsViewModel(application))
    }
}
