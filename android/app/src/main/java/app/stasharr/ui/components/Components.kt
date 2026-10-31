package app.stasharr.ui.components

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowForward
import androidx.compose.material.icons.rounded.MusicNote
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.Star
import androidx.compose.material.icons.rounded.StarBorder
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.TextUnit
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.stasharr.data.Performer
import app.stasharr.data.PerformerRef
import app.stasharr.data.Scene
import app.stasharr.data.Studio
import app.stasharr.player.PreviewPlayer
import app.stasharr.player.ScenePreview
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage

// ---------- formatting

fun formatDuration(seconds: Double): String {
    val s = seconds.toLong().coerceAtLeast(0)
    val h = s / 3600
    val m = (s % 3600) / 60
    val sec = s % 60
    return if (h > 0) "%d:%02d:%02d".format(h, m, sec) else "%d:%02d".format(m, sec)
}

fun formatRuntime(seconds: Double): String {
    val m = (seconds / 60).toInt()
    return if (m >= 60) "${m / 60}h ${m % 60}m" else "${m}m"
}

fun resolutionLabel(w: Int, h: Int): String? {
    val short = minOf(w, h)
    return when {
        short <= 0 -> null
        short >= 2100 -> "4K"
        short >= 1400 -> "1440p"
        short >= 1000 -> "1080p"
        short >= 700 -> "720p"
        else -> "${short}p"
    }
}

fun Scene.metaLine(): String = listOfNotNull(
    year,
    duration.takeIf { it > 0 }?.let { formatRuntime(it) },
    if (isAudio) "Audio" else file?.let { resolutionLabel(it.width, it.height) },
    studio?.name,
).joinToString("  •  ")

// ---------- small pieces

