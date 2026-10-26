package app.stasharr.ui.screens

import android.text.format.Formatter
import android.widget.Toast
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
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.rounded.CheckCircle
import androidx.compose.material.icons.rounded.DeleteOutline
import androidx.compose.material.icons.rounded.Download
import androidx.compose.material.icons.rounded.DownloadDone
import androidx.compose.material.icons.rounded.ErrorOutline
import androidx.compose.material.icons.rounded.PlayArrow
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import app.stasharr.app
import app.stasharr.data.OfflineItem
import app.stasharr.data.OfflineQuality
import app.stasharr.data.Scene
import app.stasharr.ui.LocalNav
import app.stasharr.ui.components.GradientText
import app.stasharr.ui.components.rememberLoad
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage
import java.io.File

private fun bytes(context: android.content.Context, n: Long) = Formatter.formatShortFileSize(context, n)

@Composable
fun DownloadsScreen() {
    val nav = LocalNav.current
    val context = LocalContext.current
    val items by app.downloads.items.collectAsState()
    val progress by app.downloads.progress.collectAsState()
    var confirm by remember { mutableStateOf<OfflineItem?>(null) }

    val sorted = items.sortedByDescending { it.addedAt }
    val used = items.filter { it.done }.sumOf { it.size }

    Column(Modifier.fillMaxSize()) {
        Row(
            Modifier.fillMaxWidth().statusBarsPadding().padding(horizontal = 4.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = { nav.back() }) { Icon(Icons.AutoMirrored.Rounded.ArrowBack, "Back") }
            Column(Modifier.weight(1f)) {
                Text("Downloads", style = MaterialTheme.typography.titleLarge)
                Text(
                    "${bytes(context, used)} used · ${bytes(context, app.downloads.freeSpace())} free",
                    style = MaterialTheme.typography.bodySmall,
                    color = Brand.TextMuted,
                )
            }
        }

        if (sorted.isEmpty()) {
            Column(
                Modifier.fillMaxSize().padding(32.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Icon(Icons.Rounded.Download, null, tint = Brand.TextMuted, modifier = Modifier.size(56.dp))
                Spacer(Modifier.height(12.dp))
                GradientText("Nothing downloaded yet", fontSize = MaterialTheme.typography.titleLarge.fontSize)
                Spacer(Modifier.height(8.dp))
                Text(
                    "Tap Download on a scene to watch it without the server, on a plane or anywhere with no Wi-Fi.",
                    color = Brand.TextMuted,
                    style = MaterialTheme.typography.bodyMedium,
                )
            }
            return@Column
        }

        LazyColumn(
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            items(sorted, key = { it.key }) { item ->
                DownloadRow(
                    item = item,
                    progress = progress[item.key]?.fraction,
                    progressText = progress[item.key]?.let { p ->
                        if (p.total > 0) "${bytes(context, p.bytes)} of ${bytes(context, p.total)}" else null
                    },
                    onPlay = { nav.play(item.sceneId, item.scene.resumeTime ?: 0.0) },
                    onRetry = { app.downloads.retry(item) },
                    onDelete = { confirm = item },
                )
            }
            item { Spacer(Modifier.navigationBarsPadding().height(24.dp)) }
        }
    }

    confirm?.let { item ->
        AlertDialog(
            onDismissRequest = { confirm = null },
            containerColor = Brand.Surface,
            title = { Text(if (item.done) "Delete download?" else "Cancel download?") },
            text = { Text("${item.scene.displayTitle} (${item.qualityLabel}) is removed from this phone. It stays on the server.") },
            confirmButton = {
                TextButton(onClick = {
                    app.downloads.delete(item)
                    confirm = null
                    Toast.makeText(context, "Removed", Toast.LENGTH_SHORT).show()
                }) { Text("Delete") }
            },
            dismissButton = { TextButton(onClick = { confirm = null }) { Text("Keep") } },
        )
    }
}

@Composable
private fun DownloadRow(
    item: OfflineItem,
    progress: Float?,
    progressText: String?,
    onPlay: () -> Unit,
    onRetry: () -> Unit,
    onDelete: () -> Unit,
) {
    val context = LocalContext.current
    Row(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .background(Brand.Surface)
            .clickable(enabled = item.done, onClick = onPlay)
            .padding(10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .width(132.dp)
                .aspectRatio(16f / 9f)
                .clip(RoundedCornerShape(10.dp))
                .background(Brand.SurfaceHigh),
            contentAlignment = Alignment.Center,
        ) {
            AsyncImage(
                model = item.thumb?.let { File(it) } ?: item.scene.paths.screenshot,
                contentDescription = null,
                contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxSize(),
            )
            if (item.done) {
                Box(
                    Modifier.size(34.dp).clip(CircleShape).background(Color.Black.copy(alpha = 0.55f)),
                    contentAlignment = Alignment.Center,
                ) { Icon(Icons.Rounded.PlayArrow, "Play", tint = Color.White) }
            }
        }
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Text(item.scene.displayTitle, style = MaterialTheme.typography.titleSmall, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Spacer(Modifier.height(2.dp))
            val (icon, text, tint) = when (item.state) {
                OfflineItem.DONE -> Triple(Icons.Rounded.DownloadDone, "${item.qualityLabel} · ${bytes(context, item.size)}", Brand.TextMuted)
                OfflineItem.FAILED -> Triple(Icons.Rounded.ErrorOutline, item.error ?: "Failed", MaterialTheme.colorScheme.error)
                OfflineItem.PREPARING -> Triple(
                    Icons.Rounded.Download,
                    "Getting ${item.qualityLabel} ready on the server" + (progress?.let { " · ${(it * 100).toInt()}%" } ?: ""),
                    Brand.Orange,
                )
                else -> Triple(
                    Icons.Rounded.Download,
                    "Downloading ${item.qualityLabel}" + (progressText?.let { " · $it" } ?: ""),
                    Brand.Orange,
                )
            }
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(icon, null, tint = tint, modifier = Modifier.size(14.dp))
                Spacer(Modifier.width(4.dp))
                Text(text, style = MaterialTheme.typography.bodySmall, color = tint, maxLines = 2, overflow = TextOverflow.Ellipsis)
            }
            if (!item.done && item.state != OfflineItem.FAILED) {
                Spacer(Modifier.height(6.dp))
                if (progress != null && progress > 0f) {
                    LinearProgressIndicator(progress = { progress }, color = Brand.Pink, trackColor = Brand.SurfaceHigh, modifier = Modifier.fillMaxWidth())
                } else {
                    LinearProgressIndicator(color = Brand.Pink, trackColor = Brand.SurfaceHigh, modifier = Modifier.fillMaxWidth())
                }
            }
        }
        if (item.state == OfflineItem.FAILED) {
            IconButton(onClick = onRetry) { Icon(Icons.Rounded.Refresh, "Try again", tint = Color.White) }
        }
        IconButton(onClick = onDelete) { Icon(Icons.Rounded.DeleteOutline, "Delete", tint = Brand.TextMuted) }
    }
}

