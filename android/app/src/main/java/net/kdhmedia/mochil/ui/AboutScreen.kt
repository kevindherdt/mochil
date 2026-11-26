package net.kdhmedia.mochil.ui

import androidx.annotation.RawRes
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import net.kdhmedia.mochil.BuildConfig
import net.kdhmedia.mochil.R
import net.kdhmedia.mochil.ui.theme.MochilSpace

/**
 * About, with the notices the app owes the work it is built on: Lucide's icons (ISC) and the two bundled fonts, Source Sans 3
 * and Newsreader, each under the SIL Open Font Licence 1.1. The licence texts ship in `res/raw` and are shown here, so a copy
 * of the app carries them (docs/android-ui-plan.md step 8).
 */
@Composable
internal fun AboutContent(modifier: Modifier = Modifier) {
    // No scroll of its own: the settings page already scrolls, and a second verticalScroll inside it is measured with
    // an unbounded height and throws — which is how About closed the app the first time it was opened.
    Column(
        modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(MochilSpace.s12),
    ) {
        Text("Mochil ${BuildConfig.VERSION_NAME}", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
        Text(
            "Your own catalogue, on this device: books, comics, audio and video from your server, with the ones you open often kept offline.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
        Text("Notices", style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold)
        LicenceSection("Lucide icons", "ISC licence", R.raw.licence_lucide_isc)
        LicenceSection("Source Sans 3", "SIL Open Font Licence 1.1", R.raw.licence_source_sans_ofl)
        LicenceSection("Newsreader", "SIL Open Font Licence 1.1", R.raw.licence_newsreader_ofl)
    }
}

/** One notice, closed by default so the page opens as a short list rather than four pages of licence. */
@Composable
private fun LicenceSection(title: String, subtitle: String, @RawRes text: Int) {
    var open by rememberSaveable(title) { mutableStateOf(false) }
    val context = LocalContext.current
    val body = remember(text) { context.resources.openRawResource(text).bufferedReader().use { it.readText() } }
    Column(Modifier.fillMaxWidth()) {
        TextButton(onClick = { open = !open }, modifier = Modifier.fillMaxWidth()) {
            Column(Modifier.fillMaxWidth()) {
                Text(title, style = MaterialTheme.typography.bodyLarge)
                Text(subtitle, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        if (open) Text(
            body,
            Modifier.padding(start = MochilSpace.s12, end = MochilSpace.s12, bottom = MochilSpace.s12),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}
