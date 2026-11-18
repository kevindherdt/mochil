package net.kdhmedia.mochil.auth

import android.app.Application
import android.os.Build
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay

data class LoginState(
    val serverUrl: String = "",
    val username: String = "",
    val password: String = "",
    val loading: Boolean = false,
    val error: String? = null,
    val connected: Boolean = false,
    val sessionVersion: Long = 0,
    val pairingCode: String = "",
    val pairingUrl: String = "",
    val notice: String? = null,
)

class LoginViewModel(application: Application) : AndroidViewModel(application) {
    private val store = DeviceCredentialStore(application)
    private val client = LoginClient()
    // A saved device credential normally means "resume straight into the library". If the last
    // launch crashed, resuming would reproduce that crash before anyone could read the report, so a
    // pending crash report parks the app signed out: the sign-in screen shows the stack, and the
    // person can decide to dismiss it and try again.
    private val mutableState = MutableStateFlow(
        LoginState(
            connected = store.load() != null &&
                !runCatching { net.kdhmedia.mochil.diagnostics.CrashReportStore(application).hasPending() }.getOrDefault(false),
        ),
    )
    val state: StateFlow<LoginState> = mutableState.asStateFlow()
    private var sessionVersion = 0L
    private var pairingJob: Job? = null

    init {
        var observed = DeviceSessionEvents.generation.value
        viewModelScope.launch {
            DeviceSessionEvents.generation.collect { generation ->
                if (generation <= observed) return@collect
                observed = generation
                val serverUrl = store.load()?.serverUrl.orEmpty()
                store.clear()
                sessionVersion += 1
                mutableState.value = LoginState(serverUrl = serverUrl, error = "This device connection expired. Sign in again.", sessionVersion = sessionVersion)
            }
        }
    }

    fun updateServer(value: String) = update { copy(serverUrl = value, error = null) }
    fun updateUsername(value: String) = update { copy(username = value, error = null) }
    fun updatePassword(value: String) = update { copy(password = value, error = null) }
    fun connectedServerUrl(): String = store.load()?.serverUrl.orEmpty()

    fun changeServerAddress(value: String) {
        val credential = store.load() ?: return
        if (mutableState.value.loading) return
        update { copy(loading = true, error = null, notice = null) }
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.changeServerAddress(credential, value) } }
                .onSuccess { updated ->
                    store.save(updated)
                    sessionVersion += 1
                    mutableState.value = LoginState(serverUrl = updated.serverUrl, connected = true, sessionVersion = sessionVersion, notice = "Server address updated.")
                }
                .onFailure { failure -> update { copy(loading = false, error = failure.message ?: "Could not change the server address.") } }
        }
    }

    fun login() {
        val current = mutableState.value
        if (current.loading) return
        if (current.serverUrl.isBlank() || current.username.isBlank() || current.password.isBlank()) {
            update { copy(error = "Enter the server address, username, and password.") }
            return
        }
        update { copy(loading = true, error = null) }
        viewModelScope.launch {
            runCatching {
                withContext(Dispatchers.IO) {
                    val credential = client.login(current.serverUrl, current.username, current.password, Build.MODEL.ifBlank { "Android device" })
                    // Encrypting with the platform Keystore is disk and CPU work, and it can fail on a
                    // device whose key was invalidated. Doing it here keeps it off the main thread and
                    // inside the failure path instead of crashing the app after a successful sign-in.
                    store.save(credential)
                    credential
                }
            }.onSuccess { credential ->
                DeviceSessionEvents.connected()
                sessionVersion += 1
                mutableState.value = LoginState(connected = true, sessionVersion = sessionVersion)
            }.onFailure { failure ->
                update { copy(password = "", loading = false, error = failure.message ?: "Could not sign in.") }
            }
        }
    }

    fun startPairing() {
        val current = mutableState.value
        if (current.loading || current.serverUrl.isBlank()) {
            if (current.serverUrl.isBlank()) update { copy(error = "Enter the Mochil server address.") }
            return
        }
        pairingJob?.cancel()
        update { copy(loading = true, error = null, pairingCode = "", pairingUrl = "") }
        pairingJob = viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.startPairing(current.serverUrl, "${Build.MODEL.ifBlank { "Android TV" }} TV") } }
                .onSuccess { request ->
                    update { copy(loading = false, pairingCode = request.code, pairingUrl = request.verificationUri) }
                    repeat(300) {
                        delay(2_000)
                        val result = runCatching { withContext(Dispatchers.IO) { client.pollPairing(current.serverUrl, request.id) } }
                        result.onFailure { failure ->
                            update { copy(loading = false, pairingCode = "", pairingUrl = "", error = failure.message ?: "Could not pair this TV.") }
                            return@launch
                        }
                        result.getOrNull()?.let { credential ->
                            store.save(credential); DeviceSessionEvents.connected(); sessionVersion += 1
                            mutableState.value = LoginState(connected = true, sessionVersion = sessionVersion)
                            return@launch
                        }
                    }
                    update { copy(pairingCode = "", pairingUrl = "", error = "This pairing code expired. Start again.") }
                }
                .onFailure { failure -> update { copy(loading = false, error = failure.message ?: "Could not start pairing.") } }
        }
    }

    fun cancelPairing() {
        pairingJob?.cancel(); pairingJob = null
        update { copy(loading = false, pairingCode = "", pairingUrl = "", error = null) }
    }

    fun disconnect() {
        val credential = store.load() ?: return
        if (mutableState.value.loading) return
        update { copy(loading = true, error = null) }
        viewModelScope.launch {
            val result = runCatching { withContext(Dispatchers.IO) { client.disconnect(credential) } }
            store.clear()
            sessionVersion += 1
            mutableState.value = LoginState(
                serverUrl = credential.serverUrl,
                error = result.exceptionOrNull()?.message,
                connected = false,
                sessionVersion = sessionVersion,
            )
        }
    }

    private fun update(transform: LoginState.() -> LoginState) {
        mutableState.value = mutableState.value.transform()
    }
}
