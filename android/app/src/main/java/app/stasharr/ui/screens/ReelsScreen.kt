package app.stasharr.ui.screens

import android.view.ViewGroup
import androidx.annotation.OptIn
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.pager.VerticalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.VolumeOff
import androidx.compose.material.icons.automirrored.rounded.VolumeUp
import androidx.compose.material.icons.rounded.Info
import androidx.compose.material.icons.rounded.PlayArrow
import androidx.compose.material.icons.rounded.SmartDisplay
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.media3.common.MediaItem
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.ui.AspectRatioFrameLayout
import androidx.media3.ui.PlayerView
import app.stasharr.app
import app.stasharr.data.Reel
import app.stasharr.player.newPlayer
import app.stasharr.ui.LocalNav
import app.stasharr.ui.components.Badge
import app.stasharr.ui.components.GradientText
import app.stasharr.ui.components.LoadingBox
import app.stasharr.ui.components.rememberPaged
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage

@OptIn(UnstableApi::class)
@Composable
fun ReelsScreen() {
    val context = LocalContext.current
    val nav = LocalNav.current
    val reels = rememberPaged(Unit) { page -> app.repo.reels(page, 15) }
    val pager = rememberPagerState(pageCount = { reels.items.size })
    var muted by remember { mutableStateOf(false) }
    var paused by remember { mutableStateOf(false) }

    val player = remember {
        newPlayer(context).apply {
            repeatMode = Player.REPEAT_MODE_ONE
        }
    }
    DisposableEffect(Unit) { onDispose { player.release() } }

    // pause when the app goes to the background
    val lifecycle = LocalLifecycleOwner.current.lifecycle
    DisposableEffect(lifecycle) {
        val obs = LifecycleEventObserver { _, e ->
            if (e == Lifecycle.Event.ON_PAUSE) player.pause()
            if (e == Lifecycle.Event.ON_RESUME && !paused) player.play()
        }
        lifecycle.addObserver(obs)
        onDispose { lifecycle.removeObserver(obs) }
    }

    LaunchedEffect(muted) { player.volume = if (muted) 0f else 1f }

    // play the reel on screen; load more near the end
    LaunchedEffect(pager.settledPage, reels.items.size) {
        val i = pager.settledPage
        val reel = reels.items.getOrNull(i) ?: return@LaunchedEffect
        player.setMediaItem(MediaItem.fromUri(reel.stream))
        player.prepare()
        paused = false
        player.play()
        if (i >= reels.items.size - 4) reels.loadMore()
    }

    Box(Modifier.fillMaxSize().background(Color.Black)) {
        when {
            reels.items.isEmpty() && reels.loading -> LoadingBox()
            reels.items.isEmpty() -> EmptyReels(reels.error)
            else -> VerticalPager(state = pager, modifier = Modifier.fillMaxSize(), key = { reels.items[it].id }) { page ->
                val reel = reels.items[page]
                ReelPage(
                    reel = reel,
                    active = page == pager.settledPage,
                    player = player,
                    muted = muted,
                    paused = paused,
                    onToggleMute = { muted = !muted },
                    onTogglePause = {
                        paused = !paused
                        if (paused) player.pause() else player.play()
                    },
                    onOpen = { player.pause(); nav.scene(reel.scene.id) },
                    onPlayScene = { player.pause(); nav.play(reel.scene.id, reel.start) },
                )
            }
        }
        Row(
            Modifier.statusBarsPadding().padding(horizontal = 18.dp, vertical = 10.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            GradientText("Reels")
        }
    }
}

@Composable
private fun EmptyReels(error: String?) {
    Column(
        Modifier.fillMaxSize().padding(32.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(Icons.Rounded.SmartDisplay, null, tint = Brand.TextMuted, modifier = Modifier.size(56.dp))
        Spacer(Modifier.height(12.dp))
        Text(
            error ?: "No reels yet. Make them on your server under Settings → Auto-import → Make reels.",
            color = Brand.TextMuted,
            style = MaterialTheme.typography.bodyLarge,
        )
    }
}

@OptIn(UnstableApi::class)
@Composable
private fun ReelPage(
    reel: Reel,
    active: Boolean,
    player: ExoPlayer,
    muted: Boolean,
    paused: Boolean,
    onToggleMute: () -> Unit,
    onTogglePause: () -> Unit,
    onOpen: () -> Unit,
    onPlayScene: () -> Unit,
) {
    Box(
        Modifier
            .fillMaxSize()
            .clickable(interactionSource = remember { MutableInteractionSource() }, indication = null, onClick = onTogglePause)
    ) {
        AsyncImage(
            model = reel.scene.paths.screenshot,
            contentDescription = null,
            contentScale = ContentScale.Crop,
            modifier = Modifier.fillMaxSize(),
        )
        if (active) {
            AndroidView(
                factory = { ctx ->
                    PlayerView(ctx).apply {
                        useController = false
                        resizeMode = AspectRatioFrameLayout.RESIZE_MODE_ZOOM
                        setShutterBackgroundColor(android.graphics.Color.TRANSPARENT)
                        layoutParams = ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT)
                        this.player = player
                    }
                },
                update = { it.player = player },
                onRelease = { it.player = null },
                modifier = Modifier.fillMaxSize(),
            )
        }
        Box(
            Modifier.fillMaxSize().background(
                Brush.verticalGradient(0f to Color.Black.copy(alpha = 0.35f), 0.25f to Color.Transparent, 0.6f to Color.Transparent, 1f to Color.Black.copy(alpha = 0.85f))
            )
        )
        if (active && paused) {
            Icon(
                Icons.Rounded.PlayArrow, "Play",
                tint = Color.White.copy(alpha = 0.9f),
                modifier = Modifier.align(Alignment.Center).size(84.dp),
            )
        }

        // details, bottom left
        Column(
            Modifier
                .align(Alignment.BottomStart)
                .fillMaxWidth(0.78f)
                .padding(start = 18.dp, bottom = 24.dp)
        ) {
            reel.scene.studio?.let { Badge(it.name.uppercase(), gradient = true) }
            Spacer(Modifier.height(8.dp))
            Text(
                reel.scene.title ?: "Untitled",
                color = Color.White,
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Black,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            if (reel.scene.performers.isNotEmpty()) {
                Text(
                    reel.scene.performers.joinToString(", ") { it.name },
                    color = Color.White.copy(alpha = 0.8f),
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }

        // actions, bottom right
        Column(
            Modifier.align(Alignment.BottomEnd).padding(end = 10.dp, bottom = 24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            ReelAction(if (muted) Icons.AutoMirrored.Rounded.VolumeOff else Icons.AutoMirrored.Rounded.VolumeUp, if (muted) "Muted" else "Sound", onToggleMute)
            ReelAction(Icons.Rounded.PlayArrow, "Full scene", onPlayScene)
            ReelAction(Icons.Rounded.Info, "Details", onOpen)
        }
    }
}

@Composable
private fun ReelAction(icon: androidx.compose.ui.graphics.vector.ImageVector, label: String, onClick: () -> Unit) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        IconButton(
            onClick = onClick,
            modifier = Modifier.size(50.dp).clip(CircleShape).background(Color.Black.copy(alpha = 0.4f)),
        ) { Icon(icon, label, tint = Color.White, modifier = Modifier.size(28.dp)) }
        Text(label, color = Color.White, style = MaterialTheme.typography.labelSmall)
    }
}
