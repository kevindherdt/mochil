package net.kdhmedia.mochil.video

import android.content.Context
import net.kdhmedia.mochil.catalogue.BrowseLayoutStore

enum class VideoLayout { Covers, List }

internal class VideoBrowsePreferences(context: Context) {
    private val shared = BrowseLayoutStore(context)

    fun layout(kind: String): VideoLayout = if (shared.isList()) VideoLayout.List else VideoLayout.Covers

    fun setLayout(kind: String, layout: VideoLayout) = shared.setList(layout == VideoLayout.List)

    fun listen(onChange: (VideoLayout) -> Unit) = shared.listen { onChange(if (it) VideoLayout.List else VideoLayout.Covers) }
}
