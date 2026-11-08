package app.stasharr.ui.screens

import android.text.format.DateUtils
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import app.stasharr.alerts.FavoriteAlertsState
import app.stasharr.app
import app.stasharr.data.FavoriteAlert
import app.stasharr.ui.LocalNav
import app.stasharr.ui.components.LoadingBox
import app.stasharr.ui.theme.Brand
import coil.compose.AsyncImage
import java.time.OffsetDateTime

private fun ago(iso: String): String = runCatching {
    val t = OffsetDateTime.parse(iso).toInstant().toEpochMilli()
    DateUtils.getRelativeTimeSpanString(t, System.currentTimeMillis(), DateUtils.MINUTE_IN_MILLIS).toString()
}.getOrDefault("")

/** New scenes with your favourite pornstars. Opening it marks them seen. */
@Composable
fun AlertsScreen() {
    val nav = LocalNav.current
    var items by remember { mutableStateOf<List<FavoriteAlert>?>(null) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) {
        runCatching { app.repo.favoriteAlerts(limit = 50) }
            .onSuccess { items = it.alerts }
            .onFailure { error = it.message; items = emptyList() }
        FavoriteAlertsState.markSeen(app)
    }

    Column(Modifier.fillMaxSize()) {
        Row(
            Modifier.fillMaxWidth().statusBarsPadding().padding(horizontal = 4.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = { nav.back() }) { Icon(Icons.AutoMirrored.Rounded.ArrowBack, "Back") }
            Text("New from your favourites", style = MaterialTheme.typography.titleLarge, modifier = Modifier.weight(1f))
        }
        val ctx = androidx.compose.ui.platform.LocalContext.current
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                "Phone notifications for new scenes",
                style = MaterialTheme.typography.bodyMedium,
                color = Brand.TextMuted,
                modifier = Modifier.weight(1f),
            )
            androidx.compose.material3.Switch(
                checked = FavoriteAlertsState.notify,
                onCheckedChange = { FavoriteAlertsState.setNotify(ctx, it) },
            )
        }
        val list = items
        when {
            list == null -> LoadingBox()
            list.isEmpty() -> Text(
                error ?: "Nothing new since you last looked. Tap “Add to favourites” on a pornstar's page to hear about her new scenes.",
                color = Brand.TextMuted,
                modifier = Modifier.padding(24.dp),
            )
            else -> LazyColumn(
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                items(list, key = { it.scene.id }) { a ->
                    Row(
                        Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(12.dp))
                            .clickable { nav.scene(a.scene.id) }
                            .padding(4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(
                            Modifier
                                .width(140.dp)
                                .aspectRatio(16f / 9f)
                                .clip(RoundedCornerShape(10.dp))
                                .background(Brand.SurfaceHigh)
                        ) {
                            AsyncImage(a.scene.paths.screenshot, null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
                        }
                        Spacer(Modifier.width(12.dp))
                        Column(Modifier.weight(1f)) {
                            Text(
                                a.scene.title ?: "Untitled",
                                style = MaterialTheme.typography.titleSmall,
                                fontWeight = FontWeight.Bold,
                                maxLines = 2,
                                overflow = TextOverflow.Ellipsis,
                            )
                            Text("with ${a.performer}", color = Brand.Pink, style = MaterialTheme.typography.bodySmall)
                            Text(ago(a.addedAt), color = Brand.TextMuted, style = MaterialTheme.typography.bodySmall)
                        }
                    }
                }
            }
        }
    }
}
