package app.stasharr.ui.store

import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.Favorite
import androidx.compose.material.icons.rounded.FavoriteBorder
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.Tune
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
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
import androidx.compose.ui.unit.sp
import app.stasharr.app
import app.stasharr.data.Performer
import app.stasharr.data.Reel
import app.stasharr.data.Scene
import app.stasharr.data.SiteHomeRow
import app.stasharr.data.Studio
import app.stasharr.data.Tag
import app.stasharr.ui.LocalNav
import app.stasharr.ui.SiteMode
import app.stasharr.ui.components.Chip
import app.stasharr.ui.components.ChipRow
import app.stasharr.ui.components.GradientButton
import app.stasharr.ui.components.Loaded
import app.stasharr.ui.components.PagedGrid
import app.stasharr.ui.components.StudioTile
import app.stasharr.ui.components.fullWidth
import app.stasharr.ui.components.rememberLoad
import app.stasharr.ui.components.rememberPaged
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonArray
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonObject

private val femaleOnly: Boolean get() = SiteMode.look?.femaleOnly ?: true
private val starsWord: String get() = if (femaleOnly) "Pornstars" else "Models"

/** The heading a home row gets when the admin gave none, as on the website. */
fun rowTitle(r: SiteHomeRow): String = r.title.ifBlank {
    when (r.type) {
        "continue" -> "Continue watching"
        "latest" -> "Latest updates"
        "models" -> if (femaleOnly) "Top pornstars" else "Top models"
        "trending" -> "Trending now"
        "popular" -> "Most viewed"
        "recommended" -> "Picked for you"
        "favorites" -> "New from your favourites"
        "liked" -> "Your liked scenes"
        "reels" -> "Reels"
        "top_rated" -> "Top rated"
        "categories" -> "Categories"
        "channels" -> "Channels"
        else -> r.tag?.name ?: ""
    }
}

private val defaultRows = listOf("continue", "latest", "models", "trending", "favorites", "recommended", "reels", "top_rated", "categories", "channels")
    .map { SiteHomeRow(type = it) }

/** Everything the storefront home shows. */
private data class StoreHomeData(
    val featured: List<Scene>,
    val ticker: List<Scene>,
    val rows: List<Pair<SiteHomeRow, Any>>,
)

private suspend fun loadStoreHome(rows: List<SiteHomeRow>): StoreHomeData = coroutineScope {
    val repo = app.repo
    suspend fun <T> safe(block: suspend () -> T, empty: T): T = runCatching { block() }.getOrDefault(empty)
    val featured = async { safe({ repo.sitePage("FEATURED", 1, 7).items.filter { it.paths.screenshot != null }.take(6) }, emptyList()) }
    val ticker = async { safe({ repo.scenes(perPage = 15).items }, emptyList()) }
    val loaded = rows.map { r ->
        r to async<Any> {
            when (r.type) {
                "continue" -> safe({ repo.continueWatching(20) }, emptyList<Scene>())
                "latest" -> safe({ repo.scenes(perPage = 12).items }, emptyList<Scene>())
                "models" -> safe({ repo.models(femaleOnly, "scenes_count", true, 1, 16).items }, emptyList<Performer>())
                "trending" -> safe({ repo.sitePage("TRENDING", 1, 16).items }, emptyList<Scene>())
                "popular" -> safe({ repo.sitePage("POPULAR", 1, 16).items }, emptyList<Scene>())
                "top_rated" -> safe({ repo.sitePage("TOP_RATED", 1, 16).items }, emptyList<Scene>())
                "recommended" -> safe({ repo.recommended(20) }, emptyList<Scene>())
                "favorites" -> safe({ repo.fromFavorites(16) }, emptyList<Scene>())
                "liked" -> safe({ repo.likedScenes(1, 16).items }, emptyList<Scene>())
                "reels" -> safe({ repo.reels(1, 16).items }, emptyList<Reel>())
                "categories" -> safe({ repo.topTags(12) }, emptyList<Tag>())
                "channels" -> safe({ repo.studios(perPage = 12).items.filter { it.sceneCount > 0 } }, emptyList<Studio>())
                "tag" -> r.tag?.let { t ->
                    safe({ repo.storeScenes("created_at", true, 1, 16, tagFilter(t.id)).items }, emptyList<Scene>())
                } ?: emptyList<Scene>()
                else -> emptyList<Scene>()
            }
        }
    }
    StoreHomeData(featured.await(), ticker.await(), loaded.map { (r, d) -> r to d.await() })
}

