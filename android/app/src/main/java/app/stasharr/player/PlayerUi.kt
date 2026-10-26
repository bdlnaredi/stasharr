package app.stasharr.player

import android.view.ContextThemeWrapper
import android.view.View
import androidx.mediarouter.app.MediaRouteButton
import com.google.android.gms.cast.framework.CastButtonFactory
import androidx.annotation.OptIn
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.gestures.calculatePan
import androidx.compose.foundation.gestures.calculateZoom
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.automirrored.rounded.VolumeUp
import androidx.compose.material.icons.rounded.AspectRatio
import androidx.compose.material.icons.rounded.BrightnessMedium
import androidx.compose.material.icons.rounded.Cast
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.FastForward
import androidx.compose.material.icons.rounded.FastRewind
import androidx.compose.material.icons.rounded.Forward10
import androidx.compose.material.icons.rounded.Lock
import androidx.compose.material.icons.rounded.LockOpen
import androidx.compose.material.icons.rounded.Pause
import androidx.compose.material.icons.rounded.PictureInPictureAlt
import androidx.compose.material.icons.rounded.PlayArrow
import androidx.compose.material.icons.rounded.Replay10
import androidx.compose.material.icons.rounded.ScreenRotation
import androidx.compose.material.icons.rounded.SkipNext
import androidx.compose.material.icons.rounded.Speed
import androidx.compose.material.icons.rounded.Subtitles
import androidx.compose.material.icons.rounded.Toc
import androidx.compose.material.icons.rounded.Tune
import androidx.compose.material.icons.rounded.ZoomIn
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Slider
import androidx.compose.material3.SliderDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableDoubleStateOf
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.input.pointer.positionChange
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.media3.common.C
import androidx.media3.common.Player
import androidx.media3.common.Tracks
import androidx.media3.common.util.UnstableApi
import app.stasharr.ui.components.GradientButton
import app.stasharr.ui.components.formatDuration
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.util.Locale
import kotlin.math.abs

private enum class Sheet { Speed, Playback, Tracks, Chapters, Screen }
private enum class Drag { None, Seek, Brightness, Volume, Pan, Zoom }

private val speeds = listOf(0.25f, 0.5f, 0.75f, 1f, 1.25f, 1.5f, 1.75f, 2f, 3f)

private fun speedLabel(s: Float) = if (s == s.toInt().toFloat()) "${s.toInt()}x" else "${s}x"

