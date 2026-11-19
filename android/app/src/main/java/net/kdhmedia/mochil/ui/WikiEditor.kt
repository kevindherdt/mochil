package net.kdhmedia.mochil.ui

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.unit.dp
import net.kdhmedia.mochil.wiki.acceptsWikiDraft

internal fun acceptWikiEditorValue(previous: TextFieldValue, next: TextFieldValue): TextFieldValue =
    if (acceptsWikiDraft(next.text)) next else previous

internal fun applyWikiMarkup(value: TextFieldValue, prefix: String, suffix: String, placeholder: String): TextFieldValue {
    val start = value.selection.min.coerceIn(0, value.text.length)
    val end = value.selection.max.coerceIn(start, value.text.length)
    val selected = value.text.substring(start, end)
    val body = selected.ifBlank { placeholder }
    val replacement = prefix + body + suffix
    val text = value.text.replaceRange(start, end, replacement)
    val selection = if (selected.isBlank()) TextRange(start + prefix.length, start + prefix.length + body.length)
    else TextRange(start + replacement.length)
    return TextFieldValue(text, selection)
}

/** Applies block Markdown at line boundaries so toolbar actions remain valid in Obsidian. */
internal fun applyWikiLinePrefix(value: TextFieldValue, prefix: String, placeholder: String): TextFieldValue {
    val text = value.text
    val selectionStart = value.selection.min.coerceIn(0, text.length)
    val selectionEnd = value.selection.max.coerceIn(selectionStart, text.length)
    val lineStart = text.lastIndexOf('\n', (selectionStart - 1).coerceAtLeast(0)).let { if (selectionStart == 0) 0 else it + 1 }
    val selectedLastCharacter = if (selectionEnd > selectionStart) selectionEnd - 1 else selectionStart
    val lineEnd = text.indexOf('\n', selectedLastCharacter.coerceAtMost(text.length)).let { if (it < 0) text.length else it }
    val body = text.substring(lineStart, lineEnd)
    if (selectionStart == selectionEnd && body.isBlank()) {
        val replacement = prefix + placeholder
        return TextFieldValue(
            text.replaceRange(lineStart, lineEnd, replacement),
            TextRange(lineStart + prefix.length, lineStart + replacement.length),
        )
    }
    val prefixed = body.lineSequence().joinToString("\n") { prefix + it }
    return TextFieldValue(text.replaceRange(lineStart, lineEnd, prefixed), TextRange(lineStart + prefixed.length))
}

/** Wraps complete selected lines so fenced blocks never split Markdown syntax mid-line. */
internal fun applyWikiFencedBlock(value: TextFieldValue, language: String = ""): TextFieldValue {
    val text = value.text
    val selectionStart = value.selection.min.coerceIn(0, text.length)
    val selectionEnd = value.selection.max.coerceIn(selectionStart, text.length)
    val lineStart = text.lastIndexOf('\n', (selectionStart - 1).coerceAtLeast(0)).let { if (selectionStart == 0) 0 else it + 1 }
    val selectedLastCharacter = if (selectionEnd > selectionStart) selectionEnd - 1 else selectionStart
    val lineEnd = text.indexOf('\n', selectedLastCharacter.coerceAtMost(text.length)).let { if (it < 0) text.length else it }
    val selected = text.substring(lineStart, lineEnd).ifBlank { "code" }
    val opening = "```$language\n"
    val replacement = "$opening$selected\n```"
    return TextFieldValue(
        text.replaceRange(lineStart, lineEnd, replacement),
        if (selectionStart == selectionEnd) TextRange(lineStart + opening.length, lineStart + opening.length + selected.length)
        else TextRange(lineStart + replacement.length),
    )
}

@Composable
internal fun WikiEditor(pageId: String, draft: String, onDraft: (String) -> Unit) {
    var value by remember(pageId) { mutableStateOf(TextFieldValue(draft, TextRange(draft.length))) }
    LaunchedEffect(draft) {
        if (draft != value.text) value = TextFieldValue(draft, TextRange(draft.length))
    }
    fun submit(next: TextFieldValue) {
        value = acceptWikiEditorValue(value, next)
        onDraft(next.text)
    }
    fun apply(prefix: String, suffix: String, placeholder: String) {
        submit(applyWikiMarkup(value, prefix, suffix, placeholder))
    }
    fun applyLine(prefix: String, placeholder: String) {
        submit(applyWikiLinePrefix(value, prefix, placeholder))
    }
    fun applyCodeBlock() {
        submit(applyWikiFencedBlock(value))
    }
    Column(Modifier.fillMaxSize().padding(bottom = 16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Row(
            Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            TextButton(onClick = { applyLine("# ", "Heading") }) { Text("Heading") }
            TextButton(onClick = { apply("**", "**", "bold text") }) { Text("Bold") }
            TextButton(onClick = { apply("*", "*", "italic text") }) { Text("Italic") }
            TextButton(onClick = { apply("~~", "~~", "struck text") }) { Text("Strike") }
            TextButton(onClick = { apply("`", "`", "code") }) { Text("Inline code") }
            TextButton(onClick = ::applyCodeBlock) { Text("Code block") }
            TextButton(onClick = { applyLine("- ", "list item") }) { Text("List") }
            TextButton(onClick = { applyLine("1. ", "list item") }) { Text("Numbered") }
            TextButton(onClick = { applyLine("- [ ] ", "task") }) { Text("Task") }
            TextButton(onClick = { applyLine("> ", "quote") }) { Text("Quote") }
            TextButton(onClick = { apply("[", "](https://)", "link text") }) { Text("Link") }
            TextButton(onClick = { apply("[[", "]]", "Page name") }) { Text("Wiki link") }
        }
        OutlinedTextField(
            value = value,
            onValueChange = ::submit,
            modifier = Modifier.fillMaxSize(),
            textStyle = androidx.compose.material3.MaterialTheme.typography.bodyMedium.copy(fontFamily = FontFamily.Monospace),
            label = { Text("Markdown") },
        )
    }
}
