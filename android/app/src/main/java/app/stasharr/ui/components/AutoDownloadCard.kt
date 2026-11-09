package app.stasharr.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import app.stasharr.alerts.AutoDownloads
import app.stasharr.ui.theme.Brand

/** Auto-download settings, at the top of Downloads. */
@Composable
fun AutoDownloadCard(modifier: Modifier = Modifier) {
    val ctx = LocalContext.current
    Column(
        modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .background(Brand.Surface)
            .padding(vertical = 8.dp),
    ) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text("Download new scenes from favourites", style = MaterialTheme.typography.titleSmall)
                Text(
                    "New scenes with pornstars you follow download by themselves, checked about every 30 minutes. Only the newest ${AutoDownloads.keep} are kept.",
                    style = MaterialTheme.typography.bodySmall,
                    color = Brand.TextMuted,
                )
            }
            Switch(checked = AutoDownloads.enabled, onCheckedChange = { AutoDownloads.set(ctx, enabled = it) })
        }
        if (AutoDownloads.enabled) {
            Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text("Only on Wi-Fi", style = MaterialTheme.typography.titleSmall)
                    Text("Waits for Wi-Fi so mobile data is not used", style = MaterialTheme.typography.bodySmall, color = Brand.TextMuted)
                }
                Switch(checked = AutoDownloads.wifiOnly, onCheckedChange = { AutoDownloads.set(ctx, wifiOnly = it) })
            }
            Text("Keep", style = MaterialTheme.typography.labelLarge, color = Brand.TextMuted, modifier = Modifier.padding(start = 16.dp, top = 6.dp))
            ChipRow {
                AutoDownloads.keepChoices.forEach { n -> Chip("$n scenes", selected = AutoDownloads.keep == n) { AutoDownloads.set(ctx, keep = n) } }
            }
            Text("Quality", style = MaterialTheme.typography.labelLarge, color = Brand.TextMuted, modifier = Modifier.padding(start = 16.dp, top = 2.dp))
            ChipRow {
                listOf("480" to "480p", "720" to "720p", "1080" to "1080p", "original" to "Original").forEach { (id, label) ->
                    Chip(label, selected = AutoDownloads.quality == id) { AutoDownloads.set(ctx, quality = id) }
                }
            }
        }
    }
}
