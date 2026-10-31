package app.stasharr.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

/** A colour set the app can wear: its own, or one of the site styles. */
data class Palette(
    val accent: Color,
    val accent2: Color,
    val background: Color,
    val surface: Color,
    val surfaceHigh: Color,
    val outline: Color,
    val text: Color,
    val textMuted: Color,
)

val DefaultPalette = Palette(
    accent = Color(0xFFFF3D7F),
    accent2 = Color(0xFFFFA53D),
    background = Color(0xFF0B0B10),
    surface = Color(0xFF15151D),
    surfaceHigh = Color(0xFF1F1F2A),
    outline = Color(0xFF2C2C3A),
    text = Color(0xFFF4F4F8),
    textMuted = Color(0xFFA0A0B2),
)

/** The site styles, in dark form for the app. */
val SitePalettes = mapOf(
    "velvet" to Palette(Color(0xFFC9A45C), Color(0xFFF0D695), Color(0xFF0B0A09), Color(0xFF15130F), Color(0xFF1F1B16), Color(0xFF3A3226), Color(0xFFF3EDE2), Color(0xFFA59B8B)),
    "neon" to Palette(Color(0xFFFF2E93), Color(0xFF21E6FF), Color(0xFF0D0718), Color(0xFF170D29), Color(0xFF22133B), Color(0xFF3A2457), Color(0xFFF5ECFF), Color(0xFFA491C7)),
    "crimson" to Palette(Color(0xFFE10600), Color(0xFFFF4D3D), Color(0xFF111111), Color(0xFF1A1A1A), Color(0xFF242424), Color(0xFF333333), Color(0xFFFFFFFF), Color(0xFF9A9A9A)),
    "blush" to Palette(Color(0xFFE84A7F), Color(0xFFFF9EB9), Color(0xFF140C10), Color(0xFF1F141A), Color(0xFF2A1B23), Color(0xFF3D2832), Color(0xFFFFF7F9), Color(0xFFC4A3B2)),
    "onyx" to Palette(Color(0xFF8C8C8C), Color(0xFFD9D9D9), Color(0xFF000000), Color(0xFF0D0D0D), Color(0xFF161616), Color(0xFF2A2A2A), Color(0xFFFFFFFF), Color(0xFF8A8A8A)),
)

/**
 * The app's colours. They follow the site's style for users the site is set
 * up for, and change on screen as soon as the style is known.
 */
object Brand {
    private var palette by mutableStateOf(DefaultPalette)

    fun use(p: Palette) {
        palette = p
    }

    val Pink: Color get() = palette.accent
    val Orange: Color get() = palette.accent2
    val Background: Color get() = palette.background
    val Surface: Color get() = palette.surface
    val SurfaceHigh: Color get() = palette.surfaceHigh
    val Outline: Color get() = palette.outline
    val Text: Color get() = palette.text
    val TextMuted: Color get() = palette.textMuted

    /** The accent gradient used for buttons, accents and the logo. */
    val Gradient: Brush get() = Brush.linearGradient(listOf(Pink, Orange))
    val GradientHorizontal: Brush get() = Brush.horizontalGradient(listOf(Pink, Orange))
}

private fun colors() = darkColorScheme(
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
    MaterialTheme(colorScheme = colors(), typography = Type, content = content)
}
