package app.stasharr.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.Sort
import androidx.compose.material.icons.rounded.ArrowDownward
import androidx.compose.material.icons.rounded.ArrowUpward
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.Shuffle
import androidx.compose.material.icons.rounded.Tune
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewmodel.compose.viewModel
import app.stasharr.app
import app.stasharr.data.AddedFilter
import app.stasharr.data.Gender
import app.stasharr.data.LengthFilter
import app.stasharr.data.OrientationFilter
import app.stasharr.data.PerformerFilters
import app.stasharr.data.ResolutionFilter
import app.stasharr.data.SceneFilters
import app.stasharr.data.SceneScope
import app.stasharr.data.SortOption
import app.stasharr.data.SortState
import app.stasharr.data.Sorts
import app.stasharr.data.StudioFilters
import app.stasharr.data.TagFilters
import app.stasharr.data.TagRef
import app.stasharr.data.WatchFilter
import app.stasharr.data.mergeFilters
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
import app.stasharr.ui.components.rememberLoad
import app.stasharr.ui.components.rememberPaged
import app.stasharr.ui.theme.Brand
import kotlinx.coroutines.delay
import kotlin.random.Random

enum class BrowseTab(val label: String) {
    Scenes("Scenes"), Performers("Performers"), Studios("Studios"), Tags("Tags"), Audio("Audio"),
}

private fun newSeed() = Random.nextInt(1, 99_999_999)

/** What Browse shows; kept while you move between tabs of the app. */
class BrowseModel : ViewModel() {
    var tab by mutableStateOf(BrowseTab.Scenes)
    var query by mutableStateOf("")

    var sceneSort by mutableStateOf(SortState(Sorts.scenes.first()))
    var sceneFilters by mutableStateOf(SceneFilters())
    var audioSort by mutableStateOf(SortState(Sorts.scenes.first()))
    var audioFilters by mutableStateOf(SceneFilters())
    var performerSort by mutableStateOf(SortState(Sorts.performers.first()))
    var performerFilters by mutableStateOf(PerformerFilters())
    var studioSort by mutableStateOf(SortState(Sorts.studios.first()))
    var studioFilters by mutableStateOf(StudioFilters())
    var tagSort by mutableStateOf(SortState(Sorts.tags.first()))
    var tagFilters by mutableStateOf(TagFilters())

    val sort: SortState
        get() = when (tab) {
            BrowseTab.Scenes -> sceneSort
            BrowseTab.Audio -> audioSort
            BrowseTab.Performers -> performerSort
            BrowseTab.Studios -> studioSort
            BrowseTab.Tags -> tagSort
        }

    fun setSort(s: SortState) {
        when (tab) {
            BrowseTab.Scenes -> sceneSort = s
            BrowseTab.Audio -> audioSort = s
            BrowseTab.Performers -> performerSort = s
            BrowseTab.Studios -> studioSort = s
            BrowseTab.Tags -> tagSort = s
        }
    }

    val sortOptions: List<SortOption>
        get() = when (tab) {
            BrowseTab.Scenes, BrowseTab.Audio -> Sorts.scenes
            BrowseTab.Performers -> Sorts.performers
            BrowseTab.Studios -> Sorts.studios
            BrowseTab.Tags -> Sorts.tags
        }

    val filterCount: Int
        get() = when (tab) {
            BrowseTab.Scenes -> sceneFilters.count
            BrowseTab.Audio -> audioFilters.count
            BrowseTab.Performers -> performerFilters.count
            BrowseTab.Studios -> studioFilters.count
            BrowseTab.Tags -> tagFilters.count
        }

    fun clearFilters() {
        when (tab) {
            BrowseTab.Scenes -> sceneFilters = SceneFilters()
            BrowseTab.Audio -> audioFilters = SceneFilters()
            BrowseTab.Performers -> performerFilters = PerformerFilters()
            BrowseTab.Studios -> studioFilters = StudioFilters()
            BrowseTab.Tags -> tagFilters = TagFilters()
        }
    }
}

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

/** A pill button with an icon, used for the sort and filter buttons. */
@Composable
private fun ToolPill(text: String, icon: androidx.compose.ui.graphics.vector.ImageVector, highlighted: Boolean = false, onClick: () -> Unit) {
    Row(
        Modifier
            .clip(RoundedCornerShape(50))
            .then(
                if (highlighted) Modifier.background(Brand.GradientHorizontal)
                else Modifier.background(Brand.Surface)
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, null, tint = Color.White, modifier = Modifier.size(18.dp))
        Spacer(Modifier.width(6.dp))
        Text(text, style = MaterialTheme.typography.labelLarge, color = Color.White, maxLines = 1)
    }
}

