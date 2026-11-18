package net.kdhmedia.mochil.ui

import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.IconButton
import androidx.compose.material3.Icon
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.ColorFilter
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.R
import net.kdhmedia.mochil.auth.LoginState
import net.kdhmedia.mochil.ui.theme.LocalReduceMotion

@Composable
fun LoginScreen(
    state: LoginState,
    onServerChange: (String) -> Unit,
    onUsernameChange: (String) -> Unit,
    onPasswordChange: (String) -> Unit,
    onLogin: () -> Unit,
    onBack: (() -> Unit)? = null,
    crashReport: String? = null,
    onDismissCrashReport: () -> Unit = {},
) {
    Surface(modifier = Modifier.fillMaxSize()) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .statusBarsPadding()
                .navigationBarsPadding()
                .imePadding()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 24.dp, vertical = 28.dp),
            verticalArrangement = Arrangement.Center,
        ) {
            onBack?.let { back ->
                IconButton(onClick = back, modifier = Modifier.align(Alignment.Start)) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back to this device")
                }
            }
            Image(
                painter = painterResource(R.drawable.mochil_mark_monochrome),
                contentDescription = "Mochil",
                colorFilter = ColorFilter.tint(MaterialTheme.colorScheme.primary),
                modifier = Modifier.size(64.dp),
            )
            Spacer(Modifier.height(18.dp))
            Text("Connect a Mochil server", style = MaterialTheme.typography.headlineLarge)
            Text(
                "Add its libraries to this device. Local media remains available if the server is offline or disconnected.",
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                style = MaterialTheme.typography.bodyLarge,
                modifier = Modifier.padding(top = 8.dp, bottom = 28.dp),
            )
            OutlinedTextField(
                value = state.serverUrl,
                onValueChange = onServerChange,
                label = { Text("Server address") },
                placeholder = { Text("http://mochil.local:8099") },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri, imeAction = ImeAction.Next),
                singleLine = true,
                enabled = !state.loading,
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = state.username,
                onValueChange = onUsernameChange,
                label = { Text("Username") },
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
                singleLine = true,
                enabled = !state.loading,
                modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
            )
            OutlinedTextField(
                value = state.password,
                onValueChange = onPasswordChange,
                label = { Text("Password") },
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Done),
				keyboardActions = KeyboardActions(onDone = { onLogin() }),
                singleLine = true,
                enabled = !state.loading,
                isError = state.error != null,
                supportingText = state.error?.let { message -> { Text(message) } },
                modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
            )
            crashReport?.let { report ->
                Surface(
                    color = MaterialTheme.colorScheme.surfaceVariant,
                    shape = MaterialTheme.shapes.medium,
                    modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                ) {
                    Column(modifier = Modifier.padding(12.dp)) {
                        Text("Mochil closed unexpectedly last time", style = MaterialTheme.typography.titleSmall)
                        Text(
                            report.lineSequence().take(14).joinToString("\n"),
                            style = MaterialTheme.typography.bodySmall,
                            modifier = Modifier.padding(top = 6.dp),
                        )
                        Text(
                            "Copy this text when reporting the problem.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 6.dp),
                        )
                        androidx.compose.material3.TextButton(onClick = onDismissCrashReport) { Text("Dismiss") }
                    }
                }
            }
            Button(
                onClick = onLogin,
                enabled = !state.loading,
				modifier = Modifier.fillMaxWidth().padding(top = 8.dp).height(52.dp),
            ) {
                if (state.loading) {
                    if (LocalReduceMotion.current) CircularProgressIndicator(
                        progress = { 1f },
                        modifier = Modifier.height(20.dp),
                        strokeWidth = 2.dp,
                        color = MaterialTheme.colorScheme.onPrimary,
                    ) else CircularProgressIndicator(
                        modifier = Modifier.height(20.dp),
                        strokeWidth = 2.dp,
                        color = MaterialTheme.colorScheme.onPrimary,
                    )
                } else {
                    Text("Sign in")
                }
            }
            Text(
                "Your password is used only to create this device connection and is not saved.",
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                style = MaterialTheme.typography.bodySmall,
                modifier = Modifier.padding(top = 16.dp).align(Alignment.Start),
            )
        }
    }
}
