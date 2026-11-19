package app.stasharr.ui.screens

// The tops of the home screen for each storefront layout, the same ones as
// on the website. Classic is the banner in HomeScreen.

import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.ExperimentalLayoutApi
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.Favorite
import androidx.compose.material.icons.rounded.Info
import androidx.compose.material.icons.rounded.PlayArrow
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.stasharr.app
import app.stasharr.data.Scene
import app.stasharr.data.Tag
import app.stasharr.player.ScenePreview
import app.stasharr.ui.LocalNav
import app.stasharr.ui.SiteMode
import app.stasharr.ui.components.Badge
import app.stasharr.ui.components.GlassButton
import app.stasharr.ui.components.GradientButton
import app.stasharr.ui.components.SceneThumb
import app.stasharr.ui.components.SpotRow
import app.stasharr.ui.components.formatDuration
import app.stasharr.ui.components.hardShadow
import app.stasharr.ui.components.metaLine
import app.stasharr.ui.components.portraitFilter
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import kotlin.math.absoluteValue
import kotlin.math.roundToInt

private fun starring(s: Scene, max: Int = 2): String = s.performers.take(max).joinToString(" & ") { it.name }

/** A scene's picture that plays its preview while shown. */
@Composable
private fun LivePicture(scene: Scene, playing: Boolean, modifier: Modifier = Modifier, grey: Boolean = false) {
    Box(modifier.background(Brand.SurfaceHigh)) {
        if (scene.paths.screenshot != null) {
            AsyncImage(
                scene.paths.screenshot, null,
                contentScale = ContentScale.Crop,
                colorFilter = if (grey) portraitFilter() else null,
                modifier = Modifier.fillMaxSize(),
            )
        }
        ScenePreview(scene.paths.preview, active = playing, modifier = Modifier.fillMaxSize(), delayMs = 400)
    }
}

/** Moves on every few seconds. */
@Composable
private fun rememberRotation(count: Int, everyMs: Long = 7000): Int {
    var i by remember { mutableIntStateOf(0) }
    LaunchedEffect(count) {
        while (count > 1) {
            delay(everyMs)
            i = (i + 1) % count
        }
    }
    return if (count == 0) 0 else i.coerceAtMost(count - 1)
}

private val fade = Brush.verticalGradient(0.35f to Color.Transparent, 1f to Color.Black.copy(alpha = 0.9f))

// ---------------------------------------------------------------- Bento

@Composable
private fun BentoTile(modifier: Modifier, onClick: () -> Unit, content: @Composable () -> Unit) {
    Box(
        modifier
            .clip(RoundedCornerShape(24.dp))
            .background(Brand.Surface)
            .border(1.dp, Brand.Outline, RoundedCornerShape(24.dp))
            .clickable(onClick = onClick)
    ) { content() }
}