/** A chip for an active filter; tapping it removes the filter. */
@Composable
private fun ActiveChip(text: String, onRemove: () -> Unit) {
    Row(
        Modifier
            .clip(RoundedCornerShape(50))
            .background(Brand.Pink.copy(alpha = 0.18f))
            .clickable(onClick = onRemove)
            .padding(start = 12.dp, end = 8.dp, top = 6.dp, bottom = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(text, style = MaterialTheme.typography.labelLarge, color = Brand.Text)
        Spacer(Modifier.width(4.dp))
        Icon(Icons.Rounded.Close, "Remove", tint = Brand.TextMuted, modifier = Modifier.size(16.dp))
    }
}

private fun stars(n: Int) = "★ $n+"

/** The removable chips for whatever filters are on. */
private fun activeChips(m: BrowseModel): List<Pair<String, () -> Unit>> {
    fun scene(f: SceneFilters, set: (SceneFilters) -> Unit) = buildList<Pair<String, () -> Unit>> {
        if (f.minRating > 0) add(stars(f.minRating) to { set(f.copy(minRating = 0)) })
        if (f.resolution != ResolutionFilter.Any) add(f.resolution.label to { set(f.copy(resolution = ResolutionFilter.Any)) })
        if (f.length != LengthFilter.Any) add(f.length.label to { set(f.copy(length = LengthFilter.Any)) })
        if (f.watched != WatchFilter.Any) add(f.watched.label to { set(f.copy(watched = WatchFilter.Any)) })
        if (f.orientation != OrientationFilter.Any) add(f.orientation.label to { set(f.copy(orientation = OrientationFilter.Any)) })
        if (f.added != AddedFilter.Any) add("Added ${f.added.label.lowercase()}" to { set(f.copy(added = AddedFilter.Any)) })
        if (f.organized) add("Organized" to { set(f.copy(organized = false)) })
        if (f.favoritePerformers) add("Favorite performers" to { set(f.copy(favoritePerformers = false)) })
        if (f.hasMarkers) add("Has chapters" to { set(f.copy(hasMarkers = false)) })
        if (f.interactive) add("Interactive" to { set(f.copy(interactive = false)) })
        f.tags.forEach { t -> add("#${t.name}" to { set(f.copy(tags = f.tags - t)) }) }
    }
    return when (m.tab) {
        BrowseTab.Scenes -> scene(m.sceneFilters) { m.sceneFilters = it }
        BrowseTab.Audio -> scene(m.audioFilters) { m.audioFilters = it }
        BrowseTab.Performers -> buildList<Pair<String, () -> Unit>> {
            val f = m.performerFilters
            f.genders.forEach { g -> add(g.label to { m.performerFilters = f.copy(genders = f.genders - g) }) }
            if (f.favorites) add("Favorites" to { m.performerFilters = f.copy(favorites = false) })
            if (f.minRating > 0) add(stars(f.minRating) to { m.performerFilters = f.copy(minRating = 0) })
            if (f.minScenes > 0) add("${f.minScenes}+ scenes" to { m.performerFilters = f.copy(minScenes = 0) })
        }
        BrowseTab.Studios -> buildList<Pair<String, () -> Unit>> {
            val f = m.studioFilters
            if (f.favorites) add("Favorites" to { m.studioFilters = f.copy(favorites = false) })
            if (f.minRating > 0) add(stars(f.minRating) to { m.studioFilters = f.copy(minRating = 0) })
            if (f.minScenes > 0) add("${f.minScenes}+ scenes" to { m.studioFilters = f.copy(minScenes = 0) })
        }
        BrowseTab.Tags -> buildList<Pair<String, () -> Unit>> {
            val f = m.tagFilters
            if (f.favorites) add("Favorites" to { m.tagFilters = f.copy(favorites = false) })
            if (f.minScenes > 0) add("${f.minScenes}+ scenes" to { m.tagFilters = f.copy(minScenes = 0) })
        }
    }
}

@Composable
fun BrowseScreen() {
    val m: BrowseModel = viewModel()
    val nav = LocalNav.current
    var q by remember { mutableStateOf(m.query.trim()) }
    var showSort by remember { mutableStateOf(false) }
    var showFilters by remember { mutableStateOf(false) }

    // search as you type, after a short pause
    LaunchedEffect(m.query) {
        delay(350)
        q = m.query.trim()
    }

    Column(Modifier.fillMaxSize().statusBarsPadding()) {
        Spacer(Modifier.height(8.dp))
        GradientText("Browse", modifier = Modifier.padding(horizontal = 18.dp, vertical = 6.dp))
        SearchField(m.query, { m.query = it }, "Search ${m.tab.label.lowercase()}")
        ChipRow {
            BrowseTab.entries.forEach { t -> Chip(t.label, selected = t == m.tab) { m.tab = t } }
        }

        // sort and filter
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            ToolPill(m.sort.option.label, Icons.AutoMirrored.Rounded.Sort) { showSort = true }
            if (m.sort.option.key == "random") {
                ToolPill("Shuffle", Icons.Rounded.Shuffle) { m.setSort(m.sort.copy(seed = newSeed())) }
            } else {
                ToolPill(
                    if (m.sort.desc) "Desc" else "Asc",
                    if (m.sort.desc) Icons.Rounded.ArrowDownward else Icons.Rounded.ArrowUpward,
                ) { m.setSort(m.sort.copy(desc = !m.sort.desc)) }
            }
            Spacer(Modifier.weight(1f))
            ToolPill(
                if (m.filterCount > 0) "Filters · ${m.filterCount}" else "Filters",
                Icons.Rounded.Tune,
                highlighted = m.filterCount > 0,
            ) { showFilters = true }
        }
        val chips = activeChips(m)
        if (chips.isNotEmpty()) {
            ChipRow {
                chips.forEach { (label, remove) -> ActiveChip(label, remove) }
                Text(
                    "Clear all",
                    color = Brand.Orange,
                    style = MaterialTheme.typography.labelLarge,
                    modifier = Modifier
                        .align(Alignment.CenterVertically)
                        .clip(RoundedCornerShape(50))
                        .clickable { m.clearFilters() }
                        .padding(horizontal = 8.dp, vertical = 6.dp),
                )
            }
        }

        when (m.tab) {
            BrowseTab.Scenes, BrowseTab.Audio -> {
                val audio = m.tab == BrowseTab.Audio
                val sort = if (audio) m.audioSort else m.sceneSort
                val filters = if (audio) m.audioFilters else m.sceneFilters
                val json = mergeFilters(if (audio) SceneScope.Audio.toFilter() else null, filters.toJson())
                SceneGrid(keys = arrayOf(m.tab, sort, filters, q), fetch = { page ->
                    app.repo.findScenes(sort, page, 30, q, json)
                })
            }
            BrowseTab.Performers -> {
                val sort = m.performerSort
                val f = m.performerFilters
                val state = rememberPaged(q, sort, f) { page -> app.repo.findPerformers(sort, page, 40, q, f.toJson()) }
                PagedGrid(state, key = { it.id }, minCell = 104.dp, empty = "No performers match") { p ->
                    PerformerBubble(p, onClick = { nav.performer(p.id) }, size = 92.dp)
                }
            }
            BrowseTab.Studios -> {
                val sort = m.studioSort
                val f = m.studioFilters
                val state = rememberPaged(q, sort, f) { page -> app.repo.findStudios(sort, page, 40, q, f.toJson()) }
                PagedGrid(state, key = { it.id }, minCell = 160.dp, empty = "No studios match") { s ->
                    StudioTile(s, onClick = { nav.studio(s.id) }, width = null)
                }
            }
            BrowseTab.Tags -> {
                val sort = m.tagSort
                val f = m.tagFilters
                val state = rememberPaged(q, sort, f) { page -> app.repo.findTags(sort, page, 80, q, f.toJson()) }
                PagedGrid(state, key = { it.id }, minCell = 150.dp, empty = "No tags match") { t ->
                    TagTile(t.name, t.sceneCount) { nav.tag(t.id, t.name) }
                }
            }
        }
    }

    if (showSort) {
        SortSheet(
            options = m.sortOptions,
            current = m.sort,
            onPick = { m.setSort(it) },
            onDismiss = { showSort = false },
        )
    }
    if (showFilters) {
        FilterSheet(m, onDismiss = { showFilters = false })
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

// ---------------------------------------------------------------- sheets

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SortSheet(options: List<SortOption>, current: SortState, onPick: (SortState) -> Unit, onDismiss: () -> Unit) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        containerColor = Brand.Surface,
    ) {
        Column(Modifier.navigationBarsPadding().padding(bottom = 12.dp)) {
            Text("Sort by", style = MaterialTheme.typography.titleLarge, modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp))
            options.forEach { o ->
                val selected = o.key == current.option.key
                Row(
                    Modifier
                        .fillMaxWidth()
                        .clickable {
                            onPick(
                                if (selected) current
                                else SortState(o, o.defaultDesc, if (o.key == "random") newSeed() else current.seed)
                            )
                            onDismiss()
                        }
                        .padding(horizontal = 20.dp, vertical = 14.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        o.label,
                        style = MaterialTheme.typography.bodyLarge,
                        color = if (selected) Brand.Pink else Brand.Text,
                        modifier = Modifier.weight(1f),
                    )
                    if (selected) Icon(Icons.Rounded.Check, null, tint = Brand.Pink)
                }
            }
            if (current.option.key != "random") {
                Row(
                    Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 10.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Chip("Descending", selected = current.desc) { onPick(current.copy(desc = true)) }
                    Chip("Ascending", selected = !current.desc) { onPick(current.copy(desc = false)) }
                }
            }
        }
    }
}

