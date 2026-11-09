package app.stasharr.player

import android.view.TextureView
import androidx.annotation.OptIn
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.layout.boundsInWindow
import androidx.compose.ui.layout.onGloballyPositioned
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import androidx.media3.common.MediaItem
import androidx.media3.common.Player
import androidx.media3.common.VideoSize
import androidx.media3.common.util.UnstableApi
import androidx.media3.exoplayer.ExoPlayer
import coil.compose.AsyncImage

/**
 * A few short silent loops (tag clips) can play at once, unlike scene
 * previews where one plays at a time. Each tile on screen takes a slot while
 * it is mostly visible and gives it back when it scrolls away.
 */
object ClipSlots {
    private const val MAX = 4
    private val holders = mutableSetOf<Any>()

    /** Changes whenever a slot frees up, so waiting tiles try again. */
    var freed by mutableIntStateOf(0)
        private set

    fun take(who: Any): Boolean {
        if (who in holders) return true
        if (holders.size >= MAX) return false
        holders += who
        return true
    }

    fun give(who: Any) {
        if (holders.remove(who)) freed++
    }
}

/**
 * A tag's looping clip filling its tile: the still picture first, then the
 * clip once the tile is mostly on screen and a slot is free. The video is
 * cropped to fill the tile. When previews are switched off under You, only
 * the still shows.
 */
@OptIn(UnstableApi::class)
@Composable
fun LoopClip(video: String, still: String?, modifier: Modifier = Modifier) {
    val context = LocalContext.current
    val token = remember { Any() }
    var visible by remember { mutableStateOf(false) }
    var player by remember { mutableStateOf<ExoPlayer?>(null) }
    var showing by remember { mutableStateOf(false) }
    var tileW by remember { mutableStateOf(0f) }
    var tileH by remember { mutableStateOf(0f) }
    var videoAspect by remember { mutableStateOf(16f / 9f) }
    val alpha by animateFloatAsState(if (showing) 1f else 0f, tween(400), label = "clip")
    val freed = ClipSlots.freed
    val enabled = AutoPreview.enabled

    // take a player while visible, give it back when not
    LaunchedEffect(visible, freed, enabled, video) {
        if (visible && enabled && player == null && ClipSlots.take(token)) {
            player = newPlayer(context.applicationContext).apply {
                volume = 0f
                repeatMode = Player.REPEAT_MODE_ONE
                addListener(object : Player.Listener {
                    override fun onRenderedFirstFrame() {
                        showing = true
                    }

                    override fun onVideoSizeChanged(videoSize: VideoSize) {
                        if (videoSize.width > 0 && videoSize.height > 0) {
                            videoAspect = videoSize.width * videoSize.pixelWidthHeightRatio / videoSize.height
                        }
                    }
                })
                setMediaItem(MediaItem.fromUri(video))
                prepare()
                play()
            }
        } else if ((!visible || !enabled) && player != null) {
            player?.release()
            player = null
            showing = false
            ClipSlots.give(token)
        }
    }

    DisposableEffect(Unit) {
        onDispose {
            player?.release()
            player = null
            ClipSlots.give(token)
        }
    }

    Box(
        modifier
            .clipToBounds()
            .onGloballyPositioned { c ->
                val b = c.boundsInWindow()
                tileW = c.size.width.toFloat()
                tileH = c.size.height.toFloat()
                val area = tileW * tileH
                visible = area > 0 && (b.width * b.height) / area >= 0.5f
            }
    ) {
        if (still != null) {
            AsyncImage(still, null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
        }
        val p = player
        if (p != null) {
            // crop the video to fill the tile, like the picture under it
            val tileAspect = if (tileH > 0) tileW / tileH else videoAspect
            val sx = if (videoAspect > tileAspect) videoAspect / tileAspect else 1f
            val sy = if (videoAspect < tileAspect) tileAspect / videoAspect else 1f
            AndroidView(
                factory = { ctx -> TextureView(ctx) },
                update = { v -> p.setVideoTextureView(v) },
                modifier = Modifier
                    .fillMaxSize()
                    .graphicsLayer { scaleX = sx; scaleY = sy }
                    .alpha(alpha),
            )
        }
    }
}
