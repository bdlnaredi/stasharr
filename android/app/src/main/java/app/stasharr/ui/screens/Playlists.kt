package app.stasharr.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.automirrored.rounded.Logout
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Bookmark
import androidx.compose.material.icons.rounded.Dns
import androidx.compose.material.icons.rounded.PlayArrow
import androidx.compose.material.icons.rounded.Shuffle
import androidx.compose.material.icons.rounded.VerifiedUser
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.stasharr.BuildConfig
import app.stasharr.app
import androidx.compose.runtime.collectAsState
import androidx.compose.material.icons.rounded.DownloadForOffline
import app.stasharr.data.Me
import app.stasharr.data.Playlist
import app.stasharr.ui.LocalNav
import app.stasharr.ui.SiteMode
import app.stasharr.player.AutoPreview
import app.stasharr.ui.components.GlassButton
import app.stasharr.ui.components.GradientButton
import app.stasharr.ui.components.GradientText
import app.stasharr.ui.components.LocalAppActions
import app.stasharr.ui.components.Loaded
import app.stasharr.ui.components.SceneThumb
import app.stasharr.ui.components.formatDuration
import app.stasharr.ui.components.rememberLoad
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage
import kotlinx.coroutines.launch

@Composable
private fun PlaylistCard(p: Playlist, onClick: () -> Unit) {
    Column(Modifier.fillMaxWidth().clickable(onClick = onClick)) {
        Box(
            Modifier
                .fillMaxWidth()
                .aspectRatio(16f / 10f)
                .clip(RoundedCornerShape(14.dp))
                .background(Brush.linearGradient(listOf(Brand.SurfaceHigh, Brand.Surface)))
        ) {
            val cover = p.cover?.paths?.screenshot
            if (cover != null) {
                AsyncImage(cover, null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
            }
            Box(
                Modifier
                    .fillMaxSize()
                    .background(Brush.verticalGradient(0.4f to Color.Transparent, 1f to Color.Black.copy(alpha = 0.8f)))
            )
            if (p.smart) {
                Text(
                    "SMART",
                    color = Color.Black,
                    fontWeight = FontWeight.Black,
                    fontSize = 10.sp,
                    modifier = Modifier.align(Alignment.TopStart).padding(8.dp).clip(RoundedCornerShape(50)).background(Brand.Gradient).padding(horizontal = 8.dp, vertical = 3.dp),
                )
            }
            if (p.watchLater) {
                Icon(
                    Icons.Rounded.Bookmark, null, tint = Color.White,
                    modifier = Modifier.align(Alignment.TopStart).padding(8.dp).clip(CircleShape).background(Brand.Gradient).padding(6.dp).size(16.dp),
                )
            }
            Text(
                "${p.sceneCount}",
                color = Color.White,
                fontWeight = FontWeight.Black,
                fontSize = 22.sp,
                modifier = Modifier.align(Alignment.BottomEnd).padding(10.dp),
            )
        }
        Spacer(Modifier.height(6.dp))
        Text(p.name, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
        if (p.description.isNotBlank()) {
            Text(p.description, style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}

@Composable
fun PlaylistsScreen() {
    val nav = LocalNav.current
    val scope = rememberCoroutineScope()
    val loader = rememberLoad(Unit) { app.repo.playlists() }
    var creating by remember { mutableStateOf(false) }
    var name by remember { mutableStateOf("") }

    Column(Modifier.fillMaxSize().statusBarsPadding()) {
        Row(Modifier.fillMaxWidth().padding(start = 18.dp, end = 8.dp, top = 14.dp, bottom = 6.dp), verticalAlignment = Alignment.CenterVertically) {
            GradientText("Playlists", modifier = Modifier.weight(1f))
            IconButton(onClick = { creating = true }) { Icon(Icons.Rounded.Add, "New playlist") }
        }
        Loaded(loader) { list ->
            LazyVerticalGrid(
                columns = GridCells.Adaptive(160.dp),
                contentPadding = PaddingValues(16.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                items(list, key = { it.id }) { p -> PlaylistCard(p) { nav.playlist(p.id) } }
                if (list.size <= 1) {
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        Text(
                            "Make playlists from a scene's page, with the playlist button.",
                            color = Brand.TextMuted,
                            modifier = Modifier.padding(top = 12.dp),
                        )
                    }
                }
            }
        }
    }

    if (creating) {
        AlertDialog(
            onDismissRequest = { creating = false },
            containerColor = Brand.Surface,
            title = { Text("New playlist") },
            text = {
                OutlinedTextField(name, { name = it }, singleLine = true, label = { Text("Name") })
            },
            confirmButton = {
                TextButton(enabled = name.isNotBlank(), onClick = {
                    scope.launch {
                        runCatching { app.repo.createPlaylist(name.trim(), null) }
                        creating = false
                        name = ""
                        loader.reload()
                    }
                }) { Text("Create") }
            },
            dismissButton = { TextButton(onClick = { creating = false }) { Text("Cancel") } },
        )
    }
}

@Composable
fun PlaylistScreen(id: String) {
    val nav = LocalNav.current
    val loader = rememberLoad(id) { app.repo.playlist(id) ?: error("Playlist not found") }
    Loaded(loader) { (p, scenes) ->
        LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(bottom = 24.dp)) {
            item {
                Box(Modifier.fillMaxWidth().height(260.dp)) {
                    val cover = scenes.firstOrNull()?.paths?.screenshot
                    if (cover != null) {
                        AsyncImage(cover, null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
                    }
                    Box(Modifier.fillMaxSize().background(Brush.verticalGradient(listOf(Brand.Background.copy(alpha = 0.3f), Brand.Background))))
                    IconButton(onClick = { nav.back() }, modifier = Modifier.statusBarsPadding().padding(4.dp)) {
                        Icon(Icons.AutoMirrored.Rounded.ArrowBack, "Back", tint = Color.White)
                    }
                    Column(Modifier.align(Alignment.BottomStart).padding(20.dp)) {
                        Text(p.name, style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Black)
                        Text("${scenes.size} scenes • ${formatDuration(scenes.sumOf { it.duration })}", color = Brand.TextMuted)
                        Spacer(Modifier.height(12.dp))
                        Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                            GradientButton("Play all", icon = Icons.Rounded.PlayArrow, onClick = {
                                if (scenes.isNotEmpty()) nav.play(scenes.first().id, 0.0, scenes.drop(1).map { it.id })
                            })
                            GlassButton("Shuffle", icon = Icons.Rounded.Shuffle, onClick = {
                                val s = scenes.shuffled()
                                if (s.isNotEmpty()) nav.play(s.first().id, 0.0, s.drop(1).map { it.id })
                            })
                        }
                    }
                }
            }
            itemsIndexed(scenes, key = { _, s -> s.id }) { i, s ->
                Row(
                    Modifier
                        .fillMaxWidth()
                        .clickable { nav.play(s.id, 0.0, scenes.drop(i + 1).map { it.id }) }
                        .padding(horizontal = 16.dp, vertical = 8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text("${i + 1}", color = Brand.TextMuted, modifier = Modifier.width(28.dp))
                    SceneThumb(s, Modifier.width(140.dp))
                    Spacer(Modifier.width(12.dp))
                    Column(Modifier.weight(1f).clickable { nav.scene(s.id) }) {
                        Text(s.displayTitle, style = MaterialTheme.typography.titleSmall, maxLines = 2, overflow = TextOverflow.Ellipsis)
                        Text(
                            listOfNotNull(s.studio?.name, s.year).joinToString(" • "),
                            style = MaterialTheme.typography.bodySmall,
                            color = Brand.TextMuted,
                        )
                    }
                }
            }
            if (scenes.isEmpty()) {
                item {
                    Text(
                        if (p.watchLater) "Tap the bookmark on any scene to save it here." else "This playlist is empty.",
                        color = Brand.TextMuted,
                        modifier = Modifier.padding(24.dp),
                    )
                }
            }
        }
    }
}

@Composable
private fun InfoRow(icon: ImageVector, title: String, value: String) {
    Row(
        Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 6.dp)
            .clip(RoundedCornerShape(14.dp))
            .background(Brand.Surface)
            .padding(16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, null, tint = Brand.Pink)
        Spacer(Modifier.width(14.dp))
        Column {
            Text(title, style = MaterialTheme.typography.labelLarge, color = Brand.TextMuted)
            Text(value, style = MaterialTheme.typography.bodyLarge)
        }
    }
}

@Composable
fun ProfileScreen(me: Me?) {
    val actions = LocalAppActions.current
    val name = me?.username ?: app.session.username ?: "You"
    Column(Modifier.fillMaxSize().statusBarsPadding()) {
        Spacer(Modifier.height(30.dp))
        Box(
            Modifier
                .align(Alignment.CenterHorizontally)
                .size(110.dp)
                .clip(CircleShape)
                .background(Brand.Gradient),
            contentAlignment = Alignment.Center,
        ) {
            Text(name.take(1).uppercase(), fontSize = 46.sp, fontWeight = FontWeight.Black, color = Color.White)
        }
        Spacer(Modifier.height(14.dp))
        Text(name, style = MaterialTheme.typography.headlineSmall, modifier = Modifier.align(Alignment.CenterHorizontally))
        Text(
            when (me?.role) {
                "ADMIN" -> "Admin"
                "EDITOR" -> "Editor"
                "VIEWER" -> "Viewer"
                else -> ""
            },
            color = Brand.TextMuted,
            modifier = Modifier.align(Alignment.CenterHorizontally),
        )
        Spacer(Modifier.height(24.dp))
        val nav = LocalNav.current
        val offline by app.downloads.items.collectAsState()
        GradientButton(
            if (offline.isEmpty()) "Downloads" else "Downloads · ${offline.count { it.done }}",
            onClick = { nav.downloads() },
            icon = Icons.Rounded.DownloadForOffline,
            modifier = Modifier.align(Alignment.CenterHorizontally),
        )
        Spacer(Modifier.height(16.dp))
        val look = SiteMode.look
        if (look?.enabled == true && !look.active) {
            val ctx = androidx.compose.ui.platform.LocalContext.current
            Row(
                Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 6.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(Modifier.weight(1f)) {
                    Text("Site layout", style = MaterialTheme.typography.titleSmall)
                    Text(
                        "Use the website's storefront look in the app, as site users see it",
                        style = MaterialTheme.typography.bodySmall,
                        color = Brand.TextMuted,
                    )
                }
                androidx.compose.material3.Switch(
                    checked = SiteMode.preview,
                    onCheckedChange = { SiteMode.setPreview(ctx, it) },
                )
            }
        }
        run {
            val ctx = androidx.compose.ui.platform.LocalContext.current
            Row(
                Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 6.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(Modifier.weight(1f)) {
                    Text("Play previews automatically", style = MaterialTheme.typography.titleSmall)
                    Text(
                        "Banners and the scene at the top of the screen play a short muted preview. Turn off to save data; press and hold a scene to preview it.",
                        style = MaterialTheme.typography.bodySmall,
                        color = Brand.TextMuted,
                    )
                }
                androidx.compose.material3.Switch(
                    checked = AutoPreview.enabled,
                    onCheckedChange = { AutoPreview.setEnabled(ctx, it) },
                )
            }
        }
        InfoRow(Icons.Rounded.Dns, "Server", app.session.serverUrl.orEmpty())
        InfoRow(Icons.Rounded.VerifiedUser, "App version", BuildConfig.VERSION_NAME)
        Spacer(Modifier.height(24.dp))
        GlassButton(
            "Sign out",
            icon = Icons.AutoMirrored.Rounded.Logout,
            onClick = actions.signOut,
            modifier = Modifier.align(Alignment.CenterHorizontally),
        )
    }
}
