package net.kdhmedia.mochil.hermes

import android.app.Application
import android.content.Context
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import java.util.UUID
import kotlinx.coroutines.CancellationException
import org.json.JSONObject
import kotlinx.coroutines.Job
import kotlinx.coroutines.cancelAndJoin
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.auth.DeviceCredentialStore

data class HermesChatState(
    val loading: Boolean = true,
    val available: Boolean = true,
    val name: String = "Hermes",
    val memoryOn: Boolean = false,
    val voiceAllowed: Boolean = true,
    val quickOn: Boolean = true,
    val chats: List<HermesChat> = emptyList(),
    val currentId: String = newHermesChatId(),
    val messages: List<HermesMessage> = emptyList(),
    val steps: List<HermesStep> = emptyList(),
    val busy: Boolean = false,
    val error: String? = null,
    val notice: String? = null,
    val memories: List<HermesMemory>? = null,
    val memoriesError: String? = null,
    val voice: HermesVoiceState = HermesVoiceState(),
    val speaking: Boolean = false,
    val talk: TalkMode = TalkMode.Off,
    val avatar: HermesAvatarInfo? = null,
    val showAvatar: Boolean = true,
    /** The libraries, for the switcher in the top bar that leads back into the app. */
    val libraries: List<net.kdhmedia.mochil.catalogue.LibrarySummary> = emptyList(),
)

/** What the voice side offers and what is chosen. Stored on the device, not on the server. */
data class HermesVoiceState(
    val canSpeak: Boolean = false,
    val canListen: Boolean = false,
    val voices: List<HermesVoice> = emptyList(),
    val defaultVoice: String = "",
    val speakReplies: Boolean = false,
    val voiceId: String = "",
    val handsFree: Boolean = true,
    val bargeIn: Boolean = false,
)

internal fun newHermesChatId(): String = "c" + UUID.randomUUID().toString().replace("-", "").take(24)

class HermesChatViewModel(application: Application) : AndroidViewModel(application) {
    private val credentials = DeviceCredentialStore(application)
    private val client = HermesClient()
    private val mutable = MutableStateFlow(HermesChatState())
    val state: StateFlow<HermesChatState> = mutable.asStateFlow()
    private var sending: Job? = null
    private val preferences = application.getSharedPreferences("hermes_voice", Context.MODE_PRIVATE)
    private val speaker = HermesSpeaker(
        client, ::credential, { mutable.value.voice.voiceId.ifBlank { mutable.value.avatar?.voice?.ifBlank { null } } },
        onChange = { speaking -> mutable.update { it.copy(speaking = speaking) }; talk.refresh() },
        onError = { error -> mutable.update { it.copy(notice = error.message ?: "Speech failed") } },
    )
    private val talk: HermesTalk = HermesTalk(
        scope = viewModelScope,
        makeMicrophone = { HermesMicrophone(it) },
        isSpeaking = { speaker.speaking },
        transcribe = { wav -> client.transcribe(credential() ?: throw HermesException("This device is not signed in"), wav) },
        send = { text, interrupted -> send(text, interrupted = interrupted) },
        interrupt = { speaker.stop(); stop() },
        options = { TalkOptions(mutable.value.voice.handsFree, mutable.value.voice.bargeIn) },
        onMode = { mode -> mutable.update { it.copy(talk = mode) } },
        onError = { error -> mutable.update { it.copy(notice = error.message ?: "The microphone failed") } },
    )

    init {
        mutable.update { it.copy(showAvatar = preferences.getBoolean("avatar", true), voice = it.voice.copy(
            speakReplies = preferences.getBoolean("speak", false),
            voiceId = preferences.getString("voice", "").orEmpty(),
            handsFree = preferences.getBoolean("hands_free", true),
            bargeIn = preferences.getBoolean("barge_in", false),
        )) }
    }

    private fun credential(): DeviceCredential? = credentials.load()

    /** For the avatar page, which signs its own requests. */
    fun deviceCredential(): DeviceCredential? = credential()

    fun load() {
        viewModelScope.launch {
            val credential = credential() ?: return@launch mutable.update { it.copy(loading = false, available = false) }
            runCatching { client.status(credential) }
                .onSuccess { status ->
                    if (status == null) mutable.update { it.copy(loading = false, available = false) }
                    else {
                        mutable.update { it.copy(loading = false, name = status.name, memoryOn = status.memory, voiceAllowed = status.voice, quickOn = status.quick,
                            error = if (!status.configured) "Not set up on the server yet" else if (!status.reachable) "${status.name} is not reachable right now" else null) }
                        refreshChats()
                        loadVoice(credential)
                        loadLibraries(credential)
                    }
                }
                .onFailure { error -> mutable.update { it.copy(loading = false, error = error.message ?: "Could not reach the server") } }
        }
    }

