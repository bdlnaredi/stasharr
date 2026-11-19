package app.stasharr.ui

import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * The storefront layouts, the same ones as on the website. A layout changes
 * where things sit and how cards look; the site's style still sets the
 * colours.
 */
data class AppLayout(
    val id: String,
    val name: String,
    val description: String,
    /** corner rounding of cards and pictures */
    val corner: Dp = 12.dp,
    /** thick outline and a hard offset shadow (Brutal) */
    val hard: Boolean = false,
    /** the title sits on the picture instead of under it */
    val infoInside: Boolean = false,
    /** serif headlines (Editorial) */
    val serif: Boolean = false,
    /** capital-letter titles (Brutal) */
    val caps: Boolean = false,
    /** black-and-white portraits (Editorial) */
    val grayPortraits: Boolean = false,
    /** frosted glass panels (Immersive) */
    val glass: Boolean = false,
)

object AppLayouts {
    val all = listOf(
        AppLayout("classic", "Classic", "The banner on top, then rows of scenes."),
        AppLayout("bento", "Bento", "A home of mixed-size tiles and rounded cards with the words inside.", corner = 22.dp, infoInside = true),
        AppLayout("mosaic", "Mosaic", "A masonry wall of scenes and pornstars, with category chips.", corner = 16.dp, infoInside = true),
        AppLayout("spotlight", "Spotlight", "A big chosen scene with a numbered top-10 list under it; lists become numbered.", corner = 8.dp),
        AppLayout("showcase", "Showcase", "A 3D cover-flow of featured scenes; cards lean in when pressed.", corner = 18.dp),
        AppLayout("immersive", "Immersive", "The featured scene fills the screen with frosted panels over it.", corner = 18.dp, glass = true),
        AppLayout("editorial", "Editorial", "A magazine: cover story, “In this issue”, serif headlines, black-and-white portraits.", corner = 0.dp, serif = true, grayPortraits = true),
        AppLayout("brutal", "Brutal", "Thick borders, hard shadows, giant capitals and a scrolling marquee.", corner = 0.dp, hard = true, caps = true),
        AppLayout("swipe", "Swipe", "Swipe through a deck of scenes: right to like, left to skip.", corner = 26.dp, infoInside = true),
    )

    fun find(id: String?): AppLayout = all.firstOrNull { it.id == id } ?: all.first()

    /** The layout in use now. */
    val current: AppLayout get() = find(SiteMode.layout)
}