@Composable
fun GradientButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    icon: ImageVector? = null,
    height: Dp = 48.dp,
) {
    Row(
        modifier
            .height(height)
            .clip(RoundedCornerShape(50))
            .background(Brand.GradientHorizontal)
            .clickable(onClick = onClick)
            .padding(horizontal = 22.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.Center,
    ) {
        if (icon != null) {
            Icon(icon, null, tint = Color.White, modifier = Modifier.size(22.dp))
            Spacer(Modifier.width(6.dp))
        }
        Text(text, color = Color.White, style = MaterialTheme.typography.labelLarge, fontSize = 15.sp)
    }
}

@Composable
fun GlassButton(
    text: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    icon: ImageVector? = null,
    height: Dp = 48.dp,
) {
    Row(
        modifier
            .height(height)
            .clip(RoundedCornerShape(50))
            .background(Color.White.copy(alpha = 0.12f))
            .border(1.dp, Color.White.copy(alpha = 0.18f), RoundedCornerShape(50))
            .clickable(onClick = onClick)
            .padding(horizontal = 18.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.Center,
    ) {
        if (icon != null) {
            Icon(icon, null, tint = Color.White, modifier = Modifier.size(20.dp))
            Spacer(Modifier.width(6.dp))
        }
        Text(text, color = Color.White, style = MaterialTheme.typography.labelLarge)
    }
}

@Composable
fun GradientText(text: String, modifier: Modifier = Modifier, fontSize: TextUnit = 26.sp, maxLines: Int = Int.MAX_VALUE) {
    Text(
        text,
        modifier = modifier,
        maxLines = maxLines,
        overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis,
        style = MaterialTheme.typography.headlineMedium.copy(
            brush = Brand.GradientHorizontal,
            fontSize = fontSize,
            fontWeight = FontWeight.Black,
        ),
    )
}

@Composable
fun Badge(text: String, modifier: Modifier = Modifier, gradient: Boolean = false) {
    Box(
        modifier
            .clip(RoundedCornerShape(5.dp))
            .then(
                if (gradient) Modifier.background(Brand.GradientHorizontal)
                else Modifier.background(Color.Black.copy(alpha = 0.65f))
            )
            .padding(horizontal = 6.dp, vertical = 2.dp)
    ) {
        Text(text, color = Color.White, style = MaterialTheme.typography.labelSmall)
    }
}

@Composable
fun SectionHeader(title: String, modifier: Modifier = Modifier, onSeeAll: (() -> Unit)? = null) {
    Row(
        modifier.fillMaxWidth().padding(start = 16.dp, end = 8.dp, top = 22.dp, bottom = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier
                .width(4.dp)
                .height(20.dp)
                .clip(RoundedCornerShape(2.dp))
                .background(Brand.Gradient)
        )
        Spacer(Modifier.width(10.dp))
        Text(title, style = MaterialTheme.typography.titleLarge, modifier = Modifier.weight(1f))
        if (onSeeAll != null) {
            Row(
                Modifier
                    .clip(RoundedCornerShape(50))
                    .clickable(onClick = onSeeAll)
                    .padding(horizontal = 8.dp, vertical = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text("See all", color = Brand.TextMuted, style = MaterialTheme.typography.labelLarge)
                Icon(Icons.AutoMirrored.Rounded.ArrowForward, null, tint = Brand.TextMuted, modifier = Modifier.size(16.dp))
            }
        }
    }
}

/** A horizontally scrolling shelf, like the rows on Plex and Netflix. */
@Composable
fun <T> Shelf(
    title: String,
    items: List<T>,
    key: (T) -> Any,
    onSeeAll: (() -> Unit)? = null,
    content: @Composable (T) -> Unit,
) {
    if (items.isEmpty()) return
    SectionHeader(title, onSeeAll = onSeeAll)
    LazyRow(
        contentPadding = PaddingValues(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        items(items, key = key) { content(it) }
    }
}

@Composable
fun ProgressLine(progress: Float, modifier: Modifier = Modifier) {
    Box(
        modifier
            .fillMaxWidth()
            .height(3.dp)
            .background(Color.White.copy(alpha = 0.25f))
    ) {
        Box(
            Modifier
                .fillMaxWidth(progress.coerceIn(0f, 1f))
                .fillMaxHeight()
                .background(Brand.GradientHorizontal)
        )
    }
}

private val thumbScrim = Brush.verticalGradient(
    0.5f to Color.Transparent,
    1f to Color.Black.copy(alpha = 0.75f),
)

/** The 16:9 picture of a scene, with its length, quality and progress. */
@Composable
fun SceneThumb(
    scene: Scene,
    modifier: Modifier = Modifier,
    showProgress: Boolean = true,
    preview: Boolean = false,
) {
    Box(
        modifier
            .aspectRatio(16f / 9f)
            .clip(RoundedCornerShape(12.dp))
            .background(Brand.SurfaceHigh)
    ) {
        if (scene.paths.screenshot != null) {
            AsyncImage(
                model = scene.paths.screenshot,
                contentDescription = null,
                contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxSize(),
            )
        }
        ScenePreview(scene.paths.preview, active = preview, modifier = Modifier.fillMaxSize())
        if (scene.isAudio) {
            Icon(
                Icons.Rounded.MusicNote, null,
                tint = Color.White.copy(alpha = 0.85f),
                modifier = Modifier.align(Alignment.Center).size(34.dp),
            )
        }
        Box(Modifier.fillMaxSize().background(thumbScrim))
        Row(
            Modifier.align(Alignment.TopEnd).padding(6.dp),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            val res = scene.file?.let { resolutionLabel(it.width, it.height) }
            if (res == "4K") Badge(res, gradient = true) else if (res != null) Badge(res)
            if (scene.isAudio) Badge("AUDIO", gradient = true)
        }
        if (scene.duration > 0) {
            Badge(formatDuration(scene.duration), Modifier.align(Alignment.BottomEnd).padding(6.dp))
        }
        if (showProgress && scene.progress > 0.01f) {
            ProgressLine(scene.progress, Modifier.align(Alignment.BottomStart))
        }
    }
}

/**
 * A scene with its title underneath, for shelves and grids. Press and hold
 * to play its preview; press and hold again to stop.
 */
@OptIn(ExperimentalFoundationApi::class)
@Composable
fun SceneCard(scene: Scene, onClick: () -> Unit, modifier: Modifier = Modifier, width: Dp? = 240.dp) {
    val haptics = LocalHapticFeedback.current
    val previewing = PreviewPlayer.activeCardId == scene.id && scene.paths.preview != null
    Column(
        (if (width != null) modifier.width(width) else modifier.fillMaxWidth())
            .clip(RoundedCornerShape(12.dp))
            .combinedClickable(
                onClick = {
                    PreviewPlayer.activeCardId = null
                    onClick()
                },
                onLongClick = {
                    if (scene.paths.preview != null) {
                        haptics.performHapticFeedback(HapticFeedbackType.LongPress)
                        PreviewPlayer.activeCardId = if (previewing) null else scene.id
                    }
                },
            )
    ) {
        SceneThumb(scene, preview = previewing)
        Spacer(Modifier.height(8.dp))
        Text(
            scene.displayTitle,
            style = MaterialTheme.typography.titleSmall,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.padding(horizontal = 2.dp),
        )
        val sub = listOfNotNull(scene.studio?.name, scene.year).joinToString(" • ")
        if (sub.isNotBlank()) {
            Text(
                sub,
                style = MaterialTheme.typography.bodySmall,
                color = Brand.TextMuted,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.padding(horizontal = 2.dp),
            )
        }
        Spacer(Modifier.height(4.dp))
    }
}

/** A round performer picture with a gradient ring. */
@Composable
fun PerformerBubble(
    name: String,
    imagePath: String?,
    onClick: () -> Unit,
    size: Dp = 84.dp,
    subtitle: String? = null,
) {
    Column(
        Modifier.width(size + 12.dp).clickable(onClick = onClick),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .size(size)
                .clip(CircleShape)
                .background(Brand.Gradient)
                .padding(2.5.dp)
                .clip(CircleShape)
                .background(Brand.Background)
                .padding(2.dp)
                .clip(CircleShape)
                .background(Brand.SurfaceHigh),
            contentAlignment = Alignment.Center,
        ) {
            Icon(Icons.Rounded.Person, null, tint = Brand.TextMuted, modifier = Modifier.size(size / 2))
            if (imagePath != null) {
                AsyncImage(
                    model = imagePath,
                    contentDescription = name,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier.fillMaxSize(),
                )
            }
        }
        Spacer(Modifier.height(6.dp))
        Text(name, style = MaterialTheme.typography.labelLarge, maxLines = 1, overflow = TextOverflow.Ellipsis)
        if (subtitle != null) {
            Text(subtitle, style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted, maxLines = 1)
        }
    }
}

@Composable
fun PerformerBubble(p: PerformerRef, onClick: () -> Unit, size: Dp = 84.dp) =
    PerformerBubble(p.name, p.imagePath, onClick, size)

@Composable
fun PerformerBubble(p: Performer, onClick: () -> Unit, size: Dp = 84.dp) =
    PerformerBubble(p.name, p.imagePath, onClick, size, subtitle = "${p.sceneCount} scenes")

/** A studio logo on a dark tile. */
@Composable
fun StudioTile(studio: Studio, onClick: () -> Unit, modifier: Modifier = Modifier, width: Dp? = 180.dp) {
    Column((if (width != null) modifier.width(width) else modifier.fillMaxWidth()).clickable(onClick = onClick)) {
        Box(
            Modifier
                .fillMaxWidth()
                .aspectRatio(16f / 9f)
                .clip(RoundedCornerShape(12.dp))
                .background(Brush.linearGradient(listOf(Brand.SurfaceHigh, Brand.Surface)))
                .border(1.dp, Brand.Outline, RoundedCornerShape(12.dp))
                .padding(18.dp),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                studio.name,
                style = MaterialTheme.typography.titleMedium,
                color = Brand.TextMuted,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            if (studio.imagePath != null && !studio.imagePath.contains("default=true")) {
                AsyncImage(
                    model = studio.imagePath,
                    contentDescription = studio.name,
                    contentScale = ContentScale.Fit,
                    modifier = Modifier.fillMaxSize().background(Brand.Surface),
                )
            }
        }
        Spacer(Modifier.height(6.dp))
        Text(studio.name, style = MaterialTheme.typography.titleSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
        Text("${studio.sceneCount} scenes", style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted)
    }
}

/** 1 to 5 stars; ratings are stored out of 100. */
@Composable
fun RatingStars(rating100: Int?, onRate: (Int?) -> Unit, size: Dp = 28.dp) {
    val stars = ((rating100 ?: 0) + 10) / 20
    Row(horizontalArrangement = Arrangement.spacedBy(2.dp)) {
        for (i in 1..5) {
            val on = i <= stars
            Icon(
                if (on) Icons.Rounded.Star else Icons.Rounded.StarBorder,
                contentDescription = "$i stars",
                tint = if (on) Brand.Orange else Brand.TextMuted,
                modifier = Modifier
                    .size(size)
                    .clip(CircleShape)
                    .clickable { onRate(if (i == stars) null else i * 20) },
            )
        }
    }
}

@Composable
fun Chip(text: String, selected: Boolean = false, onClick: () -> Unit) {
    Box(
        Modifier
            .clip(RoundedCornerShape(50))
            .then(
                if (selected) Modifier.background(Brand.GradientHorizontal)
                else Modifier
                    .background(Brand.Surface)
                    .border(1.dp, Brand.Outline, RoundedCornerShape(50))
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 8.dp)
    ) {
        Text(
            text,
            style = MaterialTheme.typography.labelLarge,
            color = if (selected) Color.White else Brand.Text,
        )
    }
}

@Composable
fun ChipRow(content: @Composable RowScope.() -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 16.dp, vertical = 6.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        content = content,
    )
}