private fun tagFilter(id: String): JsonObject = buildJsonObject {
    putJsonObject("tags") {
        put("value", buildJsonArray { add(JsonPrimitive(id)) })
        put("modifier", "INCLUDES")
        put("depth", 0)
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StoreHome() {
    val nav = LocalNav.current
    val look = SiteMode.look
    val rows = (look?.homeRows?.filter { it.enabled }?.takeIf { it.isNotEmpty() } ?: defaultRows)
    val loader = rememberLoad(rows) { loadStoreHome(rows) }
    val list = rememberLazyListState()
    val scrolled by remember { derivedStateOf { list.firstVisibleItemIndex > 0 || list.firstVisibleItemScrollOffset > 300 } }

    Box(Modifier.fillMaxSize()) {
        Loaded(loader) { data ->
            PullToRefreshBox(isRefreshing = loader.refreshing, onRefresh = loader.reload) {
                LazyColumn(state = list, modifier = Modifier.fillMaxSize()) {
                    item(key = "hero") { StoreHero(data.featured) }
                    item(key = "ticker") { StoreTicker(data.ticker) }
                    item(key = "promo") { look?.promo?.let { StorePromo(it) } }
                    data.rows.forEachIndexed { i, (row, content) ->
                        item(key = "row-$i-${row.type}") { HomeRowView(row, content) }
                    }
                    item(key = "foot") { StoreFooter() }
                }
            }
        }
        StoreTopBar(solid = scrolled, onSearch = { nav.tab("scenes") }, onAccount = { nav.tab("profile") })
    }
}

@Suppress("UNCHECKED_CAST")
@Composable
private fun HomeRowView(row: SiteHomeRow, content: Any) {
    val nav = LocalNav.current
    val title = rowTitle(row)
    when (row.type) {
        "models" -> ModelRow(title, content as List<Performer>, ranked = true, onAll = { nav.tab("models") })
        "categories" -> {
            val tags = content as List<Tag>
            if (tags.isEmpty()) return
            StoreHeader(title)
            LazyRow(contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                items(tags, key = { it.id }) { t -> CategoryTile(t.name, t.sceneCount, Modifier.width(170.dp)) { nav.tag(t.id, t.name) } }
            }
        }
        "channels" -> {
            val studios = content as List<Studio>
            if (studios.isEmpty()) return
            StoreHeader(title)
            LazyRow(contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                items(studios, key = { it.id }) { s -> StudioTile(s, onClick = { nav.studio(s.id) }) }
            }
        }
        "reels" -> {
            val reels = content as List<Reel>
            if (reels.isEmpty()) return
            StoreHeader(title, onAll = { nav.tab("reels") }, allText = "Watch reels")
            LazyRow(contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                items(reels, key = { it.id }) { r ->
                    Box(
                        Modifier
                            .width(130.dp)
                            .aspectRatio(9f / 14f)
                            .clip(RoundedCornerShape(12.dp))
                            .background(Brand.SurfaceHigh)
                            .clickable { nav.tab("reels") },
                    ) {
                        AsyncImage(r.scene.paths.screenshot, null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
                        Box(Modifier.fillMaxSize().background(Brush.verticalGradient(0.5f to Color.Transparent, 1f to Color.Black.copy(alpha = 0.85f))))
                        Text(
                            "▶ REEL",
                            color = Color.White,
                            fontSize = 10.sp,
                            fontWeight = FontWeight.Black,
                            modifier = Modifier.padding(8.dp).clip(RoundedCornerShape(4.dp)).background(Brand.Pink).padding(horizontal = 6.dp, vertical = 2.dp),
                        )
                        Text(
                            r.scene.title?.takeIf { it.isNotBlank() } ?: "Reel",
                            color = Color.White,
                            style = MaterialTheme.typography.labelLarge,
                            maxLines = 2,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.align(Alignment.BottomStart).padding(8.dp),
                        )
                    }
                }
            }
        }
        "latest" -> {
            val scenes = content as List<Scene>
            if (scenes.isEmpty()) return
            StoreHeader(title, onAll = { nav.tab("scenes") })
            Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
                scenes.take(6).chunked(2).forEach { pair ->
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        pair.forEach { s -> StoreSceneCard(s, Modifier.weight(1f), width = null) }
                        if (pair.size == 1) Spacer(Modifier.weight(1f))
                    }
                }
            }
        }
        "continue" -> {
            val scenes = content as List<Scene>
            SceneRow(title, scenes, onAll = { nav.tab("mylist") })
        }
        "tag" -> SceneRow(title, content as List<Scene>, onAll = row.tag?.let { t -> { nav.tag(t.id, t.name) } })
        "trending", "popular", "top_rated" -> SceneRow(title, content as List<Scene>, onAll = { nav.tab("scenes") })
        else -> SceneRow(title, content as List<Scene>)
    }
}

@Composable
private fun StoreFooter() {
    val look = SiteMode.look
    Column(
        Modifier.fillMaxWidth().padding(top = 36.dp, bottom = 28.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        StoreLogo(look?.name?.ifBlank { "Stasharr" } ?: "Stasharr", size = 18)
        look?.tagline?.takeIf { it.isNotBlank() }?.let {
            Spacer(Modifier.height(4.dp))
            Text(it, color = Brand.TextMuted, style = MaterialTheme.typography.bodySmall)
        }
        Spacer(Modifier.height(8.dp))
        Text(
            "All models were 18 years of age or older at the time of filming.",
            color = Brand.TextMuted.copy(alpha = 0.7f),
            fontSize = 10.sp,
        )
    }
}

// ---------------------------------------------------------------- scenes

private data class StoreSort(val id: String, val label: String, val kind: String? = null, val sort: String = "created_at", val desc: Boolean = true)

private val storeSorts = listOf(
    StoreSort("latest", "Latest"),
    StoreSort("trending", "Trending", kind = "TRENDING"),
    StoreSort("popular", "Most viewed", kind = "POPULAR"),
    StoreSort("top", "Top rated", kind = "TOP_RATED"),
    StoreSort("release", "Release date", sort = "date"),
    StoreSort("longest", "Longest", sort = "duration"),
    StoreSort("random", "Random", sort = "random"),
)

private data class StoreFilter(
    val tag: Tag? = null,
    val length: String? = null,
    val quality: String? = null,
    val studio: Studio? = null,
) {
    val count: Int get() = listOfNotNull(tag, length, quality, studio).size

    fun toJson(): JsonObject? {
        if (count == 0) return null
        return buildJsonObject {
            tag?.let {
                putJsonObject("tags") { put("value", buildJsonArray { add(JsonPrimitive(it.id)) }); put("modifier", "INCLUDES"); put("depth", 0) }
            }
            studio?.let {
                putJsonObject("studios") { put("value", buildJsonArray { add(JsonPrimitive(it.id)) }); put("modifier", "INCLUDES"); put("depth", -1) }
            }
            when (length) {
                "short" -> putJsonObject("duration") { put("value", 600); put("modifier", "LESS_THAN") }
                "medium" -> putJsonObject("duration") { put("value", 600); put("value2", 1800); put("modifier", "BETWEEN") }
                "long" -> putJsonObject("duration") { put("value", 1800); put("value2", 3600); put("modifier", "BETWEEN") }
                "xlong" -> putJsonObject("duration") { put("value", 3600); put("modifier", "GREATER_THAN") }
            }
            when (quality) {
                "4k" -> putJsonObject("resolution") { put("value", "QUAD_HD"); put("modifier", "GREATER_THAN") }
                "fhd" -> putJsonObject("resolution") { put("value", "STANDARD_HD"); put("modifier", "GREATER_THAN") }
                "hd" -> putJsonObject("resolution") { put("value", "STANDARD"); put("modifier", "GREATER_THAN") }
            }
        }
    }
}

private val lengthChoices = listOf("short" to "Under 10 min", "medium" to "10–30 min", "long" to "30–60 min", "xlong" to "Over 1 hour")
private val qualityChoices = listOf("4k" to "4K", "fhd" to "1080p and up", "hd" to "720p and up")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StoreScenes() {
    var sortId by rememberSaveable { mutableStateOf("latest") }
    var query by rememberSaveable { mutableStateOf("") }
    var filter by remember { mutableStateOf(StoreFilter()) }
    var showFilters by remember { mutableStateOf(false) }
    val seed = remember { (1..99_999_999).random() }
    val q = query.trim()

    // lists counted across everyone have no filters
    val sorts = if (filter.count > 0 || q.isNotEmpty()) storeSorts.filter { it.kind == null } else storeSorts
    val sort = sorts.firstOrNull { it.id == sortId } ?: sorts.first()

    val state = rememberPaged(sort, filter, q) { page ->
        if (sort.kind != null) app.repo.sitePage(sort.kind, page, 30)
        else app.repo.storeScenes(if (sort.sort == "random") "random_$seed" else sort.sort, sort.desc, page, 30, filter.toJson(), q.ifBlank { null })
    }

    Column(Modifier.fillMaxSize()) {
        StorePageTitle("Scenes")
        SearchBox(query, { query = it }, "Search scenes and stars")
        ChipRow {
            sorts.forEach { s -> Chip(s.label, selected = s.id == sort.id) { sortId = s.id } }
        }
        Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 2.dp), verticalAlignment = Alignment.CenterVertically) {
            Row(
                Modifier.clip(RoundedCornerShape(50)).background(if (filter.count > 0) Brand.Pink else Brand.Surface)
                    .clickable { showFilters = true }.padding(horizontal = 14.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(Icons.Rounded.Tune, null, tint = Color.White, modifier = Modifier.size(16.dp))
                Spacer(Modifier.width(6.dp))
                Text(if (filter.count > 0) "Filters · ${filter.count}" else "Filters", color = Color.White, style = MaterialTheme.typography.labelLarge)
            }
            Spacer(Modifier.weight(1f))
            if (state.total > 0) Text("${state.total} scenes", color = Brand.TextMuted, style = MaterialTheme.typography.bodySmall)
        }
        if (q.isNotEmpty()) SearchModels(q)
        PagedGrid(state, key = { it.id }, minCell = 170.dp, empty = "No scenes match") { s -> StoreSceneCard(s, width = null) }
    }

    if (showFilters) {
        FilterSheet(filter, onChange = { filter = it }, onDismiss = { showFilters = false })
    }
}

@Composable
private fun SearchModels(q: String) {
    val models = rememberLoad(q) { app.repo.models(femaleOnly, "scenes_count", true, 1, 12, q = q).items }
    val list = models.value.orEmpty()
    if (list.isEmpty()) return
    LazyRow(contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        items(list, key = { it.id }) { p -> ModelCard(p, width = 96.dp) }
    }
}

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
private fun FilterSheet(filter: StoreFilter, onChange: (StoreFilter) -> Unit, onDismiss: () -> Unit) {
    val tags = rememberLoad(Unit) { app.repo.topTags(60) }
    val studios = rememberLoad(Unit) { app.repo.studios(perPage = 60).items.filter { it.sceneCount > 0 } }
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true), containerColor = Brand.Surface) {
        Column(Modifier.fillMaxWidth().verticalScroll(rememberScrollState()).navigationBarsPadding().padding(horizontal = 20.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("Filters", style = MaterialTheme.typography.titleLarge, modifier = Modifier.weight(1f))
                if (filter.count > 0) {
                    Text("Clear", color = Brand.Pink, style = MaterialTheme.typography.labelLarge, modifier = Modifier.clickable { onChange(StoreFilter()) }.padding(8.dp))
                }
            }
            FilterGroup("Length") {
                lengthChoices.forEach { (id, label) -> Chip(label, selected = filter.length == id) { onChange(filter.copy(length = if (filter.length == id) null else id)) } }
            }
            FilterGroup("Quality") {
                qualityChoices.forEach { (id, label) -> Chip(label, selected = filter.quality == id) { onChange(filter.copy(quality = if (filter.quality == id) null else id)) } }
            }
            FilterGroup("Category") {
                tags.value.orEmpty().forEach { t -> Chip(t.name, selected = filter.tag?.id == t.id) { onChange(filter.copy(tag = if (filter.tag?.id == t.id) null else t)) } }
            }
            FilterGroup("Channel") {
                studios.value.orEmpty().forEach { s -> Chip(s.name, selected = filter.studio?.id == s.id) { onChange(filter.copy(studio = if (filter.studio?.id == s.id) null else s)) } }
            }
            Spacer(Modifier.height(16.dp))
            GradientButton("Show scenes", onClick = onDismiss, modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(16.dp))
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun FilterGroup(title: String, content: @Composable () -> Unit) {
    Text(title.uppercase(), color = Brand.TextMuted, style = MaterialTheme.typography.labelMedium, modifier = Modifier.padding(top = 16.dp, bottom = 8.dp))
    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) { content() }
}

@Composable
private fun StorePageTitle(title: String) {
    Text(
        title,
        style = MaterialTheme.typography.headlineMedium,
        fontWeight = FontWeight.Black,
        modifier = Modifier.statusBarsPadding().padding(start = 16.dp, top = 12.dp, bottom = 6.dp),
    )
}

@Composable
private fun SearchBox(value: String, onChange: (String) -> Unit, hint: String) {
    TextField(
        value = value,
        onValueChange = onChange,
        singleLine = true,
        placeholder = { Text(hint) },
        leadingIcon = { Icon(Icons.Rounded.Search, null) },
        trailingIcon = {
            if (value.isNotEmpty()) IconButton(onClick = { onChange("") }) { Icon(Icons.Rounded.Close, "Clear") }
        },
        shape = RoundedCornerShape(50),
        colors = TextFieldDefaults.colors(
            focusedContainerColor = Brand.Surface,
            unfocusedContainerColor = Brand.Surface,
            focusedIndicatorColor = Color.Transparent,
            unfocusedIndicatorColor = Color.Transparent,
        ),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp),
    )
}

