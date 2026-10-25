package app.stasharr.ui.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyGridScope
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.grid.rememberLazyGridState
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import app.stasharr.data.Page
import app.stasharr.ui.theme.Brand
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.launch

/** A list loaded from the server a page at a time. */
class PagedState<T>(
    private val scope: CoroutineScope,
    private val fetch: suspend (page: Int) -> Page<T>,
) {
    val items = mutableStateListOf<T>()
    var total by mutableIntStateOf(-1)
        private set
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    private var page = 0
    private var job: Job? = null

    val end: Boolean get() = total >= 0 && items.size >= total

    fun loadMore() {
        if (loading || end) return
        loading = true
        error = null
        job = scope.launch {
            try {
                val p = fetch(page + 1)
                page++
                items.addAll(p.items)
                total = if (p.items.isEmpty()) items.size else p.count
            } catch (e: CancellationException) {
                throw e
            } catch (e: Throwable) {
                error = e.message ?: "Couldn't load more"
            } finally {
                loading = false
            }
        }
    }

    fun refresh() {
        job?.cancel()
        loading = false
        items.clear()
        page = 0
        total = -1
        loadMore()
    }
}

@Composable
fun <T> rememberPaged(vararg keys: Any?, fetch: suspend (page: Int) -> Page<T>): PagedState<T> {
    val scope = rememberCoroutineScope()
    val state = remember(*keys) { PagedState(scope, fetch) }
    LaunchedEffect(state) { state.loadMore() }
    return state
}

/** A grid that loads more as you scroll, with an optional header above. */
@Composable
fun <T> PagedGrid(
    state: PagedState<T>,
    key: (T) -> Any,
    minCell: Dp = 170.dp,
    modifier: Modifier = Modifier,
    contentPadding: PaddingValues = PaddingValues(start = 16.dp, end = 16.dp, bottom = 24.dp),
    empty: String = "Nothing here yet",
    header: (LazyGridScope.() -> Unit)? = null,
    cell: @Composable (T) -> Unit,
) {
    val grid = rememberLazyGridState()

    LaunchedEffect(grid, state) {
        snapshotFlow {
            val last = grid.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: 0
            last >= grid.layoutInfo.totalItemsCount - 8
        }.distinctUntilChanged().collect { nearEnd -> if (nearEnd) state.loadMore() }
    }

    LazyVerticalGrid(
        columns = GridCells.Adaptive(minCell),
        state = grid,
        modifier = modifier.fillMaxSize(),
        contentPadding = contentPadding,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        header?.invoke(this)
        items(state.items, key = key) { cell(it) }
        item(span = { GridItemSpan(maxLineSpan) }) {
            Box(Modifier.fillMaxWidth().padding(16.dp), contentAlignment = Alignment.Center) {
                when {
                    state.loading -> CircularProgressIndicator(color = Brand.Pink, modifier = Modifier.size(28.dp))
                    state.error != null -> TextButton(onClick = { state.loadMore() }) {
                        Text("${state.error} – tap to retry")
                    }
                    state.end && state.items.isEmpty() -> Text(empty, color = Brand.TextMuted, style = MaterialTheme.typography.bodyLarge)
                }
            }
        }
    }
}

/** A full width item in a grid. */
fun LazyGridScope.fullWidth(key: Any? = null, content: @Composable () -> Unit) {
    item(key = key, span = { GridItemSpan(maxLineSpan) }) { content() }
}
