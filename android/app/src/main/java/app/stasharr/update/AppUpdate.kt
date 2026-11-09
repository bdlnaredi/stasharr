package app.stasharr.update

import android.content.Context
import android.content.Intent
import android.net.Uri
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import app.stasharr.BuildConfig
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import java.util.concurrent.TimeUnit

/**
 * Tells you when there is a newer app: looks at the project's releases on
 * GitHub, shows what's new, and opens the download in the browser. Android
 * installs it from there, as with the first install.
 */
object AppUpdate {
    private const val RELEASES = "https://api.github.com/repos/bdlnaredi/stasharr/releases?per_page=20"
    private const val TAG_PREFIX = "android-v"
    private const val PREFS = "app_update"
    private const val CHECK_EVERY_MS = 12 * 60 * 60 * 1000L

    @Serializable
    data class Asset(
        val name: String,
        @SerialName("browser_download_url") val url: String,
        val size: Long = 0,
    )

    @Serializable
    data class Release(
        @SerialName("tag_name") val tag: String,
        val name: String? = null,
        val body: String? = null,
        val draft: Boolean = false,
        val prerelease: Boolean = false,
        val assets: List<Asset> = emptyList(),
    ) {
        val version: String get() = tag.removePrefix(TAG_PREFIX)
        val apk: Asset? get() = assets.firstOrNull { it.name.endsWith(".apk") }
    }

    /** A newer release, once found. */
    var available by mutableStateOf<Release?>(null)
        private set

    /** Whether the update dialog is showing. */
    var showDialog by mutableStateOf(false)

    var checking by mutableStateOf(false)
        private set

    /** A message for the dialog: errors, or what to do next. */
    var message by mutableStateOf<String?>(null)

    val current: String get() = BuildConfig.VERSION_NAME

    private val json = Json { ignoreUnknownKeys = true }
    private val http by lazy {
        OkHttpClient.Builder()
            .connectTimeout(20, TimeUnit.SECONDS)
            .readTimeout(60, TimeUnit.SECONDS)
            .build()
    }

    /** 1.0.42 → [1, 0, 42]; compares part by part. */
    fun newer(a: String, b: String): Boolean {
        val x = a.split('.', '-').map { it.toIntOrNull() ?: 0 }
        val y = b.split('.', '-').map { it.toIntOrNull() ?: 0 }
        for (i in 0 until maxOf(x.size, y.size)) {
            val p = x.getOrElse(i) { 0 }
            val q = y.getOrElse(i) { 0 }
            if (p != q) return p > q
        }
        return false
    }

    private fun prefs(context: Context) = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    /**
     * Looks for a newer release. auto: the check on opening the app, done
     * at most every 12 hours and quiet about versions put off with Later.
     */
    suspend fun check(context: Context, auto: Boolean) {
        val p = prefs(context)
        if (auto && System.currentTimeMillis() - p.getLong("checked_at", 0) < CHECK_EVERY_MS) {
            // still offer what was found last time
            p.getString("found", null)?.let { saved ->
                runCatching { json.decodeFromString<Release>(saved) }.getOrNull()?.let { r ->
                    if (newer(r.version, current)) {
                        available = r
                        if (p.getString("later", null) != r.version) showDialog = true
                    }
                }
            }
            return
        }
        checking = true
        message = null
        val found = withContext(Dispatchers.IO) {
            runCatching {
                http.newCall(
                    Request.Builder().url(RELEASES)
                        .header("Accept", "application/vnd.github+json")
                        .header("User-Agent", "Stasharr-Android")
                        .build()
                ).execute().use { res ->
                    if (!res.isSuccessful) error("GitHub answered ${res.code}")
                    json.decodeFromString<List<Release>>(res.body?.string().orEmpty())
                }
            }
        }
        checking = false
        found.onFailure {
            if (!auto) message = "Could not check for updates: ${it.message}"
            return
        }
        p.edit().putLong("checked_at", System.currentTimeMillis()).apply()
        val latest = found.getOrThrow()
            .filter { !it.draft && !it.prerelease && it.tag.startsWith(TAG_PREFIX) && it.apk != null }
            .maxWithOrNull { a, b -> if (newer(a.version, b.version)) 1 else if (a.version == b.version) 0 else -1 }
        if (latest != null && newer(latest.version, current)) {
            available = latest
            p.edit().putString("found", json.encodeToString(Release.serializer(), latest)).apply()
            if (!auto || p.getString("later", null) != latest.version) showDialog = true
        } else {
            available = null
            p.edit().remove("found").apply()
            if (!auto) {
                message = "You have the latest version ($current)."
                showDialog = true
            }
        }
    }

    /** Put off this version: no dialog on opening the app until a newer one. */
    fun later(context: Context) {
        available?.let { prefs(context).edit().putString("later", it.version).apply() }
        showDialog = false
        message = null
    }

    /**
     * Opens the new version's download in the browser. Android then offers
     * to install it when it has downloaded; the app itself never installs
     * anything, which keeps Play Protect happy.
     */
    fun install(context: Context) {
        val asset = available?.apk ?: return
        runCatching {
            context.startActivity(
                Intent(Intent.ACTION_VIEW, Uri.parse(asset.url)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            )
        }.onSuccess {
            message = "Downloading in your browser. When it finishes, tap the download to install; your login is kept."
        }.onFailure {
            message = "Could not open the download: ${it.message}"
        }
    }
}
