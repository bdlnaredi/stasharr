package app.stasharr.player

import android.app.PictureInPictureParams
import android.content.Context
import android.content.Intent
import android.content.pm.ActivityInfo
import android.content.res.Configuration
import android.media.AudioManager
import android.os.Bundle
import android.os.SystemClock
import android.provider.Settings
import android.util.Rational
import android.view.LayoutInflater
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.annotation.OptIn
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import androidx.lifecycle.lifecycleScope
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.MediaMetadata
import androidx.media3.common.MimeTypes
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.TrackSelectionOverride
import androidx.media3.common.Tracks
import androidx.media3.common.util.UnstableApi
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.SeekParameters
import androidx.media3.ui.PlayerView
import app.stasharr.R
import app.stasharr.app
import app.stasharr.data.Scene
import app.stasharr.ui.theme.StasharrTheme
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

/** One way of streaming a scene: the original file, or a conversion made by the server. */
data class Source(val label: String, val url: String) {
    val isDirect: Boolean get() = url.substringBefore('?').endsWith("/stream")
    val isHls: Boolean get() = url.substringBefore('?').endsWith(".m3u8")
    /** Whether the player can jump around in it; other conversions restart from a time. */
    val seekable: Boolean get() = isDirect || isHls
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

    private lateinit var prefs: PlayerPrefs
    internal var player by mutableStateOf<ExoPlayer?>(null)
    internal var playerView: PlayerView? = null

    // what the screen shows
    internal var scene by mutableStateOf<Scene?>(null)
    internal var error by mutableStateOf<String?>(null)
    internal var inPip by mutableStateOf(false)
    internal var upNext by mutableIntStateOf(0)
    internal var tracks by mutableStateOf(Tracks.EMPTY)
    internal var sources by mutableStateOf<List<Source>>(emptyList())
    internal var sourceIndex by mutableIntStateOf(0)
    internal var decoder by mutableStateOf(Decoder.Hardware)
    internal var aspect by mutableStateOf(Aspect.Fit)
    internal var speed by mutableFloatStateOf(1f)
    internal var subtitlesOn by mutableStateOf(false)

    private var sceneId = ""
    private var queue = ArrayList<String>()
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

        override fun onTracksChanged(t: Tracks) {
            tracks = t
        }

        override fun onPlayerError(e: PlaybackException) {
            // the phone could not play it: let the server convert it, from where we were
            val at = currentSeconds()
            val next = sources.indexOfFirst { !it.isDirect }.takeIf { it > sourceIndex }
                ?: (sourceIndex + 1).takeIf { it < sources.size }
            if (next != null) {
                sourceIndex = next
                playSource(at)
            } else {
                error = "This video can't be played.\n${e.errorCodeName}"
            }
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        WindowCompat.setDecorFitsSystemWindows(window, false)
        hideSystemBars()

        prefs = PlayerPrefs(this)
        decoder = prefs.decoder
        aspect = prefs.aspect
        createPlayer()

        readIntent(intent)

        setContent {
            StasharrTheme {
                PlayerScreen(this)
            }
        }

        // save the position now and then while playing
        lifecycleScope.launch {
            while (isActive) {
                delay(15_000)
                if (player?.isPlaying == true) {
                    addPlayTime()
                    playingSince = SystemClock.elapsedRealtime()
                    saveProgress()
                }
            }
        }
    }

    private fun createPlayer() {
        val p = newPlayer(this, if (decoder == Decoder.Software) Decoder.Software else Decoder.Hardware)
        p.addListener(listener)
        p.setPlaybackSpeed(speed)
        playerView?.player = p
        player = p
    }