@Composable
private fun Section(title: String, content: @Composable () -> Unit) {
    Text(
        title,
        style = MaterialTheme.typography.titleSmall,
        color = Brand.TextMuted,
        modifier = Modifier.padding(start = 20.dp, end = 20.dp, top = 16.dp, bottom = 8.dp),
    )
    content()
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun <T> Choices(options: List<T>, selected: (T) -> Boolean, label: (T) -> String, onClick: (T) -> Unit) {
    FlowRow(
        Modifier.padding(horizontal = 20.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        options.forEach { o -> Chip(label(o), selected = selected(o)) { onClick(o) } }
    }
}

@Composable
private fun StarChoices(value: Int, onChange: (Int) -> Unit) {
    Choices((0..5).toList(), { it == value }, { if (it == 0) "Any" else "★ $it+" }) { onChange(it) }
}

@Composable
private fun SceneCountChoices(value: Int, onChange: (Int) -> Unit) {
    Choices(listOf(0, 1, 5, 10, 25, 50), { it == value }, { if (it == 0) "Any" else "$it+" }) { onChange(it) }
}

@Composable
private fun Toggle(title: String, checked: Boolean, onChange: (Boolean) -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .clickable { onChange(!checked) }
            .padding(horizontal = 20.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(title, style = MaterialTheme.typography.bodyLarge, modifier = Modifier.weight(1f))
        Switch(
            checked = checked,
            onCheckedChange = onChange,
            colors = SwitchDefaults.colors(checkedTrackColor = Brand.Pink, checkedThumbColor = Color.White),
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun FilterSheet(m: BrowseModel, onDismiss: () -> Unit) {
    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        containerColor = Brand.Surface,
    ) {
        Column(
            Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .navigationBarsPadding()
        ) {
            Row(Modifier.fillMaxWidth().padding(horizontal = 20.dp), verticalAlignment = Alignment.CenterVertically) {
                Text("Filter ${m.tab.label.lowercase()}", style = MaterialTheme.typography.titleLarge, modifier = Modifier.weight(1f))
                if (m.filterCount > 0) {
                    Text(
                        "Reset",
                        color = Brand.Orange,
                        style = MaterialTheme.typography.labelLarge,
                        modifier = Modifier.clip(RoundedCornerShape(50)).clickable { m.clearFilters() }.padding(8.dp),
                    )
                }
            }

            when (m.tab) {
                BrowseTab.Scenes -> SceneFilterContent(m.sceneFilters) { m.sceneFilters = it }
                BrowseTab.Audio -> SceneFilterContent(m.audioFilters, audio = true) { m.audioFilters = it }
                BrowseTab.Performers -> {
                    val f = m.performerFilters
                    Section("Gender") {
                        Choices(Gender.entries, { it in f.genders }, { it.label }) { g ->
                            m.performerFilters = f.copy(genders = if (g in f.genders) f.genders - g else f.genders + g)
                        }
                    }
                    Section("Rating") { StarChoices(f.minRating) { m.performerFilters = f.copy(minRating = it) } }
                    Section("Scenes") { SceneCountChoices(f.minScenes) { m.performerFilters = f.copy(minScenes = it) } }
                    Spacer(Modifier.height(8.dp))
                    Toggle("Favorites only", f.favorites) { m.performerFilters = f.copy(favorites = it) }
                }
                BrowseTab.Studios -> {
                    val f = m.studioFilters
                    Section("Rating") { StarChoices(f.minRating) { m.studioFilters = f.copy(minRating = it) } }
                    Section("Scenes") { SceneCountChoices(f.minScenes) { m.studioFilters = f.copy(minScenes = it) } }
                    Spacer(Modifier.height(8.dp))
                    Toggle("Favorites only", f.favorites) { m.studioFilters = f.copy(favorites = it) }
                }
                BrowseTab.Tags -> {
                    val f = m.tagFilters
                    Section("Scenes") { SceneCountChoices(f.minScenes) { m.tagFilters = f.copy(minScenes = it) } }
                    Spacer(Modifier.height(8.dp))
                    Toggle("Favorites only", f.favorites) { m.tagFilters = f.copy(favorites = it) }
                }
            }

            Spacer(Modifier.height(20.dp))
            Row(Modifier.fillMaxWidth().padding(horizontal = 20.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                GlassButton("Clear", onClick = { m.clearFilters() })
                GradientButton("Show results", onClick = onDismiss, modifier = Modifier.weight(1f))
            }
            Spacer(Modifier.height(16.dp))
        }
    }
}

@Composable
private fun SceneFilterContent(f: SceneFilters, audio: Boolean = false, set: (SceneFilters) -> Unit) {
    Section("Rating") { StarChoices(f.minRating) { set(f.copy(minRating = it)) } }
    Section("Watched") { Choices(WatchFilter.entries, { it == f.watched }, { it.label }) { set(f.copy(watched = it)) } }
    Section("Length") { Choices(LengthFilter.entries, { it == f.length }, { it.label }) { set(f.copy(length = it)) } }
    Section("Added") { Choices(AddedFilter.entries, { it == f.added }, { it.label }) { set(f.copy(added = it)) } }
    if (!audio) {
        Section("Quality") { Choices(ResolutionFilter.entries, { it == f.resolution }, { it.label }) { set(f.copy(resolution = it)) } }
        Section("Shape") { Choices(OrientationFilter.entries, { it == f.orientation }, { it.label }) { set(f.copy(orientation = it)) } }
    }
    Section("Tags") { TagPicker(f.tags) { set(f.copy(tags = it)) } }
    Spacer(Modifier.height(8.dp))
    Toggle("Favorite performers", f.favoritePerformers) { set(f.copy(favoritePerformers = it)) }
    if (!audio) Toggle("Has chapters", f.hasMarkers) { set(f.copy(hasMarkers = it)) }
    Toggle("Organized", f.organized) { set(f.copy(organized = it)) }
    if (!audio) Toggle("Interactive", f.interactive) { set(f.copy(interactive = it)) }
}

/** Search tags and pick several; scenes must have all of them. */
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun TagPicker(selected: List<TagRef>, onChange: (List<TagRef>) -> Unit) {
    var search by remember { mutableStateOf("") }
    var q by remember { mutableStateOf("") }
    LaunchedEffect(search) {
        delay(300)
        q = search.trim()
    }
    val results = rememberLoad(q) {
        app.repo.findTags(SortState(Sorts.tags.first()), 1, 24, q.ifBlank { null }, null).items
    }

    Column {
        if (selected.isNotEmpty()) {
            FlowRow(
                Modifier.padding(horizontal = 20.dp).padding(bottom = 8.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                selected.forEach { t -> ActiveChip("#${t.name}") { onChange(selected - t) } }
            }
        }
        TextField(
            value = search,
            onValueChange = { search = it },
            singleLine = true,
            placeholder = { Text("Find a tag") },
            leadingIcon = { Icon(Icons.Rounded.Search, null) },
            shape = RoundedCornerShape(50),
            colors = TextFieldDefaults.colors(
                focusedContainerColor = Brand.SurfaceHigh,
                unfocusedContainerColor = Brand.SurfaceHigh,
                focusedIndicatorColor = Color.Transparent,
                unfocusedIndicatorColor = Color.Transparent,
                cursorColor = Brand.Pink,
            ),
            modifier = Modifier.fillMaxWidth().padding(horizontal = 20.dp),
        )
        Spacer(Modifier.height(8.dp))
        val ids = selected.map { it.id }.toSet()
        FlowRow(
            Modifier.padding(horizontal = 20.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            results.value.orEmpty().filter { it.id !in ids }.forEach { t ->
                Chip(t.name) { onChange(selected + TagRef(t.id, t.name)) }
            }
        }
    }
}
