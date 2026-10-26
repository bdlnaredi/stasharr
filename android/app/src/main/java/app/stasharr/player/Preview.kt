package app.stasharr.player

import android.view.TextureView
import android.view.ViewGroup
import android.widget.FrameLayout
import androidx.annotation.OptIn
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.tween
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.media3.common.MediaItem
import androidx.media3.common.Player
import androidx.media3.common.VideoSize
import androidx.media3.common.util.UnstableApi
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.ui.AspectRatioFrameLayout
import kotlinx.coroutines.delay

/**
 * One muted, looping player shared by every scene preview, so only one
 * preview plays at a time however many cards are on screen.
 */
object PreviewPlayer {
    private var player: ExoPlayer? = null
    private var owner: Any? = null

    /** The id of the card currently previewing, set by a long press. */
    var activeCardId by mutableStateOf<String?>(null)

    fun get(context: android.content.Context): ExoPlayer =
        player ?: newPlayer(context.applicationContext).also {
            it.volume = 0f
            it.repeatMode = Player.REPEAT_MODE_ONE
            player = it
        }

    fun claim(who: Any, context: android.content.Context, url: String): ExoPlayer {
        val p = get(context)
        owner = who
        p.setMediaItem(MediaItem.fromUri(url))
        p.prepare()
        p.play()
        return p
    }

    fun isOwner(who: Any) = owner === who

    fun release(who: Any) {
        if (owner === who) {
            owner = null
            player?.stop()
            player?.clearVideoSurface()
        }
    }

    fun pause() {
        player?.pause()
    }

    fun resume(who: Any) {
        if (owner === who) player?.play()
    }
}

/**
 * Plays a scene's preview clip, muted and looping, filling the space it is
 * given. It fades in once the first frame is ready, so the picture under it
 * shows until then. delayMs waits before starting, so swiping past does not
 * start previews.
 */
@OptIn(UnstableApi::class)
@Composable
fun ScenePreview(url: String?, active: Boolean, modifier: Modifier = Modifier, delayMs: Long = 0) {
    if (url == null || !active) return
    val context = LocalContext.current
    val token = remember { Any() }
    var showing by remember { mutableStateOf(false) }
    var started by remember { mutableStateOf(false) }
    var aspect by remember { mutableStateOf(16f / 9f) }
    var view by remember { mutableStateOf<TextureView?>(null) }
    val alpha by animateFloatAsState(if (showing) 1f else 0f, tween(350), label = "preview")

    LaunchedEffect(url) {
        if (delayMs > 0) delay(delayMs)
        started = true
    }

    val listener = remember {
        object : Player.Listener {
            override fun onRenderedFirstFrame() {
                showing = true
            }

            override fun onVideoSizeChanged(videoSize: VideoSize) {
                if (videoSize.width > 0 && videoSize.height > 0) {
                    aspect = videoSize.width * videoSize.pixelWidthHeightRatio / videoSize.height
                }
            }
        }
    }

    // start once the view and the delay are both ready
    LaunchedEffect(started, view, url) {
        val v = view ?: return@LaunchedEffect
        if (!started) return@LaunchedEffect
        val p = PreviewPlayer.claim(token, context, url)
        p.removeListener(listener)
        p.addListener(listener)
        p.setVideoTextureView(v)
    }

    DisposableEffect(url) {
        onDispose {
            PreviewPlayer.get(context).removeListener(listener)
            PreviewPlayer.release(token)
        }
    }

    val lifecycle = LocalLifecycleOwner.current.lifecycle
    DisposableEffect(lifecycle) {
        val obs = LifecycleEventObserver { _, e ->
            if (e == Lifecycle.Event.ON_PAUSE) PreviewPlayer.pause()
            if (e == Lifecycle.Event.ON_RESUME) PreviewPlayer.resume(token)
        }
        lifecycle.addObserver(obs)
        onDispose { lifecycle.removeObserver(obs) }
    }

    AndroidView(
        factory = { ctx ->
            AspectRatioFrameLayout(ctx).apply {
                resizeMode = AspectRatioFrameLayout.RESIZE_MODE_ZOOM
                layoutParams = ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT)
                val tv = TextureView(ctx)
                addView(tv, FrameLayout.LayoutParams(FrameLayout.LayoutParams.MATCH_PARENT, FrameLayout.LayoutParams.MATCH_PARENT))
                view = tv
            }
        },
        update = { it.setAspectRatio(aspect) },
        modifier = modifier.alpha(alpha),
    )
}
