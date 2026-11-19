package app.stasharr.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ColorFilter
import androidx.compose.ui.graphics.ColorMatrix
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.stasharr.data.Performer
import app.stasharr.data.Scene
import app.stasharr.player.ScenePreview
import app.stasharr.ui.AppLayouts
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage

/** Brutal's hard shadow: a solid block offset down and right. */
fun Modifier.hardShadow(color: Color, offset: Dp = 5.dp): Modifier = drawBehind {
    val o = offset.toPx()
    drawRect(color, topLeft = Offset(o, o), size = size)
}

/** Brutal's thick outline and hard shadow, or nothing in the other layouts. */
@Composable
fun Modifier.layoutFrame(shadow: Dp = 5.dp): Modifier {
    val lay = AppLayouts.current
    return if (lay.hard) this.hardShadow(Brand.Text, shadow).border(3.dp, Brand.Text).background(Brand.Surface) else this
}

/** Editorial's black-and-white portraits. */
val grayscale: ColorFilter = ColorFilter.colorMatrix(ColorMatrix().apply { setToSaturation(0f) })

@Composable
fun portraitFilter(): ColorFilter? = if (AppLayouts.current.grayPortraits) grayscale else null

/** Spotlight's outlined numbers: 01, 02 … */
@Composable
fun OutlinedNumber(n: Int, size: Int = 34, filled: Boolean = false) {
    Text(
        n.toString().padStart(2, '0'),
        style = MaterialTheme.typography.headlineLarge.copy(
            fontSize = size.sp,
            fontWeight = FontWeight.Black,
            drawStyle = if (filled) null else Stroke(width = 3f),
        ),
        color = if (filled) Brand.Pink else Brand.TextMuted,
    )
}

/** Spotlight: a numbered row with the picture, title and who is in it. */
@Composable
fun SpotRow(scene: Scene, n: Int, selected: Boolean = false, onClick: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .background(if (selected) Brand.Surface else Color.Transparent)
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Box(Modifier.width(44.dp), contentAlignment = Alignment.Center) { OutlinedNumber(n, 28, filled = selected) }
        SceneThumb(scene, Modifier.width(128.dp), showProgress = false)
        Column(Modifier.weight(1f)) {
            Text(scene.displayTitle, style = MaterialTheme.typography.titleSmall, maxLines = 2, overflow = TextOverflow.Ellipsis)
            if (scene.performers.isNotEmpty()) {
                Text(
                    scene.performers.take(2).joinToString(" & ") { it.name },
                    style = MaterialTheme.typography.bodySmall,
                    color = Brand.Pink,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Text(scene.metaLine(), style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}

/** Mosaic: a card of any shape with the title on the picture. */
@Composable
fun MosaicCard(scene: Scene, aspect: Float, onClick: () -> Unit, modifier: Modifier = Modifier) {
    Box(
        modifier
            .fillMaxWidth()
            .aspectRatio(aspect)
            .clip(RoundedCornerShape(16.dp))
            .background(Brand.SurfaceHigh)
            .clickable(onClick = onClick)
    ) {
        if (scene.paths.screenshot != null) {
            AsyncImage(scene.paths.screenshot, null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
        }
        Box(Modifier.fillMaxSize().background(Brush.verticalGradient(0.45f to Color.Transparent, 1f to Color.Black.copy(alpha = 0.88f))))
        SceneRibbon(scene.badges, Modifier.align(Alignment.TopStart))
        Column(Modifier.align(Alignment.BottomStart).padding(10.dp)) {
            Text(scene.displayTitle, color = Color.White, style = MaterialTheme.typography.titleSmall, maxLines = 2, overflow = TextOverflow.Ellipsis)
            val sub = listOfNotNull(scene.performers.firstOrNull()?.name, scene.duration.takeIf { it > 0 }?.let { formatDuration(it) })
            if (sub.isNotEmpty()) {
                Text(sub.joinToString(" · "), color = Color.White.copy(alpha = 0.75f), style = MaterialTheme.typography.bodySmall, maxLines = 1)
            }
        }
    }
}

/** Mosaic: a pornstar in the wall, with an accent outline. */
@Composable
fun MosaicModel(p: Performer, onClick: () -> Unit) {
    Box(
        Modifier
            .fillMaxWidth()
            .aspectRatio(2f / 3f)
            .clip(RoundedCornerShape(16.dp))
            .border(2.dp, Brand.Pink, RoundedCornerShape(16.dp))
            .background(Brand.SurfaceHigh)
            .clickable(onClick = onClick)
    ) {
        if (p.imagePath != null) {
            AsyncImage(p.imagePath, null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
        }
        Box(Modifier.fillMaxSize().background(Brush.verticalGradient(0.5f to Color.Transparent, 1f to Color.Black.copy(alpha = 0.85f))))
        Text(
            "PORNSTAR",
            color = Color.White,
            style = MaterialTheme.typography.labelSmall,
            modifier = Modifier.align(Alignment.TopStart).padding(8.dp).clip(RoundedCornerShape(50)).background(Color.Black.copy(alpha = 0.5f)).padding(horizontal = 8.dp, vertical = 3.dp),
        )
        Column(Modifier.align(Alignment.BottomStart).padding(10.dp)) {
            Text(p.name, color = Color.White, style = MaterialTheme.typography.titleMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
            Text("${p.sceneCount} scenes", color = Color.White.copy(alpha = 0.75f), style = MaterialTheme.typography.bodySmall)
        }
    }
}

// The shapes repeat so each pair of columns comes out about the same height.
private val leftShapes = listOf(4f / 5f, 16f / 10f, 1f)
private val rightShapes = listOf(1f, 3f / 4f, 16f / 10f)

/**
 * Mosaic: six scenes laid as two uneven columns, so the page looks like a
 * masonry wall. A pornstar can take the first right-hand spot.
 */
@Composable
fun MosaicChunk(scenes: List<Scene>, onScene: (Scene) -> Unit, model: Performer? = null, onModel: (Performer) -> Unit = {}) {
    val left = scenes.filterIndexed { i, _ -> i % 2 == 0 }
    val right = scenes.filterIndexed { i, _ -> i % 2 == 1 }
    Row(Modifier.fillMaxWidth().padding(horizontal = 12.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            left.forEachIndexed { i, s -> MosaicCard(s, leftShapes[i % 3], onClick = { onScene(s) }) }
        }
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            if (model != null) MosaicModel(model, onClick = { onModel(model) })
            right.forEachIndexed { i, s -> MosaicCard(s, rightShapes[(i + if (model != null) 1 else 0) % 3], onClick = { onScene(s) }) }
        }
    }
    Spacer(Modifier.height(10.dp))
}
