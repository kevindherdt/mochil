package net.kdhmedia.mochil

import android.content.res.Configuration
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TelevisionDeviceTest {
    @Test fun televisionUiModeUsesTheTvPresentation() {
        assertTrue(isTelevisionUi(Configuration.UI_MODE_TYPE_TELEVISION, false))
    }

    @Test fun leanbackDevicesUseTheTvPresentationEvenWithAnUnhelpfulUiMode() {
        assertTrue(isTelevisionUi(Configuration.UI_MODE_TYPE_NORMAL, true))
    }

    @Test fun ordinaryPhonesStayOnTheMobilePresentation() {
        assertFalse(isTelevisionUi(Configuration.UI_MODE_TYPE_NORMAL, false))
    }
}
