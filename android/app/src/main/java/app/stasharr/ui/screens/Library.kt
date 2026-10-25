package app.stasharr.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.LazyGridScope
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.PlayArrow
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.Shuffle
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.blur
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import app.stasharr.app
import app.stasharr.data.Page
import app.stasharr.data.Scene
import app.stasharr.data.SceneScope
import app.stasharr.data.SceneSort
import app.stasharr.ui.LocalNav
import app.stasharr.ui.components.Chip
import app.stasharr.ui.components.ChipRow
import app.stasharr.ui.components.GlassButton
import app.stasharr.ui.components.GradientButton
import app.stasharr.ui.components.GradientText
import app.stasharr.ui.components.PagedGrid
import app.stasharr.ui.components.PerformerBubble
import app.stasharr.ui.components.SceneCard
import app.stasharr.ui.components.StudioTile
import app.stasharr.ui.components.fullWidth
import app.stasharr.ui.components.rememberLoad
import app.stasharr.ui.components.rememberPaged
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage
import kotlinx.coroutines.delay
import kotlin.random.Random

// ---------------------------------------------------------------- browse

private enum class BrowseTab(val label: String) { Scenes("Scenes"), Performers("Performers"), Studios("Studios"), Tags("Tags"), Audio("Audio") }

@Composable
private fun SearchField(value: String, onChange: (String) -> Unit, placeholder: String) {
    TextField(
        value = value,
        onValueChange = onChange,
        singleLine = true,
        placeholder = { Text(placeholder) },
        leadingIcon = { Icon(Icons.Rounded.Search, null) },
        trailingIcon = {
            if (value.isNotEmpty()) {
                IconButton(onClick = { onChange("") }) { Icon(Icons.Rounded.Close, "Clear") }
            }
        },
        shape = RoundedCornerShape(50),
        colors = TextFieldDefaults.colors(
            focusedContainerColor = Brand.SurfaceHigh,
            unfocusedContainerColor = Brand.Surface,
            focusedIndicatorColor = Color.Transparent,
            unfocusedIndicatorColor = Color.Transparent,
            cursorColor = Brand.Pink,
        ),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp),
    )
}

@Composable
fun BrowseScreen() {
    val nav = LocalNav.current
    var tab by rememberSaveable { mutableStateOf(BrowseTab.Scenes) }
    var query by rememberSaveable { mutableStateOf("") }
    var q by rememberSaveable { mutableStateOf("") }
    var sort by rememberSaveable { mutableStateOf(SceneSort.RecentlyAdded) }
    var seed by rememberSaveable { mutableIntStateOf(Random.nextInt(1, 99_999_999)) }

    // search as you type, after a short pause
    LaunchedEffect(query) {
        delay(350)
        q = query.trim()
    }

    Column(Modifier.fillMaxSize().statusBarsPadding()) {
        Spacer(Modifier.height(8.dp))
        GradientText("Browse", modifier = Modifier.padding(horizontal = 18.dp, vertical = 6.dp))
        SearchField(query, { query = it }, "Search ${tab.label.lowercase()}")
        ChipRow {
            BrowseTab.entries.forEach { t -> Chip(t.label, selected = t == tab) { tab = t } }
        }

        when (tab) {
            BrowseTab.Scenes, BrowseTab.Audio -> {
                ChipRow {
                    SceneSort.entries.forEach { s ->
                        Chip(s.label, selected = s == sort) {
                            if (s == SceneSort.Random) seed = Random.nextInt(1, 99_999_999)
                            sort = s
                        }
                    }
                }
                val scope = if (tab == BrowseTab.Audio) SceneScope.Audio else SceneScope.All
                SceneGrid(keys = arrayOf(tab, sort, q, seed), fetch = { page ->
                    app.repo.scenes(sort, page = page, perPage = 30, q = q, scope = scope, seed = seed)
                })
            }
            BrowseTab.Performers -> {
                val state = rememberPaged(q) { page -> app.repo.performers(page = page, perPage = 40, q = q) }
                PagedGrid(state, key = { it.id }, minCell = 104.dp, empty = "No performers found") { p ->
                    PerformerBubble(p, onClick = { nav.performer(p.id) }, size = 92.dp)
                }
            }
            BrowseTab.Studios -> {
                val state = rememberPaged(q) { page -> app.repo.studios(page = page, perPage = 40, q = q) }
                PagedGrid(state, key = { it.id }, minCell = 160.dp, empty = "No studios found") { s ->
                    StudioTile(s, onClick = { nav.studio(s.id) }, width = null)
                }
            }
            BrowseTab.Tags -> {
                val state = rememberPaged(q) { page -> app.repo.tags(page = page, perPage = 80, q = q) }
                PagedGrid(state, key = { it.id }, minCell = 150.dp, empty = "No tags found") { t ->
                    TagTile(t.name, t.sceneCount) { nav.tag(t.id, t.name) }
                }
            }
        }
    }
}