    private fun loadLibraries(credential: DeviceCredential) {
        viewModelScope.launch {
            val libraries = runCatching { kotlinx.coroutines.withContext(kotlinx.coroutines.Dispatchers.IO) { net.kdhmedia.mochil.catalogue.CatalogueClient().libraries(credential) } }.getOrNull().orEmpty()
            mutable.update { it.copy(libraries = libraries) }
        }
    }

    private fun loadVoice(credential: DeviceCredential) {
        viewModelScope.launch {
            val avatar = runCatching { client.avatar(credential) }.getOrNull()
            mutable.update { it.copy(avatar = avatar) }
            val status = if (mutable.value.voiceAllowed) runCatching { client.voiceStatus(credential) }.getOrNull() else null
            mutable.update { it.copy(voice = it.voice.copy(
                canSpeak = status?.speak == true, canListen = status?.speak == true && status.listen,
                voices = status?.voices.orEmpty(), defaultVoice = status?.defaultVoice.orEmpty(),
            )) }
            if (status?.speak == true && mutable.value.voice.speakReplies) client.warm(credential, mutable.value.voice.voiceId.ifBlank { null })
        }
    }

    fun refreshChats() {
        viewModelScope.launch {
            val credential = credential() ?: return@launch
            runCatching { client.chats(credential) }.onSuccess { chats -> mutable.update { it.copy(chats = chats) } }
        }
    }

    fun newChat() {
        if (mutable.value.busy) return
        mutable.update { it.copy(currentId = newHermesChatId(), messages = emptyList(), steps = emptyList(), error = null) }
    }

    fun openChat(id: String) {
        if (mutable.value.busy) return
        viewModelScope.launch {
            val credential = credential() ?: return@launch
            mutable.update { it.copy(currentId = id, messages = emptyList(), steps = emptyList(), error = null) }
            runCatching { client.messages(credential, id) }
                .onSuccess { messages -> mutable.update { if (it.currentId == id) it.copy(messages = messages) else it } }
                .onFailure { error -> mutable.update { it.copy(error = error.message) } }
        }
    }

    fun send(text: String, interrupted: Boolean = false) {
        val message = text.trim()
        if (message.isEmpty() || mutable.value.busy) return
        val voice = mutable.value.voice
        val spoken = voice.canSpeak && (voice.speakReplies || talk.mode != TalkMode.Off)
        if (spoken) speaker.begin() else speaker.stop()
        val id = mutable.value.currentId
        val before = mutable.value.messages.size
        mutable.update { it.copy(messages = it.messages + HermesMessage("user", message) + HermesMessage("assistant", ""), steps = emptyList(), busy = true, error = null) }
        sending = viewModelScope.launch {
            val credential = credential()
            var failure: String? = null
            var cancelled = false
            var received = false
            try {
                if (credential == null) throw HermesException("This device is not signed in")
                val stream: suspend () -> Unit = {
                    client.chat(credential, id, message, spoken, interrupted) { update ->
                        // Side effects stay out of the state update below, which may run more than once.
                        if (update.text != null) received = true
                        if (spoken) update.text?.let(speaker::push)
                        update.error?.let { failure = it }
                        mutable.update { current ->
                            if (current.currentId != id) return@update current
                            var next = current
                            update.text?.let { piece ->
                                val last = next.messages.last()
                                next = next.copy(messages = next.messages.dropLast(1) + last.copy(text = last.text + piece))
                            }
                            update.cards?.let { cards ->
                                val last = next.messages.last()
                                next = next.copy(messages = next.messages.dropLast(1) + last.copy(cards = cards))
                            }
                            update.steps?.let { next = next.copy(steps = it) }
                            next
                        }
                    }
                }
                try { stream() } catch (error: java.io.IOException) {
                    // The request may never have arrived. If the saved thread lacks the message, send it once more.
                    if (error is HermesException || received || !recoverNeedsResend(credential, id, before)) throw error
                    stream()
                }
            } catch (stopped: CancellationException) {
                cancelled = true
                throw stopped
            } catch (error: Exception) {
                // The connection broke (a network change, the phone sleeping its Wi-Fi). The server keeps going and saves the
                // whole reply, so look for it there before reporting a failure.
                val saved = if (error is java.io.IOException && error !is HermesException && credential != null) recoverReply(credential, id, before) else null
                if (saved != null) mutable.update { if (it.currentId == id) it.copy(messages = saved) else it }
                else failure = error.message ?: "${mutable.value.name} did not answer"
            } finally {
                if (spoken) { if (failure != null || cancelled) speaker.stop() else speaker.end() }
                talk.replyDone()
                mutable.update { current ->
                    if (current.currentId != id) return@update current.copy(busy = false)
                    val empty = current.messages.lastOrNull()?.let { it.role == "assistant" && it.text.isEmpty() } == true
                    current.copy(busy = false, steps = emptyList(), error = failure,
                        messages = if (empty) current.messages.dropLast(1) else current.messages)
                }
                refreshChats()
            }
        }
    }

