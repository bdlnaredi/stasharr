package app.stasharr.alerts

import app.stasharr.widget.SceneWidget
import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.NetworkType
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import app.stasharr.MainActivity
import app.stasharr.R
import app.stasharr.StasharrApp
import java.time.OffsetDateTime
import java.util.concurrent.TimeUnit

/**
 * Alerts for new scenes with your favourite pornstars: a count for the
 * badge in the app, and a phone notification when the server has new ones.
 * The server checks every 30 minutes or so, whenever Android lets it.
 */
object FavoriteAlertsState {
    /** New scenes since you last opened the alerts. */
    var count by mutableIntStateOf(0)

    /** Phone notifications on or off. Saved. */
    var notify by mutableStateOf(true)
        private set

    private const val PREFS = "alerts"
    private const val CHANNEL = "favourites"
    private const val WORK = "favourite-alerts"
    const val EXTRA_SCENE = "open_scene"

    /** A scene a notification asked to open; the app opens it and clears this. */
    var openScene by mutableStateOf<String?>(null)

    private fun prefs(context: Context) = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)

    fun restore(context: Context) {
        notify = prefs(context).getBoolean("notify", true)
        AutoDownloads.restore(context)
        reschedule(context)
    }

    fun setNotify(context: Context, on: Boolean) {
        notify = on
        prefs(context).edit().putBoolean("notify", on).apply()
        reschedule(context)
    }

    /** The check runs while notifications or auto-download are on. */
    fun reschedule(context: Context) {
        if (notify || AutoDownloads.enabled) schedule(context)
        else WorkManager.getInstance(context).cancelUniqueWork(WORK)
    }

    /** Fetches the count for the badge. */
    suspend fun refresh(app: StasharrApp) {
        if (app.session.serverUrl.isNullOrBlank()) return
        runCatching { app.repo.favoriteAlerts(limit = 1) }.onSuccess { count = it.count }
    }

    suspend fun markSeen(app: StasharrApp) {
        runCatching { app.repo.markAlertsSeen() }.onSuccess { count = 0 }
    }

    fun schedule(context: Context) {
        val req = PeriodicWorkRequestBuilder<FavoriteAlertsWorker>(30, TimeUnit.MINUTES)
            .setConstraints(Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build())
            .build()
        WorkManager.getInstance(context).enqueueUniquePeriodicWork(WORK, ExistingPeriodicWorkPolicy.KEEP, req)
    }

    fun canPost(context: Context): Boolean =
        Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU ||
            ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED

    internal fun lastNotified(context: Context): String? = prefs(context).getString("last", null)

    internal fun setLastNotified(context: Context, at: String) {
        prefs(context).edit().putString("last", at).apply()
    }

    internal fun channel(context: Context) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val nm = context.getSystemService(NotificationManager::class.java)
            if (nm.getNotificationChannel(CHANNEL) == null) {
                nm.createNotificationChannel(
                    NotificationChannel(CHANNEL, "New from your favourites", NotificationManager.IMPORTANCE_DEFAULT).apply {
                        description = "New scenes with the pornstars you follow"
                    }
                )
            }
        }
    }

    internal fun post(context: Context, id: Int, title: String, text: String, sceneId: String?) {
        channel(context)
        val open = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            if (sceneId != null) putExtra(EXTRA_SCENE, sceneId)
        }
        val pi = PendingIntent.getActivity(
            context, id, open, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        val n = NotificationCompat.Builder(context, CHANNEL)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(text)
            .setStyle(NotificationCompat.BigTextStyle().bigText(text))
            .setAutoCancel(true)
            .setContentIntent(pi)
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
            .build()
        try {
            NotificationManagerCompat.from(context).notify(id, n)
        } catch (_: SecurityException) {
            // notifications not allowed
        }
    }
}

private fun parse(s: String): Long = runCatching { OffsetDateTime.parse(s).toInstant().toEpochMilli() }.getOrDefault(0L)

/** Checks the server for new scenes from favourites and posts a notification. */
class FavoriteAlertsWorker(context: Context, params: WorkerParameters) : CoroutineWorker(context, params) {
    override suspend fun doWork(): Result {
        val app = applicationContext as StasharrApp
        if (app.session.serverUrl.isNullOrBlank()) return Result.success()
        val ctx = applicationContext
        val last = FavoriteAlertsState.lastNotified(ctx)
        val res = runCatching { app.repo.favoriteAlerts(limit = 10, since = last) }.getOrElse { return Result.retry() }
        FavoriteAlertsState.count = res.count
        runCatching { SceneWidget.refreshAll(ctx) }
        val fresh = res.alerts.filter { last == null || parse(it.addedAt) > parse(last) }
        val newest = res.alerts.maxByOrNull { parse(it.addedAt) }?.addedAt
        // the first check only remembers where things stand, so a new install
        // does not get a pile of notifications
        if (last == null) {
            if (newest != null) FavoriteAlertsState.setLastNotified(ctx, newest)
            else FavoriteAlertsState.setLastNotified(ctx, OffsetDateTime.now().toString())
            return Result.success()
        }
        runCatching { AutoDownloads.handle(app, fresh) }
        if (fresh.isEmpty() || !FavoriteAlertsState.notify || !FavoriteAlertsState.canPost(ctx)) {
            if (newest != null) FavoriteAlertsState.setLastNotified(ctx, newest)
            return Result.success()
        }
        if (fresh.size == 1) {
            val a = fresh.first()
            FavoriteAlertsState.post(ctx, 7001, "New scene with ${a.performer}", a.scene.title ?: "Tap to watch", a.scene.id)
        } else {
            val names = fresh.map { it.performer }.distinct()
            val who = if (names.size <= 3) names.joinToString(", ") else names.take(3).joinToString(", ") + " and more"
            FavoriteAlertsState.post(ctx, 7001, "${fresh.size} new scenes from your favourites", "With $who", null)
        }
        if (newest != null) FavoriteAlertsState.setLastNotified(ctx, newest)
        return Result.success()
    }
}
