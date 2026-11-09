package app.stasharr.ui.screens

import android.text.format.Formatter
import androidx.compose.foundation.Canvas
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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.PlayArrow
import androidx.compose.material.icons.rounded.Refresh
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import app.stasharr.app
import app.stasharr.data.AiSuggestion
import app.stasharr.data.Scene
import app.stasharr.data.ServerStats
import app.stasharr.ui.LocalNav
import app.stasharr.ui.components.Chip
import app.stasharr.ui.components.ChipRow
import app.stasharr.ui.components.GradientButton
import app.stasharr.ui.components.Loaded
import app.stasharr.ui.components.PagedGrid
import app.stasharr.ui.components.SceneCard
import app.stasharr.ui.components.formatDuration
import app.stasharr.ui.components.fullWidth
import app.stasharr.ui.components.formatRuntime
import app.stasharr.ui.components.rememberLoad
import app.stasharr.ui.components.rememberPaged
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.time.OffsetDateTime
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

/** Admin tools: AI marker review, auto-import, server tasks and statistics. */
@Composable
fun AdminScreen() {
    val nav = LocalNav.current
    var tab by rememberSaveable { mutableStateOf("review") }
    Column(Modifier.fillMaxSize()) {
        Row(
            Modifier.fillMaxWidth().statusBarsPadding().padding(horizontal = 4.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = { nav.back() }) { Icon(Icons.AutoMirrored.Rounded.ArrowBack, "Back") }
            Text("Admin", style = MaterialTheme.typography.titleLarge)
        }
        ChipRow {
            listOf("review" to "AI markers", "import" to "Auto-import", "tasks" to "Tasks", "stats" to "Statistics").forEach { (id, label) ->
                Chip(label, selected = tab == id) { tab = id }
            }
        }
        Box(Modifier.weight(1f)) {
            when (tab) {
                "review" -> ReviewTab()
                "import" -> ImportTab()
                "tasks" -> TasksTab()
                else -> StatsTab()
            }
        }
    }
}

private fun whenText(iso: String?): String = iso?.let {
    runCatching {
        OffsetDateTime.parse(it).format(DateTimeFormatter.ofLocalizedDateTime(FormatStyle.MEDIUM, FormatStyle.SHORT))
    }.getOrNull()
} ?: "never"

/** A picture of the scene at that moment, made by the server. */
private fun Scene.at(seconds: Double): String = app.api.baseUrl.trimEnd('/') + "/scene/$id/frame/${(seconds + 1).toInt()}"

// ---------------------------------------------------------------- AI markers

@Composable
private fun ReviewTab() {
    val nav = LocalNav.current
    val scope = rememberCoroutineScope()
    var nonce by remember { mutableIntStateOf(0) }
    val load = rememberLoad(nonce) { app.repo.aiSuggestions(1, 60) }
    var gone by remember(nonce) { mutableStateOf(setOf<String>()) }
    var busy by remember { mutableStateOf(false) }

    fun act(ids: List<String>, keep: Boolean) {
        if (busy || ids.isEmpty()) return
        busy = true
        scope.launch {
            runCatching { if (keep) app.repo.aiApprove(ids) else app.repo.aiReject(ids) }
                .onSuccess { gone = gone + ids }
            busy = false
            // fetch more once a page is done
            if ((load.value?.suggestions?.count { it.id !in gone } ?: 0) == 0) nonce++
        }
    }

    Loaded(load) { res ->
        val left = res.suggestions.filter { it.id !in gone }
        if (left.isEmpty()) {
            Empty("Nothing to review", "Suggestions from the Stasharr Tagger show here. Turn on review under Settings → AI markers on the server.")
            return@Loaded
        }
        val byScene = left.groupBy { it.scene.id }
        LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
            item {
                Text(
                    "${res.count - gone.size} suggestions in ${res.scenes} scenes",
                    color = Brand.TextMuted,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
            byScene.forEach { (_, list) ->
                val scene = list.first().scene
                item(key = "s" + scene.id) {
                    Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(14.dp)).background(Brand.Surface).padding(12.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                scene.displayTitle,
                                style = MaterialTheme.typography.titleSmall,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                                modifier = Modifier.weight(1f).clickable { nav.scene(scene.id) },
                            )
                            TextButton(enabled = !busy, onClick = { act(list.map { it.id }, true) }) { Text("Keep all") }
                            TextButton(enabled = !busy, onClick = { act(list.map { it.id }, false) }) { Text("Throw all", color = Brand.TextMuted) }
                        }
                        list.forEach { s -> SuggestionRow(s, busy, onPlay = { nav.play(scene.id, s.seconds) }, onKeep = { act(listOf(s.id), true) }, onThrow = { act(listOf(s.id), false) }) }
                    }
                }
            }
            item { Spacer(Modifier.navigationBarsPadding().height(16.dp)) }
        }
    }
}

