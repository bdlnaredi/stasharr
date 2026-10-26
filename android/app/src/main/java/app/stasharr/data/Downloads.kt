package app.stasharr.data

import android.app.DownloadManager
import android.content.Context
import android.net.Uri
import android.os.Environment
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import okhttp3.OkHttpClient
import okhttp3.Request
import java.io.File

/** A scene saved on the phone to watch without the server. */
@Serializable
data class OfflineItem(
    val sceneId: String,
    val quality: String,
    val qualityLabel: String,
    val scene: Scene,
    val state: String = PREPARING,
    val addedAt: Long = System.currentTimeMillis(),
    val downloadId: Long = -1,
    val path: String = "",
    val size: Long = 0,
    val error: String? = null,
    val thumb: String? = null,
) {
    val key: String get() = "${sceneId}_$quality"
    val done: Boolean get() = state == DONE

    companion object {
        const val PREPARING = "PREPARING"
        const val DOWNLOADING = "DOWNLOADING"
        const val DONE = "DONE"
        const val FAILED = "FAILED"
    }
}

/** Live progress, not saved: 0..1, and bytes so far. */
data class OfflineProgress(val fraction: Float, val bytes: Long = 0, val total: Long = 0)

/**
 * Downloads scenes for offline viewing. The server first converts the file
 * to the chosen size if needed, then Android's download manager fetches it,
 * with its own notification, even if the app is closed.
 */