@OptIn(UnstableApi::class)
@Composable
fun PlayerScreen(a: PlayerActivity) {
    val scope = rememberCoroutineScope()
    val scene = a.scene
    val player = a.player

    // playback position, read a few times a second
    var position by remember { mutableDoubleStateOf(0.0) }
    var duration by remember { mutableDoubleStateOf(0.0) }
    var playing by remember { mutableStateOf(false) }
    var buffering by remember { mutableStateOf(false) }
    LaunchedEffect(player) {
        while (true) {
            val p = a.activePlayer()
            if (p != null) {
                position = a.currentSeconds()
                duration = a.durationSeconds()
                playing = p.isPlaying
                buffering = p.playbackState == Player.STATE_BUFFERING
            }
            delay(250)
        }
    }

    var controls by remember { mutableStateOf(true) }
    var locked by remember { mutableStateOf(false) }
    var touches by remember { mutableIntStateOf(0) }
    var sheet by remember { mutableStateOf<Sheet?>(null) }

    // zoom and move
    var zoom by remember { mutableFloatStateOf(1f) }
    var pan by remember { mutableStateOf(Offset.Zero) }

    // gesture feedback
    var seekBubble by remember { mutableStateOf<Pair<Boolean, Int>?>(null) } // forward?, seconds
    var bubbleJob by remember { mutableStateOf<Job?>(null) }
    var hold by remember { mutableStateOf<String?>(null) } // "2x" or "rewind"
    var holdSpeed by remember { mutableFloatStateOf(1f) }
    var scrubTo by remember { mutableStateOf<Double?>(null) }
    var level by remember { mutableStateOf<Pair<ImageVector, Float>?>(null) }
    var sliderValue by remember { mutableStateOf<Float?>(null) }

    fun poke() { touches++ }

    // hide the controls after a while
    LaunchedEffect(controls, playing, touches, sheet) {
        if (controls && playing && sheet == null) {
            delay(3500)
            controls = false
        }
    }

    BoxWithConstraints(Modifier.fillMaxSize().background(Color.Black)) {
        val wPx = constraints.maxWidth.toFloat()
        val hPx = constraints.maxHeight.toFloat()

        // audio files show their cover behind everything
        if (scene != null && scene.isAudio) {
            AsyncImage(
                model = scene.paths.screenshot,
                contentDescription = null,
                contentScale = ContentScale.Fit,
                modifier = Modifier.fillMaxSize(),
            )
        }

        // the video, zoomed and moved as asked
        Box(
            Modifier
                .fillMaxSize()
                .clipToBounds()
                .graphicsLayer {
                    scaleX = zoom
                    scaleY = zoom
                    translationX = pan.x
                    translationY = pan.y
                }
        ) {
            AndroidView(
                factory = { ctx -> a.makePlayerView(ctx) },
                update = { v ->
                    v.player = a.player
                    v.resizeMode = a.aspect.resizeMode
                    v.videoSurfaceView?.visibility = if (scene?.isAudio == true) View.GONE else View.VISIBLE
                },
                modifier = Modifier.fillMaxSize(),
            )
        }

        // casting: the picture shows on the TV
        a.castingTo?.let { tv ->
            Box(Modifier.fillMaxSize().background(Color.Black)) {
                AsyncImage(
                    model = scene?.paths?.screenshot,
                    contentDescription = null,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.fillMaxSize().graphicsLayer { alpha = 0.35f },
                )
                Column(
                    Modifier.align(Alignment.Center),
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Icon(Icons.Rounded.Cast, null, tint = Brand.Orange, modifier = Modifier.size(56.dp))
                    Spacer(Modifier.height(8.dp))
                    Text("Playing on $tv", color = Color.White, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
                }
            }
        }

        // ---------------------------------------------------------- gestures
        if (!a.inPip) {
            Box(
                Modifier
                    .fillMaxSize()
                    .pointerInput(locked) {
                        if (locked) return@pointerInput
                        awaitEachGesture {
                            val down = awaitFirstDown(requireUnconsumed = false)
                            var mode = Drag.None
                            var total = Offset.Zero
                            val startX = down.position.x
                            val slop = viewConfiguration.touchSlop
                            val startBrightness = a.brightness()
                            val startVolume = a.volume()
                            var vertical = 0f
                            var seekStart = 0.0
                            var seekDelta = 0.0
                            do {
                                val event = awaitPointerEvent()
                                val pressed = event.changes.filter { it.pressed }
                                if (pressed.size >= 2) {
                                    // pinch to zoom, and move while zoomed
                                    mode = Drag.Zoom
                                    val z = (zoom * event.calculateZoom()).coerceIn(1f, 5f)
                                    val p = pan + event.calculatePan()
                                    val maxX = (z - 1) * wPx / 2
                                    val maxY = (z - 1) * hPx / 2
                                    zoom = z
                                    pan = Offset(p.x.coerceIn(-maxX, maxX), p.y.coerceIn(-maxY, maxY))
                                    if (z <= 1.01f) pan = Offset.Zero
                                    event.changes.forEach { it.consume() }
                                } else if (pressed.size == 1 && mode != Drag.Zoom) {
                                    val c = pressed[0]
                                    val d = c.positionChange()
                                    total += d
                                    if (mode == Drag.None && total.getDistance() > slop) {
                                        mode = when {
                                            zoom > 1.01f -> Drag.Pan
                                            abs(total.x) > abs(total.y) -> Drag.Seek.also {
                                                seekStart = a.currentSeconds()
                                            }
                                            startX < wPx / 2 -> Drag.Brightness
                                            else -> Drag.Volume
                                        }
                                    }
                                    when (mode) {
                                        Drag.Seek -> {
                                            // a full swipe across the screen moves two minutes
                                            seekDelta += d.x / wPx * 120.0
                                            scrubTo = (seekStart + seekDelta).coerceIn(0.0, maxOf(0.0, duration))
                                        }
                                        Drag.Brightness -> {
                                            vertical -= d.y / hPx
                                            val v = (startBrightness + vertical).coerceIn(0.01f, 1f)
                                            a.setBrightness(v)
                                            level = Icons.Rounded.BrightnessMedium to v
                                        }
                                        Drag.Volume -> {
                                            vertical -= d.y / hPx
                                            val v = (startVolume + vertical).coerceIn(0f, 1f)
                                            a.setVolume(v)
                                            level = Icons.AutoMirrored.Rounded.VolumeUp to v
                                        }
                                        Drag.Pan -> {
                                            val p = pan + d
                                            val maxX = (zoom - 1) * wPx / 2
                                            val maxY = (zoom - 1) * hPx / 2
                                            pan = Offset(p.x.coerceIn(-maxX, maxX), p.y.coerceIn(-maxY, maxY))
                                        }
                                        else -> Unit
                                    }
                                    if (mode != Drag.None) c.consume()
                                }
                            } while (event.changes.any { it.pressed })

                            if (mode == Drag.Seek) {
                                scrubTo?.let { a.seekTo(it) }
                                scrubTo = null
                            }
                            if (mode == Drag.Brightness || mode == Drag.Volume) {
                                scope.launch {
                                    delay(600)
                                    level = null
                                }
                            }
                        }
                    }
                    .pointerInput(locked) {
                        detectTapGestures(
                            onTap = {
                                controls = !controls
                                poke()
                            },
                            onDoubleTap = { o ->
                                if (locked) return@detectTapGestures
                                when {
                                    o.x < wPx / 3 -> {
                                        a.seekBy(-10.0)
                                        val prev = seekBubble?.takeIf { !it.first }?.second ?: 0
                                        seekBubble = false to prev + 10
                                    }
                                    o.x > wPx * 2 / 3 -> {
                                        a.seekBy(10.0)
                                        val prev = seekBubble?.takeIf { it.first }?.second ?: 0
                                        seekBubble = true to prev + 10
                                    }
                                    else -> a.togglePlay()
                                }
                                bubbleJob?.cancel()
                                bubbleJob = scope.launch {
                                    delay(800)
                                    seekBubble = null
                                }
                            },
                            onLongPress = { o ->
                                if (locked) return@detectTapGestures
                                // hold: 2x forwards on the right, rewind on the left
                                if (o.x >= wPx / 2) {
                                    holdSpeed = a.speed
                                    a.setPlaybackSpeed(2f)
                                    a.activePlayer()?.play()
                                    hold = "2x"
                                } else {
                                    a.startRewind()
                                    hold = "rewind"
                                }
                            },
                            onPress = {
                                tryAwaitRelease()
                                when (hold) {
                                    "2x" -> a.setPlaybackSpeed(holdSpeed)
                                    "rewind" -> a.stopRewind()
                                }
                                hold = null
                            },
                        )
                    }
            )
        }

        // ---------------------------------------------------------- feedback
        seekBubble?.let { (forward, secs) ->
            Box(
                Modifier
                    .align(if (forward) Alignment.CenterEnd else Alignment.CenterStart)
                    .fillMaxHeight()
                    .width(maxWidth / 3)
                    .clip(if (forward) RoundedCornerShape(topStartPercent = 50, bottomStartPercent = 50) else RoundedCornerShape(topEndPercent = 50, bottomEndPercent = 50))
                    .background(Color.White.copy(alpha = 0.12f)),
                contentAlignment = Alignment.Center,
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Icon(if (forward) Icons.Rounded.FastForward else Icons.Rounded.FastRewind, null, tint = Color.White, modifier = Modifier.size(36.dp))
                    Text("${if (forward) "+" else "−"}$secs s", color = Color.White, fontWeight = FontWeight.Bold)
                }
            }
        }

        hold?.let { h ->
            Row(
                Modifier
                    .align(Alignment.TopCenter)
                    .safeDrawingPadding()
                    .padding(top = 24.dp)
                    .clip(RoundedCornerShape(50))
                    .background(Brand.GradientHorizontal)
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(if (h == "2x") Icons.Rounded.FastForward else Icons.Rounded.FastRewind, null, tint = Color.White)
                Spacer(Modifier.width(6.dp))
                Text(if (h == "2x") "2x speed" else "Rewinding", color = Color.White, fontWeight = FontWeight.Bold)
            }
        }

        scrubTo?.let { t ->
            val diff = t - position
            Column(
                Modifier
                    .align(Alignment.Center)
                    .clip(RoundedCornerShape(16.dp))
                    .background(Color.Black.copy(alpha = 0.6f))
                    .padding(horizontal = 22.dp, vertical = 12.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(formatDuration(t), color = Color.White, fontSize = 28.sp, fontWeight = FontWeight.Black)
                Text(
                    "${if (diff >= 0) "+" else "−"}${formatDuration(abs(diff))}",
                    color = Brand.Orange,
                    fontWeight = FontWeight.Bold,
                )
            }
        }

        level?.let { (icon, v) ->
            Row(
                Modifier
                    .align(Alignment.TopCenter)
                    .safeDrawingPadding()
                    .padding(top = 24.dp)
                    .clip(RoundedCornerShape(50))
                    .background(Color.Black.copy(alpha = 0.65f))
                    .padding(horizontal = 16.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(icon, null, tint = Color.White)
                Spacer(Modifier.width(10.dp))
                Box(Modifier.width(140.dp).height(6.dp).clip(RoundedCornerShape(3.dp)).background(Color.White.copy(alpha = 0.25f))) {
                    Box(Modifier.fillMaxWidth(v).fillMaxHeight().background(Brand.GradientHorizontal))
                }
                Spacer(Modifier.width(10.dp))
                Text("${(v * 100).toInt()}%", color = Color.White, fontWeight = FontWeight.Bold)
            }
        }

        if ((buffering || scene == null) && a.error == null) {
            CircularProgressIndicator(color = Brand.Pink, modifier = Modifier.align(Alignment.Center))
        }

        // ---------------------------------------------------------- controls
        AnimatedVisibility(
            visible = controls && !locked && !a.inPip && scene != null,
            enter = fadeIn(),
            exit = fadeOut(),
            modifier = Modifier.fillMaxSize(),
        ) {
            Box(Modifier.fillMaxSize()) {
                Box(
                    Modifier.fillMaxSize().background(
                        Brush.verticalGradient(
                            0f to Color.Black.copy(alpha = 0.7f),
                            0.25f to Color.Transparent,
                            0.65f to Color.Transparent,
                            1f to Color.Black.copy(alpha = 0.8f),
                        )
                    )
                )

                // top bar
                Row(
                    Modifier.align(Alignment.TopStart).fillMaxWidth().safeDrawingPadding().padding(4.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    IconButton(onClick = { a.finish() }) { Icon(Icons.AutoMirrored.Rounded.ArrowBack, "Back", tint = Color.White) }
                    Column(Modifier.weight(1f)) {
                        Text(
                            scene?.displayTitle.orEmpty(),
                            color = Color.White,
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                        val sub = listOfNotNull(scene?.studio?.name, a.sources.getOrNull(a.sourceIndex)?.label, a.decoder.label)
                            .joinToString(" • ")
                        Text(sub, color = Color.White.copy(alpha = 0.7f), style = MaterialTheme.typography.bodySmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                    if (a.castAvailable) {
                        AndroidView(
                            factory = { ctx ->
                                MediaRouteButton(ContextThemeWrapper(ctx, androidx.appcompat.R.style.Theme_AppCompat)).also {
                                    CastButtonFactory.setUpMediaRouteButton(ctx.applicationContext, it)
                                }
                            },
                            modifier = Modifier.size(48.dp),
                        )
                    }
                    if (a.castingTo == null) {
                        IconButton(onClick = { a.enterPip() }) { Icon(Icons.Rounded.PictureInPictureAlt, "Small window", tint = Color.White) }
                    }
                    IconButton(onClick = { locked = true; poke() }) { Icon(Icons.Rounded.Lock, "Lock", tint = Color.White) }
                }

                // middle: back 10, play, forward 10
                Row(
                    Modifier.align(Alignment.Center),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(36.dp),
                ) {
                    IconButton(onClick = { a.seekBy(-10.0); poke() }, modifier = Modifier.size(56.dp)) {
                        Icon(Icons.Rounded.Replay10, "Back 10 seconds", tint = Color.White, modifier = Modifier.size(40.dp))
                    }
                    Box(
                        Modifier
                            .size(76.dp)
                            .clip(CircleShape)
                            .background(Brand.Gradient)
                            .clickable { a.togglePlay(); poke() },
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(if (playing) Icons.Rounded.Pause else Icons.Rounded.PlayArrow, if (playing) "Pause" else "Play", tint = Color.White, modifier = Modifier.size(46.dp))
                    }
                    IconButton(onClick = { a.seekBy(10.0); poke() }, modifier = Modifier.size(56.dp)) {
                        Icon(Icons.Rounded.Forward10, "Forward 10 seconds", tint = Color.White, modifier = Modifier.size(40.dp))
                    }
                }

                // bottom: seek bar and buttons
                Column(
                    Modifier.align(Alignment.BottomStart).fillMaxWidth().safeDrawingPadding().padding(horizontal = 16.dp, vertical = 6.dp)
                ) {
                    val shown = sliderValue?.toDouble() ?: position
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(formatDuration(shown), color = Color.White, style = MaterialTheme.typography.labelLarge)
                        Spacer(Modifier.weight(1f))
                        Text(formatDuration(duration), color = Color.White.copy(alpha = 0.7f), style = MaterialTheme.typography.labelLarge)
                    }
                    Box(contentAlignment = Alignment.Center) {
                        // chapter marks on the seek bar
                        val marks = scene?.markers.orEmpty()
                        if (marks.isNotEmpty() && duration > 0) {
                            Canvas(Modifier.fillMaxWidth().height(10.dp).padding(horizontal = 10.dp)) {
                                marks.forEach { m ->
                                    val x = (m.seconds / duration).toFloat().coerceIn(0f, 1f) * size.width
                                    drawCircle(Brand.Orange, radius = 3.dp.toPx(), center = Offset(x, size.height / 2))
                                }
                            }
                        }
                        Slider(
                            value = (shown / maxOf(duration, 1.0)).toFloat().coerceIn(0f, 1f),
                            onValueChange = { sliderValue = (it * duration).toFloat(); poke() },
                            onValueChangeFinished = {
                                sliderValue?.let { a.seekTo(it.toDouble()) }
                                sliderValue = null
                            },
                            colors = SliderDefaults.colors(
                                thumbColor = Color.White,
                                activeTrackColor = Brand.Pink,
                                inactiveTrackColor = Color.White.copy(alpha = 0.25f),
                            ),
                        )
                    }
                    Row(
                        Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Pill(Icons.Rounded.Speed, speedLabel(a.speed)) { sheet = Sheet.Speed }
                        Pill(Icons.Rounded.AspectRatio, if (zoom > 1.01f) "Zoom ${"%.1f".format(zoom)}x" else a.aspect.label) { sheet = Sheet.Screen }
                        Pill(Icons.Rounded.Tune, a.sources.getOrNull(a.sourceIndex)?.label ?: "Quality") { sheet = Sheet.Playback }
                        Pill(Icons.Rounded.Subtitles, null) { sheet = Sheet.Tracks }
                        if (scene?.markers?.isNotEmpty() == true) Pill(Icons.Rounded.Toc, null) { sheet = Sheet.Chapters }
                        Spacer(Modifier.weight(1f))
                        IconButton(onClick = { a.rotate(); poke() }) { Icon(Icons.Rounded.ScreenRotation, "Rotate", tint = Color.White) }
                        if (a.upNext > 0) {
                            IconButton(onClick = { a.playNext() }) { Icon(Icons.Rounded.SkipNext, "Next", tint = Color.White) }
                        }
                    }
                }
            }
        }

        // locked: only the unlock button
        if (locked && controls && !a.inPip) {
            Column(
                Modifier.align(Alignment.TopStart).safeDrawingPadding().padding(12.dp),
            ) {
                Row(
                    Modifier
                        .clip(RoundedCornerShape(50))
                        .background(Color.Black.copy(alpha = 0.6f))
                        .clickable { locked = false; poke() }
                        .padding(horizontal = 14.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(Icons.Rounded.LockOpen, null, tint = Color.White)
                    Spacer(Modifier.width(8.dp))
                    Text("Tap to unlock", color = Color.White, fontWeight = FontWeight.Bold)
                }
            }
        }

        a.error?.let { msg ->
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
                Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    if (a.decoder != Decoder.Server) {
                        GradientButton("Try server transcode", onClick = { a.chooseDecoder(Decoder.Server) })
                    }
                    GradientButton("Close", onClick = { a.finish() })
                }
            }
        }
    }

    // ---------------------------------------------------------- sheets
    when (sheet) {
        Sheet.Speed -> OptionSheet("Playback speed", onDismiss = { sheet = null }) {
            speeds.forEach { s ->
                OptionRow(speedLabel(s), selected = s == a.speed) { a.setPlaybackSpeed(s); sheet = null }
            }
        }
        Sheet.Screen -> OptionSheet("Screen", onDismiss = { sheet = null }) {
            Aspect.entries.forEach { asp ->
                OptionRow(asp.label, selected = asp == a.aspect && zoom <= 1.01f) {
                    a.chooseAspect(asp)
                    zoom = 1f
                    pan = Offset.Zero
                    sheet = null
                }
            }
            OptionRow("Zoom in 1.5x", icon = Icons.Rounded.ZoomIn, selected = zoom in 1.4f..1.6f) {
                zoom = 1.5f
                pan = Offset.Zero
                sheet = null
            }
            Hint("Pinch the video to zoom, and drag to move around while zoomed.")
        }
        Sheet.Playback -> OptionSheet("Playback", onDismiss = { sheet = null }) {
            SheetLabel("Decoding")
            Decoder.entries.forEach { d ->
                OptionRow(d.label, subtitle = d.description, selected = d == a.decoder) {
                    a.chooseDecoder(d)
                    sheet = null
                }
            }
            if (a.sources.size > 1) {
                SheetLabel("Quality")
                a.sources.forEachIndexed { i, src ->
                    OptionRow(
                        src.label,
                        subtitle = if (src.isDirect) "The file as it is, played by your phone" else "Converted by your server",
                        selected = i == a.sourceIndex,
                    ) {
                        a.chooseSource(i)
                        sheet = null
                    }
                }
            }
        }
        Sheet.Tracks -> OptionSheet("Audio and subtitles", onDismiss = { sheet = null }) {
            val audio = a.tracks.groups.filter { it.type == C.TRACK_TYPE_AUDIO }
            val text = a.tracks.groups.filter { it.type == C.TRACK_TYPE_TEXT }
            SheetLabel("Audio")
            if (audio.isEmpty()) Hint("No audio tracks")
            TrackRows(audio) { g, i -> a.chooseTrack(C.TRACK_TYPE_AUDIO, g, i); sheet = null }
            SheetLabel("Subtitles")
            OptionRow("Off", selected = text.none { g -> (0 until g.length).any { g.isTrackSelected(it) } }) {
                a.chooseTrack(C.TRACK_TYPE_TEXT, null)
                sheet = null
            }
            if (text.isEmpty()) Hint("This video has no subtitles")
            TrackRows(text) { g, i -> a.chooseTrack(C.TRACK_TYPE_TEXT, g, i); sheet = null }
        }
        Sheet.Chapters -> OptionSheet("Chapters", onDismiss = { sheet = null }) {
            scene?.markers.orEmpty().sortedBy { it.seconds }.forEach { m ->
                OptionRow(m.label, subtitle = formatDuration(m.seconds), selected = false) {
                    a.seekTo(m.seconds)
                    sheet = null
                }
            }
        }
        null -> Unit
    }
}

@Composable
private fun Pill(icon: ImageVector, text: String?, onClick: () -> Unit) {
    Row(
        Modifier
            .clip(RoundedCornerShape(50))
            .background(Color.White.copy(alpha = 0.14f))
            .clickable(onClick = onClick)
            .padding(horizontal = if (text == null) 10.dp else 12.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, null, tint = Color.White, modifier = Modifier.size(18.dp))
        if (text != null) {
            Spacer(Modifier.width(6.dp))
            Text(text, color = Color.White, style = MaterialTheme.typography.labelLarge, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(end = 2.dp))
        }
    }
}

@kotlin.OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun OptionSheet(title: String, onDismiss: () -> Unit, content: @Composable () -> Unit) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        containerColor = Brand.Surface,
    ) {
        Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding().padding(bottom = 16.dp)) {
            Text(title, style = MaterialTheme.typography.titleLarge, modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp))
            content()
        }
    }
}

