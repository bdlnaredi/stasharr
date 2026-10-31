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

/**
 * The site front end's look in the app: for users the site is set up for,
 * the app takes the site's name, colours and storefront rows.
 */
object SiteMode {
    var look by mutableStateOf<SiteLook?>(null)
        private set

    val active: Boolean get() = look?.active == true

    private fun prefs(context: Context) = context.getSharedPreferences("site_look", Context.MODE_PRIVATE)

    private fun parse(hex: String): Color? = runCatching { Color(android.graphics.Color.parseColor(hex)) }.getOrNull()

    private fun paint(l: SiteLook?) {
        if (l == null || !l.active) {
            Brand.use(DefaultPalette)
            return
        }
        var p = SitePalettes[l.style] ?: DefaultPalette
        parse(l.accent)?.let { a -> p = p.copy(accent = a, accent2 = lerp(a, Color.White, 0.35f)) }
        Brand.use(p)
    }

    /** Uses the look saved last time, so the app opens in the right colours. */
    fun restore(context: Context) {
        val p = prefs(context)
        if (!p.getBoolean("active", false)) return
        val l = SiteLook(
            enabled = true,
            style = p.getString("style", "velvet") ?: "velvet",
            name = p.getString("name", "") ?: "",
            accent = p.getString("accent", "") ?: "",
            modelsGender = p.getString("gender", "FEMALE") ?: "FEMALE",
            appliesToMe = true,
            appLook = true,
        )
        look = l
        paint(l)
    }

    fun apply(context: Context, l: SiteLook?) {
        look = l
        paint(l)
        prefs(context).edit()
            .putBoolean("active", l?.active == true)
            .putString("style", l?.style)
            .putString("name", l?.name)
            .putString("accent", l?.accent)
            .putString("gender", l?.modelsGender)
            .apply()
    }

    fun clear(context: Context) = apply(context, null)
}