@Composable
private fun SuggestionRow(s: AiSuggestion, busy: Boolean, onPlay: () -> Unit, onKeep: () -> Unit, onThrow: () -> Unit) {
    Row(Modifier.fillMaxWidth().padding(top = 8.dp), verticalAlignment = Alignment.CenterVertically) {
        Box(
            Modifier.width(112.dp).aspectRatio(16f / 9f).clip(RoundedCornerShape(8.dp)).background(Brand.SurfaceHigh).clickable(onClick = onPlay),
            contentAlignment = Alignment.Center,
        ) {
            AsyncImage(s.scene.at(s.seconds), null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
            Icon(Icons.Rounded.PlayArrow, "Play this moment", tint = androidx.compose.ui.graphics.Color.White)
        }
        Column(Modifier.weight(1f).padding(horizontal = 10.dp)) {
            Text(s.label, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold)
            Text(
                "${formatDuration(s.seconds)}–${formatDuration(s.endSeconds)} · ${(s.score * 100).toInt()}% sure",
                style = MaterialTheme.typography.bodySmall,
                color = Brand.TextMuted,
            )
        }
        IconButton(enabled = !busy, onClick = onKeep) { Icon(Icons.Rounded.Check, "Keep", tint = Brand.Pink) }
        IconButton(enabled = !busy, onClick = onThrow) { Icon(Icons.Rounded.Close, "Throw away", tint = Brand.TextMuted) }
    }
}

// ---------------------------------------------------------------- auto-import

@Composable
private fun ImportTab() {
    val nav = LocalNav.current
    val scope = rememberCoroutineScope()
    var nonce by remember { mutableIntStateOf(0) }
    val status = rememberLoad(nonce) { app.repo.autoImportStatus() }
    val attention = rememberPaged(nonce) { page -> app.repo.needsAttention(page, 30) }
    var message by remember { mutableStateOf<String?>(null) }

    PagedGrid(attention, key = { it.id }, minCell = 170.dp, empty = "Every scene has a studio and performers", header = {
        fullWidth("status") {
            Column(Modifier.padding(vertical = 8.dp)) {
                val st = status.value
                Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(14.dp)).background(Brand.Surface).padding(14.dp)) {
                    Text(
                        when {
                            st == null -> "Loading…"
                            !st.enabled -> "Auto-import is off"
                            st.running -> "Auto-import is on and watching ${st.watchedPaths.size} folder(s)"
                            else -> "Auto-import is on but not running"
                        },
                        style = MaterialTheme.typography.titleSmall,
                    )
                    if (st != null) {
                        Text("Last check: ${whenText(st.lastCheckAt)}", style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted)
                        st.lastResult?.takeIf { it.isNotBlank() }?.let { Text(it, style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted) }
                    }
                    message?.let { Text(it, style = MaterialTheme.typography.bodySmall, color = Brand.Pink, modifier = Modifier.padding(top = 4.dp)) }
                    Row(Modifier.padding(top = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        GradientButton("Check now", onClick = {
                            scope.launch {
                                message = runCatching { app.repo.autoImportCheckNow() }.fold({ "Checking for new files…" }, { it.message })
                                delay(4000)
                                nonce++
                            }
                        })
                        TextButton(onClick = {
                            scope.launch {
                                message = runCatching { app.repo.autoImportReidentify() }.fold({ "Identifying again. See Tasks for progress." }, { it.message })
                            }
                        }) { Text("Identify again") }
                    }
                }
                Text(
                    "Needs attention: no studio or no performers",
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.padding(top = 16.dp, bottom = 4.dp),
                )
            }
        }
    }) { s -> SceneCard(s, onClick = { nav.scene(s.id) }, width = null) }
}

