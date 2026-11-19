package net.kdhmedia.mochil.reader.epub

import android.content.Context
import android.view.View
import android.view.ViewGroup
import android.widget.Scroller
import java.lang.reflect.Modifier
import java.util.Collections
import java.util.WeakHashMap
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch
import org.readium.r2.navigator.epub.EpubNavigatorFragment

/** A scroller that arrives at once: a swipe then lands on the next page without sliding there. */
internal class InstantScroller(context: Context) : Scroller(context, null) {
    override fun startScroll(startX: Int, startY: Int, dx: Int, dy: Int, duration: Int) = super.startScroll(startX, startY, dx, dy, 0)
    override fun startScroll(startX: Int, startY: Int, dx: Int, dy: Int) = super.startScroll(startX, startY, dx, dy, 0)
}

/**
 * Makes swipes turn the page without sliding, for "Reduce animations". Readium settles a swipe with a scroller
 * of its own, in its page views, so the scroller in each is replaced. Found by type, not by name, so it still
 * works in a release build that renames the library's classes; a view it cannot change is left alone.
 */
internal class EpubInstantPages(private val scope: CoroutineScope) {
    private var following: Job? = null
    private val done: MutableSet<View> = Collections.newSetFromMap(WeakHashMap())

    fun apply(navigator: EpubNavigatorFragment?) {
        navigator ?: return
        patch(navigator.publicationView)
        if (following == null) following = scope.launch {
            // Each chapter turned to is a new page view.
            navigator.currentLocator.collectLatest { delay(100); patch(navigator.publicationView) }
        }
    }

    private fun patch(view: View) {
        if (done.add(view)) replaceScrollers(view)
        if (view is ViewGroup) for (i in 0 until view.childCount) patch(view.getChildAt(i))
    }

    private fun replaceScrollers(view: View) {
        var type: Class<*>? = view.javaClass
        // Only the library's own classes are looked into, never Android's.
        while (type != null && !type.name.startsWith("android.")) {
            type.declaredFields.filter { it.type == Scroller::class.java && !Modifier.isStatic(it.modifiers) }.forEach { field ->
                runCatching {
                    field.isAccessible = true
                    if (field.get(view) !is InstantScroller) field.set(view, InstantScroller(view.context))
                }
            }
            type = type.superclass
        }
    }
}
