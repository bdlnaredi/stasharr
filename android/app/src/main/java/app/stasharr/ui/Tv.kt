package app.stasharr.ui

import android.app.UiModeManager
import android.content.Context
import android.content.pm.PackageManager
import android.content.res.Configuration
import androidx.compose.foundation.IndicationNodeFactory
import androidx.compose.foundation.interaction.FocusInteraction
import androidx.compose.foundation.interaction.InteractionSource
import androidx.compose.foundation.interaction.PressInteraction
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.ContentDrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.node.DelegatableNode
import androidx.compose.ui.node.DrawModifierNode
import androidx.compose.ui.node.invalidateDraw
import androidx.compose.ui.unit.dp
import app.stasharr.ui.theme.Brand
import kotlinx.coroutines.launch

/** Running on Android TV (or another big screen with a remote). */
object Tv {
    var isTv = false
        private set

    fun detect(context: Context) {
        val ui = context.getSystemService(Context.UI_MODE_SERVICE) as? UiModeManager
        isTv = ui?.currentModeType == Configuration.UI_MODE_TYPE_TELEVISION ||
            context.packageManager.hasSystemFeature(PackageManager.FEATURE_LEANBACK)
    }
}

/**
 * What the remote is on: a pink outline and a light glow around whatever
 * has focus, so it can be seen from the sofa. Used for every clickable on TV.
 */
object TvFocusIndication : IndicationNodeFactory {
    override fun create(interactionSource: InteractionSource): DelegatableNode = TvFocusNode(interactionSource)
    override fun equals(other: Any?): Boolean = other === this
    override fun hashCode(): Int = 7
}

private class TvFocusNode(private val source: InteractionSource) : Modifier.Node(), DrawModifierNode {
    private var focused = false
    private var pressed = false

    override fun onAttach() {
        coroutineScope.launch {
            var focusCount = 0
            var pressCount = 0
            source.interactions.collect { i ->
                when (i) {
                    is FocusInteraction.Focus -> focusCount++
                    is FocusInteraction.Unfocus -> focusCount--
                    is PressInteraction.Press -> pressCount++
                    is PressInteraction.Release, is PressInteraction.Cancel -> pressCount--
                }
                val f = focusCount > 0
                val p = pressCount > 0
                if (f != focused || p != pressed) {
                    focused = f
                    pressed = p
                    invalidateDraw()
                }
            }
        }
    }

    override fun ContentDrawScope.draw() {
        drawContent()
        if (focused || pressed) {
            val w = 3.dp.toPx()
            val r = CornerRadius(12.dp.toPx())
            drawRoundRect(Color.White.copy(alpha = if (pressed) 0.25f else 0.12f), cornerRadius = r)
            drawRoundRect(
                Brand.Pink,
                topLeft = Offset(w / 2, w / 2),
                size = size.copy(width = size.width - w, height = size.height - w),
                cornerRadius = r,
                style = Stroke(width = w),
            )
        }
    }
}