/** Picks a size and starts downloading a scene. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DownloadSheet(scene: Scene, onDismiss: () -> Unit) {
    val context = LocalContext.current
    val nav = LocalNav.current
    val qualities = rememberLoad(Unit) { app.repo.offlineQualities() }
    val items by app.downloads.items.collectAsState()
    val mine = items.filter { it.sceneId == scene.id }
    val f = scene.file
    val sheet = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheet, containerColor = Brand.Surface) {
        Column(Modifier.padding(horizontal = 20.dp).padding(bottom = 28.dp)) {
            Text("Download", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
            Text(
                "Watch without the server. Smaller sizes are converted on the server first, which takes a few minutes.",
                style = MaterialTheme.typography.bodySmall,
                color = Brand.TextMuted,
            )
            Spacer(Modifier.height(12.dp))
            val list = qualities.value.orEmpty().filter { q ->
                // no point converting to a size bigger than the file
                q.height == 0 || f == null || f.height == 0 || q.height <= minOf(f.width, f.height) || q.height == 480
            }.let { if (scene.isAudio) it.filter { q -> q.height == 0 } else it }
            if (qualities.value == null) {
                LinearProgressIndicator(color = Brand.Pink, trackColor = Brand.SurfaceHigh, modifier = Modifier.fillMaxWidth())
            }
            list.forEach { q -> QualityRow(scene, q, mine.firstOrNull { it.quality == q.id }) { onDismiss() } }
            Spacer(Modifier.height(8.dp))
            Text(
                "${bytes(context, app.downloads.freeSpace())} free on this phone",
                style = MaterialTheme.typography.bodySmall,
                color = Brand.TextMuted,
            )
            if (items.isNotEmpty()) {
                TextButton(onClick = { onDismiss(); nav.downloads() }) { Text("See all downloads") }
            }
        }
    }
}

@Composable
private fun QualityRow(scene: Scene, q: OfflineQuality, existing: OfflineItem?, onStarted: () -> Unit) {
    val context = LocalContext.current
    val f = scene.file
    val estimate = when {
        q.height == 0 && f != null && f.size > 0 -> bytes(context, f.size)
        q.height > 0 && scene.duration > 0 -> {
            // rough: average bitrate of the conversion
            val mbps = when (q.height) { 1080 -> 4.5; 720 -> 2.2; else -> 1.1 }
            "about " + bytes(context, (scene.duration * mbps * 1_000_000 / 8).toLong())
        }
        else -> null
    }
    val busy = existing != null && existing.state != OfflineItem.FAILED
    Row(
        Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp)
            .clip(RoundedCornerShape(12.dp))
            .background(Brand.SurfaceHigh)
            .clickable(enabled = !busy) {
                app.downloads.start(scene, q)
                Toast.makeText(context, "Downloading ${q.label}", Toast.LENGTH_SHORT).show()
                onStarted()
            }
            .padding(14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(q.label, style = MaterialTheme.typography.titleSmall)
            if (estimate != null) Text(estimate, style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted)
        }
        when {
            existing?.done == true -> Icon(Icons.Rounded.CheckCircle, "Downloaded", tint = Brand.Pink)
            busy -> Text("In progress", style = MaterialTheme.typography.labelMedium, color = Brand.Orange)
            else -> Icon(Icons.Rounded.Download, "Download", tint = Color.White)
        }
    }
}
