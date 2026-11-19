package net.kdhmedia.mochil.ui

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.test.core.app.ApplicationProvider
import net.kdhmedia.mochil.audio.MusicQueueViewModel
import net.kdhmedia.mochil.audio.MusicViewModel
import net.kdhmedia.mochil.catalogue.LibraryViewModel
import net.kdhmedia.mochil.dashboard.DashboardViewModel
import net.kdhmedia.mochil.playback.PlaybackViewModel
import net.kdhmedia.mochil.podcast.PodcastViewModel
import net.kdhmedia.mochil.video.VideoViewModel
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.junit.Assert.assertTrue

/**
 * MochilApp builds these with `viewModel()`, which reaches them through the platform's
 * reflection-based factory. That factory only ever asks for a constructor of one exact shape:
 * `(Application)`. Kotlin does not generate that overload for a class whose constructor has
 * further parameters with defaults, so such a class compiles, passes its own tests (they call the
 * full constructor), and then throws "Cannot create an instance of ..." the first time a signed-in
 * screen composes. Asserting the shape here keeps that failure in the build instead of on a phone.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class ViewModelFactoryContractTest {

    private val sessionViewModels = listOf(
        LibraryViewModel::class.java,
        MusicViewModel::class.java,
        MusicQueueViewModel::class.java,
        PlaybackViewModel::class.java,
        PodcastViewModel::class.java,
        VideoViewModel::class.java,
        DashboardViewModel::class.java,
        net.kdhmedia.mochil.update.AppUpdateViewModel::class.java,
    )

    @Test
    fun everyViewModelCreatedByTheAppHasTheConstructorTheFactoryAsksFor() {
        val missing = sessionViewModels.mapNotNull { type ->
            val hasApplicationConstructor = runCatching { type.getConstructor(Application::class.java) }.isSuccess
            if (hasApplicationConstructor) null else type.name
        }
        assertTrue(
            "The ViewModel factory asks for (Application) only. Add @JvmOverloads, or a plain " +
                "(Application) constructor, to: $missing",
            missing.isEmpty(),
        )
    }

    @Test
    fun everyViewModelIsAnAndroidViewModelSoTheFactoryUsesThatConstructor() {
        val wrong = sessionViewModels.filterNot { AndroidViewModel::class.java.isAssignableFrom(it) }
        assertTrue("Not an AndroidViewModel, so (Application) is never asked for: ${wrong.map { it.name }}", wrong.isEmpty())
    }

    @Test
    fun theApplicationParameterIsTheOnlyOneThatHasToBeReal() {
        // Guards the assertion itself: a class that does have (Application) must report present.
        assertTrue(runCatching { MusicViewModel::class.java.getConstructor(Application::class.java) }.isSuccess)
        assertTrue(ApplicationProvider.getApplicationContext<Application>() != null)
    }
}