// ---------------------------------------------------------------- tasks

@Composable
private fun TasksTab() {
    val scope = rememberCoroutineScope()
    var nonce by remember { mutableIntStateOf(0) }
    val jobs = rememberLoad(nonce) { app.repo.jobs() }
    // follow progress while the tab is open
    LaunchedEffect(Unit) {
        while (true) {
            delay(3000)
            nonce++
        }
    }
    Loaded(jobs) { list ->
        if (list.isEmpty()) {
            Empty("No tasks running", "Scans, reels, AI markers, backups and other server tasks show here while they run.")
            return@Loaded
        }
        LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            items(list, key = { it.id }) { j ->
                Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(Brand.Surface).padding(12.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Column(Modifier.weight(1f)) {
                            Text(j.description, style = MaterialTheme.typography.titleSmall)
                            Text(
                                j.status.lowercase().replaceFirstChar { it.uppercase() } + (j.subTasks?.firstOrNull()?.let { " · $it" } ?: ""),
                                style = MaterialTheme.typography.bodySmall,
                                color = Brand.TextMuted,
                                maxLines = 2,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        if (j.status == "RUNNING" || j.status == "READY") {
                            TextButton(onClick = { scope.launch { runCatching { app.repo.stopJob(j.id) }; nonce++ } }) { Text("Stop") }
                        }
                    }
                    j.progress?.let { p ->
                        LinearProgressIndicator(
                            progress = { p.toFloat().coerceIn(0f, 1f) },
                            color = Brand.Pink,
                            trackColor = Brand.SurfaceHigh,
                            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                        )
                    }
                    j.error?.let { Text(it, color = Brand.Pink, style = MaterialTheme.typography.bodySmall) }
                }
            }
        }
    }
}

// ---------------------------------------------------------------- statistics

