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
import androidx.compose.foundation.LocalIndication
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.graphics.RectangleShape
import androidx.compose.ui.graphics.drawscope.Stroke
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
import app.stasharr.player.AutoPreview
import app.stasharr.player.PreviewPlayer
import app.stasharr.player.previewSpot
import app.stasharr.player.rememberPreviewKey
import app.stasharr.player.ScenePreview
import app.stasharr.ui.AppLayouts
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
    val lay = AppLayouts.current
    // Editorial: a thin rule above an italic serif heading
    if (lay.serif) {
        Box(Modifier.padding(start = 16.dp, end = 16.dp, top = 22.dp).fillMaxWidth().height(1.dp).background(Brand.Text))
    }
    Row(
        modifier.fillMaxWidth().padding(start = 16.dp, end = 8.dp, top = if (lay.serif) 8.dp else 22.dp, bottom = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        when (lay.id) {
            "classic", "bento", "showcase", "immersive", "swipe" -> {
                Box(
                    Modifier
                        .width(4.dp)
                        .height(20.dp)
                        .clip(RoundedCornerShape(2.dp))
                        .background(Brand.Gradient)
                )
                Spacer(Modifier.width(10.dp))
            }
        }
        when {
            // Brutal: the heading in a solid block
            lay.hard -> Box(Modifier.weight(1f)) {
                Text(
                    title.uppercase(),
                    style = MaterialTheme.typography.titleLarge,
                    color = Brand.Background,
                    modifier = Modifier.background(Brand.Text).padding(horizontal = 10.dp, vertical = 2.dp),
                )
            }
            // Spotlight: big outlined capitals
            lay.id == "spotlight" -> Text(
                title.uppercase(),
                style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Black, drawStyle = Stroke(width = 2.5f)),
                modifier = Modifier.weight(1f),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            lay.id == "mosaic" -> Text(title, style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Black), modifier = Modifier.weight(1f))
            else -> Text(title, style = MaterialTheme.typography.titleLarge, modifier = Modifier.weight(1f))
        }
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
    corner: Dp = AppLayouts.current.corner,
) {
    Box(
        modifier
            .aspectRatio(16f / 9f)
            .clip(if (corner == 0.dp) RectangleShape else RoundedCornerShape(corner))
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
        SceneRibbon(scene.badges, Modifier.align(Alignment.TopStart))
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
    val key = rememberPreviewKey("card")
    val previewing = PreviewPlayer.activeCardId == key && scene.paths.preview != null
    val lay = AppLayouts.current
    val shape = if (lay.corner == 0.dp) RectangleShape else RoundedCornerShape(lay.corner)

    // Showcase leans the card back as you press it; Brutal presses it in
    val press = remember { MutableInteractionSource() }
    val pressed by press.collectIsPressedAsState()
    val lean by animateFloatAsState(if (pressed && lay.id == "showcase") 1f else 0f, label = "lean")
    val sink by animateFloatAsState(if (pressed && lay.hard) 1f else 0f, label = "sink")

    Column(
        (if (width != null) modifier.width(width) else modifier.fillMaxWidth())
            .graphicsLayer {
                rotationX = lean * 10f
                scaleX = 1f - lean * 0.04f
                scaleY = 1f - lean * 0.04f
                cameraDistance = 14f * density
                translationX = sink * 4.dp.toPx()
                translationY = sink * 4.dp.toPx()
            }
            .layoutFrame(shadow = (5f * (1f - sink)).dp)
            .clip(shape)
            .combinedClickable(
                interactionSource = press,
                indication = LocalIndication.current,
                onClick = {
                    PreviewPlayer.activeCardId = null
                    onClick()
                },
                onLongClick = {
                    if (scene.paths.preview != null) {
                        haptics.performHapticFeedback(HapticFeedbackType.LongPress)
                        PreviewPlayer.activeCardId = if (previewing) null else key
                    }
                },
            )
    ) {
        val thumbMod = Modifier.previewSpot(key, AutoPreview.enabled && scene.paths.preview != null)
        val title = if (lay.caps) scene.displayTitle.uppercase() else scene.displayTitle
        val sub = listOfNotNull(scene.studio?.name, scene.year).joinToString(" • ")
        if (lay.infoInside) {
            // the words sit on the picture: a frosted pill in Swipe, a fade elsewhere
            Box {
                SceneThumb(scene, preview = previewing, modifier = thumbMod)
                Column(
                    Modifier
                        .align(Alignment.BottomStart)
                        .fillMaxWidth()
                        .then(
                            if (lay.id == "swipe") Modifier.padding(8.dp).clip(RoundedCornerShape(16.dp)).background(Brand.Background.copy(alpha = 0.72f))
                            else Modifier.background(Brush.verticalGradient(listOf(Color.Transparent, Color.Black.copy(alpha = 0.8f))))
                        )
                        .padding(horizontal = 10.dp, vertical = 8.dp)
                ) {
                    Text(title, style = MaterialTheme.typography.titleSmall, color = if (lay.id == "swipe") Brand.Text else Color.White, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    if (sub.isNotBlank()) {
                        Text(sub, style = MaterialTheme.typography.bodySmall, color = (if (lay.id == "swipe") Brand.TextMuted else Color.White.copy(alpha = 0.75f)), maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                }
            }
            SceneBadgePills(scene.badges, Modifier.padding(horizontal = 2.dp, vertical = 2.dp))
            Spacer(Modifier.height(2.dp))
        } else {
            SceneThumb(scene, preview = previewing, modifier = thumbMod)
            Spacer(Modifier.height(8.dp))
            val pad = if (lay.hard) 10.dp else 2.dp
            Text(
                title,
                style = MaterialTheme.typography.titleSmall,
                maxLines = if (lay.serif) 2 else 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.padding(horizontal = pad),
            )
            SceneBadgePills(scene.badges, Modifier.padding(horizontal = pad, vertical = 2.dp))
            if (sub.isNotBlank()) {
                Text(
                    if (lay.serif) sub.uppercase() else sub,
                    style = MaterialTheme.typography.bodySmall.copy(letterSpacing = if (lay.serif) 1.2.sp else MaterialTheme.typography.bodySmall.letterSpacing),
                    color = Brand.TextMuted,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.padding(horizontal = pad),
                )
            }
            Spacer(Modifier.height(if (lay.hard) 10.dp else 4.dp))
        }
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
    val lay = AppLayouts.current
    // round with a ring, except: square and outlined in Brutal, a soft
    // squircle in Swipe and Bento, a square portrait in Editorial
    val shape = when {
        lay.hard || lay.serif -> RectangleShape
        lay.id == "swipe" || lay.id == "bento" -> RoundedCornerShape(30)
        else -> CircleShape
    }
    Column(
        Modifier.width(size + 12.dp).clickable(onClick = onClick),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .size(size)
                .then(
                    if (lay.hard) Modifier.hardShadow(Brand.Text, 4.dp).border(3.dp, Brand.Text).background(Brand.SurfaceHigh)
                    else if (lay.serif) Modifier.background(Brand.SurfaceHigh)
                    else Modifier
                        .clip(shape)
                        .background(Brand.Gradient)
                        .padding(2.5.dp)
                        .clip(shape)
                        .background(Brand.Background)
                        .padding(2.dp)
                        .clip(shape)
                        .background(Brand.SurfaceHigh)
                )
                .clip(shape),
            contentAlignment = Alignment.Center,
        ) {
            Icon(Icons.Rounded.Person, null, tint = Brand.TextMuted, modifier = Modifier.size(size / 2))
            if (imagePath != null) {
                AsyncImage(
                    model = imagePath,
                    contentDescription = name,
                    contentScale = ContentScale.Crop,
                    colorFilter = portraitFilter(),
                    modifier = Modifier.fillMaxSize(),
                )
            }
        }
        Spacer(Modifier.height(if (lay.hard) 8.dp else 6.dp))
        Text(
            if (lay.caps) name.uppercase() else name,
            style = if (lay.serif) MaterialTheme.typography.titleSmall else MaterialTheme.typography.labelLarge,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
        )
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
                .layoutFrame()
                .clip(RoundedCornerShape(AppLayouts.current.corner))
                .background(Brush.linearGradient(listOf(Brand.SurfaceHigh, Brand.Surface)))
                .border(1.dp, Brand.Outline, RoundedCornerShape(AppLayouts.current.corner))
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
