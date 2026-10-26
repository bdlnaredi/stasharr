package app.stasharr.ui.screens

import androidx.compose.animation.animateColorAsState
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Info
import androidx.compose.material.icons.rounded.PlayArrow
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.stasharr.app
import app.stasharr.data.Performer
import app.stasharr.data.Scene
import app.stasharr.data.SceneScope
import app.stasharr.data.SceneSort
import app.stasharr.data.Studio
import app.stasharr.player.ScenePreview
import app.stasharr.ui.LocalNav
import app.stasharr.ui.components.Badge
import app.stasharr.ui.components.GlassButton
import app.stasharr.ui.components.GradientButton
import app.stasharr.ui.components.GradientText
import app.stasharr.ui.components.Loaded
import app.stasharr.ui.components.PerformerBubble
import app.stasharr.ui.components.SceneCard
import app.stasharr.ui.components.Shelf
import app.stasharr.ui.components.StudioTile
import app.stasharr.ui.components.metaLine
import app.stasharr.ui.components.rememberLoad
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlin.math.absoluteValue

private data class HomeData(
    val hero: List<Scene>,
    val continueWatching: List<Scene>,
    val recommended: List<Scene>,
    val recent: List<Scene>,
    val watchLater: List<Scene>,
    val topRated: List<Scene>,
    val studios: List<Studio>,
    val performers: List<Performer>,
    val audio: List<Scene>,
)

