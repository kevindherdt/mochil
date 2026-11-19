package net.kdhmedia.mochil.wiki

sealed interface WikiMarkdownBlock {
    data class Heading(val level: Int, val text: String) : WikiMarkdownBlock
    data class Paragraph(val text: String) : WikiMarkdownBlock
    data class ListItem(
        val text: String,
        val marker: String = "•",
        val checked: Boolean? = null,
        val depth: Int = 0,
        val taskOrdinal: Int = -1,
    ) : WikiMarkdownBlock
    data class Quote(val text: String) : WikiMarkdownBlock
    data class Image(val target: String, val description: String) : WikiMarkdownBlock
    data class Properties(val fields: List<WikiProperty>) : WikiMarkdownBlock
    data class Table(val headers: List<String>, val rows: List<List<String>>) : WikiMarkdownBlock
    data object Rule : WikiMarkdownBlock
    data class Code(val language: String, val text: String) : WikiMarkdownBlock
}

data class WikiProperty(val name: String, val value: String)
data class WikiOutlineEntry(val level: Int, val title: String, val blockIndex: Int)
data class WikiMarkdownDocument(val blocks: List<WikiMarkdownBlock>, val outline: List<WikiOutlineEntry>)

enum class WikiInlineStyle { Plain, Bold, Italic, Strikethrough, Code, Link, WikiLink }

data class WikiInlineSegment(
    val text: String,
    val target: String? = null,
    val style: WikiInlineStyle = if (target == null) WikiInlineStyle.Plain else WikiInlineStyle.WikiLink,
)

private val taskListPattern = Regex("""^(\s*)[-*+]\s+\[([ xX])]\s+(.*)$""")
private val unorderedListPattern = Regex("""^(\s*)[-*+]\s+(.*)$""")
private val orderedListPattern = Regex("""^(\s*)(\d+)[.)]\s+(.*)$""")
private val horizontalRulePattern = Regex("""^\s{0,3}((\*\s*){3,}|(-\s*){3,}|(_\s*){3,})$""")
private val obsidianImagePattern = Regex("""^!\[\[([^\]\n|]+)(?:\|([^\]\n]+))?]]\s*$""")
private val markdownImagePattern = Regex("""^!\[([^\]\n]*)]\(([^)\n]+)\)\s*$""")
private val wikiTaskMarker = Regex("""^(\s*[-*+]\s+\[)([ xX])(]\s+.*)$""", RegexOption.MULTILINE)
private val wikiFrontmatterRange = Regex("""\A\uFEFF?---\r?\n.*?^---(?:\r?\n|\z)""", setOf(RegexOption.DOT_MATCHES_ALL, RegexOption.MULTILINE))
private val tableDividerCell = Regex("""^:?-{3,}:?$""")
private const val MAX_WIKI_TABLE_COLUMNS = 20
private const val MAX_WIKI_TABLE_ROWS = 100
private const val MAX_WIKI_TABLE_CELL_CHARS = 500

