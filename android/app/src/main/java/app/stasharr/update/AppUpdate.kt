package app.stasharr.update

import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.pm.PackageInstaller
import android.net.Uri
import android.os.Build
import android.provider.Settings
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
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
import java.io.File
import java.util.concurrent.TimeUnit

/**
 * Updates the app from inside the app: looks at the project's releases on
 * GitHub, and when there is a newer Android build, downloads it and hands it
 * to Android's installer (Android asks once to allow installing from
 * Stasharr, then asks to confirm each update).
 */
object AppUpdate {
    private const val RELEASES = "https://api.github.com/repos/bdlnaredi/stasharr/releases?per_page=20"
    private const val TAG_PREFIX = "android-v"
    private const val PREFS = "app_update"
    private const val CHECK_EVERY_MS = 12 * 60 * 60 * 1000L
    internal const val ACTION_INSTALLED = "app.stasharr.update.INSTALL_STATUS"

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

    /** Download progress from 0 to 1 while downloading, otherwise -1. */
    var progress by mutableFloatStateOf(-1f)
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

    /** Whether Android lets Stasharr install apps; when not, opens the setting. */
    private fun mayInstall(context: Context): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return true
        if (context.packageManager.canRequestPackageInstalls()) return true
        message = "Allow Stasharr to install updates in the screen that opened, then come back and tap Update again."
        context.startActivity(
            Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES, Uri.parse("package:${context.packageName}"))
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        )
        return false
    }

    /** Downloads the new version and starts installing it. */
    suspend fun install(context: Context) {
        val release = available ?: return
        val asset = release.apk ?: return
        if (!mayInstall(context)) return
        message = null
        progress = 0f
        val file = File(context.cacheDir, "update.apk")
        val ok = withContext(Dispatchers.IO) {
            runCatching {
                http.newCall(Request.Builder().url(asset.url).header("User-Agent", "Stasharr-Android").build())
                    .execute().use { res ->
                        if (!res.isSuccessful) error("download failed (${res.code})")
                        val body = res.body ?: error("empty download")
                        val total = body.contentLength().takeIf { it > 0 } ?: asset.size
                        file.outputStream().use { out ->
                            body.byteStream().use { input ->
                                val buf = ByteArray(64 * 1024)
                                var done = 0L
                                while (true) {
                                    val n = input.read(buf)
                                    if (n < 0) break
                                    out.write(buf, 0, n)
                                    done += n
                                    if (total > 0) progress = (done.toFloat() / total).coerceIn(0f, 1f)
                                }
                            }
                        }
                    }
                installFile(context, file)
            }
        }
        progress = -1f
        ok.onFailure { message = "Update failed: ${it.message}" }
    }

    private fun installFile(context: Context, file: File) {
        val installer = context.packageManager.packageInstaller
        val params = PackageInstaller.SessionParams(PackageInstaller.SessionParams.MODE_FULL_INSTALL).apply {
            setAppPackageName(context.packageName)
        }
        val id = installer.createSession(params)
        installer.openSession(id).use { session ->
            session.openWrite("stasharr.apk", 0, file.length()).use { out ->
                file.inputStream().use { it.copyTo(out) }
                session.fsync(out)
            }
            val intent = Intent(context, InstallStatusReceiver::class.java).setAction(ACTION_INSTALLED)
            val flags = PendingIntent.FLAG_UPDATE_CURRENT or
                (if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) PendingIntent.FLAG_MUTABLE else 0)
            val pi = PendingIntent.getBroadcast(context, id, intent, flags)
            session.commit(pi.intentSender)
        }
        message = "Installing… Android will ask you to confirm."
    }
}

/** Hears back from Android's installer: asks the person to confirm, or reports a problem. */
class InstallStatusReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        when (intent.getIntExtra(PackageInstaller.EXTRA_STATUS, PackageInstaller.STATUS_FAILURE)) {
            PackageInstaller.STATUS_PENDING_USER_ACTION -> {
                @Suppress("DEPRECATION")
                val confirm = if (Build.VERSION.SDK_INT >= 33) {
                    intent.getParcelableExtra(Intent.EXTRA_INTENT, Intent::class.java)
                } else {
                    intent.getParcelableExtra(Intent.EXTRA_INTENT)
                }
                confirm?.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)?.let { context.startActivity(it) }
            }
            PackageInstaller.STATUS_SUCCESS -> AppUpdate.message = null
            PackageInstaller.STATUS_FAILURE_CONFLICT, PackageInstaller.STATUS_FAILURE_INCOMPATIBLE ->
                AppUpdate.message = "This update cannot replace the installed app (it was installed from a differently signed copy). Uninstall Stasharr, then install the new version from GitHub."
            else -> {
                val why = intent.getStringExtra(PackageInstaller.EXTRA_STATUS_MESSAGE)
                AppUpdate.message = "Update not installed" + (why?.let { ": $it" } ?: ".")
            }
        }
    }
}
