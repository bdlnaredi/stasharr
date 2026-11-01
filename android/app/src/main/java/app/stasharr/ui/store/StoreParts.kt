package app.stasharr.ui.store

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.basicMarquee
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowForward
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Favorite
import androidx.compose.material.icons.rounded.FavoriteBorder
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.PlayArrow
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.Star
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
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
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.stasharr.app
import app.stasharr.data.Performer
import app.stasharr.data.Scene
import app.stasharr.data.SitePromo
import app.stasharr.player.AutoPreview
import app.stasharr.player.PreviewPlayer
import app.stasharr.player.previewSpot
import app.stasharr.player.rememberPreviewKey
import app.stasharr.player.ScenePreview
import app.stasharr.ui.LocalNav
import app.stasharr.ui.SiteMode
import app.stasharr.ui.components.Badge
import app.stasharr.ui.components.GradientText
import app.stasharr.ui.components.SceneThumb
import app.stasharr.ui.components.formatDuration
import app.stasharr.ui.components.resolutionLabel
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/** The storefront's top bar: logo or site name, search and the account button. */
@Composable
fun StoreTopBar(solid: Boolean, onSearch: () -> Unit, onAccount: () -> Unit) {
    val look = SiteMode.look
    val name = look?.name?.takeIf { it.isNotBlank() } ?: "Stasharr"
    Row(
        Modifier
            .fillMaxWidth()
            .background(
                if (solid) Brush.verticalGradient(listOf(Brand.Background.copy(alpha = 0.96f), Brand.Background.copy(alpha = 0.96f)))
                else Brush.verticalGradient(listOf(Color.Black.copy(alpha = 0.6f), Color.Transparent))
            )
            .statusBarsPadding()
            .padding(start = 18.dp, end = 6.dp, top = 6.dp, bottom = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(Modifier.weight(1f)) {
            if (look?.logoUrl != null) {
                AsyncImage(
                    model = look.logoUrl,
                    contentDescription = name,
                    contentScale = ContentScale.Fit,
                    modifier = Modifier.height(32.dp).width(180.dp),
                    alignment = Alignment.CenterStart,
                )
            } else {
                StoreLogo(name)
            }
        }
        IconButton(onClick = onSearch) { Icon(Icons.Rounded.Search, "Search", tint = Color.White) }
        Box(
            Modifier
                .padding(end = 8.dp)
                .size(34.dp)
                .clip(CircleShape)
                .background(Brand.Pink)
                .clickable(onClick = onAccount),
            contentAlignment = Alignment.Center,
        ) {
            Icon(Icons.Rounded.Person, "Account", tint = Color.White, modifier = Modifier.size(20.dp))
        }
    }
}

/** The site name with its last word in the accent colour, like the website. */
@Composable
fun StoreLogo(name: String, size: Int = 22) {
    val words = name.trim().split(Regex("\\s+"))
    Row(verticalAlignment = Alignment.CenterVertically) {
        if (words.size > 1) {
            Text(words.dropLast(1).joinToString(" ") + " ", color = Color.White, fontSize = size.sp, fontWeight = FontWeight.Black, maxLines = 1)
            GradientText(words.last(), fontSize = size.sp, maxLines = 1)
        } else {
            GradientText(name, fontSize = size.sp, maxLines = 1)
        }
    }
}

/** A heading for a storefront row, with an optional "View all". */
@Composable
fun StoreHeader(title: String, onAll: (() -> Unit)? = null, allText: String = "View all") {
    Row(
        Modifier.fillMaxWidth().padding(start = 16.dp, end = 10.dp, top = 24.dp, bottom = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.ExtraBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Spacer(Modifier.height(5.dp))
            Box(Modifier.width(36.dp).height(3.dp).clip(RoundedCornerShape(2.dp)).background(Brand.GradientHorizontal))
        }
        if (onAll != null) {
            Row(
                Modifier.clip(RoundedCornerShape(50)).clickable(onClick = onAll).padding(horizontal = 8.dp, vertical = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(allText.uppercase(), color = Brand.Pink, style = MaterialTheme.typography.labelMedium, fontWeight = FontWeight.Bold)
                Icon(Icons.AutoMirrored.Rounded.ArrowForward, null, tint = Brand.Pink, modifier = Modifier.size(14.dp))
            }
        }
    }
}

/** A scene tile like the website's: picture with badges, title, stars, studio and date. */
@OptIn(ExperimentalFoundationApi::class)
@Composable
fun StoreSceneCard(scene: Scene, modifier: Modifier = Modifier, width: Dp? = 260.dp, onClick: (() -> Unit)? = null) {
    val nav = LocalNav.current
    val haptics = LocalHapticFeedback.current
    val key = rememberPreviewKey("card")
    val previewing = PreviewPlayer.activeCardId == key && scene.paths.preview != null
    Column(
        (if (width != null) modifier.width(width) else modifier.fillMaxWidth())
            .clip(RoundedCornerShape(12.dp))
            .combinedClickable(
                onClick = {
                    PreviewPlayer.activeCardId = null
                    if (onClick != null) onClick() else nav.scene(scene.id)
                },
                onLongClick = {
                    if (scene.paths.preview != null) {
                        haptics.performHapticFeedback(HapticFeedbackType.LongPress)
                        PreviewPlayer.activeCardId = if (previewing) null else key
                    }
                },
            )
    ) {
        Box {
            SceneThumb(scene, preview = previewing, modifier = Modifier.previewSpot(key, AutoPreview.enabled && scene.paths.preview != null))
            if (scene.isNew) {
                Badge("NEW", Modifier.align(Alignment.TopStart).padding(6.dp), gradient = true)
            }
        }
        Spacer(Modifier.height(8.dp))
        Text(
            scene.displayTitle,
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.Bold,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.padding(horizontal = 2.dp),
        )
        val stars = scene.stars.take(3).joinToString(", ") { it.name }
        if (stars.isNotBlank()) {
            Text(stars, style = MaterialTheme.typography.bodySmall, color = Brand.Pink, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(horizontal = 2.dp))
        }
        val sub = listOfNotNull(scene.studio?.name, scene.date).joinToString("  ·  ")
        if (sub.isNotBlank()) {
            Text(sub, style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.padding(horizontal = 2.dp))
        }
        Spacer(Modifier.height(4.dp))
    }
}

/** A tall portrait card for a pornstar, with an optional rank number. */
@Composable
fun ModelCard(p: Performer, modifier: Modifier = Modifier, width: Dp? = 140.dp, rank: Int? = null, subtitle: String? = null) {
    val nav = LocalNav.current
    Column(
        (if (width != null) modifier.width(width) else modifier.fillMaxWidth())
            .clip(RoundedCornerShape(12.dp))
            .clickable { nav.performer(p.id) },
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .fillMaxWidth()
                .aspectRatio(2f / 3f)
                .clip(RoundedCornerShape(12.dp))
                .background(Brand.SurfaceHigh),
            contentAlignment = Alignment.Center,
        ) {
            Icon(Icons.Rounded.Person, null, tint = Brand.TextMuted, modifier = Modifier.size(48.dp))
            if (p.imagePath != null) {
                AsyncImage(p.imagePath, p.name, contentScale = ContentScale.Crop, alignment = Alignment.TopCenter, modifier = Modifier.fillMaxSize())
            }
            if (rank != null) {
                Box(
                    Modifier
                        .align(Alignment.BottomStart)
                        .fillMaxWidth()
                        .background(Brush.verticalGradient(listOf(Color.Transparent, Color.Black.copy(alpha = 0.85f))))
                        .padding(start = 10.dp, top = 28.dp, bottom = 4.dp)
                ) {
                    Text("$rank", color = Color.White, fontSize = 38.sp, fontWeight = FontWeight.Black)
                }
            }
        }
        Spacer(Modifier.height(8.dp))
        Text(p.name, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold, maxLines = 1, overflow = TextOverflow.Ellipsis)
        Text(subtitle ?: "${p.sceneCount} scenes", style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted, maxLines = 1)
    }
}

/** A coloured category tile. */
@Composable
fun CategoryTile(name: String, count: Int, modifier: Modifier = Modifier, onClick: () -> Unit) {
    val hue = (name.hashCode() and 0xffff) % 360
    val tint = Color.hsv(hue.toFloat(), 0.5f, 0.45f)
    Box(
        modifier
            .aspectRatio(3f / 2f)
            .clip(RoundedCornerShape(12.dp))
            .background(Brush.linearGradient(listOf(tint, Brand.SurfaceHigh)))
            .border(1.dp, Brand.Outline, RoundedCornerShape(12.dp))
            .clickable(onClick = onClick)
            .padding(12.dp),
    ) {
        Column(Modifier.align(Alignment.BottomStart)) {
            Text(name, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold, color = Color.White, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text("$count scenes", style = MaterialTheme.typography.bodySmall, color = Color.White.copy(alpha = 0.7f))
        }
    }
}

/** A sideways row of scenes. */
@Composable
fun SceneRow(title: String, scenes: List<Scene>, onAll: (() -> Unit)? = null) {
    if (scenes.isEmpty()) return
    StoreHeader(title, onAll)
    LazyRow(contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(14.dp)) {
        itemsIndexed(scenes, key = { _, s -> s.id }) { _, s -> StoreSceneCard(s) }
    }
}

/** A sideways row of pornstars. */
@Composable
fun ModelRow(title: String, models: List<Performer>, ranked: Boolean = false, onAll: (() -> Unit)? = null, subtitle: ((Performer) -> String)? = null) {
    if (models.isEmpty()) return
    StoreHeader(title, onAll)
    LazyRow(contentPadding = PaddingValues(horizontal = 16.dp), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        itemsIndexed(models, key = { _, p -> p.id }) { i, p -> ModelCard(p, rank = if (ranked) i + 1 else null, subtitle = subtitle?.invoke(p)) }
    }
}

/** The big featured banner, changing every few seconds. */
@Composable
fun StoreHero(scenes: List<Scene>) {
    if (scenes.isEmpty()) {
        Spacer(Modifier.statusBarsPadding().height(72.dp))
        return
    }
    val nav = LocalNav.current
    val pager = rememberPagerState(pageCount = { scenes.size })
    LaunchedEffect(pager, scenes.size) {
        while (scenes.size > 1) {
            delay(8000)
            if (!pager.isScrollInProgress) pager.animateScrollToPage((pager.currentPage + 1) % scenes.size)
        }
    }
    Box(Modifier.fillMaxWidth().height(560.dp)) {
        HorizontalPager(state = pager, modifier = Modifier.fillMaxSize()) { page ->
            val s = scenes[page]
            val key = rememberPreviewKey("hero")
            val current = pager.currentPage == page && !pager.isScrollInProgress
            Box(
                Modifier
                    .fillMaxSize()
                    .previewSpot(key, AutoPreview.enabled && current && s.paths.preview != null)
                    .clickable { nav.scene(s.id) }
            ) {
                AsyncImage(s.paths.screenshot, null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
                ScenePreview(
                    s.paths.preview,
                    active = if (AutoPreview.enabled) PreviewPlayer.activeCardId == key else current,
                    modifier = Modifier.fillMaxSize(),
                    delayMs = 300,
                )
                Box(
                    Modifier.fillMaxSize().background(
                        Brush.verticalGradient(
                            0f to Color.Black.copy(alpha = 0.35f),
                            0.45f to Color.Transparent,
                            0.75f to Brand.Background.copy(alpha = 0.85f),
                            1f to Brand.Background,
                        )
                    )
                )
                HeroText(s, Modifier.align(Alignment.BottomStart))
            }
        }
        if (scenes.size > 1) {
            Row(
                Modifier.align(Alignment.BottomEnd).padding(end = 18.dp, bottom = 14.dp),
                horizontalArrangement = Arrangement.spacedBy(6.dp),
            ) {
                scenes.indices.forEach { i ->
                    Box(
                        Modifier
                            .height(4.dp)
                            .width(if (i == pager.currentPage) 26.dp else 10.dp)
                            .clip(RoundedCornerShape(2.dp))
                            .background(if (i == pager.currentPage) Brand.Pink else Color.White.copy(alpha = 0.35f))
                    )
                }
            }
        }
    }
}

@Composable
private fun HeroText(s: Scene, modifier: Modifier) {
    val nav = LocalNav.current
    val scope = rememberCoroutineScope()
    var inList by remember(s.id) { mutableStateOf(s.inWatchLater) }
    Column(modifier.padding(start = 18.dp, end = 18.dp, bottom = 36.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                "FEATURED",
                color = Brand.Pink,
                fontSize = 11.sp,
                fontWeight = FontWeight.Black,
                letterSpacing = 2.sp,
                modifier = Modifier.border(1.dp, Brand.Pink, RoundedCornerShape(50)).padding(horizontal = 10.dp, vertical = 2.dp),
            )
            s.studio?.let {
                Spacer(Modifier.width(10.dp))
                Text(it.name.uppercase(), color = Color.White.copy(alpha = 0.8f), fontSize = 11.sp, fontWeight = FontWeight.Bold, letterSpacing = 2.sp, maxLines = 1)
            }
        }
        Spacer(Modifier.height(8.dp))
        Text(
            s.displayTitle,
            color = Color.White,
            fontSize = if (s.displayTitle.length > 45) 24.sp else 30.sp,
            lineHeight = if (s.displayTitle.length > 45) 28.sp else 34.sp,
            fontWeight = FontWeight.Black,
            maxLines = 3,
            overflow = TextOverflow.Ellipsis,
        )
        val stars = s.stars.take(3)
        if (stars.isNotEmpty()) {
            Spacer(Modifier.height(6.dp))
            Text(
                "Starring " + stars.joinToString(" & ") { it.name },
                color = Color.White.copy(alpha = 0.85f),
                style = MaterialTheme.typography.bodyMedium,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        val meta = listOfNotNull(s.date, s.duration.takeIf { it > 0 }?.let { formatDuration(it) }, s.file?.let { resolutionLabel(it.width, it.height) })
        if (meta.isNotEmpty()) {
            Text(meta.joinToString("   "), color = Color.White.copy(alpha = 0.7f), style = MaterialTheme.typography.bodySmall)
        }
        Spacer(Modifier.height(14.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
            Row(
                Modifier.height(44.dp).clip(RoundedCornerShape(50)).background(Brand.GradientHorizontal)
                    .clickable { nav.play(s.id, 0.0) }.padding(horizontal = 20.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(Icons.Rounded.PlayArrow, null, tint = Color.White)
                Text("WATCH NOW", color = Color.White, fontWeight = FontWeight.Bold, fontSize = 13.sp, letterSpacing = 1.sp)
            }
            Row(
                Modifier.height(44.dp).clip(RoundedCornerShape(50)).background(Color.White.copy(alpha = 0.12f))
                    .border(1.dp, Color.White.copy(alpha = 0.35f), RoundedCornerShape(50))
                    .clickable {
                        scope.launch { runCatching { app.repo.toggleWatchLater(s.id) }.onSuccess { inList = it } }
                    }.padding(horizontal = 16.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(if (inList) Icons.Rounded.Check else Icons.Rounded.Add, null, tint = Color.White, modifier = Modifier.size(18.dp))
                Spacer(Modifier.width(4.dp))
                Text("MY LIST", color = Color.White, fontWeight = FontWeight.Bold, fontSize = 13.sp, letterSpacing = 1.sp)
            }
            LikeHeart(s.id)
        }
    }
}

/** A round heart that likes or unlikes a scene. */
@Composable
fun LikeHeart(sceneId: String, initial: Boolean? = null, size: Dp = 44.dp) {
    val scope = rememberCoroutineScope()
    var liked by remember(sceneId) { mutableStateOf(initial ?: false) }
    if (initial == null) {
        LaunchedEffect(sceneId) { runCatching { app.repo.sceneExtras(sceneId) }.getOrNull()?.let { liked = it.liked } }
    }
    Box(
        Modifier
            .size(size)
            .clip(CircleShape)
            .background(Color.White.copy(alpha = 0.12f))
            .border(1.dp, Color.White.copy(alpha = 0.35f), CircleShape)
            .clickable {
                val was = liked
                liked = !was
                scope.launch { runCatching { app.repo.toggleLike(sceneId) }.onSuccess { liked = it }.onFailure { liked = was } }
            },
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            if (liked) Icons.Rounded.Favorite else Icons.Rounded.FavoriteBorder,
            if (liked) "Unlike" else "Like",
            tint = if (liked) Brand.Pink else Color.White,
            modifier = Modifier.size(size * 0.5f),
        )
    }
}

/** The "Just added" strip that scrolls on its own. */
@Composable
fun StoreTicker(scenes: List<Scene>) {
    if (scenes.isEmpty()) return
    Row(
        Modifier.fillMaxWidth().height(40.dp).background(Brand.Surface),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(Modifier.fillMaxHeight().background(Brand.Pink).padding(horizontal = 12.dp), contentAlignment = Alignment.Center) {
            Text("JUST ADDED", color = Color.White, fontSize = 11.sp, fontWeight = FontWeight.Black, letterSpacing = 1.5.sp)
        }
        Text(
            scenes.joinToString("      ●  ") { s ->
                s.displayTitle + (s.stars.take(2).takeIf { it.isNotEmpty() }?.let { " — " + it.joinToString(" & ") { p -> p.name } } ?: "")
            },
            color = Brand.Text,
            style = MaterialTheme.typography.bodyMedium,
            maxLines = 1,
            modifier = Modifier.weight(1f).padding(start = 12.dp).basicMarquee(iterations = Int.MAX_VALUE, velocity = 40.dp),
        )
    }
}

/** The admin's promo banner. */
@Composable
fun StorePromo(promo: SitePromo) {
    if (!promo.enabled || promo.text.isBlank()) return
    val nav = LocalNav.current
    Box(
        Modifier
            .padding(start = 16.dp, end = 16.dp, top = 20.dp)
            .fillMaxWidth()
            .height(96.dp)
            .clip(RoundedCornerShape(14.dp))
            .background(Brand.GradientHorizontal)
            .clickable(enabled = promo.link.isNotBlank()) { nav.siteLink(promo.link) },
    ) {
        if (promo.image.isNotBlank()) {
            AsyncImage(promo.image, null, contentScale = ContentScale.Crop, alpha = 0.35f, modifier = Modifier.fillMaxSize())
        }
        Row(Modifier.fillMaxSize().padding(horizontal = 18.dp), verticalAlignment = Alignment.CenterVertically) {
            Text(promo.text, color = Color.White, fontSize = 18.sp, fontWeight = FontWeight.Black, maxLines = 2, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f))
            if (promo.link.isNotBlank()) {
                Text(
                    "EXPLORE",
                    color = Color.White,
                    fontSize = 12.sp,
                    fontWeight = FontWeight.Black,
                    modifier = Modifier.border(1.dp, Color.White, RoundedCornerShape(50)).padding(horizontal = 12.dp, vertical = 6.dp),
                )
            }
        }
    }
}

/** Stars showing everyone's average rating. */
@Composable
fun AverageStars(average: Double, count: Int) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        val full = (average / 20).toFloat()
        (1..5).forEach { n ->
            Icon(
                Icons.Rounded.Star,
                null,
                tint = if (full >= n - 0.25f) Brand.Pink else Brand.Outline,
                modifier = Modifier.size(18.dp),
            )
        }
        Spacer(Modifier.width(6.dp))
        Text(String.format("%.1f", average / 20), fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleSmall)
        Spacer(Modifier.width(6.dp))
        Text("$count rating${if (count == 1) "" else "s"}", color = Brand.TextMuted, style = MaterialTheme.typography.bodySmall)
    }
}
