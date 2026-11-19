package net.kdhmedia.mochil.hermes

import java.nio.ByteBuffer
import java.nio.ByteOrder
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.CoroutineScope
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class HermesSpeechTest {
    @Test fun speechTextDropsCodeLinksMarkdownEmojiAndStageDirections() {
        assertEquals("Hello there", speechText("**Hello** there"))
        assertEquals("See the docs now", speechText("See [the docs](https://x.example/a) now"))
        assertEquals("Try this", speechText("Try ```val x = 1``` this"))
        assertEquals("I smile", speechText("I *smile*"))
        assertEquals("Good morning", speechText("*smiles softly* Good morning"))
        assertEquals("Hi", speechText("Hi 😊"))
        assertEquals("One two", speechText("# One\n- two"))
        assertEquals("", speechText("https://example.com"))
    }

    @Test fun sentencesAreCompleteOnlyAfterAnEndMarkAndShortOnesJoinTheNext() {
        val first = takeSentences("Good morning, Kevin. How did you sleep last night? I hope wel")
        assertEquals(listOf("Good morning, Kevin. How did you sleep last night?"), first.sentences)
        assertEquals("I hope wel", first.rest)
        val short = takeSentences("Hi. Yes. That is a longer one that stands alone. ")
        assertEquals(listOf("Hi. Yes. That is a longer one that stands alone."), short.sentences)
        val flushed = takeSentences("Almost there", final = true)
        assertEquals(listOf("Almost there"), flushed.sentences)
        assertEquals("", flushed.rest)
        assertEquals(emptyList<String>(), takeSentences("   ", final = true).sentences)
    }

    @Test fun theDetectorStartsAfterAMomentOfSpeechAndEndsAfterAShortSilence() {
        val vad = HermesVad()
        var events = mutableListOf<HermesVad.Event>()
        repeat(40) { vad.feed(0.001, 20.0) }
        repeat(14) { vad.feed(0.2, 20.0)?.let(events::add) } // 280 ms of speech
        assertEquals(listOf(HermesVad.Event.Start), events)
        assertTrue(vad.isSpeaking)
        events = mutableListOf()
        repeat(60) { vad.feed(0.001, 20.0)?.let(events::add) } // 1200 ms of quiet
        assertEquals(listOf(HermesVad.Event.End), events)
        assertFalse(vad.isSpeaking)
    }

    @Test fun aShortNoiseDoesNotStartSpeech() {
        val vad = HermesVad()
        repeat(5) { assertNull(vad.feed(0.3, 20.0)) }
        repeat(10) { assertNull(vad.feed(0.001, 20.0)) }
        assertFalse(vad.isSpeaking)
    }

    @Test fun whatWasHeardIsClassified() {
        assertEquals(HeardKind.Empty, classifyHeard("  [BLANK_AUDIO] "))
        assertEquals(HeardKind.Stop, classifyHeard("Stop listening!"))
        assertEquals(HeardKind.Phantom, classifyHeard("Thanks for watching.", 1200))
        assertEquals(HeardKind.Say, classifyHeard("Thanks for watching.", 4000))
        assertEquals(HeardKind.Say, classifyHeard("What's the weather like?"))
    }

    @Test fun theWavIsMono16kAndSizedRight() {
        val wav = encodeWav16k(shortArrayOf(1, -2, 3))
        val b = ByteBuffer.wrap(wav).order(ByteOrder.LITTLE_ENDIAN)
        assertEquals(44 + 6, wav.size)
        assertEquals("RIFF", String(wav, 0, 4))
        assertEquals("WAVE", String(wav, 8, 4))
        assertEquals(16000, b.getInt(24))
        assertEquals(1, b.getShort(22).toInt())
        assertEquals(6, b.getInt(40))
        assertEquals(-2, b.getShort(46).toInt())
    }

    @Test fun theSampleSplitterNeverBreaksASample() {
        val splitter = SampleSplitter()
        assertEquals(2, splitter.feed(byteArrayOf(1, 2, 3)).size)
        assertEquals(4, splitter.feed(byteArrayOf(4, 5, 6)).size)
        assertEquals(0, splitter.feed(byteArrayOf()).size)
    }

    private class FakeMicrophone : TalkMicrophone {
        var started = false; var mutedNow = false; var pickierNow = false
        override fun start() { started = true }
        override fun stop() { started = false }
        override fun setMuted(muted: Boolean) { mutedNow = muted }
        override fun hearAbove(speaking: Boolean) { pickierNow = speaking }
    }

    private fun talk(handsFree: Boolean, bargeIn: Boolean, heard: String, speaking: () -> Boolean = { false }, sent: MutableList<Pair<String, Boolean>>, mic: FakeMicrophone, interrupts: IntArray, modes: MutableList<TalkMode> = mutableListOf()): Pair<HermesTalk, () -> TalkHandlers> {
        var handlers: TalkHandlers? = null
        val talk = HermesTalk(
            scope = CoroutineScope(Dispatchers.Unconfined),
            makeMicrophone = { handlers = it; mic },
            isSpeaking = speaking,
            transcribe = { heard },
            send = { text, interrupted -> sent += text to interrupted },
            interrupt = { interrupts[0] += 1 },
            options = { TalkOptions(handsFree, bargeIn) },
            onMode = { modes += it },
        )
        return talk to { handlers!! }
    }

    @Test fun aHandsFreeTurnListensHearsSendsAndGoesBackToListening() {
        val sent = mutableListOf<Pair<String, Boolean>>(); val mic = FakeMicrophone(); val interrupts = intArrayOf(0)
        val (talk, handlers) = talk(true, false, "Good morning", sent = sent, mic = mic, interrupts = interrupts)
        talk.toggle()
        assertEquals(TalkMode.Listening, talk.mode); assertTrue(mic.started)
        handlers().onSpeechStart()
        assertEquals(TalkMode.Hearing, talk.mode)
        handlers().onUtterance(ByteArray(10), 3000)
        assertEquals(listOf("Good morning" to false), sent)
        assertEquals(TalkMode.Waiting, talk.mode)
        assertTrue("muted while she answers unless barge-in", mic.mutedNow)
        talk.replyDone()
        assertEquals(TalkMode.Listening, talk.mode); assertTrue(mic.started)
    }

    @Test fun withoutHandsFreeTheMicrophoneClosesAfterOneTurn() {
        val sent = mutableListOf<Pair<String, Boolean>>(); val mic = FakeMicrophone()
        val (talk, handlers) = talk(false, false, "Hello", sent = sent, mic = mic, interrupts = intArrayOf(0))
        talk.toggle(); handlers().onUtterance(ByteArray(2), 1000)
        assertFalse(mic.started)
        talk.replyDone()
        assertEquals(TalkMode.Off, talk.mode)
    }

    @Test fun silenceAndPhantomsAreNotSentAndStopEndsTheConversation() {
        val sent = mutableListOf<Pair<String, Boolean>>(); val mic = FakeMicrophone()
        val (phantom, h1) = talk(true, false, "Thanks for watching", sent = sent, mic = mic, interrupts = intArrayOf(0))
        phantom.toggle(); h1().onUtterance(ByteArray(2), 800)
        assertTrue(sent.isEmpty()); assertEquals(TalkMode.Listening, phantom.mode)
        val (stopper, h2) = talk(true, false, "stop listening", sent = sent, mic = mic, interrupts = intArrayOf(0))
        stopper.toggle(); h2().onUtterance(ByteArray(2), 800)
        assertTrue(sent.isEmpty()); assertEquals(TalkMode.Off, stopper.mode); assertFalse(mic.started)
    }

    @Test fun talkingOverHerInterruptsHerAndTellsHerNextTime() {
        val sent = mutableListOf<Pair<String, Boolean>>(); val mic = FakeMicrophone(); val interrupts = intArrayOf(0)
        var speaking = true
        val (talk, handlers) = talk(true, true, "Wait a moment", speaking = { speaking }, sent = sent, mic = mic, interrupts = interrupts)
        talk.toggle()
        assertFalse("barge-in keeps the microphone open while she speaks", mic.mutedNow)
        assertTrue("and it is pickier then", mic.pickierNow)
        handlers().onSpeechStart()
        assertEquals(1, interrupts[0])
        speaking = false
        handlers().onUtterance(ByteArray(2), 2000)
        assertEquals(listOf("Wait a moment" to true), sent)
    }
}