    /** Builds the video view; used once by the screen. */
    internal fun makePlayerView(context: Context): PlayerView =
        (LayoutInflater.from(context).inflate(R.layout.player_texture_view, null) as PlayerView).also {
            it.player = player
            it.resizeMode = aspect.resizeMode
            playerView = it
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

    internal fun hideSystemBars() {
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
        player?.stop()
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
                sourceIndex = if (decoder == Decoder.Server) firstServerSource() else 0
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

    private fun firstServerSource(): Int = sources.indexOfFirst { !it.isDirect }.coerceAtLeast(0)

    /** The original file first (the phone plays far more than a browser), then the server's conversions. */
    private fun buildSources(s: Scene): List<Source> {
        val list = mutableListOf<Source>()
        s.paths.stream?.let { list += Source("Original", it) }
        val streams = s.streams.filter { !it.url.substringBefore('?').endsWith(".mpd") }
        // server conversions: HLS first as it can jump around, then MP4
        streams.sortedBy {
            val u = it.url.substringBefore('?')
            when {
                u.endsWith("/stream") -> 0
                u.endsWith(".m3u8") -> 1
                u.endsWith(".mp4") || u.endsWith(".mp3") -> 2
                else -> 3
            }
        }.forEach { st ->
            val label = (st.label ?: "Server")
                .replace("Direct stream", "Original")
                .replace("HLS", "Server")
                .replace("MP4", "Server MP4")
                .replace("WEBM", "Server WebM")
            list += Source(label, st.url)
        }
        return list.distinctBy { it.url }
    }

    internal fun playSource(startSeconds: Double) {
        val s = scene ?: return
        val p = player ?: return
        val src = sources.getOrNull(sourceIndex) ?: return
        var url = src.url
        offsetSeconds = 0.0
        if (!src.seekable && startSeconds > 1) {
            url += (if (url.contains('?')) "&" else "?") + "start=" + startSeconds.toInt()
            offsetSeconds = startSeconds.toInt().toDouble()
        }

        // subtitles from the server, if the scene has caption files
        val subs = s.captions.orEmpty().mapNotNull { c ->
            val base = s.paths.caption ?: return@mapNotNull null
            MediaItem.SubtitleConfiguration.Builder(android.net.Uri.parse("$base?lang=${c.lang}&type=${c.type}"))
                .setMimeType(MimeTypes.TEXT_VTT)
                .setLanguage(c.lang.ifBlank { null })
                .setLabel(c.lang.uppercase().ifBlank { "Subtitles" })
                .build()
        }

        val item = MediaItem.Builder()
            .setUri(url)
            .apply { if (src.isHls) setMimeType(MimeTypes.APPLICATION_M3U8) }
            .setSubtitleConfigurations(subs)
            .setMediaMetadata(
                MediaMetadata.Builder()
                    .setTitle(s.displayTitle)
                    .setArtist(s.studio?.name)
                    .build()
            )
            .build()
        val startMs = if (src.seekable) (startSeconds * 1000).toLong() else 0L
        p.setMediaItem(item, startMs)
        p.prepare()
        p.play()
        error = null
    }

    // ---------------------------------------------------------------- controls

    internal fun currentSeconds(): Double = offsetSeconds + (player?.currentPosition ?: 0L) / 1000.0

    internal fun durationSeconds(): Double {
        val d = scene?.duration ?: 0.0
        if (d > 0) return d
        val ms = player?.duration ?: C.TIME_UNSET
        return if (ms == C.TIME_UNSET) 0.0 else ms / 1000.0
    }

    /** Jumps to a time in the scene, restarting a conversion there if needed. */
    internal fun seekTo(seconds: Double) {
        val p = player ?: return
        val src = sources.getOrNull(sourceIndex) ?: return
        val target = seconds.coerceIn(0.0, maxOf(0.0, durationSeconds() - 1))
        if (src.seekable) {
            p.seekTo((target * 1000).toLong())
        } else {
            playSource(target)
        }
    }

    internal fun seekBy(seconds: Double) = seekTo(currentSeconds() + seconds)

    internal fun togglePlay() {
        val p = player ?: return
        if (p.isPlaying) p.pause() else {
            if (p.playbackState == Player.STATE_ENDED) seekTo(0.0)
            p.play()
        }
    }

    internal fun setPlaybackSpeed(s: Float) {
        speed = s
        player?.setPlaybackSpeed(s)
    }

    internal fun chooseAspect(a: Aspect) {
        aspect = a
        prefs.aspect = a
        playerView?.resizeMode = a.resizeMode
    }

    internal fun chooseSource(index: Int) {
        if (index == sourceIndex) return
        val at = currentSeconds()
        sourceIndex = index
        if (!sources[index].isDirect && decoder != Decoder.Server) {
            decoder = Decoder.Server
        } else if (sources[index].isDirect && decoder == Decoder.Server) {
            decoder = prefs.decoder.takeIf { it != Decoder.Server } ?: Decoder.Hardware
        }
        playSource(at)
    }

    /** Switches between the phone's hardware and software decoders, or the server. */
    internal fun chooseDecoder(d: Decoder) {
        if (d == decoder) return
        val at = currentSeconds()
        val wasPlaying = player?.playWhenReady ?: true
        val rebuild = (d == Decoder.Software) != (decoder == Decoder.Software)
        decoder = d
        prefs.decoder = d
        if (rebuild) {
            player?.removeListener(listener)
            player?.release()
            createPlayer()
        }
        sourceIndex = if (d == Decoder.Server) firstServerSource() else 0
        playSource(at)
        if (!wasPlaying) player?.pause()
    }

    /** Picks an audio or subtitle track, or turns subtitles off with null. */
    internal fun chooseTrack(type: Int, group: Tracks.Group?, index: Int = 0) {
        val p = player ?: return
        val b = p.trackSelectionParameters.buildUpon()
        if (group == null) {
            b.setTrackTypeDisabled(type, true)
            if (type == C.TRACK_TYPE_TEXT) subtitlesOn = false
        } else {
            b.setTrackTypeDisabled(type, false)
            b.setOverrideForType(TrackSelectionOverride(group.mediaTrackGroup, index))
            if (type == C.TRACK_TYPE_TEXT) subtitlesOn = true
        }
        p.trackSelectionParameters = b.build()
    }

    // rewinding: the player cannot play backwards, so it steps back in jumps
    private var rewindJob: Job? = null

    internal fun startRewind() {
        val p = player ?: return
        val resume = p.playWhenReady
        p.pause()
        p.setSeekParameters(SeekParameters.CLOSEST_SYNC)
        rewindJob?.cancel()
        rewindJob = lifecycleScope.launch {
            try {
                while (isActive) {
                    seekBy(-1.0)
                    delay(500)
                }
            } finally {
                p.setSeekParameters(SeekParameters.DEFAULT)
                if (resume) p.play()
            }
        }
    }

    internal fun stopRewind() {
        rewindJob?.cancel()
        rewindJob = null
    }

    internal fun rotate() {
        requestedOrientation = if (resources.configuration.orientation == Configuration.ORIENTATION_LANDSCAPE) {
            ActivityInfo.SCREEN_ORIENTATION_SENSOR_PORTRAIT
        } else {
            ActivityInfo.SCREEN_ORIENTATION_SENSOR_LANDSCAPE
        }
    }

    // brightness and volume, for the swipe gestures

    internal fun brightness(): Float {
        val b = window.attributes.screenBrightness
        if (b >= 0) return b
        return runCatching {
            Settings.System.getInt(contentResolver, Settings.System.SCREEN_BRIGHTNESS) / 255f
        }.getOrDefault(0.5f)
    }

    internal fun setBrightness(value: Float) {
        val attrs = window.attributes
        attrs.screenBrightness = value.coerceIn(0.01f, 1f)
        window.attributes = attrs
    }

    private val audio by lazy { getSystemService(Context.AUDIO_SERVICE) as AudioManager }

    internal fun volume(): Float {
        val max = audio.getStreamMaxVolume(AudioManager.STREAM_MUSIC).coerceAtLeast(1)
        return audio.getStreamVolume(AudioManager.STREAM_MUSIC).toFloat() / max
    }

    internal fun setVolume(value: Float) {
        val max = audio.getStreamMaxVolume(AudioManager.STREAM_MUSIC)
        audio.setStreamVolume(AudioManager.STREAM_MUSIC, (value.coerceIn(0f, 1f) * max).toInt(), 0)
    }

    // ---------------------------------------------------------------- progress

    private fun addPlayTime() {
        if (playingSince > 0) {
            val secs = (SystemClock.elapsedRealtime() - playingSince) / 1000.0
            unsavedPlay += secs
            watched += secs
            playingSince = if (player?.isPlaying == true) SystemClock.elapsedRealtime() else 0L
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

    internal fun playNext() {
        addPlayTime()
        saveProgress()
        if (queue.isEmpty()) {
            finish()
            return
        }
        val next = queue.removeAt(0)
        upNext = queue.size
        loadScene(next, 0.0)
    }

    // ---------------------------------------------------------------- lifecycle

    override fun onStop() {
        super.onStop()
        if (!inPip) player?.pause()
        addPlayTime()
        saveProgress()
    }

    override fun onDestroy() {
        stopRewind()
        player?.removeListener(listener)
        player?.release()
        player = null
        super.onDestroy()
    }

    internal fun enterPip() {
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

    override fun onUserLeaveHint() {
        super.onUserLeaveHint()
        // keep playing in a small window when going home
        if (player?.isPlaying == true) enterPip()
    }

    override fun onPictureInPictureModeChanged(isInPictureInPictureMode: Boolean, newConfig: Configuration) {
        super.onPictureInPictureModeChanged(isInPictureInPictureMode, newConfig)
        inPip = isInPictureInPictureMode
        if (!isInPictureInPictureMode) hideSystemBars()
    }
}