class Downloads(
    private val context: Context,
    private val repo: Repository,
    private val http: OkHttpClient,
    private val scope: CoroutineScope,
) {
    private val api get() = repo
    private val store = File(context.filesDir, "downloads.json")
    private val thumbs = File(context.filesDir, "offline_thumbs").apply { mkdirs() }
    private val dm = context.getSystemService(Context.DOWNLOAD_SERVICE) as DownloadManager
    private val json = kotlinx.serialization.json.Json { ignoreUnknownKeys = true; coerceInputValues = true }

    private val _items = MutableStateFlow<List<OfflineItem>>(emptyList())
    val items: StateFlow<List<OfflineItem>> = _items

    private val _progress = MutableStateFlow<Map<String, OfflineProgress>>(emptyMap())
    val progress: StateFlow<Map<String, OfflineProgress>> = _progress

    private val preparing = mutableMapOf<String, Job>()
    private var monitor: Job? = null
    private val lock = Any()

    init {
        val saved = runCatching { json.decodeFromString<List<OfflineItem>>(store.readText()) }.getOrDefault(emptyList())
        _items.value = saved
        // carry on where the app left off
        saved.filter { it.state == OfflineItem.PREPARING }.forEach { prepare(it) }
        if (saved.any { it.state == OfflineItem.DOWNLOADING }) watchDownloads()
    }

    private fun save() {
        val list = _items.value
        runCatching {
            val tmp = File(store.path + ".tmp")
            tmp.writeText(json.encodeToString(list))
            tmp.renameTo(store)
        }
    }

    private fun update(key: String, fn: (OfflineItem) -> OfflineItem) {
        synchronized(lock) {
            _items.value = _items.value.map { if (it.key == key) fn(it) else it }
            save()
        }
    }

    fun forScene(sceneId: String): List<OfflineItem> = _items.value.filter { it.sceneId == sceneId }

    /** The downloaded file of a scene, best quality first, if any. */
    fun localFile(sceneId: String): File? {
        val order = listOf("original", "1080", "720", "480")
        return forScene(sceneId)
            .filter { it.done }
            .sortedBy { order.indexOf(it.quality).let { i -> if (i < 0) 99 else i } }
            .map { File(it.path) }
            .firstOrNull { it.exists() && it.length() > 0 }
    }

    fun storedScene(sceneId: String): Scene? = forScene(sceneId).firstOrNull { it.done }?.scene

    fun freeSpace(): Long = runCatching {
        (context.getExternalFilesDir(Environment.DIRECTORY_MOVIES) ?: context.filesDir).usableSpace
    }.getOrDefault(0L)

    /** Starts downloading a scene in a quality. */
    fun start(scene: Scene, quality: OfflineQuality) {
        val item = OfflineItem(sceneId = scene.id, quality = quality.id, qualityLabel = quality.label, scene = scene)
        synchronized(lock) {
            if (_items.value.any { it.key == item.key && it.state != OfflineItem.FAILED }) return
            _items.value = _items.value.filter { it.key != item.key } + item
            save()
        }
        scope.launch { saveThumb(item) }
        prepare(item)
    }

    fun retry(item: OfflineItem) {
        update(item.key) { it.copy(state = OfflineItem.PREPARING, error = null, downloadId = -1) }
        prepare(item)
    }

    fun delete(item: OfflineItem) {
        preparing.remove(item.key)?.cancel()
        if (item.downloadId >= 0) runCatching { dm.remove(item.downloadId) }
        if (item.path.isNotBlank()) runCatching { File(item.path).delete() }
        synchronized(lock) {
            _items.value = _items.value.filter { it.key != item.key }
            save()
        }
        if (_items.value.none { it.sceneId == item.sceneId }) item.thumb?.let { runCatching { File(it).delete() } }
        _progress.value = _progress.value - item.key
    }

    private suspend fun saveThumb(item: OfflineItem) {
        val url = item.scene.paths.screenshot ?: return
        val file = File(thumbs, "${item.sceneId}.jpg")
        runCatching {
            if (!file.exists()) {
                http.newCall(Request.Builder().url(url).build()).execute().use { res ->
                    if (!res.isSuccessful) return@runCatching
                    file.outputStream().use { out -> res.body?.byteStream()?.copyTo(out) }
                }
            }
            update(item.key) { it.copy(thumb = file.path) }
        }
    }

    /** Asks the server to get the file ready, waits for it, then downloads it. */
    private fun prepare(item: OfflineItem) {
        preparing[item.key]?.cancel()
        preparing[item.key] = scope.launch {
            var failures = 0
            var state: OfflineDownload? = null
            while (isActive) {
                try {
                    state = if (state == null) api.prepareOfflineDownload(item.sceneId, item.quality)
                    else api.offlineDownload(item.sceneId, item.quality)
                    failures = 0
                } catch (e: ApiException) {
                    // the server said no: stop; a lost connection: keep trying
                    if (e.message?.contains("reach the server") != true || ++failures > 60) {
                        update(item.key) { it.copy(state = OfflineItem.FAILED, error = e.message) }
                        return@launch
                    }
                    delay(10_000)
                    continue
                }
                val s = state!!
                when (s.status) {
                    "READY" -> {
                        val url = s.url
                        if (url == null) {
                            update(item.key) { it.copy(state = OfflineItem.FAILED, error = "The server gave no address") }
                        } else {
                            enqueue(item, url)
                        }
                        return@launch
                    }
                    "FAILED" -> {
                        update(item.key) { it.copy(state = OfflineItem.FAILED, error = s.error ?: "The server could not convert the file") }
                        return@launch
                    }
                    "NONE" -> {
                        // the server restarted: ask again
                        state = null
                    }
                    else -> {
                        _progress.value = _progress.value + (item.key to OfflineProgress(s.progress.toFloat()))
                    }
                }
                delay(3000)
            }
        }
    }

    private fun enqueue(item: OfflineItem, url: String) {
        val ext = if (item.quality == "original") {
            item.scene.file?.path?.substringAfterLast('.', "mp4")?.lowercase()?.take(5) ?: "mp4"
        } else "mp4"
        val name = "${item.sceneId}_${item.quality}.$ext"
        val dir = context.getExternalFilesDir(Environment.DIRECTORY_MOVIES) ?: context.filesDir
        val dest = File(dir, name)
        runCatching { dest.delete() }
        val req = DownloadManager.Request(Uri.parse(url))
            .setTitle(item.scene.displayTitle)
            .setDescription("Stasharr · ${item.qualityLabel}")
            .setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED)
            .setDestinationInExternalFilesDir(context, Environment.DIRECTORY_MOVIES, name)
            .setAllowedOverMetered(true)
            .setAllowedOverRoaming(false)
        val id = try {
            dm.enqueue(req)
        } catch (e: Exception) {
            update(item.key) { it.copy(state = OfflineItem.FAILED, error = e.message ?: "Couldn't start the download") }
            return
        }
        update(item.key) { it.copy(state = OfflineItem.DOWNLOADING, downloadId = id, path = dest.path) }
        watchDownloads()
    }

    /** Follows Android's downloads until they finish. */
    private fun watchDownloads() {
        if (monitor?.isActive == true) return
        monitor = scope.launch {
            while (isActive) {
                val active = _items.value.filter { it.state == OfflineItem.DOWNLOADING && it.downloadId >= 0 }
                if (active.isEmpty()) break
                val prog = _progress.value.toMutableMap()
                for (item in active) {
                    val c = runCatching { dm.query(DownloadManager.Query().setFilterById(item.downloadId)) }.getOrNull()
                    if (c == null || !c.moveToFirst()) {
                        c?.close()
                        // removed from the system's list, for example by the user
                        update(item.key) { it.copy(state = OfflineItem.FAILED, error = "The download was cancelled") }
                        continue
                    }
                    c.use {
                        val status = it.getInt(it.getColumnIndexOrThrow(DownloadManager.COLUMN_STATUS))
                        val got = it.getLong(it.getColumnIndexOrThrow(DownloadManager.COLUMN_BYTES_DOWNLOADED_SO_FAR))
                        val total = it.getLong(it.getColumnIndexOrThrow(DownloadManager.COLUMN_TOTAL_SIZE_BYTES))
                        when (status) {
                            DownloadManager.STATUS_SUCCESSFUL -> {
                                val size = File(item.path).length().takeIf { s -> s > 0 } ?: total
                                update(item.key) { i -> i.copy(state = OfflineItem.DONE, size = size, error = null) }
                                prog.remove(item.key)
                            }
                            DownloadManager.STATUS_FAILED -> {
                                val reason = it.getInt(it.getColumnIndexOrThrow(DownloadManager.COLUMN_REASON))
                                update(item.key) { i -> i.copy(state = OfflineItem.FAILED, error = failureText(reason)) }
                                prog.remove(item.key)
                            }
                            else -> {
                                val f = if (total > 0) got.toFloat() / total else 0f
                                prog[item.key] = OfflineProgress(f, got, total)
                            }
                        }
                    }
                }
                _progress.value = prog
                delay(1000)
            }
        }
    }

    private fun failureText(reason: Int): String = when (reason) {
        DownloadManager.ERROR_INSUFFICIENT_SPACE -> "Not enough space on the phone"
        DownloadManager.ERROR_HTTP_DATA_ERROR, DownloadManager.ERROR_CANNOT_RESUME -> "The connection dropped"
        401, 403 -> "The server refused the download. Try again."
        404 -> "The file is no longer on the server. Try again."
        409 -> "The file was not ready. Try again."
        else -> "Download failed ($reason)"
    }
}