@Composable
private fun SheetLabel(text: String) {
    Text(text, color = Brand.TextMuted, style = MaterialTheme.typography.titleSmall, modifier = Modifier.padding(start = 20.dp, top = 14.dp, bottom = 4.dp))
}

@Composable
private fun Hint(text: String) {
    Text(text, color = Brand.TextMuted, style = MaterialTheme.typography.bodySmall, modifier = Modifier.padding(horizontal = 20.dp, vertical = 6.dp))
}

@Composable
private fun OptionRow(
    title: String,
    subtitle: String? = null,
    selected: Boolean,
    icon: ImageVector? = null,
    onClick: () -> Unit,
) {
    Row(
        Modifier.fillMaxWidth().clickable(onClick = onClick).padding(horizontal = 20.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (icon != null) {
            Icon(icon, null, tint = Brand.TextMuted)
            Spacer(Modifier.width(12.dp))
        }
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.bodyLarge, color = if (selected) Brand.Pink else Brand.Text)
            if (subtitle != null) Text(subtitle, style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted)
        }
        if (selected) Icon(Icons.Rounded.Check, null, tint = Brand.Pink)
    }
}

@OptIn(UnstableApi::class)
@Composable
private fun TrackRows(groups: List<Tracks.Group>, onPick: (Tracks.Group, Int) -> Unit) {
    var n = 0
    groups.forEach { g ->
        for (i in 0 until g.length) {
            if (!g.isTrackSupported(i)) continue
            n++
            val f = g.getTrackFormat(i)
            val lang = f.language?.let { runCatching { Locale.forLanguageTag(it).displayLanguage }.getOrNull() }?.takeIf { it.isNotBlank() }
            val parts = listOfNotNull(
                f.label,
                lang,
                f.channelCount.takeIf { it > 0 }?.let { if (it >= 6) "5.1" else if (it == 2) "Stereo" else if (it == 1) "Mono" else "$it ch" },
                f.sampleMimeType?.substringAfter('/')?.uppercase(),
            ).distinct()
            OptionRow(parts.firstOrNull() ?: "Track $n", subtitle = parts.drop(1).joinToString(" • ").ifBlank { null }, selected = g.isTrackSelected(i)) {
                onPick(g, i)
            }
        }
    }
}
