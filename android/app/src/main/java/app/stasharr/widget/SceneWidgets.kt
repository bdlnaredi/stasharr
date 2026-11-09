package app.stasharr.widget

import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.drawable.BitmapDrawable
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.Image
import androidx.glance.ImageProvider
import androidx.glance.LocalSize
import androidx.glance.action.clickable
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.GlanceAppWidgetManager
import androidx.glance.appwidget.GlanceAppWidgetReceiver
import androidx.glance.appwidget.SizeMode
import androidx.glance.appwidget.action.actionStartActivity
import androidx.glance.appwidget.cornerRadius
import androidx.glance.appwidget.provideContent
import androidx.glance.appwidget.updateAll
import androidx.glance.background
import androidx.glance.layout.Alignment
import androidx.glance.layout.Box
import androidx.glance.layout.Column
import androidx.glance.layout.ContentScale
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.layout.padding
import androidx.glance.layout.size
import androidx.glance.layout.width
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import androidx.glance.unit.ColorProvider
import app.stasharr.MainActivity
import app.stasharr.StasharrApp
import app.stasharr.alerts.FavoriteAlertsState
import app.stasharr.data.Scene
import coil.imageLoader
import coil.request.ImageRequest
import coil.size.Scale

/** What a widget shows: Continue watching, or new scenes from favourites. */
enum class WidgetKind(val title: String, val empty: String) {
    Continue("Continue watching", "Scenes you started show here"),
    Favourites("New from favourites", "New scenes with pornstars you follow show here"),
}

private class WidgetItem(val id: String, val title: String, val sub: String, val thumb: Bitmap?, val progress: Float)

private val bg = ColorProvider(Color(0xFF14101A))
private val card = ColorProvider(Color(0xFF221B2B))
private val text = ColorProvider(Color.White)
private val muted = ColorProvider(Color(0xFFB4A9C2))
private val pink = ColorProvider(Color(0xFFFF3D8B))

/** A home screen widget with a few scenes; tapping one opens it in the app. */
abstract class SceneWidget(private val kind: WidgetKind) : GlanceAppWidget() {
    override val sizeMode = SizeMode.Responsive(setOf(DpSize(180.dp, 110.dp), DpSize(250.dp, 180.dp), DpSize(320.dp, 260.dp)))

    override suspend fun provideGlance(context: Context, id: GlanceId) {
        val items = load(context)
        provideContent { Content(context, items) }
    }

    private suspend fun load(context: Context): List<WidgetItem>? {
        val app = context.applicationContext as StasharrApp
        if (app.session.serverUrl.isNullOrBlank()) return null
        val scenes: List<Scene> = runCatching {
            when (kind) {
                WidgetKind.Continue -> app.repo.continueWatching(4)
                WidgetKind.Favourites -> app.repo.favoriteAlerts(limit = 4).alerts.map { it.scene }
            }
        }.getOrElse { return emptyList() }
        return scenes.take(4).map { s ->
            WidgetItem(
                id = s.id,
                title = s.displayTitle,
                sub = listOfNotNull(s.stars.firstOrNull()?.name, s.studio?.name).joinToString(" · "),
                thumb = s.paths.screenshot?.let { thumb(context, it) },
                progress = if (kind == WidgetKind.Continue) s.progress else 0f,
            )
        }
    }

    private suspend fun thumb(context: Context, url: String): Bitmap? = runCatching {
        val req = ImageRequest.Builder(context)
            .data(url)
            .size(320, 180)
            .scale(Scale.FILL)
            .allowHardware(false)
            .build()
        (context.imageLoader.execute(req).drawable as? BitmapDrawable)?.bitmap
    }.getOrNull()

    private fun open(context: Context, sceneId: String?): Intent =
        Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            if (sceneId != null) putExtra(FavoriteAlertsState.EXTRA_SCENE, sceneId)
            // a different data address keeps each scene's tap separate
            data = android.net.Uri.parse("stasharr://scene/${sceneId ?: "home"}")
        }

    @Composable
    private fun Content(context: Context, items: List<WidgetItem>?) {
        val size = LocalSize.current
        val rows = when {
            size.height >= 250.dp -> 4
            size.height >= 170.dp -> 3
            else -> 1
        }
        Column(
            GlanceModifier.fillMaxSize().background(bg).cornerRadius(16.dp).padding(10.dp)
                .clickable(actionStartActivity(open(context, null))),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("STASHARR", style = TextStyle(color = pink, fontSize = 11.sp, fontWeight = FontWeight.Bold))
                Spacer(GlanceModifier.width(6.dp))
                Text(kind.title, style = TextStyle(color = text, fontSize = 13.sp, fontWeight = FontWeight.Medium), maxLines = 1)
            }
            Spacer(GlanceModifier.height(6.dp))
            when {
                items == null -> Text("Sign in to the app", style = TextStyle(color = muted, fontSize = 12.sp))
                items.isEmpty() -> Text(kind.empty, style = TextStyle(color = muted, fontSize = 12.sp))
                else -> items.take(rows).forEach { item ->
                    Row(
                        GlanceModifier.fillMaxWidth().padding(vertical = 3.dp).background(card).cornerRadius(10.dp)
                            .clickable(actionStartActivity(open(context, item.id))),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(GlanceModifier.size(96.dp, 54.dp)) {
                            if (item.thumb != null) {
                                Image(
                                    ImageProvider(item.thumb),
                                    contentDescription = null,
                                    contentScale = ContentScale.Crop,
                                    modifier = GlanceModifier.fillMaxSize().cornerRadius(10.dp),
                                )
                            }
                        }
                        Column(GlanceModifier.padding(horizontal = 8.dp).defaultWeight()) {
                            Text(item.title, style = TextStyle(color = text, fontSize = 13.sp, fontWeight = FontWeight.Medium), maxLines = 2)
                            if (item.sub.isNotBlank()) {
                                Text(item.sub, style = TextStyle(color = muted, fontSize = 11.sp), maxLines = 1)
                            }
                            if (item.progress > 0.01f) {
                                Text("${(item.progress * 100).toInt()}% watched", style = TextStyle(color = pink, fontSize = 10.sp))
                            }
                        }
                    }
                }
            }
        }
    }

    companion object {
        /** Refreshes every widget, for example after signing in or a new check. */
        suspend fun refreshAll(context: Context) {
            val m = GlanceAppWidgetManager(context)
            runCatching {
                if (m.getGlanceIds(ContinueWidget::class.java).isNotEmpty()) ContinueWidget().updateAll(context)
                if (m.getGlanceIds(FavouritesWidget::class.java).isNotEmpty()) FavouritesWidget().updateAll(context)
            }
        }
    }
}

class ContinueWidget : SceneWidget(WidgetKind.Continue)
class FavouritesWidget : SceneWidget(WidgetKind.Favourites)

class ContinueWidgetReceiver : GlanceAppWidgetReceiver() {
    override val glanceAppWidget: GlanceAppWidget = ContinueWidget()
}

class FavouritesWidgetReceiver : GlanceAppWidgetReceiver() {
    override val glanceAppWidget: GlanceAppWidget = FavouritesWidget()
}