@Composable
private fun TagTile(name: String, count: Int, onClick: () -> Unit) {
    val hue = (name.hashCode() and 0xffff) % 360
    val tint = Color.hsv(hue.toFloat(), 0.55f, 0.55f)
    Box(
        Modifier
            .fillMaxWidth()
            .height(72.dp)
            .clip(RoundedCornerShape(14.dp))
            .background(Brush.linearGradient(listOf(tint, Brand.SurfaceHigh)))
            .clickable(onClick = onClick)
            .padding(12.dp),
    ) {
        Column(Modifier.align(Alignment.BottomStart)) {
            Text(name, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis, color = Color.White)
            Text("$count scenes", style = MaterialTheme.typography.bodySmall, color = Color.White.copy(alpha = 0.7f))
        }
    }
}

/** A grid of scenes that loads more as you scroll. */
@Composable
fun SceneGrid(
    keys: Array<Any?>,
    fetch: suspend (page: Int) -> Page<Scene>,
    header: (LazyGridScope.() -> Unit)? = null,
) {
    val nav = LocalNav.current
    val state = rememberPaged(*keys) { page -> fetch(page) }
    PagedGrid(state, key = { it.id }, minCell = 170.dp, header = header, empty = "No scenes here yet") { s ->
        SceneCard(s, onClick = { nav.scene(s.id) }, width = null)
    }
}

// ---------------------------------------------------------------- "see all" lists

enum class SceneListKind(val title: String) {
    Continue("Continue watching"),
    Recommended("Recommended for you"),
    Recent("Recently added"),
    WatchLater("Watch later"),
    TopRated("Top rated"),
    Audio("Audio"),
}