// ---------------------------------------------------------------- pornstars

private val modelSorts = listOf(
    Triple("scenes_count", "Most scenes", true),
    Triple("rating", "Top rated", true),
    Triple("created_at", "Newest", true),
    Triple("name", "A–Z", false),
)

@Composable
fun StoreModels() {
    var sortKey by rememberSaveable { mutableStateOf("scenes_count") }
    var letter by rememberSaveable { mutableStateOf("") }
    var query by rememberSaveable { mutableStateOf("") }
    val sort = modelSorts.first { it.first == sortKey }
    val q = query.trim()
    val state = rememberPaged(sortKey, letter, q) { page ->
        app.repo.models(femaleOnly, sort.first, sort.third, page, 36, q = q.ifBlank { null }, letter = letter.ifBlank { null })
    }
    Column(Modifier.fillMaxSize()) {
        StorePageTitle(starsWord)
        SearchBox(query, { query = it }, "Search ${starsWord.lowercase()}")
        ChipRow { modelSorts.forEach { (key, label, _) -> Chip(label, selected = key == sortKey) { sortKey = key } } }
        LazyRow(contentPadding = PaddingValues(horizontal = 16.dp, vertical = 4.dp), horizontalArrangement = Arrangement.spacedBy(4.dp)) {
            items(listOf("") + ('A'..'Z').map { it.toString() }) { l ->
                Box(
                    Modifier
                        .size(34.dp)
                        .clip(CircleShape)
                        .background(if (letter == l) Brand.Pink else Color.Transparent)
                        .clickable { letter = l },
                    contentAlignment = Alignment.Center,
                ) {
                    Text(l.ifBlank { "All" }, color = if (letter == l) Color.White else Brand.TextMuted, fontSize = 12.sp, fontWeight = FontWeight.Bold)
                }
            }
        }
        PagedGrid(state, key = { it.id }, minCell = 110.dp, empty = "No ${starsWord.lowercase()} found") { p -> ModelCard(p, width = null) }
    }
}

