package app.stasharr.player

import android.app.PictureInPictureParams
import android.content.Context
import android.content.Intent
import android.content.pm.ActivityInfo
import android.content.res.Configuration
import android.os.Bundle
import android.os.SystemClock
import android.util.Rational
import android.view.View
import android.view.ViewGroup
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.annotation.OptIn
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.rounded.SkipNext
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.lifecycle.lifecycleScope
import androidx.media3.common.AudioAttributes
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.MediaMetadata
import androidx.media3.common.MimeTypes
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultDataSource
import androidx.media3.datasource.okhttp.OkHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.ui.PlayerView
import app.stasharr.app
import app.stasharr.data.Scene
import app.stasharr.ui.components.GradientButton
import app.stasharr.ui.theme.Brand
import app.stasharr.ui.theme.StasharrTheme
import coil.compose.AsyncImage
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

/** An ExoPlayer that uses the app's login for every request. */
@OptIn(UnstableApi::class)
fun newPlayer(context: Context): ExoPlayer {
    val http = OkHttpDataSource.Factory(app.api.client)
    val data = DefaultDataSource.Factory(context, http)
    return ExoPlayer.Builder(context)
        .setMediaSourceFactory(DefaultMediaSourceFactory(data))
        .setAudioAttributes(
            AudioAttributes.Builder()
                .setUsage(C.USAGE_MEDIA)
                .setContentType(C.AUDIO_CONTENT_TYPE_MOVIE)
                .build(),
            true,
        )
        .setSeekBackIncrementMs(10_000)
        .setSeekForwardIncrementMs(10_000)
        .build()
}

@OptIn(UnstableApi::class)
class PlayerActivity : ComponentActivity() {

    companion object {
        private const val EXTRA_ID = "id"
        private const val EXTRA_START = "start"
        private const val EXTRA_QUEUE = "queue"

        fun intent(context: Context, id: String, startAt: Double, queue: List<String>): Intent =
            Intent(context, PlayerActivity::class.java)
                .putExtra(EXTRA_ID, id)
                .putExtra(EXTRA_START, startAt)
                .putStringArrayListExtra(EXTRA_QUEUE, ArrayList(queue))
    }

    private lateinit var player: ExoPlayer
    private var playerView: PlayerView? = null

    private var scene by mutableStateOf<Scene?>(null)
    private var error by mutableStateOf<String?>(null)
    private var controlsVisible by mutableStateOf(true)
    private var inPip by mutableStateOf(false)
    private var upNext by mutableStateOf(0)

    private var sceneId = ""
    private var queue = ArrayList<String>()
    private var sources: List<String> = emptyList()
    private var sourceIndex = 0
    // transcoded streams start at the requested time, so their clock is offset
    private var offsetSeconds = 0.0

    private var playingSince = 0L
    private var unsavedPlay = 0.0
    private var watched = 0.0
    private var countedPlay = false