@Composable
private fun BackHeader(title: String) {
    val nav = LocalNav.current
    Row(
        Modifier.fillMaxWidth().statusBarsPadding().padding(horizontal = 4.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        IconButton(onClick = { nav.back() }) { Icon(Icons.AutoMirrored.Rounded.ArrowBack, "Back") }
        Text(title, style = MaterialTheme.typography.titleLarge, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}

@Composable
fun SceneListScreen(kind: SceneListKind) {
    Column(Modifier.fillMaxSize()) {
        BackHeader(kind.title)
        val fixed: (suspend () -> List<Scene>)? = when (kind) {
            SceneListKind.Continue -> ({ app.repo.continueWatching(60) })
            SceneListKind.Recommended -> ({ app.repo.recommended(60) })
            SceneListKind.WatchLater -> ({ app.repo.watchLaterScenes() })
            else -> null
        }
        if (fixed != null) {
            SceneGrid(keys = arrayOf(kind), fetch = { page ->
                if (page > 1) Page(0, emptyList()) else fixed().let { Page(it.size, it) }
            })
        } else {
            val (sort, scope) = when (kind) {
                SceneListKind.TopRated -> SceneSort.TopRated to SceneScope.All
                SceneListKind.Audio -> SceneSort.RecentlyAdded to SceneScope.Audio
                else -> SceneSort.RecentlyAdded to SceneScope.All
            }
            SceneGrid(keys = arrayOf(kind), fetch = { page ->
                app.repo.scenes(sort, page = page, perPage = 30, scope = scope)
            })
        }
    }
}

// ---------------------------------------------------------------- performer, studio, tag

/** Sort chips and play-all buttons shown above a scene grid. */
@Composable
private fun SceneTools(
    sort: SceneSort,
    onSort: (SceneSort) -> Unit,
    onPlayAll: () -> Unit,
    onShuffle: () -> Unit,
) {
    Column {
        Row(Modifier.padding(horizontal = 0.dp, vertical = 8.dp)) {
            GradientButton("Play all", onClick = onPlayAll, icon = Icons.Rounded.PlayArrow, height = 42.dp)
            Spacer(Modifier.width(10.dp))
            GlassButton("Shuffle", onClick = onShuffle, icon = Icons.Rounded.Shuffle, height = 42.dp)
        }
        Row(Modifier.padding(vertical = 4.dp)) {
            listOf(SceneSort.RecentlyAdded, SceneSort.Newest, SceneSort.TopRated, SceneSort.MostPlayed, SceneSort.Title).forEach { s ->
                Box(Modifier.padding(end = 6.dp)) { Chip(s.label, selected = s == sort) { onSort(s) } }
            }
        }
    }
}

@Composable
private fun ScopedScenes(
    scope: SceneScope,
    header: @Composable () -> Unit,
) {
    val nav = LocalNav.current
    var sort by rememberSaveable { mutableStateOf(SceneSort.RecentlyAdded) }

    suspend fun ids(random: Boolean): List<String> {
        val s = if (random) SceneSort.Random else sort
        return app.repo.scenes(s, perPage = 200, scope = scope, seed = Random.nextInt(1, 99_999_999)).items.map { it.id }
    }

    val playAll = rememberLoad(scope, sort) { ids(false) }

    SceneGrid(keys = arrayOf(scope, sort), fetch = { page -> app.repo.scenes(sort, page = page, perPage = 30, scope = scope) }) {
        fullWidth("header") { header() }
        fullWidth("tools") {
            SceneTools(
                sort = sort,
                onSort = { sort = it },
                onPlayAll = { playAll.value?.let { if (it.isNotEmpty()) nav.play(it.first(), 0.0, it.drop(1)) } },
                onShuffle = { playAll.value?.shuffled()?.let { if (it.isNotEmpty()) nav.play(it.first(), 0.0, it.drop(1)) } },
            )
        }
    }
}

@Composable
private fun ProfileHeader(
    title: String,
    subtitle: String?,
    image: String?,
    round: Boolean,
    details: String? = null,
) {
    val nav = LocalNav.current
    Box(Modifier.fillMaxWidth().height(if (details.isNullOrBlank()) 300.dp else 360.dp)) {
        if (image != null) {
            AsyncImage(
                model = image,
                contentDescription = null,
                contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxSize().blur(30.dp),
            )
        }
        Box(
            Modifier.fillMaxSize().background(
                Brush.verticalGradient(
                    listOf(Brand.Pink.copy(alpha = 0.25f), Brand.Background.copy(alpha = 0.85f), Brand.Background)
                )
            )
        )
        IconButton(
            onClick = { nav.back() },
            modifier = Modifier.statusBarsPadding().padding(4.dp),
        ) { Icon(Icons.AutoMirrored.Rounded.ArrowBack, "Back", tint = Color.White) }

        Column(
            Modifier.align(Alignment.BottomStart).fillMaxWidth().padding(bottom = 8.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Box(
                Modifier
                    .size(if (round) 120.dp else 96.dp)
                    .clip(if (round) CircleShape else RoundedCornerShape(20.dp))
                    .background(Brand.Gradient)
                    .padding(3.dp)
                    .clip(if (round) CircleShape else RoundedCornerShape(18.dp))
                    .background(Brand.SurfaceHigh),
                contentAlignment = Alignment.Center,
            ) {
                Icon(Icons.Rounded.Person, null, tint = Brand.TextMuted, modifier = Modifier.size(48.dp))
                if (image != null) {
                    AsyncImage(
                        model = image,
                        contentDescription = title,
                        contentScale = if (round) ContentScale.Crop else ContentScale.Fit,
                        modifier = Modifier.fillMaxSize(),
                    )
                }
            }
            Spacer(Modifier.height(12.dp))
            Text(title, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Black)
            if (subtitle != null) Text(subtitle, color = Brand.TextMuted, style = MaterialTheme.typography.bodyMedium)
            if (!details.isNullOrBlank()) {
                Spacer(Modifier.height(8.dp))
                Text(
                    details,
                    color = Brand.Text.copy(alpha = 0.8f),
                    style = MaterialTheme.typography.bodySmall,
                    maxLines = 3,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.padding(horizontal = 8.dp),
                )
            }
        }
    }
}

@Composable
fun PerformerScreen(id: String) {
    val p = rememberLoad(id) { app.repo.performer(id) }.value
    ScopedScenes(SceneScope.ByPerformer(id)) {
        ProfileHeader(
            title = p?.name ?: "",
            subtitle = p?.let { listOfNotNull(it.country, "${it.sceneCount} scenes").joinToString(" • ") },
            image = p?.imagePath,
            round = true,
            details = p?.details,
        )
    }
}

@Composable
fun StudioScreen(id: String) {
    val s = rememberLoad(id) { app.repo.studio(id) }.value
    ScopedScenes(SceneScope.ByStudio(id)) {
        ProfileHeader(
            title = s?.name ?: "",
            subtitle = s?.let { "${it.sceneCount} scenes" },
            image = s?.imagePath?.takeIf { !it.contains("default=true") },
            round = false,
        )
    }
}

@Composable
fun TagScreen(id: String, name: String) {
    ScopedScenes(SceneScope.ByTag(id)) {
        ProfileHeader(title = name, subtitle = "Tag", image = null, round = false)
    }
}
