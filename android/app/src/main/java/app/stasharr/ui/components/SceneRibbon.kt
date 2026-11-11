package app.stasharr.ui.components

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.padding
import androidx.compose.ui.draw.clip
import androidx.compose.foundation.layout.size
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.clipRect
import androidx.compose.ui.graphics.drawscope.rotate
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.drawText
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

private val ribbonLabels = mapOf("trans" to "TRANS", "lesbian" to "LESBIAN", "bisexual" to "BI")

// deep red with a thin gold edge, white lettering
private val ribbonRed = listOf(Color(0xFFC8102E), Color(0xFF9B0A20), Color(0xFF6E0014))
private val ribbonGold = Color(0xFFD4AF37)

/**
 * A diagonal ribbon across the top left corner of a scene's picture saying
 * it is a trans, lesbian or bisexual scene: deep red, gold edges, white text.
 */
@Composable
fun SceneRibbon(badges: List<String>, modifier: Modifier = Modifier) {
    val list = badges.filter { it in ribbonLabels }
    if (list.isEmpty()) return
    val label = list.joinToString(" · ") { ribbonLabels.getValue(it) }
    val measurer = rememberTextMeasurer()
    Canvas(modifier.size(66.dp)) {
        val band = 15.dp.toPx()
        val d = size.width * 0.72f
        val c = Offset(d / 2, d / 2)
        val length = d * 1.6f
        val text = measurer.measure(
            label,
            TextStyle(
                color = Color.White,
                fontSize = 8.sp,
                fontWeight = FontWeight.ExtraBold,
                letterSpacing = 1.5.sp,
            ),
        )
        clipRect {
            rotate(-45f, pivot = c) {
                val top = c.y - band / 2
                drawRect(Color.Black.copy(alpha = 0.35f), topLeft = Offset(c.x - length / 2, top + 2.dp.toPx()), size = Size(length, band))
                drawRect(
                    Brush.verticalGradient(ribbonRed, startY = top, endY = top + band),
                    topLeft = Offset(c.x - length / 2, top),
                    size = Size(length, band),
                )
                val edge = 1.dp.toPx()
                drawRect(ribbonGold, topLeft = Offset(c.x - length / 2, top), size = Size(length, edge))
                drawRect(ribbonGold, topLeft = Offset(c.x - length / 2, top + band - edge), size = Size(length, edge))
                drawText(text, topLeft = Offset(c.x - text.size.width / 2f, c.y - text.size.height / 2f))
            }
        }
    }
}

/** The same as small red labels, under a scene's title. */
@Composable
fun SceneBadgePills(badges: List<String>, modifier: Modifier = Modifier) {
    val list = badges.filter { it in ribbonLabels }
    if (list.isEmpty()) return
    androidx.compose.foundation.layout.Row(modifier, horizontalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(6.dp)) {
        list.forEach { b ->
            androidx.compose.material3.Text(
                ribbonLabels.getValue(b),
                color = Color.White,
                fontSize = 10.sp,
                fontWeight = FontWeight.Bold,
                letterSpacing = 1.2.sp,
                modifier = Modifier
                    .clip(androidx.compose.foundation.shape.RoundedCornerShape(4.dp))
                    .background(Brush.verticalGradient(ribbonRed))
                    .border(1.dp, ribbonGold, androidx.compose.foundation.shape.RoundedCornerShape(4.dp))
                    .padding(horizontal = 7.dp, vertical = 1.dp),
            )
        }
    }
}
