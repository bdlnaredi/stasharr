package app.stasharr.ui

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.lerp
import app.stasharr.data.SiteLook
import app.stasharr.ui.theme.Brand
import app.stasharr.ui.theme.DefaultPalette
import app.stasharr.ui.theme.SitePalettes
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/**
 * The site front end in the app. Users the site is set up for get the
 * storefront: the site's name, colours, home rows and pages. Anyone else can
 * switch it on under You to preview it.
 */
object SiteMode {
    var look by mutableStateOf<SiteLook?>(null)
        private set

    /** The storefront switched on by hand, for admins to preview it. */
    var preview by mutableStateOf(false)
        private set

    /** Whether the person confirmed they are an adult (asked once). */
    var adultOk by mutableStateOf(false)
        private set

    /** Whether the app shows the storefront. */
    val storefront: Boolean get() = look?.let { it.enabled && (it.active || preview) } == true

    /** Kept for older screens: the storefront is on. */
    val active: Boolean get() = storefront

    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private fun prefs(context: Context) = context.getSharedPreferences("site_look", Context.MODE_PRIVATE)

    private fun parse(hex: String): Color? = runCatching { Color(android.graphics.Color.parseColor(hex)) }.getOrNull()

    private fun paint() {
        val l = look
        if (l == null || !storefront) {
            Brand.use(DefaultPalette)
            return
        }
        var p = SitePalettes[l.style] ?: DefaultPalette
        parse(l.accent)?.let { a -> p = p.copy(accent = a, accent2 = lerp(a, Color.White, 0.35f)) }
        Brand.use(p)
    }

    /** Uses the look saved last time, so the app opens as it was. */
    fun restore(context: Context) {
        val p = prefs(context)
        preview = p.getBoolean("preview", false)
        adultOk = p.getBoolean("adult_ok", false)
        look = p.getString("look", null)?.let { runCatching { json.decodeFromString<SiteLook>(it) }.getOrNull() }
        paint()
    }

    fun apply(context: Context, l: SiteLook?) {
        look = l
        paint()
        prefs(context).edit().putString("look", l?.let { json.encodeToString(it) }).apply()
    }

    fun setPreview(context: Context, on: Boolean) {
        preview = on
        paint()
        prefs(context).edit().putBoolean("preview", on).apply()
    }

    fun confirmAdult(context: Context) {
        adultOk = true
        prefs(context).edit().putBoolean("adult_ok", true).apply()
    }

    fun clear(context: Context) {
        look = null
        preview = false
        paint()
        prefs(context).edit().clear().apply()
    }
}
