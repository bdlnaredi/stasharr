package app.stasharr.player

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.layout.boundsInWindow
import androidx.compose.ui.layout.onGloballyPositioned
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.collectLatest
import java.util.concurrent.atomic.AtomicLong

/**
 * Plays previews by themselves, like the website does when you point at a
 * scene: once scrolling stops, the top-most picture that is fully on screen
 * (the banner, the scene page's picture, or a scene card) plays its preview.
 * Only one plays at a time, through PreviewPlayer.
 */
object AutoPreview {
    /** Whether previews start by themselves. Saved; can be turned off under You. */
    var enabled by mutableStateOf(true)
        private set

    private val spots = mutableStateMapOf<String, Rect>()
    private val counter = AtomicLong()

    fun newKey(prefix: String) = "$prefix:${counter.incrementAndGet()}"

    fun restore(context: Context) {
        enabled = context.getSharedPreferences("player", Context.MODE_PRIVATE).getBoolean("auto_preview", true)
    }

    fun setEnabled(context: Context, on: Boolean) {
        enabled = on
        context.getSharedPreferences("player", Context.MODE_PRIVATE).edit().putBoolean("auto_preview", on).apply()
        if (!on) PreviewPlayer.activeCardId = null
    }

    internal fun report(key: String, box: Rect?) {
        if (box == null) spots.remove(key) else spots[key] = box
    }

    internal fun forget(key: String) {
        spots.remove(key)
        if (PreviewPlayer.activeCardId == key) PreviewPlayer.activeCardId = null
    }

    /** The spot to play: the top-most fully visible one, left-most first. */
    private fun pick(all: Map<String, Rect>): String? =
        all.entries
            .sortedWith(compareBy({ (it.value.top / 24f).toInt() }, { it.value.left }))
            .firstOrNull()?.key

    /** Put once near the top of the app: chooses what plays after scrolling stops. */
    @Composable
    fun Controller() {
        LaunchedEffect(Unit) {
            snapshotFlow { enabled to spots.toMap() }.collectLatest { (on, all) ->
                if (!on) return@collectLatest
                // wait for scrolling to settle
                delay(650)
                val key = pick(all)
                if (key != PreviewPlayer.activeCardId) PreviewPlayer.activeCardId = key
            }
        }
    }
}

/**
 * Marks a picture that can play its preview by itself. Reports where it is
 * while it is fully on screen; eligible = false takes it out.
 */
fun Modifier.previewSpot(key: String, eligible: Boolean = true): Modifier = composed {
    DisposableEffect(key) { onDispose { AutoPreview.forget(key) } }
    if (!eligible) {
        DisposableEffect(key, eligible) {
            AutoPreview.report(key, null)
            onDispose { }
        }
        this
    } else {
        this.onGloballyPositioned { c ->
            // mostly on screen: nearly all of its width and most of its
            // height, so a tall banner on a sideways phone still counts
            val b = c.boundsInWindow()
            val w = c.size.width.toFloat()
            val h = c.size.height.toFloat()
            val shown = w > 0 && h > 0 && b.width >= w * 0.9f && b.height >= h * 0.7f
            AutoPreview.report(key, if (shown) b else null)
        }
    }
}

/** A key for one picture on screen, kept while it is shown. */
@Composable
fun rememberPreviewKey(prefix: String): String = remember { AutoPreview.newKey(prefix) }