// ---------------------------------------------------------------- one pornstar

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun StoreModel(id: String) {
    val nav = LocalNav.current
    val scope = rememberCoroutineScope()
    val profile = rememberLoad(id) { app.repo.modelProfile(id) ?: error("This model is not available") }
    val popular = rememberLoad(id) { app.repo.sitePage("POPULAR", 1, 10, performerId = id).items }
    val costars = rememberLoad(id) { app.repo.coStars(id) }
    var sortId by rememberSaveable(id) { mutableStateOf("latest") }
    val seed = remember { (1..99_999_999).random() }
    val filter = remember(id) {
        buildJsonObject {
            putJsonObject("performers") { put("value", buildJsonArray { add(JsonPrimitive(id)) }); put("modifier", "INCLUDES") }
        }
    }
    val scenes = rememberPaged(id, sortId) { page ->
        val s = storeSorts.first { it.id == sortId }
        app.repo.storeScenes(if (s.sort == "random") "random_$seed" else s.sort, s.desc, page, 30, filter)
    }

    Loaded(profile) { m ->
        var following by remember(m.id) { mutableStateOf(m.following) }
        val cover = popular.value?.firstOrNull { it.paths.screenshot != null }?.paths?.screenshot ?: m.imagePath
        PagedGrid(scenes, key = { it.id }, minCell = 170.dp, empty = "No scenes yet", contentPadding = PaddingValues(start = 16.dp, end = 16.dp, bottom = 24.dp), header = {
            fullWidth("cover") {
                Box(Modifier.fillMaxWidth().height(520.dp).padding(horizontal = 0.dp)) {
                    AsyncImage(cover, null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
                    Box(Modifier.fillMaxSize().background(Brush.verticalGradient(0f to Color.Black.copy(alpha = 0.4f), 0.5f to Color.Black.copy(alpha = 0.5f), 1f to Brand.Background)))
                    IconButton(onClick = { nav.back() }, modifier = Modifier.statusBarsPadding().padding(4.dp)) {
                        Icon(Icons.AutoMirrored.Rounded.ArrowBack, "Back", tint = Color.White)
                    }
                    Row(Modifier.align(Alignment.BottomStart).padding(bottom = 8.dp), verticalAlignment = Alignment.Bottom) {
                        Box(
                            Modifier.width(130.dp).aspectRatio(2f / 3f).clip(RoundedCornerShape(14.dp)).background(Brand.SurfaceHigh)
                                .border(2.dp, Brand.Pink.copy(alpha = 0.6f), RoundedCornerShape(14.dp)),
                        ) {
                            if (m.imagePath != null) AsyncImage(m.imagePath, m.name, contentScale = ContentScale.Crop, alignment = Alignment.TopCenter, modifier = Modifier.fillMaxSize())
                        }
                        Spacer(Modifier.width(14.dp))
                        Column(Modifier.weight(1f)) {
                            Text(if (m.gender == "FEMALE") "PORNSTAR" else "MODEL", color = Brand.Pink, fontSize = 11.sp, fontWeight = FontWeight.Black, letterSpacing = 2.sp)
                            Text(m.name, color = Color.White, fontSize = 30.sp, lineHeight = 32.sp, fontWeight = FontWeight.Black, maxLines = 2, overflow = TextOverflow.Ellipsis)
                            Text("${m.sceneCount} scenes", color = Color.White.copy(alpha = 0.8f), style = MaterialTheme.typography.bodyMedium)
                            Spacer(Modifier.height(10.dp))
                            Row(
                                Modifier.height(40.dp).clip(RoundedCornerShape(50))
                                    .background(if (following) Color.White.copy(alpha = 0.14f) else Brand.Pink)
                                    .clickable {
                                        val was = following
                                        following = !was
                                        scope.launch { runCatching { app.repo.toggleFavoriteModel(m.id) }.onSuccess { following = it }.onFailure { following = was } }
                                    }
                                    .padding(horizontal = 16.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Icon(if (following) Icons.Rounded.Favorite else Icons.Rounded.FavoriteBorder, null, tint = Color.White, modifier = Modifier.size(18.dp))
                                Spacer(Modifier.width(6.dp))
                                Text(if (following) "FAVOURITE" else "ADD TO FAVOURITES", color = Color.White, fontSize = 12.sp, fontWeight = FontWeight.Bold)
                            }
                        }
                    }
                }
            }
            fullWidth("facts") {
                val facts = listOfNotNull(
                    m.birthdate?.let { b -> runCatching { java.time.Period.between(java.time.LocalDate.parse(b), java.time.LocalDate.now()).years }.getOrNull()?.let { "Age" to "$it" } },
                    m.country?.let { "From" to it },
                    m.ethnicity?.let { "Ethnicity" to it },
                    m.hairColor?.let { "Hair" to it },
                    m.eyeColor?.let { "Eyes" to it },
                    m.heightCm?.let { "Height" to "$it cm" },
                    m.measurements?.let { "Measurements" to it },
                    m.careerStart?.let { "Career" to "$it – ${m.careerEnd ?: "now"}" },
                    m.tattoos?.let { "Tattoos" to it },
                ).filter { it.second.isNotBlank() }
                Column(Modifier.padding(top = 8.dp)) {
                    if (m.aliases.isNotEmpty()) Text("Also known as ${m.aliases.take(5).joinToString(", ")}", color = Brand.TextMuted, style = MaterialTheme.typography.bodySmall)
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.padding(top = 8.dp)) {
                        facts.forEach { (k, v) ->
                            Column(Modifier.clip(RoundedCornerShape(10.dp)).background(Brand.Surface).padding(horizontal = 12.dp, vertical = 8.dp)) {
                                Text(k.uppercase(), color = Brand.TextMuted, fontSize = 10.sp, fontWeight = FontWeight.Bold, letterSpacing = 1.sp)
                                Text(v, style = MaterialTheme.typography.bodyMedium)
                            }
                        }
                    }
                    m.details?.takeIf { it.isNotBlank() }?.let {
                        Spacer(Modifier.height(10.dp))
                        Text(it, color = Brand.TextMuted, style = MaterialTheme.typography.bodyMedium, maxLines = 6, overflow = TextOverflow.Ellipsis)
                    }
                }
            }
            fullWidth("popular") {
                val p = popular.value.orEmpty()
                if (p.size > 1) Column(Modifier.padding(horizontal = 0.dp)) {
                    StoreHeader("Most popular scenes")
                    LazyRow(horizontalArrangement = Arrangement.spacedBy(12.dp)) { items(p, key = { it.id }) { s -> StoreSceneCard(s, width = 240.dp) } }
                }
            }
            fullWidth("costars") {
                val c = costars.value.orEmpty()
                if (c.isNotEmpty()) Column {
                    StoreHeader("Frequently seen with")
                    LazyRow(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        items(c, key = { it.performer.id }) { x -> ModelCard(x.performer, width = 120.dp, subtitle = "${x.scenes} together") }
                    }
                }
            }
            fullWidth("all") {
                Column {
                    StoreHeader("All scenes with ${m.name}")
                    ChipRow { storeSorts.filter { it.kind == null }.forEach { s -> Chip(s.label, selected = s.id == sortId) { sortId = s.id } } }
                }
            }
        }) { s -> StoreSceneCard(s, width = null) }
    }
}

// ---------------------------------------------------------------- my list

private data class MyListData(
    val cont: List<Scene>,
    val favModels: List<Performer>,
    val fromFavs: List<Scene>,
    val later: List<Scene>,
    val liked: List<Scene>,
)

@Composable
fun StoreMyList() {
    val loader = rememberLoad(Unit) {
        coroutineScope {
            val repo = app.repo
            val cont = async { runCatching { repo.continueWatching(20) }.getOrDefault(emptyList()) }
            val favs = async { runCatching { repo.favoriteModels() }.getOrDefault(emptyList()) }
            val from = async { runCatching { repo.fromFavorites(16) }.getOrDefault(emptyList()) }
            val later = async { runCatching { repo.watchLaterScenes() }.getOrDefault(emptyList()) }
            val liked = async { runCatching { repo.likedScenes(1, 40).items }.getOrDefault(emptyList()) }
            MyListData(cont.await(), favs.await(), from.await(), later.await(), liked.await())
        }
    }
    Loaded(loader) { d ->
        LazyColumn(Modifier.fillMaxSize()) {
            item { StorePageTitle("My list") }
            item { SceneRow("Continue watching", d.cont) }
            item {
                if (d.favModels.isEmpty()) {
                    StoreHeader("Favourite ${starsWord.lowercase()}")
                    Text(
                        "Tap “Add to favourites” on a ${if (femaleOnly) "pornstar" else "model"}'s page to follow her.",
                        color = Brand.TextMuted,
                        modifier = Modifier.padding(horizontal = 16.dp),
                    )
                } else {
                    ModelRow("Favourite ${starsWord.lowercase()}", d.favModels)
                }
            }
            item { SceneRow("New from your favourites", d.fromFavs) }
            item { SceneRow("Saved for later", d.later) }
            item {
                if (d.liked.isEmpty()) {
                    StoreHeader("Liked scenes")
                    Text("Tap the heart on a scene to like it.", color = Brand.TextMuted, modifier = Modifier.padding(horizontal = 16.dp))
                } else {
                    SceneRow("Liked scenes", d.liked)
                }
            }
            item { Spacer(Modifier.height(32.dp)) }
        }
    }
}

// ---------------------------------------------------------------- age check

@Composable
fun AgeGate(onEnter: () -> Unit) {
    val context = LocalContext.current
    val look = SiteMode.look
    Box(Modifier.fillMaxSize().background(Brand.Background), contentAlignment = Alignment.Center) {
        Column(
            Modifier.padding(28.dp).clip(RoundedCornerShape(18.dp)).background(Brand.Surface).padding(26.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            if (look?.logoUrl != null) {
                AsyncImage(look.logoUrl, look.name, contentScale = ContentScale.Fit, modifier = Modifier.height(48.dp).width(200.dp))
            } else {
                StoreLogo(look?.name?.ifBlank { "Stasharr" } ?: "Stasharr", size = 24)
            }
            Spacer(Modifier.height(16.dp))
            Text("This is an adult app", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
            Spacer(Modifier.height(8.dp))
            Text(
                "It contains sexually explicit material. Only continue if you are at least 18 years old, or the age of majority where you live, and you want to see it.",
                color = Brand.TextMuted,
                style = MaterialTheme.typography.bodyMedium,
            )
            Spacer(Modifier.height(20.dp))
            GradientButton("I am 18 or older — Enter", onClick = onEnter, modifier = Modifier.fillMaxWidth())
            Spacer(Modifier.height(8.dp))
            Text(
                "Leave",
                color = Brand.TextMuted,
                style = MaterialTheme.typography.labelLarge,
                modifier = Modifier.clickable {
                    (context as? android.app.Activity)?.finishAffinity()
                }.padding(10.dp),
            )
        }
    }
}
