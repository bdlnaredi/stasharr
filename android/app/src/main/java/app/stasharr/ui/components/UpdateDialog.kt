package app.stasharr.ui.components

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import app.stasharr.ui.theme.Brand
import app.stasharr.update.AppUpdate
import kotlinx.coroutines.launch

/** "A new version is ready": what changed, and a button that installs it. */
@Composable
fun UpdateDialog() {
    if (!AppUpdate.showDialog) return
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val release = AppUpdate.available
    val downloading = AppUpdate.progress >= 0f

    AlertDialog(
        onDismissRequest = { if (!downloading) AppUpdate.later(context) },
        title = {
            Text(if (release != null) "Stasharr ${release.version} is ready" else "No update")
        },
        text = {
            Column {
                if (release != null) {
                    Text(
                        "You have ${AppUpdate.current}.",
                        color = Brand.TextMuted,
                        style = MaterialTheme.typography.bodySmall,
                    )
                    val notes = release.body?.substringBefore("\n---")?.trim().orEmpty()
                    if (notes.isNotEmpty()) {
                        Spacer(Modifier.height(10.dp))
                        Column(Modifier.heightIn(max = 260.dp).verticalScroll(rememberScrollState())) {
                            Text(notes.replace("**", "").replace("### ", "").replace("## ", ""), style = MaterialTheme.typography.bodyMedium)
                        }
                    }
                }
                if (downloading) {
                    Spacer(Modifier.height(14.dp))
                    LinearProgressIndicator(progress = { AppUpdate.progress }, modifier = Modifier.fillMaxWidth(), color = Brand.Pink)
                    Text("Downloading… ${(AppUpdate.progress * 100).toInt()}%", style = MaterialTheme.typography.bodySmall)
                }
                AppUpdate.message?.let {
                    Spacer(Modifier.height(10.dp))
                    Text(it, style = MaterialTheme.typography.bodySmall, color = if (release == null) MaterialTheme.colorScheme.onSurface else Brand.Pink)
                }
            }
        },
        confirmButton = {
            if (release != null) {
                TextButton(
                    enabled = !downloading,
                    onClick = { scope.launch { AppUpdate.install(context) } },
                ) { Text("Update") }
            } else {
                TextButton(onClick = { AppUpdate.showDialog = false; AppUpdate.message = null }) { Text("OK") }
            }
        },
        dismissButton = {
            if (release != null && !downloading) {
                TextButton(onClick = { AppUpdate.later(context) }) { Text("Later") }
            }
        },
    )
}
