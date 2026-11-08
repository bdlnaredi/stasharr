package app.stasharr.ui.components

import android.text.format.DateUtils
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import app.stasharr.app
import app.stasharr.data.Comment
import app.stasharr.ui.theme.Brand
import kotlinx.coroutines.launch
import java.time.OffsetDateTime

private const val PER_PAGE = 20

private fun ago(iso: String): String = runCatching {
    val t = OffsetDateTime.parse(iso).toInstant().toEpochMilli()
    DateUtils.getRelativeTimeSpanString(t, System.currentTimeMillis(), DateUtils.MINUTE_IN_MILLIS).toString()
}.getOrDefault("")

/**
 * A scene's comments: a box to write one, then the newest first. On a reel
 * the same conversation is shown, and new comments note the reel.
 */
@Composable
fun CommentsSection(
    sceneId: String,
    modifier: Modifier = Modifier,
    reelId: String? = null,
    showTitle: Boolean = true,
    onCountChange: (Int) -> Unit = {},
) {
    val scope = rememberCoroutineScope()
    val list = remember(sceneId) { mutableStateListOf<Comment>() }
    var total by remember(sceneId) { mutableIntStateOf(0) }
    var page by remember(sceneId) { mutableIntStateOf(1) }
    var loading by remember(sceneId) { mutableStateOf(true) }
    var text by remember(sceneId) { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    suspend fun load(p: Int) {
        loading = true
        runCatching { app.repo.comments(sceneId, p, PER_PAGE) }
            .onSuccess { res ->
                if (p == 1) list.clear()
                list.addAll(res.items.filter { c -> list.none { it.id == c.id } })
                total = res.count
                onCountChange(res.count)
            }
            .onFailure { error = it.message }
        loading = false
    }

    LaunchedEffect(sceneId) { load(1) }

    Column(modifier) {
        if (showTitle) {
            Text(
                if (total == 0) "Comments" else if (total == 1) "1 comment" else "$total comments",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.padding(bottom = 8.dp),
            )
        }

        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedTextField(
                value = text,
                onValueChange = { if (it.length <= 1000) text = it },
                placeholder = { Text("Add a comment…") },
                modifier = Modifier.weight(1f),
                maxLines = 4,
                shape = RoundedCornerShape(14.dp),
                colors = OutlinedTextFieldDefaults.colors(
                    focusedBorderColor = Brand.Pink,
                    unfocusedBorderColor = Brand.Outline,
                ),
            )
            Button(
                onClick = {
                    val body = text.trim()
                    if (body.isEmpty()) return@Button
                    busy = true
                    error = null
                    scope.launch {
                        runCatching { app.repo.addComment(sceneId, body, reelId) }
                            .onSuccess { c ->
                                if (c != null) list.add(0, c)
                                total += 1
                                onCountChange(total)
                                text = ""
                            }
                            .onFailure { error = it.message }
                        busy = false
                    }
                },
                enabled = text.isNotBlank() && !busy,
                colors = ButtonDefaults.buttonColors(containerColor = Brand.Pink),
            ) { Text("Post") }
        }
        error?.let { Text(it, color = Color(0xFFFF6B81), style = MaterialTheme.typography.bodySmall, modifier = Modifier.padding(top = 6.dp)) }

        if (!loading && list.isEmpty()) {
            Text("Be the first to comment.", color = Brand.TextMuted, modifier = Modifier.padding(vertical = 12.dp))
        }

        list.forEach { c ->
            Row(Modifier.fillMaxWidth().padding(vertical = 10.dp)) {
                Box(
                    Modifier
                        .size(34.dp)
                        .background(if (c.mine) Brand.Pink else Brand.SurfaceHigh, CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(c.author.take(1).uppercase(), fontWeight = FontWeight.Bold, color = Color.White)
                }
                Spacer(Modifier.width(10.dp))
                Column(Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(c.author, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.bodyMedium)
                        Spacer(Modifier.width(8.dp))
                        Text(ago(c.createdAt), color = Brand.TextMuted, style = MaterialTheme.typography.bodySmall)
                        if (c.fromReel) {
                            Spacer(Modifier.width(8.dp))
                            Text("on a reel", color = Brand.TextMuted, style = MaterialTheme.typography.labelSmall)
                        }
                    }
                    Text(c.body, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(top = 2.dp))
                    if (c.canDelete) {
                        TextButton(
                            onClick = {
                                scope.launch {
                                    runCatching { app.repo.deleteComment(c.id) }
                                        .onSuccess {
                                            list.removeAll { it.id == c.id }
                                            total = (total - 1).coerceAtLeast(0)
                                            onCountChange(total)
                                        }
                                        .onFailure { error = it.message }
                                }
                            },
                            contentPadding = PaddingValues(0.dp),
                        ) { Text("Delete", color = Brand.TextMuted, style = MaterialTheme.typography.labelMedium) }
                    }
                }
            }
        }

        if (list.size < total) {
            TextButton(onClick = { scope.launch { page += 1; load(page) } }, enabled = !loading) {
                Text("Show more comments")
            }
        }
    }
}
