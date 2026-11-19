package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import net.kdhmedia.mochil.auth.LoginState

@Composable
fun TvPairingScreen(state: LoginState, onServerChange: (String) -> Unit, onStart: () -> Unit, onCancel: () -> Unit) {
    val initialFocus = remember { FocusRequester() }
    LaunchedEffect(state.pairingCode, state.loading) {
        if (!state.loading) runCatching { initialFocus.requestFocus() }
    }
    Surface(Modifier.fillMaxSize()) {
        Column(
            Modifier.fillMaxSize().padding(horizontal = 72.dp, vertical = 48.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Text("Mochil", style = MaterialTheme.typography.displaySmall, fontWeight = FontWeight.SemiBold)
            if (state.pairingCode.isNotBlank()) {
                Text("Connect this TV", style = MaterialTheme.typography.headlineLarge, modifier = Modifier.padding(top = 28.dp))
                Text("On a phone or computer, open", color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(top = 16.dp))
                Text(state.pairingUrl.substringBefore("?"), color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.titleLarge)
                Text("and enter", color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(top = 20.dp))
                Text(state.pairingCode, style = MaterialTheme.typography.displayLarge.copy(letterSpacing = 8.sp), fontWeight = FontWeight.Bold)
                Text("Waiting for approval…", color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(top = 18.dp))
                OutlinedButton(onClick = onCancel, modifier = Modifier.padding(top = 22.dp).focusRequester(initialFocus).tvFocusBorder()) { Text("Cancel") }
            } else {
                Text("Connect without typing your password", style = MaterialTheme.typography.headlineLarge, textAlign = TextAlign.Center, modifier = Modifier.padding(top = 28.dp))
                Text("Enter your Mochil address. You will approve this TV from a signed-in phone or computer.", color = MaterialTheme.colorScheme.onSurfaceVariant, textAlign = TextAlign.Center, modifier = Modifier.padding(top = 12.dp))
                OutlinedTextField(
                    value = state.serverUrl, onValueChange = onServerChange, label = { Text("Server address") },
                    placeholder = { Text("https://books.example.com") }, singleLine = true, enabled = !state.loading,
                    modifier = Modifier.widthIn(max = 680.dp).fillMaxWidth().padding(top = 28.dp).focusRequester(initialFocus).tvFocusBorder(),
                )
                Button(onClick = onStart, enabled = !state.loading, modifier = Modifier.padding(top = 18.dp).tvFocusBorder()) {
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                        if (state.loading) CircularProgressIndicator(strokeWidth = 2.dp, modifier = Modifier.padding(2.dp))
                        Text(if (state.loading) "Starting…" else "Show pairing code")
                    }
                }
                state.error?.let { Text(it, color = MaterialTheme.colorScheme.error, modifier = Modifier.padding(top = 14.dp)) }
            }
        }
    }
}
