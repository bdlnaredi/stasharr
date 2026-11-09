package app.stasharr.alerts

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import app.stasharr.StasharrApp
import app.stasharr.data.FavoriteAlert
import app.stasharr.data.OfflineItem

/**
 * Downloads new scenes from your favourite pornstars by themselves, found by
 * the same check as the alerts. Only the newest few are kept: older
 * automatic downloads are deleted. Downloads you start yourself are never
 * touched.
 */
object AutoDownloads {
    var enabled by mutableStateOf(false)
        private set

    /** Wait for Wi-Fi before downloading. */
    var wifiOnly by mutableStateOf(true)
        private set

    /** How many automatic downloads to keep. */
    var keep by mutableIntStateOf(5)
        private set

    /** The server's quality id: original, 1080, 720 or 480. */
    var quality by mutableStateOf("720")
        private set

    val keepChoices = listOf(3, 5, 10, 20)

    private fun prefs(context: Context) = context.getSharedPreferences("auto_downloads", Context.MODE_PRIVATE)

    fun restore(context: Context) {
        val p = prefs(context)
        enabled = p.getBoolean("enabled", false)
        wifiOnly = p.getBoolean("wifi_only", true)
        keep = p.getInt("keep", 5)
        quality = p.getString("quality", "720") ?: "720"
    }

    fun set(context: Context, enabled: Boolean = this.enabled, wifiOnly: Boolean = this.wifiOnly, keep: Int = this.keep, quality: String = this.quality) {
        this.enabled = enabled
        this.wifiOnly = wifiOnly
        this.keep = keep
        this.quality = quality
        prefs(context).edit()
            .putBoolean("enabled", enabled)
            .putBoolean("wifi_only", wifiOnly)
            .putInt("keep", keep)
            .putString("quality", quality)
            .apply()
        FavoriteAlertsState.reschedule(context)
        if (enabled) prune(context.applicationContext as StasharrApp)
    }

    /** Starts downloads for new scenes, newest first, up to the number to keep. */
    suspend fun handle(app: StasharrApp, fresh: List<FavoriteAlert>) {
        if (!enabled || fresh.isEmpty()) return
        val qualities = runCatching { app.repo.offlineQualities() }.getOrDefault(emptyList())
        if (qualities.isEmpty()) return
        val q = qualities.firstOrNull { it.id == quality }
            ?: qualities.filter { it.height > 0 }.minByOrNull { kotlin.math.abs(it.height - (quality.toIntOrNull() ?: 720)) }
            ?: qualities.first()
        val have = app.downloads.items.value.map { it.sceneId }.toSet()
        fresh.map { it.scene }
            .distinctBy { it.id }
            .filter { it.id !in have }
            .take(keep)
            .forEach { s ->
                val full = runCatching { app.repo.scene(s.id) }.getOrNull() ?: s
                app.downloads.start(full, q, auto = true)
            }
        prune(app)
    }

    /** Deletes the oldest automatic downloads beyond the number to keep. */
    fun prune(app: StasharrApp) {
        app.downloads.items.value
            .filter { it.auto && it.state != OfflineItem.FAILED }
            .sortedByDescending { it.addedAt }
            .drop(keep)
            .forEach { app.downloads.delete(it) }
    }
}