internal fun parseWikiMarkdown(source: String, checkpoint: () -> Unit = {}): List<WikiMarkdownBlock> {
    val blocks = mutableListOf<WikiMarkdownBlock>()
    val (frontmatter, body) = extractWikiFrontmatter(source)
    if (frontmatter.isNotEmpty()) blocks += WikiMarkdownBlock.Properties(frontmatter)
    val paragraph = mutableListOf<String>()
    val code = mutableListOf<String>()
    var language = ""
    var fenced = false
    var taskOrdinal = 0
    fun flushParagraph() {
        if (paragraph.isNotEmpty()) blocks += WikiMarkdownBlock.Paragraph(paragraph.joinToString(" ").trim())
        paragraph.clear()
    }
    val lines = WikiLineCursor(body.lineSequence().iterator())
    while (true) {
        checkpoint()
        val raw = lines.take() ?: break
        val line = raw.trimEnd()
        if (line.startsWith("```")) {
            if (fenced) { blocks += WikiMarkdownBlock.Code(language, code.joinToString("\n")); code.clear(); fenced = false }
            else { flushParagraph(); language = line.removePrefix("```").trim().take(40); fenced = true }
        } else if (fenced) code += raw
        else when {
            line.isBlank() -> flushParagraph()
            parseWikiTableRow(line)?.let { it.size >= 2 } == true && lines.peek()?.let(::isWikiTableDivider) == true -> {
                flushParagraph()
                val headers = parseWikiTableRow(line)!!.take(MAX_WIKI_TABLE_COLUMNS)
                val rows = mutableListOf<List<String>>()
                lines.take() // delimiter
                while (rows.size < MAX_WIKI_TABLE_ROWS) {
                    checkpoint()
                    val row = lines.peek()?.let(::parseWikiTableRow) ?: break
                    rows += List(headers.size) { column -> row.getOrElse(column) { "" }.take(MAX_WIKI_TABLE_CELL_CHARS) }
                    lines.take()
                }
                // Consume surplus table rows after reaching the render bound so
                // they cannot collapse into one enormous fallback paragraph.
                while (lines.peek()?.let(::parseWikiTableRow) != null) {
                    checkpoint()
                    lines.take()
                }
                blocks += WikiMarkdownBlock.Table(headers.map { it.take(MAX_WIKI_TABLE_CELL_CHARS) }, rows)
                continue
            }
            obsidianImagePattern.matches(line.trim()) -> {
                flushParagraph()
                val match = obsidianImagePattern.matchEntire(line.trim())!!
                val target = match.groupValues[1].trim()
                blocks += WikiMarkdownBlock.Image(target, match.groupValues[2].trim().ifBlank { target.substringAfterLast('/') })
            }
            markdownImagePattern.matches(line.trim()) -> {
                flushParagraph()
                val match = markdownImagePattern.matchEntire(line.trim())!!
                blocks += WikiMarkdownBlock.Image(match.groupValues[2].trim(), match.groupValues[1].trim())
            }
            horizontalRulePattern.matches(line) -> { flushParagraph(); blocks += WikiMarkdownBlock.Rule }
            line.startsWith("#") -> {
                val level = line.takeWhile { it == '#' }.length.coerceIn(1, 6)
                if (line.getOrNull(level) == ' ') { flushParagraph(); blocks += WikiMarkdownBlock.Heading(level, line.drop(level).trim()) }
                else paragraph += line
            }
            line.trimStart().startsWith("> ") -> {
                flushParagraph()
                blocks += WikiMarkdownBlock.Quote(line.trimStart().removePrefix("> ").trim())
            }
            taskListPattern.matches(line) -> {
                flushParagraph()
                val match = taskListPattern.matchEntire(line)!!
                blocks += WikiMarkdownBlock.ListItem(
                    text = match.groupValues[3].trim(),
                    checked = match.groupValues[2].isNotBlank(),
                    depth = match.groupValues[1].length / 2,
                    taskOrdinal = taskOrdinal++,
                )
            }
            orderedListPattern.matches(line) -> {
                flushParagraph()
                val match = orderedListPattern.matchEntire(line)!!
                blocks += WikiMarkdownBlock.ListItem(
                    text = match.groupValues[3].trim(),
                    marker = "${match.groupValues[2]}.",
                    depth = match.groupValues[1].length / 2,
                )
            }
            unorderedListPattern.matches(line) -> {
                flushParagraph()
                val match = unorderedListPattern.matchEntire(line)!!
                blocks += WikiMarkdownBlock.ListItem(
                    text = match.groupValues[2].trim(),
                    depth = match.groupValues[1].length / 2,
                )
            }
            else -> paragraph += line
        }
    }
    if (fenced) blocks += WikiMarkdownBlock.Code(language, code.joinToString("\n"))
    flushParagraph()
    return blocks
}

private class WikiLineCursor(private val source: Iterator<String>) {
    private var next: String? = if (source.hasNext()) source.next() else null
    fun peek(): String? = next
    fun take(): String? = next.also { next = if (source.hasNext()) source.next() else null }
}