@Composable
private fun StatsTab() {
    val nav = LocalNav.current
    val context = LocalContext.current
    var days by rememberSaveable { mutableIntStateOf(30) }
    val stats = rememberLoad(days) { app.repo.serverStats(days) }
    fun size(b: Double) = Formatter.formatShortFileSize(context, b.toLong())

    Column(Modifier.fillMaxSize()) {
        ChipRow { listOf(7, 30, 90, 365).forEach { d -> Chip(if (d == 365) "Year" else "$d days", selected = days == d) { days = d } } }
        Loaded(stats) { s: ServerStats ->
            LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                item {
                    val t = s.totals
                    val tiles = listOf(
                        "Plays" to "${t.plays}",
                        "Scenes" to "${t.scenes}",
                        "Library" to formatRuntime(t.duration),
                        "Size" to size(t.size),
                        "Performers" to "${t.performers}",
                        "Reel views" to "${t.reelViews}",
                        "Comments" to "${t.comments}",
                        "Users" to "${t.users}",
                    )
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        tiles.chunked(4).forEach { row ->
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                row.forEach { (label, value) ->
                                    Column(Modifier.weight(1f).clip(RoundedCornerShape(12.dp)).background(Brand.Surface).padding(10.dp)) {
                                        Text(value, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold, maxLines = 1)
                                        Text(label, style = MaterialTheme.typography.labelSmall, color = Brand.TextMuted, maxLines = 1)
                                    }
                                }
                            }
                        }
                    }
                }
                item { PlaysChart(s) }
                if (s.users.isNotEmpty()) {
                    item { Heading("Who is watching") }
                    items(s.users, key = { "u" + it.id }) { u ->
                        Row(Modifier.fillMaxWidth().padding(vertical = 2.dp), verticalAlignment = Alignment.CenterVertically) {
                            Column(Modifier.weight(1f)) {
                                Text(u.name, style = MaterialTheme.typography.titleSmall)
                                Text("Last played ${whenText(u.lastPlay)} · signed in ${whenText(u.lastLogin)}", style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted)
                            }
                            Text("${u.plays} plays", style = MaterialTheme.typography.labelLarge)
                        }
                    }
                }
                if (s.topScenes.isNotEmpty()) {
                    item { Heading("Most watched scenes") }
                    items(s.topScenes, key = { "t" + it.scene.id }) { t ->
                        Row(Modifier.fillMaxWidth().clickable { nav.scene(t.scene.id) }, verticalAlignment = Alignment.CenterVertically) {
                            AsyncImage(
                                t.scene.paths.screenshot,
                                null,
                                contentScale = ContentScale.Crop,
                                modifier = Modifier.width(96.dp).aspectRatio(16f / 9f).clip(RoundedCornerShape(8.dp)),
                            )
                            Text(t.scene.displayTitle, maxLines = 2, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f).padding(horizontal = 10.dp))
                            Text("${t.plays}", style = MaterialTheme.typography.labelLarge)
                        }
                    }
                }
                if (s.topPerformers.isNotEmpty()) {
                    item { Heading("Most watched performers") }
                    item { Text(s.topPerformers.take(10).joinToString(" · ") { "${it.name} (${it.count})" }, color = Brand.TextMuted) }
                }
                if (s.topTags.isNotEmpty()) {
                    item { Heading("Most watched tags") }
                    item { Text(s.topTags.take(10).joinToString(" · ") { "${it.name} (${it.count})" }, color = Brand.TextMuted) }
                }
                if (s.generated.isNotEmpty()) {
                    item { Heading("Storage: videos ${size(s.totals.size)}, generated ${size(s.generatedTotal)}") }
                    items(s.generated, key = { "g" + it.name }) { g ->
                        Row(Modifier.fillMaxWidth()) {
                            Text(g.name, modifier = Modifier.weight(1f), color = Brand.TextMuted)
                            Text(size(g.bytes))
                        }
                    }
                }
                item { Spacer(Modifier.navigationBarsPadding().height(16.dp)) }
            }
        }
    }
}

@Composable
private fun PlaysChart(s: ServerStats) {
    val days = s.playsByDay
    if (days.isEmpty()) return
    val max = (days.maxOf { it.plays }).coerceAtLeast(1)
    Column(Modifier.fillMaxWidth().clip(RoundedCornerShape(14.dp)).background(Brand.Surface).padding(12.dp)) {
        Text("Plays per day", style = MaterialTheme.typography.titleSmall)
        Text("Most in a day: $max", style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted)
        val bar = Brand.Pink
        Canvas(Modifier.fillMaxWidth().height(110.dp).padding(top = 8.dp)) {
            val gap = 2.dp.toPx()
            val w = (size.width - gap * (days.size - 1)) / days.size
            days.forEachIndexed { i, d ->
                val h = size.height * d.plays / max
                drawRoundRect(
                    bar,
                    topLeft = Offset(i * (w + gap), size.height - h),
                    size = Size(w.coerceAtLeast(1f), h),
                    cornerRadius = CornerRadius(2.dp.toPx()),
                )
            }
        }
    }
}

@Composable
private fun Heading(text: String) {
    Text(text, style = MaterialTheme.typography.titleMedium, modifier = Modifier.padding(top = 8.dp))
}

@Composable
private fun Empty(title: String, sub: String) {
    Column(
        Modifier.fillMaxSize().padding(32.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(Icons.Rounded.Refresh, null, tint = Brand.TextMuted, modifier = Modifier.size(40.dp))
        Spacer(Modifier.height(10.dp))
        Text(title, style = MaterialTheme.typography.titleMedium)
        Text(sub, color = Brand.TextMuted, style = MaterialTheme.typography.bodySmall, modifier = Modifier.padding(top = 4.dp))
    }
}
