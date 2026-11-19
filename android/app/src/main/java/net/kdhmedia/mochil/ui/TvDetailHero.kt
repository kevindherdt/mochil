package net.kdhmedia.mochil.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.SubcomposeLayout
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Constraints
import androidx.compose.ui.unit.sp
import androidx.compose.ui.unit.dp

/** One viewport, measured in priority order. No estimates of font/line or control heights.
 * Title and actions are mandatory; story/facts use the space left by the actual episode rail.
 * The rail receives bounded height, so even a short window with the playback dock cannot
 * force focus-reveal scrolling to cut the title against the internal viewport's top edge.
 *
 * When [onStoryOpen] is supplied and the synopsis is longer than the three lines this hero will show, the block carries
 * the same one-stop Read more affordance the show page has: a two-line ellipsis the viewer can open, rather than a
 * description that just stops mid-sentence.
 */
@Composable
internal fun TvDetailHero(
    title: String,
    story: String,
    modifier: Modifier = Modifier,
    headerFraction: Float = 1f,
    facts: @Composable () -> Unit,
    controls: @Composable () -> Unit,
    rail: @Composable () -> Unit = {},
    onStoryOpen: (() -> Unit)? = null,
    storyFocus: androidx.compose.ui.focus.FocusRequester? = null,
) {
    val titleStyle = MaterialTheme.typography.headlineSmall.copy(fontSize = 26.sp, lineHeight = 30.sp)
    val storyStyle = tvDetailStoryStyle(MaterialTheme.typography.bodyMedium)
    SubcomposeLayout(modifier) { constraints ->
        val width = constraints.maxWidth
        val height = constraints.maxHeight
        val gap = 8.dp.roundToPx()
        val header = Constraints(maxWidth = (width * headerFraction).toInt(), maxHeight = height)
        val full = Constraints(maxWidth = width, maxHeight = height)
        fun slot(key: String, limits: Constraints, hidden: Boolean = false, content: @Composable () -> Unit) =
            subcompose(key) { Box(Modifier.fillMaxWidth().then(if (hidden) Modifier.clearAndSetSemantics {} else Modifier)) { content() } }.single().measure(limits)
        // Whether the synopsis is cut off is a measured fact, not an assumption: lay the whole text out unbounded.
        var fullStoryLines = 0
        if (story.isNotBlank()) slot("story-full", header.copy(maxHeight = Constraints.Infinity), hidden = true) {
            Text(story, style = storyStyle, onTextLayout = { fullStoryLines = it.lineCount })
        }
        val more = onStoryOpen != null && fullStoryLines > 3
        fun storySlot(key: String, lines: Int, expandable: Boolean, hidden: Boolean) =
            slot(key, header, hidden = hidden) { TvSynopsis(story, lines, storyStyle, expandable, { onStoryOpen?.invoke() }, readMoreFocus = storyFocus) }
        val heading = slot("title", header) { Text(title, style = titleStyle, fontWeight = FontWeight.SemiBold, maxLines = 2, overflow = TextOverflow.Ellipsis) }
        val actions = slot("controls", full, content = controls)
        // The rail budget is charged one plain line, never the Read more row: the episodes keep their height.
        val oneLine = storySlot("story-base", 1, expandable = false, hidden = true)
        val railBudget = (height - heading.height - actions.height - oneLine.height - gap * 3).coerceAtLeast(0)
        val episodes = slot("rail", full.copy(maxHeight = railBudget), content = rail)
        var remaining = height - heading.height - actions.height - episodes.height - gap * 3
        val factLine = slot("facts", header, hidden = true, content = facts)
        val useFacts = factLine.height > 0 && remaining >= factLine.height + gap + oneLine.height
        if (useFacts) remaining -= factLine.height + gap
        var synopsis = oneLine
        var storyLines = 1
        for (lines in 2..3) {
            val candidate = storySlot("story-$lines", lines, expandable = more, hidden = true)
            if (candidate.height <= remaining) { synopsis = candidate; storyLines = lines }
        }
        val useStory = story.isNotBlank() && synopsis.height <= remaining
        val visibleStory = if (useStory) storySlot("story", storyLines, expandable = more, hidden = false) else null
        val visibleFacts = if (useFacts) slot("facts-visible", header, content = facts) else null
        val factsHeight = visibleFacts?.let { it.height + gap } ?: 0
        val storyHeight = visibleStory?.let { it.height + gap } ?: 0
        val actionsTop = heading.height + gap + factsHeight + storyHeight
        val episodesTop = actionsTop + actions.height + if (episodes.height > 0) gap else 0
        layout(width, episodesTop + episodes.height) {
            heading.placeRelative(0, 0)
            visibleFacts?.placeRelative(0, heading.height + gap)
            visibleStory?.placeRelative(0, heading.height + gap + factsHeight)
            actions.placeRelative(0, actionsTop)
            episodes.placeRelative(0, episodesTop)
        }
    }
}