internal fun parseWikiTableRow(line: String): List<String>? {
    if ('|' !in line) return null
    val trimmed = line.trim().removePrefix("|").removeSuffix("|")
    val cells = mutableListOf<String>()
    val cell = StringBuilder()
    var escaped = false
    trimmed.forEach { character ->
        when {
            escaped -> { cell.append(character); escaped = false }
            character == '\\' -> escaped = true
            character == '|' -> {
                cells += cell.toString().trim()
                if (cells.size == MAX_WIKI_TABLE_COLUMNS) return cells
                cell.clear()
            }
            else -> cell.append(character)
        }
    }
    if (escaped) cell.append('\\')
    cells += cell.toString().trim()
    return cells.take(MAX_WIKI_TABLE_COLUMNS).takeIf { it.size >= 2 }
}

private fun isWikiTableDivider(line: String): Boolean =
    parseWikiTableRow(line)?.all { tableDividerCell.matches(it.replace(" ", "")) } == true

internal fun parseWikiMarkdownDocument(source: String, outlineLimit: Int = 500, checkpoint: () -> Unit = {}): WikiMarkdownDocument {
    val blocks = parseWikiMarkdown(source, checkpoint)
    return WikiMarkdownDocument(blocks, wikiOutline(blocks, outlineLimit))
}

internal fun wikiOutline(source: String, limit: Int = 500): List<WikiOutlineEntry> = wikiOutline(parseWikiMarkdown(source), limit)

internal fun wikiOutline(blocks: List<WikiMarkdownBlock>, limit: Int = 500): List<WikiOutlineEntry> =
    blocks.asSequence().mapIndexedNotNull { index, block ->
        (block as? WikiMarkdownBlock.Heading)?.let { WikiOutlineEntry(it.level, it.text, index) }
    }.take(limit.coerceIn(1, 500)).toList()

/** Changes one rendered task marker without normalizing any other vault bytes. */
internal fun toggleWikiTask(source: String, taskOrdinal: Int, checked: Boolean): String {
    if (taskOrdinal < 0) return source
    val frontmatterEnd = wikiFrontmatterRange.find(source)?.range?.last?.plus(1) ?: 0
    var fenced = false
    var ordinal = 0
    for (raw in Regex("""[^\r\n]+""").findAll(source)) {
        if (raw.range.first < frontmatterEnd) continue
        val bom = if (raw.range.first == 0 && raw.value.startsWith('\uFEFF')) 1 else 0
        val line = raw.value.drop(bom).trimEnd()
        if (line.startsWith("```")) { fenced = !fenced; continue }
        if (fenced) continue
        val task = wikiTaskMarker.matchEntire(line) ?: continue
        if (ordinal++ != taskOrdinal) continue
        val offset = raw.range.first + bom + (task.groups[2]?.range?.first ?: return source)
        return source.replaceRange(offset, offset + 1, if (checked) "x" else " ")
    }
    return source
}

internal fun extractWikiFrontmatter(source: String): Pair<List<WikiProperty>, String> {
    val rendered = source.removePrefix("\uFEFF")
    val normalized = rendered.replace("\r\n", "\n")
    if (!normalized.startsWith("---\n")) return emptyList<WikiProperty>() to rendered
    val range = wikiFrontmatterRange.find(normalized)?.range ?: return emptyList<WikiProperty>() to source
    val end = range.last + 1
    val closing = normalized.lastIndexOf("\n---", end - 1).coerceAtLeast(4)
    val fields = normalized.substring(4, closing).lineSequence().mapNotNull { line ->
        val separator = line.indexOf(':')
        if (separator <= 0) return@mapNotNull null
        val name = line.substring(0, separator).trim().take(80)
        val value = line.substring(separator + 1).trim().trim('"', '\'').take(500)
        name.takeIf(String::isNotBlank)?.let { WikiProperty(it, value) }
    }.take(100).toList()
    return fields to normalized.substring(end).trimStart('\n')
}
