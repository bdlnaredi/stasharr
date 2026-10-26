package app.stasharr.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.CloudOff
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import app.stasharr.data.ApiException
import app.stasharr.ui.theme.Brand
import kotlinx.coroutines.CancellationException

/** The state of something loaded from the server. */
sealed interface Load<out T> {
    data object Loading : Load<Nothing>
    data class Ok<T>(val value: T) : Load<T>
    data class Err(val message: String, val unauthorized: Boolean = false) : Load<Nothing>
}

fun Throwable.toErr(): Load.Err = when (this) {
    is ApiException -> Load.Err(message ?: "Something went wrong", unauthorized)
    else -> Load.Err(message ?: "Something went wrong")
}

class Loader<T>(val state: Load<T>, val reload: () -> Unit, val refreshing: Boolean = false) {
    val value: T? get() = (state as? Load.Ok<T>)?.value
}

/** Loads data when the keys change, and again when reload() is called. */
@Composable
fun <T> rememberLoad(vararg keys: Any?, block: suspend () -> T): Loader<T> {
    var state by remember(*keys) { mutableStateOf<Load<T>>(Load.Loading) }
    var nonce by remember { mutableIntStateOf(0) }
    var refreshing by remember { mutableStateOf(false) }
    LaunchedEffect(*keys, nonce) {
        // keep showing old data while refreshing
        if (state !is Load.Ok) state = Load.Loading else refreshing = true
        try {
            state = Load.Ok(block())
        } catch (e: CancellationException) {
            throw e
        } catch (e: Throwable) {
            // a failed refresh keeps what is already on screen
            if (state !is Load.Ok) state = e.toErr()
        } finally {
            refreshing = false
        }
    }
    return Loader(state, { nonce++ }, refreshing)
}

/** App wide actions any screen can use. */
class AppActions(
    val signOut: () -> Unit,
)

val LocalAppActions = compositionLocalOf { AppActions(signOut = {}) }

/** Opens the Downloads screen, where one is available. */
val LocalOpenDownloads = compositionLocalOf<(() -> Unit)?> { null }

@Composable
fun LoadingBox(modifier: Modifier = Modifier) {
    Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        CircularProgressIndicator(color = Brand.Pink, strokeWidth = 3.dp, modifier = Modifier.size(36.dp))
    }
}

@Composable
fun ErrorBox(err: Load.Err, onRetry: () -> Unit, modifier: Modifier = Modifier) {
    val actions = LocalAppActions.current
    Column(
        modifier.fillMaxSize().padding(32.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(Icons.Rounded.CloudOff, null, tint = Brand.TextMuted, modifier = Modifier.size(48.dp))
        Spacer(Modifier.height(12.dp))
        Text(
            err.message,
            style = MaterialTheme.typography.bodyLarge,
            color = Brand.TextMuted,
            textAlign = TextAlign.Center,
            modifier = Modifier.fillMaxWidth(),
        )
        Spacer(Modifier.height(16.dp))
        GradientButton(text = "Try again", onClick = onRetry)
        if (err.unauthorized) {
            TextButton(onClick = actions.signOut) { Text("Sign in again") }
        }
        val openDownloads = LocalOpenDownloads.current
        if (openDownloads != null && app.stasharr.app.downloads.items.value.any { it.done }) {
            TextButton(onClick = openDownloads) { Text("Watch your downloads") }
        }
    }
}

/** Shows loading and error states, and the content once loaded. */
@Composable
fun <T> Loaded(loader: Loader<T>, modifier: Modifier = Modifier, content: @Composable (T) -> Unit) {
    when (val s = loader.state) {
        is Load.Loading -> LoadingBox(modifier)
        is Load.Err -> ErrorBox(s, loader.reload, modifier)
        is Load.Ok -> content(s.value)
    }
}
