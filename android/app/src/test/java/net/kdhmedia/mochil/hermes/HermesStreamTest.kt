package net.kdhmedia.mochil.hermes

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class HermesStreamTest {
    private fun feedAll(reader: HermesStreamReader, vararg lines: String) = lines.mapNotNull(reader::feed)

    @Test fun textAndToolStepsAreToldApartAndAStepKeepsItsDetail() {
        val reader = HermesStreamReader()
        val results = feedAll(
            reader,
            """data: {"choices":[{"delta":{"role":"assistant"}}]}""", "",
            "event: hermes.tool.progress",
            """data: {"tool":"web_search","label":"weather in Brussels","toolCallId":"t1","status":"running"}""", "",
            "event: hermes.tool.progress",
            """data: {"tool":"web_search","toolCallId":"t1","status":"completed"}""", "",
            """data: {"choices":[{"delta":{"content":"It is "}}]}""", "",
            """data: {"choices":[{"delta":{"content":"7 degrees."}}]}""", "",
            "data: [DONE]",
        )
        assertEquals("It is 7 degrees.", results.mapNotNull { it.text }.joinToString(""))
        val last = results.mapNotNull { it.steps }.last().single()
        assertEquals(HermesStep("t1", "web_search", "weather in Brussels", HermesStepStatus.Done), last)
    }

    @Test fun aWaitingNoteShowsTheSecondsAndGoesAwayWhenTheFirstWordsArrive() {
        val reader = HermesStreamReader()
        val first = feedAll(reader, "event: hermes.waiting", """data: {"seconds":10}""", "").single()
        assertEquals(listOf(HermesStep("wait", "waiting", "10 s", HermesStepStatus.Running)), first.steps)
        val later = feedAll(reader, "event: hermes.waiting", """data: {"seconds":20}""", "").single()
        assertEquals("20 s", later.steps!!.single().detail)
        val answer = reader.feed("""data: {"choices":[{"delta":{"content":"Hi"}}]}""")!!
        assertEquals("Hi", answer.text)
        assertEquals(emptyList<HermesStep>(), answer.steps)
        assertEquals(emptyList<HermesStep>(), reader.steps())
        assertEquals("Waiting", hermesToolWords("waiting"))
    }

    @Test fun aFailureReportedByTheRelayIsHandedOnAndCommentsAreIgnored() {
        val reader = HermesStreamReader()
        assertNull(reader.feed(": ping"))
        val error = feedAll(reader, "event: hermes.error", """data: {"error":"Hermes is not reachable"}""", "").single()
        assertEquals("Hermes is not reachable", error.error)
    }

    @Test fun anEventNameDoesNotLeakIntoTheNextEvent() {
        val reader = HermesStreamReader()
        feedAll(reader, "event: hermes.status", """data: {"x":1}""", "")
        assertEquals("ok", reader.feed("""data: {"choices":[{"delta":{"content":"ok"}}]}""")!!.text)
    }

    @Test fun brokenJsonAndEmptyDeltasAreSkipped() {
        val reader = HermesStreamReader()
        assertNull(reader.feed("data: {not json"))
        assertNull(reader.feed("""data: {"choices":[{"delta":{"content":""}}]}"""))
        assertNull(reader.feed(""))
    }

    @Test fun cardsTheAssistantOffersToPlayComeThroughAndBadOnesAreDropped() {
        val reader = HermesStreamReader()
        assertNull(reader.feed("event: hermes.cards"))
        val update = reader.feed("""data: {"cards":[{"id":"7","library":"music","kind":"music","title":"Lilac","subtitle":"IU"},{"id":"","library":"music","title":"No id"}]}""")
        assertEquals(listOf(HermesCard("7", "music", "music", "Lilac", "IU")), update?.cards)
    }

    @Test fun unknownToolsGetAReadableName() {
        assertEquals("Searching the web", hermesToolWords("web_search"))
        assertEquals("Controlling the home", hermesToolWords("ha_call_service"))
        assertEquals("Fetch weather", hermesToolWords("fetch_weather"))
        assertEquals("Working", hermesToolWords(""))
    }

    @Test fun listsAndMessagesAreParsedAndStrangeRowsDropped() {
        val chats = parseHermesChats(JSONObject("""{"chats":[{"id":"c1","title":"","updated":5.0,"count":3},{"title":"no id"}]}"""))
        assertEquals(listOf(HermesChat("c1", "New chat", 5.0, 3)), chats)
        val messages = parseHermesMessages(JSONObject("""{"messages":[{"role":"user","text":"hi"},{"role":"tool","text":"x"},{"role":"assistant","text":"  "}]}"""))
        assertEquals(1, messages.size)
        assertEquals("user", messages.single().role)
        assertEquals("hi", messages.single().text)
        assertTrue(messages.single().fromUser)
        assertEquals(listOf(HermesMemory("Kevin lives in Belgium", "d")), parseHermesMemories(JSONObject("""{"items":[{"text":" Kevin lives in Belgium ","date":"d"},{"text":""}]}""")))
        val status = parseHermesStatus(JSONObject("""{"configured":true,"reachable":true,"name":"","memory":true}"""))
        assertEquals(HermesStatus(true, true, "Hermes", true), status)
    }

    @Test
    fun statusReadsTheVoiceAndQuickSwitches() {
        val status = parseHermesStatus(JSONObject("""{"configured":true,"reachable":true,"name":"L","memory":true,"features":{"voice":false,"quick":false}}"""))
        assertEquals(HermesStatus(true, true, "L", true, voice = false, quick = false), status)
    }
}