class HermesAvatarTest {
    private val origin = "https://books.example"

    @Test fun theAvatarPageCanOnlyReachAvatarFilesAndAssetsOnTheServer() {
        assertEquals("$origin/api/android/v1/avatar/core", avatarTarget(origin, origin, "/api/avatar/core"))
        assertEquals("$origin/api/android/v1/avatar/files/Ice/Ice.model3.json", avatarTarget(origin, origin, "/api/avatar/files/Ice/Ice.model3.json"))
        assertEquals("$origin/assets/hermes-avatar.js", avatarTarget(origin, origin, "/assets/hermes-avatar.js"))
        assertNull(avatarTarget(origin, "https://evil.example", "/api/avatar/core"))
        assertNull(avatarTarget(origin, origin, "/api/hermes/chats"))
        assertNull(avatarTarget(origin, origin, "/api/avatar/../hermes/chats"))
        assertNull(avatarTarget(origin, origin, "/assets/../config/config.json"))
        assertNull(avatarTarget(origin, origin, "/"))
    }

    @Test fun percentEncodedTraversalSegmentsCannotEscapeTheAssetRoot() {
        // A literal `!path.contains("..")` check never sees a traversal segment that is percent-encoded,
        // so it would have let this straight through as if it were a plain file under /assets/.
        assertNull(avatarTarget(origin, origin, "/assets/%2e%2e/%2e%2e/config/config.json"))
        assertNull(avatarTarget(origin, origin, "/assets/%2e%2e%2fconfig.json"))
        assertNull(avatarTarget(origin, origin, "/api/avatar/%2e%2e/hermes/chats"))
        // Doubly-encoded ("%252e" decodes once to "%2e", decodes again to ".") must not slip past either.
        assertNull(avatarTarget(origin, origin, "/assets/%252e%252e/config.json"))
        // A legitimate file one level deep must still resolve normally.
        assertEquals("$origin/assets/models/Ice/Ice.model3.json", avatarTarget(origin, origin, "/assets/models/Ice/Ice.model3.json"))
    }

