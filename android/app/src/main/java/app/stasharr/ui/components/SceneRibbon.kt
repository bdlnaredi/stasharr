package app.stasharr.ui.components

import androidx.compose.foundation.Canvas
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

// the flags' colours, top to bottom
private val ribbonColors = mapOf(
    "trans" to listOf(Color(0xFF5BCEFA), Color(0xFFF5A9B8), Color.White, Color(0xFFF5A9B8), Color(0xFF5BCEFA)),
    "lesbian" to listOf(Color(0xFFD52D00), Color(0xFFFF9A56), Color.White, Color(0xFFD362A4), Color(0xFFA30262)),
    "bisexual" to listOf(Color(0xFFD60270), Color(0xFFD60270), Color(0xFF9B4F96), Color(0xFF0038A8), Color(0xFF0038A8)),
)

/**
 * A diagonal ribbon across the top left corner of a scene's picture saying
 * it is a trans, lesbian or bisexual scene, in the colours of that flag.
 */
@Composable
fun SceneRibbon(badges: List<String>, modifier: Modifier = Modifier) {
    val list = badges.filter { it in ribbonLabels }
    if (list.isEmpty()) return
    val first = list.first()
    val label = list.joinToString(" · ") { ribbonLabels.getValue(it) }
    val measurer = rememberTextMeasurer()
    val dark = first != "bisexual"
    Canvas(modifier.size(66.dp)) {
        val band = 15.dp.toPx()
        val d = size.width * 0.72f
        val c = Offset(d / 2, d / 2)
        val length = d * 1.6f
        val text = measurer.measure(
            label,
            TextStyle(
                color = if (dark) Color(0xFF1D2A3A) else Color.White,
                fontSize = 8.sp,
                fontWeight = FontWeight.ExtraBold,
                letterSpacing = 1.sp,
            ),
        )
        clipRect {
            rotate(-45f, pivot = c) {
                val top = c.y - band / 2
                drawRect(Color.Black.copy(alpha = 0.35f), topLeft = Offset(c.x - length / 2, top + 2.dp.toPx()), size = Size(length, band))
                drawRect(
                    Brush.verticalGradient(ribbonColors.getValue(first), startY = top, endY = top + band),
                    topLeft = Offset(c.x - length / 2, top),
                    size = Size(length, band),
                )
                drawText(text, topLeft = Offset(c.x - text.size.width / 2f, c.y - text.size.height / 2f))
            }
        }
    }
}
