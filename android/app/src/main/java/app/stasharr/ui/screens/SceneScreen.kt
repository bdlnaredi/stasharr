package app.stasharr.ui.screens

import android.widget.Toast
import androidx.compose.animation.animateContentSize
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.automirrored.rounded.PlaylistAdd
import androidx.compose.material.icons.rounded.Bookmark
import androidx.compose.material.icons.rounded.BookmarkBorder
import androidx.compose.material.icons.rounded.PlayArrow
import androidx.compose.material.icons.rounded.Replay
import androidx.compose.material.icons.rounded.Whatshot
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.blur
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.stasharr.app
import app.stasharr.ui.SiteMode
import app.stasharr.ui.store.AverageStars
import androidx.compose.material.icons.rounded.Favorite
import androidx.compose.material.icons.rounded.FavoriteBorder
import androidx.compose.runtime.LaunchedEffect
import app.stasharr.data.OfflineItem
import androidx.compose.runtime.collectAsState
import androidx.compose.material.icons.rounded.Download
import androidx.compose.material.icons.rounded.DownloadDone
import app.stasharr.data.Playlist
import app.stasharr.data.Scene
import app.stasharr.player.ScenePreview
import app.stasharr.ui.LocalNav
import app.stasharr.ui.components.Chip
import app.stasharr.ui.components.GlassButton
import app.stasharr.ui.components.GradientButton
import app.stasharr.ui.components.Loaded
import app.stasharr.ui.components.PerformerBubble
import app.stasharr.ui.components.ProgressLine
import app.stasharr.ui.components.RatingStars
import app.stasharr.ui.components.SceneCard
import app.stasharr.ui.components.SectionHeader
import app.stasharr.ui.components.formatDuration
import app.stasharr.ui.components.metaLine
import app.stasharr.ui.components.rememberLoad
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage
import kotlinx.coroutines.launch

@Composable
fun SceneScreen(id: String) {
    val loader = rememberLoad(id) { app.repo.scene(id) ?: error("This scene no longer exists") }
    Loaded(loader) { scene -> SceneDetails(scene, onChanged = loader.reload) }
}

