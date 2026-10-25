package app.stasharr.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

object Brand {
    val Pink = Color(0xFFFF3D7F)
    val Orange = Color(0xFFFFA53D)
    val Background = Color(0xFF0B0B10)
    val Surface = Color(0xFF15151D)
    val SurfaceHigh = Color(0xFF1F1F2A)
    val Outline = Color(0xFF2C2C3A)
    val Text = Color(0xFFF4F4F8)
    val TextMuted = Color(0xFFA0A0B2)

    /** The pink to orange gradient used for buttons, accents and the logo. */
    val Gradient = Brush.linearGradient(listOf(Pink, Orange))
    val GradientHorizontal = Brush.horizontalGradient(listOf(Pink, Orange))
}

private val Colors = darkColorScheme(
    primary = Brand.Pink,
    onPrimary = Color.White,
    secondary = Brand.Orange,
    onSecondary = Color.Black,
    tertiary = Brand.Orange,
    background = Brand.Background,
    onBackground = Brand.Text,
    surface = Brand.Background,
    onSurface = Brand.Text,
    surfaceVariant = Brand.Surface,
    onSurfaceVariant = Brand.TextMuted,
    surfaceContainer = Brand.Surface,
    surfaceContainerHigh = Brand.SurfaceHigh,
    surfaceContainerHighest = Brand.SurfaceHigh,
    surfaceContainerLow = Brand.Surface,
    surfaceContainerLowest = Brand.Background,
    outline = Brand.Outline,
    outlineVariant = Brand.Outline,
    secondaryContainer = Brand.SurfaceHigh,
    onSecondaryContainer = Brand.Text,
    error = Color(0xFFFF5A5F),
)

private val base = Typography()

private val Type = Typography(
    displaySmall = base.displaySmall.copy(fontWeight = FontWeight.Black, letterSpacing = (-0.5).sp),
    headlineLarge = base.headlineLarge.copy(fontWeight = FontWeight.ExtraBold, letterSpacing = (-0.5).sp),
    headlineMedium = base.headlineMedium.copy(fontWeight = FontWeight.ExtraBold, letterSpacing = (-0.3).sp),
    headlineSmall = base.headlineSmall.copy(fontWeight = FontWeight.Bold),
    titleLarge = base.titleLarge.copy(fontWeight = FontWeight.Bold),
    titleMedium = base.titleMedium.copy(fontWeight = FontWeight.SemiBold),
    titleSmall = base.titleSmall.copy(fontWeight = FontWeight.SemiBold),
    labelLarge = base.labelLarge.copy(fontWeight = FontWeight.Bold, letterSpacing = 0.3.sp),
    labelSmall = TextStyle(fontSize = 10.sp, fontWeight = FontWeight.Bold, letterSpacing = 0.6.sp),
)

@Composable
fun StasharrTheme(content: @Composable () -> Unit) {
    // always dark, like the big streaming apps
    MaterialTheme(colorScheme = Colors, typography = Type, content = content)
}