    /** True when, a moment after a dropped connection, the server's thread does not contain the message just sent. */
    private suspend fun recoverNeedsResend(credential: DeviceCredential, id: String, before: Int): Boolean {
        delay(2500)
        val stored = runCatching { client.messages(credential, id) }.getOrNull() ?: return before == 0 // a chat that does not exist yet
        return stored.size < before + 1
    }

    /** After a dropped connection: waits for the saved thread to hold her answer. Null if it never shows up. */
    private suspend fun recoverReply(credential: DeviceCredential, id: String, before: Int): List<HermesMessage>? {
        mutable.update { it.copy(notice = "Connection dropped, waiting for the saved reply…") }
        repeat(30) {
            delay(3000)
            val stored = runCatching { client.messages(credential, id) }.getOrNull()
            if (stored != null && stored.size >= before + 2 && stored.last().role == "assistant" && stored.last().text.isNotBlank()) return stored
        }
        return null
    }

    /** Sends a quick action's question with the library context it needs; a failure to gather it still sends the question. */
    fun runAction(action: HermesAction, book: JSONObject? = null) {
        if (mutable.value.busy) return
        viewModelScope.launch {
            val credential = credential() ?: return@launch
            val context = runCatching { gatherContext(action, { path -> client.library(credential, path) }, book) }.getOrDefault("")
            send(wrapContext(action.prompt, context))
        }
    }

    /** Books matching a search, for the "Tell me about a book" picker. */
    suspend fun searchBooks(query: String): List<JSONObject> {
        val credential = credential() ?: return emptyList()
        return runCatching { client.library(credential, "books?q=${java.net.URLEncoder.encode(query.trim(), "UTF-8")}&limit=8") }.getOrDefault(emptyList())
    }

    fun stop() { sending?.cancel() }

    // --- Voice ---

    private fun savePreferences(voice: HermesVoiceState) {
        preferences.edit().putBoolean("speak", voice.speakReplies).putString("voice", voice.voiceId)
            .putBoolean("hands_free", voice.handsFree).putBoolean("barge_in", voice.bargeIn).apply()
    }

    private fun setVoice(change: (HermesVoiceState) -> HermesVoiceState) {
        mutable.update { it.copy(voice = change(it.voice).also(::savePreferences)) }
        talk.refresh()
    }