    private val listener = object : Player.Listener {
        override fun onIsPlayingChanged(isPlaying: Boolean) {
            if (isPlaying) {
                playingSince = SystemClock.elapsedRealtime()
            } else {
                addPlayTime()
                saveProgress()
            }
            playerView?.keepScreenOn = isPlaying
        }

        override fun onPlaybackStateChanged(state: Int) {
            if (state == Player.STATE_ENDED) {
                addPlayTime()
                saveProgress(finished = true)
                playNext()
            }
        }

        override fun onPlayerError(e: PlaybackException) {
            // try the next way of streaming, from where we were
            if (sourceIndex < sources.size - 1) {
                val at = currentSeconds()
                sourceIndex++
                playSource(at)
            } else {
                error = "This video can't be played on this phone.\n${e.errorCodeName}"
            }
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        WindowCompat.setDecorFitsSystemWindows(window, false)
        hideSystemBars()

        player = newPlayer(this)
        player.addListener(listener)

        readIntent(intent)

        setContent {
            StasharrTheme {
                PlayerUi()
            }
        }

        // save the position now and then while playing
        lifecycleScope.launch {
            while (isActive) {
                delay(15_000)
                if (player.isPlaying) {
                    addPlayTime()
                    playingSince = SystemClock.elapsedRealtime()
                    saveProgress()
                }
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        addPlayTime()
        saveProgress()
        readIntent(intent)
    }

    private fun readIntent(intent: Intent) {
        queue = intent.getStringArrayListExtra(EXTRA_QUEUE) ?: ArrayList()
        upNext = queue.size
        val id = intent.getStringExtra(EXTRA_ID) ?: return finish()
        loadScene(id, intent.getDoubleExtra(EXTRA_START, 0.0))
    }

    private fun hideSystemBars() {
        val c = WindowCompat.getInsetsController(window, window.decorView)
        c.systemBarsBehavior = WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
        c.hide(WindowInsetsCompat.Type.systemBars())
    }

    private fun loadScene(id: String, start: Double) {
        sceneId = id
        scene = null
        error = null
        countedPlay = false
        watched = 0.0
        unsavedPlay = 0.0
        player.stop()
        lifecycleScope.launch {
            try {
                val s = app.repo.scene(id) ?: throw IllegalStateException("This scene no longer exists")
                scene = s
                val f = s.file
                requestedOrientation = when {
                    s.isAudio || f == null || f.width == 0 -> ActivityInfo.SCREEN_ORIENTATION_UNSPECIFIED
                    f.width >= f.height -> ActivityInfo.SCREEN_ORIENTATION_SENSOR_LANDSCAPE
                    else -> ActivityInfo.SCREEN_ORIENTATION_SENSOR_PORTRAIT
                }
                sources = buildSources(s)
                sourceIndex = 0
                if (sources.isEmpty()) {
                    error = "This scene has no file to play"
                } else {
                    playSource(start.coerceAtLeast(0.0))
                }
            } catch (e: Exception) {
                error = e.message ?: "Couldn't load this scene"
            }
        }
    }

    private fun isDirect(url: String) = url.substringBefore('?').endsWith("/stream")
    private fun isHls(url: String) = url.substringBefore('?').endsWith(".m3u8")

    /** The original file first (this phone plays far more formats than a browser), then the server's conversions. */
    private fun buildSources(s: Scene): List<String> {
        val direct = s.paths.stream
        val others = s.streams.map { it.url }.sortedBy { url ->
            when {
                isDirect(url) -> 0
                isHls(url) -> 1
                url.contains(".mp4") -> 2
                url.contains(".mp3") -> 2
                else -> 3
            }
        }
        return (listOfNotNull(direct) + others).distinct()
    }

    private fun playSource(startSeconds: Double) {
        val s = scene ?: return
        var url = sources.getOrNull(sourceIndex) ?: return
        val seekable = isDirect(url) || isHls(url)
        offsetSeconds = 0.0
        if (!seekable && startSeconds > 1) {
            url += (if (url.contains('?')) "&" else "?") + "start=" + startSeconds.toInt()
            offsetSeconds = startSeconds.toInt().toDouble()
        }
        val item = MediaItem.Builder()
            .setUri(url)
            .apply { if (isHls(url)) setMimeType(MimeTypes.APPLICATION_M3U8) }
            .setMediaMetadata(
                MediaMetadata.Builder()
                    .setTitle(s.displayTitle)
                    .setArtist(s.studio?.name)
                    .build()
            )
            .build()
        val startMs = if (seekable) (startSeconds * 1000).toLong() else 0L
        player.setMediaItem(item, startMs)
        player.prepare()
        player.play()
        error = null
    }

    private fun currentSeconds(): Double = offsetSeconds + player.currentPosition / 1000.0

    private fun addPlayTime() {
        if (playingSince > 0) {
            val secs = (SystemClock.elapsedRealtime() - playingSince) / 1000.0
            unsavedPlay += secs
            watched += secs
            playingSince = if (player.isPlaying) SystemClock.elapsedRealtime() else 0L
        }
    }

    private fun saveProgress(finished: Boolean = false) {
        val s = scene ?: return
        val id = s.id
        val pos = if (finished) 0.0 else currentSeconds()
        val played = unsavedPlay
        unsavedPlay = 0.0
        val countNow = !countedPlay && watched >= minOf(30.0, s.duration * 0.1).coerceAtLeast(5.0)
        if (countNow) countedPlay = true
        app.appScope.launch {
            runCatching { app.repo.saveActivity(id, pos, played) }
            if (countNow) runCatching { app.repo.addPlay(id) }
        }
    }

    private fun playNext() {
        if (queue.isEmpty()) {
            finish()
            return
        }
        val next = queue.removeAt(0)
        upNext = queue.size
        loadScene(next, 0.0)
    }

    override fun onStop() {
        super.onStop()
        if (!inPip) player.pause()
        addPlayTime()
        saveProgress()
    }

    override fun onDestroy() {
        player.removeListener(listener)
        player.release()
        super.onDestroy()
    }

    override fun onUserLeaveHint() {
        super.onUserLeaveHint()
        // keep playing in a small window when going home
        if (player.isPlaying) {
            val f = scene?.file
            val ratio = if (f != null && f.width > 0 && f.height > 0) {
                val r = f.width.toFloat() / f.height
                when {
                    r > 2.39f -> Rational(239, 100)
                    r < 1 / 2.39f -> Rational(100, 239)
                    else -> Rational(f.width, f.height)
                }
            } else Rational(16, 9)
            runCatching {
                enterPictureInPictureMode(PictureInPictureParams.Builder().setAspectRatio(ratio).build())
            }
        }
    }

    override fun onPictureInPictureModeChanged(isInPictureInPictureMode: Boolean, newConfig: Configuration) {
        super.onPictureInPictureModeChanged(isInPictureInPictureMode, newConfig)
        inPip = isInPictureInPictureMode
        playerView?.useController = !isInPictureInPictureMode
        if (!isInPictureInPictureMode) hideSystemBars()
    }

    @Composable
    private fun PlayerUi() {
        val s = scene
        Box(Modifier.fillMaxSize().background(Color.Black)) {
            // audio files show their cover behind the controls
            if (s != null && s.isAudio) {
                AsyncImage(
                    model = s.paths.screenshot,
                    contentDescription = null,
                    contentScale = ContentScale.Fit,
                    modifier = Modifier.fillMaxSize(),
                )
            }
            AndroidView(
                factory = { ctx ->
                    PlayerView(ctx).apply {
                        layoutParams = ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT)
                        this.player = this@PlayerActivity.player
                        setShowNextButton(false)
                        setShowPreviousButton(false)
                        setShowSubtitleButton(true)
                        controllerShowTimeoutMs = 3500
                        setShutterBackgroundColor(android.graphics.Color.TRANSPARENT)
                        setControllerVisibilityListener(PlayerView.ControllerVisibilityListener { v ->
                            controlsVisible = v == View.VISIBLE
                        })
                        playerView = this
                    }
                },
                update = { view ->
                    view.videoSurfaceView?.visibility = if (s?.isAudio == true) View.GONE else View.VISIBLE
                },
                modifier = Modifier.fillMaxSize(),
            )

            if (s == null && error == null) {
                CircularProgressIndicator(color = Brand.Pink, modifier = Modifier.align(Alignment.Center))
            }

            AnimatedVisibility(
                visible = controlsVisible && !inPip && s != null,
                enter = fadeIn(),
                exit = fadeOut(),
                modifier = Modifier.align(Alignment.TopStart),
            ) {
                Row(
                    Modifier
                        .fillMaxWidth()
                        .background(Brush.verticalGradient(listOf(Color.Black.copy(alpha = 0.7f), Color.Transparent)))
                        .safeDrawingPadding()
                        .padding(horizontal = 4.dp, vertical = 4.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    IconButton(onClick = { finish() }) {
                        Icon(Icons.AutoMirrored.Rounded.ArrowBack, "Back", tint = Color.White)
                    }
                    Column(Modifier.weight(1f)) {
                        Text(
                            s?.displayTitle.orEmpty(),
                            color = Color.White,
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                        val sub = listOfNotNull(s?.studio?.name, s?.performers?.takeIf { it.isNotEmpty() }?.joinToString(", ") { it.name })
                            .joinToString(" • ")
                        if (sub.isNotEmpty()) {
                            Text(sub, color = Color.White.copy(alpha = 0.7f), style = MaterialTheme.typography.bodySmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                        }
                    }
                    if (upNext > 0) {
                        Row(
                            Modifier
                                .clip(RoundedCornerShape(50))
                                .background(Color.White.copy(alpha = 0.15f))
                                .padding(horizontal = 4.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text("$upNext up next", color = Color.White, style = MaterialTheme.typography.labelLarge, modifier = Modifier.padding(start = 10.dp))
                            IconButton(onClick = {
                                addPlayTime()
                                saveProgress()
                                playNext()
                            }) { Icon(Icons.Rounded.SkipNext, "Next", tint = Color.White) }
                        }
                        Spacer(Modifier.width(8.dp))
                    }
                }
            }

            error?.let { msg ->
                Column(
                    Modifier
                        .align(Alignment.Center)
                        .padding(32.dp)
                        .clip(RoundedCornerShape(20.dp))
                        .background(Brand.Surface.copy(alpha = 0.95f))
                        .padding(24.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Text(msg, color = Color.White, textAlign = TextAlign.Center)
                    Spacer(Modifier.height(16.dp))
                    GradientButton("Close", onClick = { finish() })
                }
            }
        }
    }
}