    @Test fun theOldSubstringCheckWouldHaveLetAnEncodedTraversalThrough() {
        // This reproduces exactly what the old `avatarTarget` check did, to document the bug it had:
        // a literal substring search for ".." never matches "%2e%2e", so it passed the guard.
        val path = "/assets/%2e%2e/%2e%2e/config/config.json"
        val oldCheckPassed = path.startsWith("/assets/") && !path.contains("..")
        assertTrue(oldCheckPassed)
        // ...while the fixed implementation now rejects the very same path.
        assertNull(avatarTarget(origin, origin, path))
    }

    @Test fun theAvatarListGivesChoicesAndEachModelsVoice() {
        val info = parseHermesAvatarInfo(org.json.JSONObject("""{"core":true,"selected":"Ice","models":[{"id":"Hiyori","name":"Hiyori"},{"id":"Ice","name":"Ice"},{"id":".","name":"Root"}],"voices":{"Ice":"alba"}}"""))!!
        assertEquals("Ice", info.id)
        assertEquals("alba", info.voice)
        assertEquals(listOf("Hiyori", "Ice", "Root"), info.choices.map { it.label })
        assertEquals(mapOf("Ice" to "alba"), info.voices)
    }

    @Test fun theSavedBackgroundAndPositionComeWithTheAvatar() {
        val info = parseHermesAvatarInfo(org.json.JSONObject("""{"core":true,"selected":"Ice","models":[{"id":"Ice"}],"scene":"night","image":42,"frames":{"Ice":{"zoom":1.4,"x":0.1,"y":-0.2},"Other":{"zoom":2,"x":0,"y":0}}}"""))!!
        assertEquals("night", info.scene)
        assertEquals(HermesAvatarFrame(1.4f, 0.1f, -0.2f), info.frame)
        val look = org.json.JSONObject(info.lookJson())
        assertEquals("night", look.getString("scene"))
        assertEquals(42L, look.getLong("image"))
        assertEquals(1.4, look.getJSONObject("frame").getDouble("zoom"), 0.001)
        val plain = parseHermesAvatarInfo(org.json.JSONObject("""{"core":true,"selected":"Ice","models":[{"id":"Ice"}]}"""))!!
        assertEquals("", plain.scene)
        assertEquals(HermesAvatarFrame(), plain.frame)
    }