    fun setSpeakReplies(on: Boolean) {
        setVoice { it.copy(speakReplies = on) }
        if (!on) speaker.stop() else credential()?.let { credential -> viewModelScope.launch { client.warm(credential, mutable.value.voice.voiceId.ifBlank { null }) } }
    }
    fun setVoiceId(id: String) = setVoice { it.copy(voiceId = id) }
    /** Switches the character on the server (as the website's settings do) and reloads it here. */
    fun chooseAvatar(id: String) {
        val credential = credential() ?: return
        val voice = mutable.value.avatar?.voices?.get(id).orEmpty()
        viewModelScope.launch {
            runCatching { client.selectAvatar(credential, id, voice) }
                .onSuccess { client.avatar(credential)?.let { fresh -> mutable.update { it.copy(avatar = fresh) } } }
                .onFailure { error -> mutable.update { it.copy(notice = error.message ?: "Could not change the avatar") } }
        }
    }
    /** Changes what is behind her; the page redraws at once and the choice is saved on the server. */
    fun setScene(scene: String) {
        val credential = credential() ?: return
        mutable.update { state -> state.copy(avatar = state.avatar?.copy(scene = scene)) }
        viewModelScope.launch {
            runCatching { client.saveAvatarLook(credential, JSONObject().put("scene", scene)) }
                .onFailure { error -> mutable.update { it.copy(notice = error.message ?: "Could not save the background") } }
        }
    }
    /** Moves the picture while a slider is dragged (nothing is sent until [saveFrame]). */
    fun previewFrame(frame: HermesAvatarFrame) = mutable.update { state -> state.copy(avatar = state.avatar?.copy(frame = frame)) }
    fun saveFrame() {
        val credential = credential() ?: return
        val avatar = mutable.value.avatar ?: return
        viewModelScope.launch {
            val frame = JSONObject().put("model", avatar.id).put("zoom", avatar.frame.zoom.toDouble()).put("x", avatar.frame.x.toDouble()).put("y", avatar.frame.y.toDouble())
            runCatching { client.saveAvatarLook(credential, JSONObject().put("frame", frame)) }
                .onFailure { error -> mutable.update { it.copy(notice = error.message ?: "Could not save the position") } }
        }
    }
    /** The avatar page was killed by the phone (memory). Hide it so the chat keeps working, and say how to bring it back. */
    fun avatarCrashed() {
        preferences.edit().putBoolean("avatar", false).apply()
        mutable.update { it.copy(showAvatar = false, notice = "The avatar stopped because the phone ran low on memory. Turn it back on in the settings.") }
    }
    fun setShowAvatar(on: Boolean) { preferences.edit().putBoolean("avatar", on).apply(); mutable.update { it.copy(showAvatar = on) } }

    /** How loud her voice is right now (0 to 1), for the avatar's mouth. */
    fun voiceLevel(): Float = speaker.level()
    fun setHandsFree(on: Boolean) = setVoice { it.copy(handsFree = on) }
    fun setBargeIn(on: Boolean) = setVoice { it.copy(bargeIn = on) }

    /** Starts or ends a spoken conversation. The screen has already made sure the microphone permission is granted. */
    fun toggleTalk() {
        if (talk.mode == TalkMode.Off && !mutable.value.voice.speakReplies) setSpeakReplies(true)
        talk.toggle()
    }

    /** Reads one finished message aloud. */
    fun readAloud(text: String) { if (mutable.value.voice.canSpeak) speaker.speakText(text) }

    fun stopSpeaking() { speaker.stop() }

    /** The screen left the foreground: nothing may keep listening or talking. */
    fun leave() { talk.stop(); speaker.stop() }

    override fun onCleared() { talk.stop(); speaker.stop(); super.onCleared() }

    fun rename(id: String, title: String) {
        viewModelScope.launch {
            val credential = credential() ?: return@launch
            runCatching { client.rename(credential, id, title.trim()) }
                .onSuccess { refreshChats() }
                .onFailure { error -> mutable.update { it.copy(notice = "Could not rename the chat: ${error.message ?: "no answer"}") } }
        }
    }

    fun delete(id: String) {
        viewModelScope.launch {
            val credential = credential() ?: return@launch
            // Stop listening to a reply that is still being written, so the screen can move on to a new chat.
            val running = mutable.value.currentId == id && mutable.value.busy
            if (running) { sending?.cancelAndJoin() }
            runCatching { client.delete(credential, id) }
                .onSuccess {
                    if (mutable.value.currentId == id) newChat()
                    refreshChats()
                }
                .onFailure { error -> mutable.update { it.copy(notice = "Could not delete the chat: ${error.message ?: "no answer"}") } }
            // Hermes carries on with a reply it has started and saves the chat when it ends, so look again later.
            if (running) { delay(60_000); runCatching { client.delete(credential, id) }.onSuccess { refreshChats() } }
        }
    }

    fun remember(text: String) {
        viewModelScope.launch {
            val credential = credential() ?: return@launch
            runCatching { client.remember(credential, text.trim()) }
                .onSuccess { mutable.update { it.copy(notice = "Sent to memory") } }
                .onFailure { error -> mutable.update { it.copy(notice = error.message ?: "Could not save it") } }
        }
    }

    fun loadMemories() {
        viewModelScope.launch {
            val credential = credential() ?: return@launch
            mutable.update { it.copy(memories = null, memoriesError = null) }
            runCatching { client.memories(credential) }
                .onSuccess { list -> mutable.update { it.copy(memories = list.orEmpty()) } }
                .onFailure { error -> mutable.update { it.copy(memoriesError = error.message ?: "Could not load her memories") } }
        }
    }

    fun say(text: String) { mutable.update { it.copy(notice = text) } }

    fun clearNotice() { mutable.update { it.copy(notice = null) } }
}