private suspend fun loadHome(): HomeData = coroutineScope {
    val repo = app.repo
    // each shelf loads on its own; one failing does not empty the page
    suspend fun <T> safe(block: suspend () -> List<T>): List<T> = runCatching { block() }.getOrDefault(emptyList())

    val cont = async { safe { repo.continueWatching(20) } }
    val rec = async { safe { repo.recommended(20) } }
    val recent = async { repo.scenes(SceneSort.RecentlyAdded, perPage = 20).items }
    val wl = async { safe { repo.watchLaterScenes() } }
    val top = async { safe { repo.scenes(SceneSort.TopRated, perPage = 20).items.filter { (it.rating100 ?: 0) > 0 } } }
    val studios = async { safe { repo.studios(perPage = 16).items.filter { it.sceneCount > 0 } } }
    val perf = async { safe { repo.performers(perPage = 20).items.filter { it.sceneCount > 0 } } }
    val audio = async { safe { repo.scenes(SceneSort.RecentlyAdded, perPage = 12, scope = SceneScope.Audio).items } }

    val recentList = recent.await()
    val recList = rec.await()
    val hero = (recList.take(4) + recentList.take(6))
        .distinctBy { it.id }
        .filter { it.paths.screenshot != null }
        .take(6)

    HomeData(
        hero = hero,
        continueWatching = cont.await(),
        recommended = recList,
        recent = recentList,
        watchLater = wl.await(),
        topRated = top.await(),
        studios = studios.await(),
        performers = perf.await(),
        audio = audio.await(),
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen() {
    val loader = rememberLoad(Unit) { loadHome() }
    val nav = LocalNav.current
    val listState = rememberLazyListState()
    val scrolled by remember { derivedStateOf { listState.firstVisibleItemIndex > 0 || listState.firstVisibleItemScrollOffset > 200 } }

    Box(Modifier.fillMaxSize()) {
        Loaded(loader) { data ->
            PullToRefreshBox(
                isRefreshing = loader.refreshing,
                onRefresh = loader.reload,
            ) {
                LazyColumn(state = listState, modifier = Modifier.fillMaxSize()) {
                    item(key = "hero") {
                        if (data.hero.isNotEmpty()) {
                            Hero(data.hero)
                        } else {
                            Spacer(Modifier.statusBarsPadding().height(72.dp))
                        }
                    }
                    item(key = "continue") {
                        Shelf("Continue watching", data.continueWatching, key = { it.id }, onSeeAll = { nav.list(SceneListKind.Continue) }) {
                            SceneCard(it, onClick = { nav.play(it.id, it.resumeTime ?: 0.0) }, width = 260.dp)
                        }
                    }
                    item(key = "recommended") {
                        Shelf("Recommended for you", data.recommended, key = { it.id }, onSeeAll = { nav.list(SceneListKind.Recommended) }) {
                            SceneCard(it, onClick = { nav.scene(it.id) })
                        }
                    }
                    item(key = "recent") {
                        Shelf("Recently added", data.recent, key = { it.id }, onSeeAll = { nav.list(SceneListKind.Recent) }) {
                            SceneCard(it, onClick = { nav.scene(it.id) })
                        }
                    }
                    item(key = "watchlater") {
                        Shelf("Watch later", data.watchLater, key = { it.id }, onSeeAll = { nav.list(SceneListKind.WatchLater) }) {
                            SceneCard(it, onClick = { nav.scene(it.id) })
                        }
                    }
                    item(key = "performers") {
                        Shelf("Popular performers", data.performers, key = { it.id }) {
                            PerformerBubble(it, onClick = { nav.performer(it.id) })
                        }
                    }
                    item(key = "toprated") {
                        Shelf("Top rated", data.topRated, key = { it.id }, onSeeAll = { nav.list(SceneListKind.TopRated) }) {
                            SceneCard(it, onClick = { nav.scene(it.id) })
                        }
                    }
                    item(key = "studios") {
                        Shelf("Studios", data.studios, key = { it.id }) {
                            StudioTile(it, onClick = { nav.studio(it.id) })
                        }
                    }
                    item(key = "audio") {
                        Shelf("Audio", data.audio, key = { it.id }, onSeeAll = { nav.list(SceneListKind.Audio) }) {
                            SceneCard(it, onClick = { nav.scene(it.id) }, width = 200.dp)
                        }
                    }
                    item(key = "end") { Spacer(Modifier.height(32.dp)) }
                }
            }
        }
        TopBar(solid = scrolled, onSearch = { nav.tab("browse") })
    }
}

@Composable
private fun TopBar(solid: Boolean, onSearch: () -> Unit) {
    val bg by animateColorAsState(if (solid) Brand.Background.copy(alpha = 0.94f) else Color.Transparent, label = "bar")
    Row(
        Modifier
            .fillMaxWidth()
            .background(
                if (solid) Brush.verticalGradient(listOf(bg, bg))
                else Brush.verticalGradient(listOf(Color.Black.copy(alpha = 0.55f), Color.Transparent))
            )
            .statusBarsPadding()
            .padding(start = 18.dp, end = 6.dp, top = 6.dp, bottom = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        GradientText("stasharr", fontSize = 24.sp, modifier = Modifier.weight(1f))
        IconButton(onClick = onSearch) {
            Icon(Icons.Rounded.Search, "Search", tint = Color.White)
        }
    }
}

/** The big swipeable banner at the top, like Plex and Netflix. */
@Composable
private fun Hero(scenes: List<Scene>) {
    val nav = LocalNav.current
    val pager = rememberPagerState(pageCount = { scenes.size })

    // move on every few seconds
    LaunchedEffect(pager, scenes.size) {
        while (scenes.size > 1) {
            delay(6500)
            if (!pager.isScrollInProgress) {
                pager.animateScrollToPage((pager.currentPage + 1) % scenes.size)
            }
        }
    }

    Box(Modifier.fillMaxWidth().height(520.dp)) {
        HorizontalPager(state = pager, modifier = Modifier.fillMaxSize()) { page ->
            val s = scenes[page]
            val offset = (pager.currentPage - page + pager.currentPageOffsetFraction).absoluteValue
            Box(
                Modifier
                    .fillMaxSize()
                    .graphicsLayer { alpha = 1f - (offset * 0.6f).coerceIn(0f, 0.6f) }
            ) {
                AsyncImage(
                    model = s.paths.screenshot,
                    contentDescription = null,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier
                        .fillMaxSize()
                        .graphicsLayer {
                            // gentle zoom as the page settles
                            val scale = 1.08f - (0.08f * (1f - offset.coerceIn(0f, 1f)))
                            scaleX = scale
                            scaleY = scale
                        },
                )
                ScenePreview(
                    s.paths.preview,
                    active = page == pager.settledPage && !pager.isScrollInProgress,
                    modifier = Modifier.fillMaxSize(),
                    delayMs = 1200,
                )
                Box(
                    Modifier
                        .fillMaxSize()
                        .background(
                            Brush.verticalGradient(
                                0f to Color.Black.copy(alpha = 0.35f),
                                0.35f to Color.Transparent,
                                0.62f to Brand.Background.copy(alpha = 0.55f),
                                1f to Brand.Background,
                            )
                        )
                )
                Column(
                    Modifier
                        .align(Alignment.BottomStart)
                        .fillMaxWidth()
                        .padding(horizontal = 20.dp, vertical = 30.dp)
                ) {
                    Badge(if (page == 0) "FEATURED" else "FOR YOU", gradient = true)
                    Spacer(Modifier.height(10.dp))
                    Text(
                        s.displayTitle,
                        style = MaterialTheme.typography.headlineLarge,
                        fontSize = 30.sp,
                        lineHeight = 34.sp,
                        fontWeight = FontWeight.Black,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                        color = Color.White,
                    )
                    Spacer(Modifier.height(6.dp))
                    Text(s.metaLine(), color = Color.White.copy(alpha = 0.8f), style = MaterialTheme.typography.bodyMedium)
                    if (s.performers.isNotEmpty()) {
                        Text(
                            s.performers.joinToString(", ") { it.name },
                            color = Color.White.copy(alpha = 0.65f),
                            style = MaterialTheme.typography.bodySmall,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                    Spacer(Modifier.height(16.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                        GradientButton(
                            if ((s.resumeTime ?: 0.0) > 5) "Resume" else "Play",
                            onClick = { nav.play(s.id, s.resumeTime ?: 0.0) },
                            icon = Icons.Rounded.PlayArrow,
                        )
                        GlassButton("More info", onClick = { nav.scene(s.id) }, icon = Icons.Rounded.Info)
                    }
                }
            }
        }
        // page dots
        Row(
            Modifier.align(Alignment.BottomCenter).padding(bottom = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            repeat(scenes.size) { i ->
                val selected = pager.currentPage == i
                val w by animateDpAsState(if (selected) 18.dp else 6.dp, label = "dot")
                Box(
                    Modifier
                        .height(6.dp)
                        .width(w)
                        .clip(CircleShape)
                        .background(if (selected) Brand.GradientHorizontal else Brush.linearGradient(listOf(Color.White.copy(alpha = 0.35f), Color.White.copy(alpha = 0.35f))))
                )
            }
        }
    }
}