@Composable
private fun TileLabel(text: String, modifier: Modifier = Modifier) {
    Text(
        text.uppercase(),
        color = Color.White,
        style = MaterialTheme.typography.labelSmall,
        modifier = modifier.clip(RoundedCornerShape(50)).background(Color.Black.copy(alpha = 0.5f)).padding(horizontal = 9.dp, vertical = 4.dp),
    )
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
internal fun BentoTop(data: HomeData) {
    val nav = LocalNav.current
    val feature = data.hero.firstOrNull() ?: return
    val trend = data.trending.firstOrNull { it.id != feature.id }
    val best = data.topRated.firstOrNull { it.id != feature.id && it.id != trend?.id }
    val star = data.performers.firstOrNull()
    Column(Modifier.statusBarsPadding().padding(top = 60.dp, start = 14.dp, end = 14.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        BentoTile(Modifier.fillMaxWidth().height(280.dp), onClick = { nav.scene(feature.id) }) {
            LivePicture(feature, playing = true, modifier = Modifier.fillMaxSize())
            Box(Modifier.fillMaxSize().background(fade))
            Column(Modifier.align(Alignment.BottomStart).padding(18.dp)) {
                Badge("FEATURED", gradient = true)
                Spacer(Modifier.height(8.dp))
                Text(feature.displayTitle, color = Color.White, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Black, maxLines = 2, overflow = TextOverflow.Ellipsis)
                Text(starring(feature, 3), color = Color.White.copy(alpha = 0.8f), style = MaterialTheme.typography.bodySmall, maxLines = 1)
                Spacer(Modifier.height(10.dp))
                GradientButton("Play", onClick = { nav.play(feature.id, feature.resumeTime ?: 0.0) }, icon = Icons.Rounded.PlayArrow, height = 40.dp)
            }
        }
        Row(Modifier.fillMaxWidth().height(230.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            if (trend != null) BentoTile(Modifier.weight(1f).fillMaxHeight(), onClick = { nav.scene(trend.id) }) {
                LivePicture(trend, playing = false, modifier = Modifier.fillMaxSize())
                Box(Modifier.fillMaxSize().background(fade))
                TileLabel("🔥 Trending #1", Modifier.padding(10.dp))
                Text(trend.displayTitle, color = Color.White, style = MaterialTheme.typography.titleMedium, maxLines = 3, overflow = TextOverflow.Ellipsis, modifier = Modifier.align(Alignment.BottomStart).padding(12.dp))
            }
            if (star != null) BentoTile(Modifier.weight(1f).fillMaxHeight(), onClick = { nav.performer(star.id) }) {
                if (star.imagePath != null) AsyncImage(star.imagePath, null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
                Box(Modifier.fillMaxSize().background(fade))
                TileLabel("Star of the week", Modifier.padding(10.dp))
                Column(Modifier.align(Alignment.BottomStart).padding(12.dp)) {
                    Text("#1", color = Color.White, fontSize = 44.sp, fontWeight = FontWeight.Black)
                    Text(star.name, color = Color.White, style = MaterialTheme.typography.titleMedium, maxLines = 1)
                    Text("${star.sceneCount} scenes", color = Color.White.copy(alpha = 0.75f), style = MaterialTheme.typography.bodySmall)
                }
            }
        }
        Row(Modifier.fillMaxWidth().height(170.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            if (best != null) BentoTile(Modifier.weight(1f).fillMaxHeight(), onClick = { nav.scene(best.id) }) {
                LivePicture(best, playing = false, modifier = Modifier.fillMaxSize())
                Box(Modifier.fillMaxSize().background(fade))
                TileLabel("Top rated", Modifier.padding(10.dp))
                Text(best.displayTitle, color = Color.White, style = MaterialTheme.typography.titleSmall, maxLines = 2, overflow = TextOverflow.Ellipsis, modifier = Modifier.align(Alignment.BottomStart).padding(12.dp))
            }
            Box(
                Modifier
                    .weight(1f)
                    .fillMaxHeight()
                    .clip(RoundedCornerShape(24.dp))
                    .background(Brush.linearGradient(listOf(Brand.Pink.copy(alpha = 0.45f), Brand.Surface)))
                    .padding(12.dp)
            ) {
                Column {
                    Text("BROWSE", style = MaterialTheme.typography.labelLarge)
                    Spacer(Modifier.height(6.dp))
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        data.categories.take(7).forEach { t ->
                            Text(
                                t.name,
                                style = MaterialTheme.typography.labelMedium,
                                modifier = Modifier.clip(RoundedCornerShape(50)).background(Brand.Background.copy(alpha = 0.55f)).clickable { nav.tag(t.id, t.name) }.padding(horizontal = 10.dp, vertical = 5.dp),
                            )
                        }
                    }
                }
            }
        }
        val news = data.recent.filter { it.id != feature.id }.take(3)
        if (news.isNotEmpty()) {
            Column(
                Modifier.fillMaxWidth().clip(RoundedCornerShape(24.dp)).background(Brand.Surface).border(1.dp, Brand.Outline, RoundedCornerShape(24.dp)).padding(14.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Text("JUST ADDED", style = MaterialTheme.typography.labelLarge)
                news.forEach { s ->
                    Row(Modifier.fillMaxWidth().clickable { nav.scene(s.id) }, verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        SceneThumb(s, Modifier.width(110.dp), showProgress = false, corner = 12.dp)
                        Text(s.displayTitle, style = MaterialTheme.typography.titleSmall, maxLines = 2, overflow = TextOverflow.Ellipsis)
                    }
                }
            }
        }
    }
}

// ---------------------------------------------------------------- Mosaic

@Composable
internal fun MosaicTop(categories: List<Tag>, count: Int) {
    val nav = LocalNav.current
    Column(Modifier.statusBarsPadding().padding(top = 60.dp)) {
        Row(Modifier.padding(horizontal = 16.dp), verticalAlignment = Alignment.Bottom) {
            Text("Discover", style = MaterialTheme.typography.displaySmall, fontWeight = FontWeight.Black)
            Spacer(Modifier.width(10.dp))
            if (count > 0) Text("$count scenes", color = Brand.TextMuted, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(bottom = 6.dp))
        }
        LazyRow(contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            items(categories, key = { it.id }) { t ->
                Text(
                    t.name,
                    style = MaterialTheme.typography.labelLarge,
                    modifier = Modifier.clip(RoundedCornerShape(50)).background(Brand.Surface).border(1.dp, Brand.Outline, RoundedCornerShape(50)).clickable { nav.tag(t.id, t.name) }.padding(horizontal = 14.dp, vertical = 8.dp),
                )
            }
        }
    }
}

// ---------------------------------------------------------------- Spotlight

@Composable
internal fun SpotlightTop(data: HomeData) {
    val nav = LocalNav.current
    val tabs = listOf("Trending" to data.trending, "New" to data.recent, "Top rated" to data.topRated, "For you" to data.recommended)
        .filter { it.second.isNotEmpty() }
    if (tabs.isEmpty()) return
    var tab by rememberSaveable { mutableIntStateOf(0) }
    var picked by rememberSaveable(tab) { mutableIntStateOf(0) }
    val list = tabs[tab.coerceAtMost(tabs.size - 1)].second.take(10)
    val current = list[picked.coerceAtMost(list.size - 1)]
    Column {
        Box(Modifier.fillMaxWidth().height(420.dp)) {
            LivePicture(current, playing = true, modifier = Modifier.fillMaxSize())
            Box(Modifier.fillMaxSize().background(Brush.verticalGradient(0f to Color.Black.copy(alpha = 0.4f), 0.4f to Color.Transparent, 1f to Brand.Background)))
            Row(Modifier.align(Alignment.TopEnd).statusBarsPadding().padding(top = 60.dp, end = 18.dp), verticalAlignment = Alignment.Bottom) {
                Text((picked + 1).toString().padStart(2, '0'), color = Color.White, fontSize = 30.sp, fontWeight = FontWeight.Black)
                Text(" / ${list.size}", color = Color.White.copy(alpha = 0.7f), style = MaterialTheme.typography.labelLarge, modifier = Modifier.padding(bottom = 4.dp))
            }
            Column(Modifier.align(Alignment.BottomStart).padding(horizontal = 18.dp, vertical = 14.dp)) {
                Text(current.displayTitle, color = Color.White, style = MaterialTheme.typography.headlineMedium, maxLines = 2, overflow = TextOverflow.Ellipsis)
                Text(current.metaLine(), color = Color.White.copy(alpha = 0.8f), style = MaterialTheme.typography.bodySmall)
                Spacer(Modifier.height(10.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    GradientButton("Play", onClick = { nav.play(current.id, current.resumeTime ?: 0.0) }, icon = Icons.Rounded.PlayArrow, height = 42.dp)
                    GlassButton("More info", onClick = { nav.scene(current.id) }, icon = Icons.Rounded.Info, height = 42.dp)
                }
            }
        }
        LazyRow(contentPadding = PaddingValues(horizontal = 14.dp, vertical = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            items(tabs.size) { i ->
                val on = i == tab
                Text(
                    tabs[i].first,
                    style = MaterialTheme.typography.labelLarge,
                    color = if (on) Color.White else Brand.TextMuted,
                    modifier = Modifier
                        .clip(RoundedCornerShape(50))
                        .then(if (on) Modifier.background(Brand.GradientHorizontal) else Modifier.border(1.dp, Brand.Outline, RoundedCornerShape(50)))
                        .clickable { tab = i }
                        .padding(horizontal = 14.dp, vertical = 7.dp),
                )
            }
        }
        // tap a row to show it big; tap the shown one to open it
        list.forEachIndexed { i, s ->
            SpotRow(s, i + 1, selected = i == picked, onClick = { if (i == picked) nav.scene(s.id) else picked = i })
        }
    }
}

// ---------------------------------------------------------------- Showcase

@Composable
internal fun ShowcaseTop(scenes: List<Scene>) {
    if (scenes.isEmpty()) return
    val nav = LocalNav.current
    val pager = rememberPagerState(pageCount = { scenes.size })
    val density = LocalDensity.current
    LaunchedEffect(pager, scenes.size) {
        while (scenes.size > 1) {
            delay(7000)
            if (!pager.isScrollInProgress) pager.animateScrollToPage((pager.currentPage + 1) % scenes.size)
        }
    }
    val current = scenes[pager.currentPage]
    Box(Modifier.fillMaxWidth()) {
        // a soft glow of the current picture behind everything
        AsyncImage(
            current.paths.screenshot, null,
            contentScale = ContentScale.Crop,
            modifier = Modifier.matchParentSize().graphicsLayer { alpha = 0.35f },
        )
        Box(Modifier.matchParentSize().background(Brush.verticalGradient(listOf(Brand.Background.copy(alpha = 0.55f), Brand.Background))))
        Column(Modifier.statusBarsPadding().padding(top = 64.dp, bottom = 8.dp)) {
            HorizontalPager(
                state = pager,
                contentPadding = PaddingValues(horizontal = 48.dp),
                pageSpacing = 0.dp,
                modifier = Modifier.fillMaxWidth(),
            ) { page ->
                val s = scenes[page]
                val o = (page - pager.currentPage) - pager.currentPageOffsetFraction
                val a = o.absoluteValue.coerceAtMost(2f)
                Box(
                    Modifier
                        .fillMaxWidth()
                        .aspectRatio(16f / 10f)
                        .graphicsLayer {
                            cameraDistance = 12f * density.density
                            rotationY = o.coerceIn(-1f, 1f) * 38f
                            scaleX = 1f - a * 0.12f
                            scaleY = 1f - a * 0.12f
                            alpha = 1f - a * 0.3f
                        }
                        .clip(RoundedCornerShape(22.dp))
                        .clickable { if (page == pager.currentPage) nav.scene(s.id) }
                ) {
                    LivePicture(s, playing = page == pager.settledPage && !pager.isScrollInProgress, modifier = Modifier.fillMaxSize())
                }
            }
            Spacer(Modifier.height(18.dp))
            Column(Modifier.fillMaxWidth().padding(horizontal = 24.dp), horizontalAlignment = Alignment.CenterHorizontally) {
                Text(current.displayTitle, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Black, textAlign = TextAlign.Center, maxLines = 2, overflow = TextOverflow.Ellipsis)
                Text(starring(current, 3), color = Brand.Pink, style = MaterialTheme.typography.bodyMedium, maxLines = 1)
                Text(current.metaLine(), color = Brand.TextMuted, style = MaterialTheme.typography.bodySmall)
                Spacer(Modifier.height(12.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    GradientButton("Play", onClick = { nav.play(current.id, current.resumeTime ?: 0.0) }, icon = Icons.Rounded.PlayArrow, height = 42.dp)
                    GlassButton("More info", onClick = { nav.scene(current.id) }, icon = Icons.Rounded.Info, height = 42.dp)
                }
                Spacer(Modifier.height(12.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    repeat(scenes.size) { i ->
                        Box(
                            Modifier.height(6.dp).width(if (i == pager.currentPage) 22.dp else 6.dp).clip(CircleShape)
                                .background(if (i == pager.currentPage) Brand.Pink else Brand.TextMuted.copy(alpha = 0.5f))
                        )
                    }
                }
            }
        }
    }
}

// ---------------------------------------------------------------- Immersive

@Composable
internal fun ImmersiveTop(scenes: List<Scene>) {
    if (scenes.isEmpty()) return
    val nav = LocalNav.current
    val i = rememberRotation(scenes.size)
    val s = scenes[i]
    val height = LocalConfiguration.current.screenHeightDp.dp
    Box(Modifier.fillMaxWidth().height(height)) {
        LivePicture(s, playing = true, modifier = Modifier.fillMaxSize())
        Box(
            Modifier.fillMaxSize().background(
                Brush.verticalGradient(0f to Color.Black.copy(alpha = 0.45f), 0.35f to Color.Transparent, 0.7f to Brand.Background.copy(alpha = 0.6f), 1f to Brand.Background)
            )
        )
        Column(Modifier.align(Alignment.BottomStart).padding(start = 22.dp, end = 22.dp, bottom = 64.dp)) {
            Badge("FEATURED", gradient = true)
            Spacer(Modifier.height(10.dp))
            Text(s.displayTitle, color = Color.White, fontSize = 42.sp, lineHeight = 44.sp, fontWeight = FontWeight.Black, maxLines = 3, overflow = TextOverflow.Ellipsis)
            if (s.performers.isNotEmpty()) Text("Starring ${starring(s, 3)}", color = Color.White.copy(alpha = 0.85f), style = MaterialTheme.typography.bodyLarge)
            Text(s.metaLine(), color = Color.White.copy(alpha = 0.7f), style = MaterialTheme.typography.bodySmall)
            Spacer(Modifier.height(16.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                GradientButton("Play", onClick = { nav.play(s.id, s.resumeTime ?: 0.0) }, icon = Icons.Rounded.PlayArrow)
                GlassButton("More info", onClick = { nav.scene(s.id) }, icon = Icons.Rounded.Info)
            }
        }
        // the frosted sheet's edge, where the rows begin
        Box(
            Modifier.align(Alignment.BottomCenter).fillMaxWidth().height(30.dp)
                .clip(RoundedCornerShape(topStart = 30.dp, topEnd = 30.dp))
                .background(Brand.Surface.copy(alpha = 0.85f)),
            contentAlignment = Alignment.Center,
        ) {
            Box(Modifier.width(46.dp).height(5.dp).clip(RoundedCornerShape(3.dp)).background(Color.White.copy(alpha = 0.3f)))
        }
    }
}

// ---------------------------------------------------------------- Editorial

@Composable
internal fun EditorialTop(data: HomeData) {
    val nav = LocalNav.current
    val cover = data.hero.firstOrNull() ?: return
    val issue = data.recent.filter { it.id != cover.id }.take(6)
    val month = remember { SimpleDateFormat("MMMM yyyy", Locale.getDefault()).format(Date()) }
    Column(Modifier.statusBarsPadding().padding(top = 60.dp, start = 16.dp, end = 16.dp)) {
        Box(Modifier.fillMaxWidth().height(1.dp).background(Brand.Text))
        Row(Modifier.fillMaxWidth().padding(vertical = 8.dp)) {
            Text("THE ISSUE", style = MaterialTheme.typography.labelSmall, color = Brand.TextMuted, modifier = Modifier.weight(1f))
            Text(month.uppercase(), style = MaterialTheme.typography.labelSmall, color = Brand.TextMuted)
        }
        Box(Modifier.fillMaxWidth().height(3.dp).background(Brand.Text))
        Spacer(Modifier.height(16.dp))
        LivePicture(cover, playing = true, modifier = Modifier.fillMaxWidth().aspectRatio(4f / 3f).clickable { nav.scene(cover.id) })
        Spacer(Modifier.height(14.dp))
        Text("COVER STORY", color = Brand.Pink, style = MaterialTheme.typography.labelLarge.copy(letterSpacing = 2.sp))
        Text(cover.displayTitle, style = MaterialTheme.typography.displaySmall, lineHeight = 40.sp)
        if (cover.performers.isNotEmpty()) {
            Text("STARRING ${starring(cover, 3).uppercase()}", style = MaterialTheme.typography.labelLarge.copy(letterSpacing = 1.5.sp), modifier = Modifier.padding(vertical = 6.dp))
        }
        cover.details?.takeIf { it.isNotBlank() }?.let {
            Text(it, style = MaterialTheme.typography.bodyLarge.copy(fontFamily = androidx.compose.ui.text.font.FontFamily.Serif), color = Brand.TextMuted, maxLines = 4, overflow = TextOverflow.Ellipsis)
        }
        Spacer(Modifier.height(12.dp))
        GradientButton("Watch now", onClick = { nav.play(cover.id, cover.resumeTime ?: 0.0) }, icon = Icons.Rounded.PlayArrow, height = 42.dp)
        if (issue.isNotEmpty()) {
            Spacer(Modifier.height(24.dp))
            Box(Modifier.fillMaxWidth().height(1.dp).background(Brand.Text))
            Text("In this issue", style = MaterialTheme.typography.headlineSmall, fontStyle = FontStyle.Italic, modifier = Modifier.padding(vertical = 10.dp))
            issue.forEachIndexed { i, s ->
                Box(Modifier.fillMaxWidth().height(1.dp).background(Brand.Outline))
                Row(Modifier.fillMaxWidth().clickable { nav.scene(s.id) }.padding(vertical = 12.dp), horizontalArrangement = Arrangement.spacedBy(14.dp)) {
                    Text((i + 1).toString().padStart(2, '0'), color = Brand.Pink, fontSize = 28.sp, fontStyle = FontStyle.Italic, fontFamily = androidx.compose.ui.text.font.FontFamily.Serif)
                    Column(Modifier.weight(1f)) {
                        Text(s.displayTitle, style = MaterialTheme.typography.titleMedium, maxLines = 2, overflow = TextOverflow.Ellipsis)
                        Text(listOfNotNull(starring(s).ifBlank { null }, s.duration.takeIf { it > 0 }?.let { formatDuration(it) }).joinToString(" · ").uppercase(), style = MaterialTheme.typography.labelSmall, color = Brand.TextMuted)
                    }
                }
            }
        }
    }
}

// ---------------------------------------------------------------- Brutal

@Composable
internal fun BrutalTop(data: HomeData) {
    val nav = LocalNav.current
    if (data.hero.isEmpty()) return
    val i = rememberRotation(data.hero.size)
    val s = data.hero[i]
    Column(Modifier.statusBarsPadding().padding(top = 64.dp)) {
        Box(Modifier.padding(horizontal = 16.dp)) {
            Column(
                Modifier
                    .fillMaxWidth()
                    .hardShadow(Brand.Text, 8.dp)
                    .border(3.dp, Brand.Text)
                    .background(Brand.Pink)
                    .padding(18.dp)
            ) {
                Spacer(Modifier.height(18.dp))
                Text(s.displayTitle.uppercase(), color = Color.White, fontSize = 38.sp, lineHeight = 38.sp, fontWeight = FontWeight.Black, maxLines = 3, overflow = TextOverflow.Ellipsis)
                if (s.performers.isNotEmpty()) {
                    Text(starring(s, 3).uppercase(), color = Color.White, fontWeight = FontWeight.ExtraBold, style = MaterialTheme.typography.bodyLarge, modifier = Modifier.padding(top = 6.dp))
                }
                Spacer(Modifier.height(14.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    BrutalButton("▶ WATCH", filled = false) { nav.play(s.id, s.resumeTime ?: 0.0) }
                    if (data.hero.size > 1) BrutalButton("INFO", filled = false) { nav.scene(s.id) }
                }
            }
            Text(
                if (isNewScene(s)) "NEW!" else "FEATURED",
                fontWeight = FontWeight.Black,
                color = Brand.Text,
                modifier = Modifier.offset(x = (-6).dp, y = (-12).dp).rotate(-8f).border(3.dp, Brand.Text).background(Brand.Background).padding(horizontal = 12.dp, vertical = 4.dp),
            )
        }
        Spacer(Modifier.height(26.dp))
        Box(Modifier.padding(horizontal = 22.dp).rotate(1.5f)) {
            LivePicture(
                s, playing = true,
                modifier = Modifier.fillMaxWidth().aspectRatio(16f / 10f).hardShadow(Brand.Text, 8.dp).border(3.dp, Brand.Text).clickable { nav.scene(s.id) },
            )
            if (s.duration > 0) {
                Text(
                    formatDuration(s.duration),
                    fontWeight = FontWeight.Black,
                    color = Brand.Text,
                    modifier = Modifier.align(Alignment.BottomStart).offset(x = 12.dp, y = 12.dp).rotate(-3f).border(3.dp, Brand.Text).background(Brand.Background).padding(horizontal = 10.dp, vertical = 3.dp),
                )
            }
        }
        Spacer(Modifier.height(30.dp))
        Marquee(data.recent.map { it.displayTitle })
    }
}

private fun isNewScene(s: Scene) = s.badges.any { it.equals("new", ignoreCase = true) }

@Composable
private fun BrutalButton(text: String, filled: Boolean, onClick: () -> Unit) {
    Text(
        text,
        fontWeight = FontWeight.Black,
        color = if (filled) Brand.Background else Brand.Text,
        modifier = Modifier
            .hardShadow(Brand.Text, 4.dp)
            .border(3.dp, Brand.Text)
            .background(if (filled) Brand.Text else Brand.Background)
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 10.dp),
    )
}

/** Titles scrolling across a black strip, forever. */
@Composable
private fun Marquee(titles: List<String>) {
    if (titles.isEmpty()) return
    val line = titles.take(12).joinToString("   ✦   ") { it.uppercase() } + "   ✦   "
    val roll = rememberInfiniteTransition(label = "marquee")
    val x by roll.animateFloat(0f, 1f, infiniteRepeatable(tween(40000, easing = LinearEasing), RepeatMode.Restart), label = "x")
    Box(
        Modifier.fillMaxWidth().rotate(-1f).background(Brand.Text).padding(vertical = 10.dp)
    ) {
        Row(Modifier.graphicsLayer { translationX = -x * 2400f }) {
            repeat(3) {
                Text(line, color = Brand.Background, fontSize = 20.sp, fontWeight = FontWeight.Black, maxLines = 1, softWrap = false)
            }
        }
    }
}

// ---------------------------------------------------------------- Swipe

@Composable
internal fun SwipeTop(data: HomeData) {
    val nav = LocalNav.current
    val scope = rememberCoroutineScope()
    val gone = remember { mutableStateListOf<String>() }
    val liked = remember { mutableStateListOf<Scene>() }
    val pool = (data.recommended + data.trending + data.recent)
        .distinctBy { it.id }
        .filter { it.paths.screenshot != null && it.id !in gone }
    val density = LocalDensity.current
    val dx = remember { Animatable(0f) }
    var dy by remember { mutableStateOf(0f) }
    val limit = with(density) { 110.dp.toPx() }

    fun finish(right: Boolean) {
        val top = pool.firstOrNull() ?: return
        scope.launch {
            dx.animateTo(if (right) limit * 5 else -limit * 5, tween(260))
            if (right) {
                liked.add(0, top)
                runCatching { app.repo.toggleLike(top.id) }
            }
            gone.add(top.id)
            dx.snapTo(0f)
            dy = 0f
        }
    }

    Column(Modifier.statusBarsPadding().padding(top = 60.dp), horizontalAlignment = Alignment.CenterHorizontally) {
        Text("Swipe to discover", style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Black)
        Text("Right to like, left to skip, tap to open", color = Brand.TextMuted, style = MaterialTheme.typography.bodySmall)
        Spacer(Modifier.height(14.dp))
        Box(Modifier.fillMaxWidth(0.86f).aspectRatio(4f / 5f)) {
            if (pool.isEmpty()) {
                Column(
                    Modifier.fillMaxSize().clip(RoundedCornerShape(30.dp)).border(2.dp, Brand.Outline, RoundedCornerShape(30.dp)).clickable { gone.clear() },
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Text("You've seen them all", style = MaterialTheme.typography.titleMedium)
                    Text("Tap to start again", color = Brand.TextMuted, style = MaterialTheme.typography.bodySmall)
                }
            }
            pool.take(3).reversed().forEach { s ->
                val depth = pool.indexOf(s)
                val top = depth == 0
                Box(
                    Modifier
                        .fillMaxSize()
                        .then(
                            if (top) Modifier
                                .offset { IntOffset(dx.value.roundToInt(), (dy * 0.3f).roundToInt()) }
                                .graphicsLayer { rotationZ = dx.value / 22f }
                                .pointerInput(s.id) {
                                    detectDragGestures(
                                        onDragEnd = {
                                            when {
                                                dx.value > limit -> finish(true)
                                                dx.value < -limit -> finish(false)
                                                else -> scope.launch { dx.animateTo(0f); dy = 0f }
                                            }
                                        },
                                    ) { change, drag ->
                                        change.consume()
                                        scope.launch { dx.snapTo(dx.value + drag.x) }
                                        dy += drag.y
                                    }
                                }
                            else Modifier.graphicsLayer {
                                translationY = depth * 14.dp.toPx()
                                scaleX = 1f - depth * 0.05f
                                scaleY = 1f - depth * 0.05f
                                alpha = 0.85f
                            }
                        )
                        .clip(RoundedCornerShape(30.dp))
                        .clickable(enabled = top) { nav.scene(s.id) }
                ) {
                    LivePicture(s, playing = top, modifier = Modifier.fillMaxSize())
                    Box(Modifier.fillMaxSize().background(fade))
                    if (top) {
                        val p = (dx.value / limit).coerceIn(-1f, 1f)
                        Stamp("LIKE", Color(0xFF3DDC84), -14f, Modifier.align(Alignment.TopStart).padding(22.dp).graphicsLayer { alpha = p.coerceAtLeast(0f) })
                        Stamp("SKIP", Color(0xFFFF4D6D), 14f, Modifier.align(Alignment.TopEnd).padding(22.dp).graphicsLayer { alpha = (-p).coerceAtLeast(0f) })
                    }
                    Column(Modifier.align(Alignment.BottomStart).padding(20.dp)) {
                        Text(s.displayTitle, color = Color.White, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Black, maxLines = 2, overflow = TextOverflow.Ellipsis)
                        Text(listOfNotNull(starring(s, 3).ifBlank { null }, s.duration.takeIf { it > 0 }?.let { formatDuration(it) }).joinToString(" · "), color = Color.White.copy(alpha = 0.8f), style = MaterialTheme.typography.bodySmall)
                    }
                }
            }
        }
        Spacer(Modifier.height(16.dp))
        if (pool.isNotEmpty()) {
            Row(horizontalArrangement = Arrangement.spacedBy(18.dp), verticalAlignment = Alignment.CenterVertically) {
                RoundAction(Icons.Rounded.Close, Color(0xFFFF4D6D), 56) { finish(false) }
                var saved by remember(pool.first().id) { mutableStateOf(false) }
                RoundAction(if (saved) Icons.Rounded.Check else Icons.Rounded.Add, Brand.Text, 46) {
                    val id = pool.first().id
                    scope.launch { saved = runCatching { app.repo.toggleWatchLater(id) }.getOrDefault(false) }
                }
                RoundAction(Icons.Rounded.PlayArrow, Brand.Text, 46) { pool.firstOrNull()?.let { nav.play(it.id, it.resumeTime ?: 0.0) } }
                RoundAction(Icons.Rounded.Favorite, Color.White, 64, gradient = true) { finish(true) }
            }
        }
        if (liked.isNotEmpty()) {
            Spacer(Modifier.height(18.dp))
            Text("LIKED JUST NOW", style = MaterialTheme.typography.labelLarge, color = Brand.TextMuted, modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp))
            LazyRow(contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                items(liked, key = { it.id }) { s ->
                    SceneThumb(s, Modifier.width(140.dp).clickable { nav.scene(s.id) }, showProgress = false, corner = 14.dp)
                }
            }
        }
    }
}

@Composable
private fun Stamp(text: String, color: Color, angle: Float, modifier: Modifier) {
    Text(
        text,
        color = color,
        fontSize = 28.sp,
        fontWeight = FontWeight.Black,
        modifier = modifier.rotate(angle).border(4.dp, color, RoundedCornerShape(10.dp)).padding(horizontal = 12.dp, vertical = 2.dp),
    )
}

@Composable
private fun RoundAction(icon: androidx.compose.ui.graphics.vector.ImageVector, tint: Color, size: Int, gradient: Boolean = false, onClick: () -> Unit) {
    Box(
        Modifier
            .size(size.dp)
            .clip(CircleShape)
            .then(if (gradient) Modifier.background(Brand.Gradient) else Modifier.background(Brand.Surface).border(1.dp, Brand.Outline, CircleShape))
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Icon(icon, null, tint = tint, modifier = Modifier.size((size * 0.45f).dp))
    }
}

/** Whether the storefront home uses its own top for the current layout. */
internal val homeLayout: String get() = SiteMode.layout