    @Test fun theOriginIsTheServersSchemeAndHost() {
        assertEquals("https://books.example", avatarOrigin("https://books.example/"))
        assertEquals("https://10.0.2.2:18443", avatarOrigin("https://10.0.2.2:18443"))
    }

    @Test fun theChosenAvatarIsReadFromTheModelList() {
        val list = org.json.JSONObject("""{"core":true,"selected":"b","models":[{"id":"a","model":"a.model3.json","expressions":[]},{"id":"b","model":"b.model3.json","expressions":[{"name":"blush","file":"b.exp3.json"}]}],"voices":{"b":"calm"}}""")
        val info = parseHermesAvatarInfo(list)!!
        assertEquals("calm", info.voice)
        assertEquals("b", org.json.JSONObject(info.model).getString("id"))
        assertNull(parseHermesAvatarInfo(org.json.JSONObject("""{"core":false,"models":[{"id":"a"}]}""")))
        assertNull(parseHermesAvatarInfo(org.json.JSONObject("""{"core":true,"models":[]}""")))
        assertEquals("a", org.json.JSONObject(parseHermesAvatarInfo(org.json.JSONObject("""{"core":true,"selected":"gone","models":[{"id":"a"}]}"""))!!.model).getString("id"))
    }
}

class HermesActionsTest {
    private fun book(json: String) = org.json.JSONObject(json)

    @Test fun aBookIsOneLineWithAuthorSeriesAndProgress() {
        assertEquals("- Dune by Frank Herbert (Dune #1), 40% read", bookLine(book("""{"title":"Dune","authors":["Frank Herbert"],"series":"Dune","series_index":1,"reading_progress":{"fraction":0.4}}""")))
        assertEquals("- Emma by Jane Austen, finished", bookLine(book("""{"title":"Emma","authors":["Jane Austen"],"reading_progress":{"finished":true,"fraction":1}}""")))
        assertEquals("- Untitled", bookLine(book("{}")))
        assertEquals("- Odd (Set #1.5)", bookLine(book("""{"title":"Odd","series":"Set","series_index":1.5}""")))
    }

    @Test fun descriptionsLoseMarkupAndAreCut() {
        assertEquals("Fish & chips", plainText("<p>Fish &amp; <b>chips</b></p>"))
        val cut = plainText("word ".repeat(400), 50)
        assertTrue(cut.endsWith("…") && cut.length <= 51)
    }

    @Test fun contextIsWrappedAndSplitBackApart() {
        val wrapped = wrapContext("What is new?", "Recently added books:\n- A")
        assertEquals(SplitContext("What is new?", "Recently added books:\n- A"), splitContext(wrapped))
        assertEquals("Plain", wrapContext("Plain", ""))
        assertEquals(SplitContext("Plain", ""), splitContext("Plain"))
    }

    @Test fun eachActionGathersItsOwnContext() = kotlinx.coroutines.runBlocking {
        val asked = mutableListOf<String>()
        val api: suspend (String) -> List<org.json.JSONObject> = { path ->
            asked += path
            if (path.startsWith("reading?finished")) throw java.io.IOException("down") else listOf(book("""{"title":"B","authors":["A"]}"""))
        }
        val recent = gatherContext(HERMES_ACTIONS.first { it.id == "recent" }, api)
        assertEquals("Recently added books:\n- B by A", recent)
        val next = gatherContext(HERMES_ACTIONS.first { it.id == "next" }, api)
        assertTrue("one failing list does not stop the others", next.contains("Reading now:") && next.contains("Recently added:") && !next.contains("Recently finished"))
        val chosen = gatherContext(HERMES_ACTIONS.first { it.id == "book" }, api, book("""{"title":"X","description":"<i>Nice</i>","library_name":"Books","language":"en"}"""))
        assertEquals("- X\nLibrary: Books, language: en\nDescription: Nice", chosen)
        assertEquals("", gatherContext(HERMES_ACTIONS.first { it.id == "polish" }, api))
    }
}