@Composable
private fun RoundAction(icon: ImageVector, label: String, active: Boolean = false, onClick: () -> Unit) {
    Column(
        Modifier
            .clip(RoundedCornerShape(12.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 10.dp, vertical = 6.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(icon, label, tint = if (active) Brand.Pink else Color.White, modifier = Modifier.size(26.dp))
        Spacer(Modifier.height(4.dp))
        Text(label, style = MaterialTheme.typography.labelSmall, color = Brand.TextMuted)
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun SceneDetails(scene: Scene, onChanged: () -> Unit) {
    val nav = LocalNav.current
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    var inWatchLater by remember(scene.id) { mutableStateOf(scene.inWatchLater) }
    var rating by remember(scene.id) { mutableStateOf(scene.rating100) }
    var oCount by remember(scene.id) { mutableIntStateOf(scene.oCounter ?: 0) }
    var showPlaylists by remember { mutableStateOf(false) }
    var showDownload by remember { mutableStateOf(false) }
    val store = SiteMode.storefront
    val extras = rememberLoad(scene.id, store) { if (store) app.repo.sceneExtras(scene.id) else null }
    var liked by remember(scene.id) { mutableStateOf(false) }
    LaunchedEffect(extras.value) { extras.value?.let { liked = it.liked } }
    val offline by app.downloads.items.collectAsState()
    val downloaded = offline.any { it.sceneId == scene.id && it.done }
    val downloading = offline.any { it.sceneId == scene.id && !it.done && it.state != OfflineItem.FAILED }
    var expanded by remember { mutableStateOf(false) }
    val similar = rememberLoad(scene.id) { app.repo.similar(scene.id, 20) }
    val resume = scene.resumeTime ?: 0.0
    val canResume = resume > 5 && resume < scene.duration - 10

    fun toast(msg: String) = Toast.makeText(context, msg, Toast.LENGTH_SHORT).show()

    Box(Modifier.fillMaxSize()) {
        Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState())) {
            // backdrop
            Box(Modifier.fillMaxWidth().height(420.dp)) {
                AsyncImage(
                    model = scene.paths.screenshot,
                    contentDescription = null,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.fillMaxSize().blur(18.dp),
                )
                Box(
                    Modifier
                        .fillMaxSize()
                        .background(
                            Brush.verticalGradient(
                                0f to Brand.Background.copy(alpha = 0.35f),
                                0.7f to Brand.Background.copy(alpha = 0.85f),
                                1f to Brand.Background,
                            )
                        )
                )
                // sharp picture on top of the blurred one
                Box(
                    Modifier
                        .align(Alignment.BottomCenter)
                        .padding(horizontal = 20.dp)
                        .padding(bottom = 8.dp)
                        .fillMaxWidth()
                        .aspectRatio(16f / 9f)
                        .clip(RoundedCornerShape(18.dp))
                        .background(Brand.SurfaceHigh)
                        .clickable { nav.play(scene.id, if (canResume) resume else 0.0) }
                ) {
                    AsyncImage(
                        model = scene.paths.screenshot,
                        contentDescription = null,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier.fillMaxSize(),
                    )
                    // the moving preview, like Netflix
                    ScenePreview(scene.paths.preview, active = true, modifier = Modifier.fillMaxSize(), delayMs = 600)
                    Box(
                        Modifier
                            .align(Alignment.Center)
                            .size(64.dp)
                            .clip(CircleShape)
                            .background(Brand.Gradient),
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(Icons.Rounded.PlayArrow, "Play", tint = Color.White, modifier = Modifier.size(40.dp))
                    }
                    if (scene.progress > 0.01f) ProgressLine(scene.progress, Modifier.align(Alignment.BottomStart))
                }
            }

            Column(Modifier.padding(horizontal = 20.dp)) {
                Spacer(Modifier.height(8.dp))
                Text(
                    scene.displayTitle,
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Black,
                    lineHeight = 32.sp,
                )
                Spacer(Modifier.height(6.dp))
                Text(scene.metaLine(), color = Brand.TextMuted, style = MaterialTheme.typography.bodyMedium)
                if (scene.studio != null) {
                    Spacer(Modifier.height(4.dp))
                    Text(
                        scene.studio.name,
                        color = Brand.Orange,
                        style = MaterialTheme.typography.titleSmall,
                        modifier = Modifier.clickable { nav.studio(scene.studio.id) },
                    )
                }

                Spacer(Modifier.height(18.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    GradientButton(
                        if (canResume) "Resume ${formatDuration(resume)}" else "Play",
                        onClick = { nav.play(scene.id, if (canResume) resume else 0.0) },
                        icon = Icons.Rounded.PlayArrow,
                        modifier = Modifier.weight(1f),
                    )
                    if (canResume) {
                        GlassButton("Start over", onClick = { nav.play(scene.id, 0.0) }, icon = Icons.Rounded.Replay)
                    }
                }

                Spacer(Modifier.height(14.dp))
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceEvenly,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    RoundAction(
                        if (inWatchLater) Icons.Rounded.Bookmark else Icons.Rounded.BookmarkBorder,
                        "Watch later",
                        active = inWatchLater,
                    ) {
                        scope.launch {
                            runCatching { app.repo.toggleWatchLater(scene.id) }
                                .onSuccess { inWatchLater = it; toast(if (it) "Saved to Watch later" else "Removed from Watch later") }
                                .onFailure { toast(it.message ?: "Couldn't update") }
                        }
                    }
                    if (store) {
                        RoundAction(
                            if (liked) Icons.Rounded.Favorite else Icons.Rounded.FavoriteBorder,
                            if (liked) "Liked" else "Like",
                            active = liked,
                        ) {
                            val was = liked
                            liked = !was
                            scope.launch {
                                runCatching { app.repo.toggleLike(scene.id) }
                                    .onSuccess { liked = it }
                                    .onFailure { liked = was; toast(it.message ?: "Couldn't update") }
                            }
                        }
                    }
                    RoundAction(Icons.AutoMirrored.Rounded.PlaylistAdd, "Playlist") { showPlaylists = true }
                    RoundAction(
                        when {
                            downloaded -> Icons.Rounded.DownloadDone
                            else -> Icons.Rounded.Download
                        },
                        when {
                            downloaded -> "Downloaded"
                            downloading -> "Downloading"
                            else -> "Download"
                        },
                        active = downloaded || downloading,
                    ) { showDownload = true }
                    RoundAction(Icons.Rounded.Whatshot, if (oCount > 0) "O · $oCount" else "O") {
                        scope.launch {
                            runCatching { app.repo.addO(scene.id) }.onSuccess { oCount = it }
                        }
                    }
                }

                val avg = extras.value?.siteRating
                if (store && avg?.average != null) {
                    Spacer(Modifier.height(10.dp))
                    AverageStars(avg.average, avg.count)
                }
                Spacer(Modifier.height(10.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("Your rating", color = Brand.TextMuted, style = MaterialTheme.typography.labelLarge, modifier = Modifier.width(100.dp))
                    RatingStars(rating, onRate = { r ->
                        val old = rating
                        rating = r
                        scope.launch {
                            runCatching { app.repo.setRating(scene.id, r) }.onFailure {
                                rating = old
                                toast(it.message ?: "Couldn't save the rating")
                            }
                        }
                    })
                }

                if (!scene.details.isNullOrBlank()) {
                    Spacer(Modifier.height(16.dp))
                    Text(
                        scene.details,
                        style = MaterialTheme.typography.bodyMedium,
                        color = Brand.Text.copy(alpha = 0.85f),
                        maxLines = if (expanded) Int.MAX_VALUE else 4,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier
                            .animateContentSize()
                            .clickable { expanded = !expanded },
                    )
                }
            }

            if (scene.performers.isNotEmpty()) {
                SectionHeader("Cast")
                LazyRow(contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    items(scene.performers, key = { it.id }) { p ->
                        PerformerBubble(p, onClick = { nav.performer(p.id) }, size = 76.dp)
                    }
                }
            }

            if (scene.markers.isNotEmpty()) {
                SectionHeader("Chapters")
                LazyRow(contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    items(scene.markers.sortedBy { it.seconds }, key = { it.id }) { m ->
                        Chip("${formatDuration(m.seconds)}  ${m.label}") { nav.play(scene.id, m.seconds) }
                    }
                }
            }

            if (scene.tags.isNotEmpty()) {
                SectionHeader("Tags")
                FlowRow(
                    Modifier.padding(horizontal = 16.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    scene.tags.forEach { t -> Chip(t.name) { nav.tag(t.id, t.name) } }
                }
            }

            val more = similar.value.orEmpty()
            if (more.isNotEmpty()) {
                SectionHeader("More like this")
                LazyRow(contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    items(more, key = { it.id }) { s -> SceneCard(s, onClick = { nav.scene(s.id) }, width = 220.dp) }
                }
            }

            scene.file?.let { f ->
                SectionHeader("File")
                Text(
                    f.path,
                    style = MaterialTheme.typography.bodySmall,
                    color = Brand.TextMuted,
                    modifier = Modifier.padding(horizontal = 20.dp),
                )
            }
            Spacer(Modifier.height(40.dp))
        }

        IconButton(
            onClick = { nav.back() },
            modifier = Modifier
                .statusBarsPadding()
                .padding(8.dp)
                .clip(CircleShape)
                .background(Color.Black.copy(alpha = 0.45f)),
        ) {
            Icon(Icons.AutoMirrored.Rounded.ArrowBack, "Back", tint = Color.White)
        }
    }

    if (showDownload) {
        DownloadSheet(scene, onDismiss = { showDownload = false })
    }

    if (showPlaylists) {
        AddToPlaylistDialog(sceneId = scene.id, onDismiss = { showPlaylists = false }, onDone = { msg ->
            showPlaylists = false
            toast(msg)
            onChanged()
        })
    }
}

@Composable
fun AddToPlaylistDialog(sceneId: String, onDismiss: () -> Unit, onDone: (String) -> Unit) {
    val scope = rememberCoroutineScope()
    val playlists = rememberLoad(Unit) { app.repo.playlists() }
    var newName by remember { mutableStateOf("") }
    var error by remember { mutableStateOf<String?>(null) }

    fun add(p: Playlist) {
        scope.launch {
            runCatching { app.repo.addToPlaylist(p.id, sceneId) }
                .onSuccess { onDone("Added to ${p.name}") }
                .onFailure { error = it.message }
        }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        containerColor = Brand.Surface,
        title = { Text("Add to playlist") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                playlists.value.orEmpty().filter { !it.watchLater && !it.smart }.forEach { p ->
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(10.dp))
                            .clickable { add(p) }
                            .background(Brand.SurfaceHigh)
                            .padding(12.dp),
                    ) {
                        Text(p.name, modifier = Modifier.weight(1f))
                        Text("${p.sceneCount}", color = Brand.TextMuted)
                    }
                }
                OutlinedTextField(
                    value = newName,
                    onValueChange = { newName = it },
                    singleLine = true,
                    label = { Text("New playlist") },
                    modifier = Modifier.fillMaxWidth(),
                )
                if (error != null) Text(error!!, color = MaterialTheme.colorScheme.error)
            }
        },
        confirmButton = {
            TextButton(
                enabled = newName.isNotBlank(),
                onClick = {
                    scope.launch {
                        runCatching { app.repo.createPlaylist(newName.trim(), sceneId) }
                            .onSuccess { onDone("Added to ${newName.trim()}") }
                            .onFailure { error = it.message }
                    }
                },
            ) { Text("Create") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
