package app.stasharr.ui

import android.content.Context
import android.net.Uri
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.PlaylistPlay
import androidx.compose.material.icons.rounded.Home
import androidx.compose.material.icons.rounded.Person
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.SmartDisplay
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import app.stasharr.app
import app.stasharr.player.PlayerActivity
import app.stasharr.ui.components.AppActions
import app.stasharr.ui.components.LocalAppActions
import app.stasharr.ui.components.LocalOpenDownloads
import app.stasharr.ui.components.rememberLoad
import app.stasharr.ui.screens.BrowseScreen
import app.stasharr.ui.screens.DownloadsScreen
import app.stasharr.ui.screens.HomeScreen
import app.stasharr.ui.screens.LoginScreen
import app.stasharr.ui.screens.PerformerScreen
import app.stasharr.ui.screens.PlaylistScreen
import app.stasharr.ui.screens.PlaylistsScreen
import app.stasharr.ui.screens.ProfileScreen
import app.stasharr.ui.screens.ReelsScreen
import app.stasharr.ui.screens.SceneListKind
import app.stasharr.ui.screens.SceneListScreen
import app.stasharr.ui.screens.SceneScreen
import app.stasharr.ui.screens.StudioScreen
import app.stasharr.ui.screens.TagScreen
import app.stasharr.ui.theme.Brand
import app.stasharr.player.AutoPreview
import app.stasharr.ui.store.AgeGate
import app.stasharr.ui.store.StoreHome
import app.stasharr.ui.store.StoreModel
import app.stasharr.ui.store.StoreModels
import app.stasharr.ui.store.StoreMyList
import app.stasharr.ui.store.StoreScenes
import androidx.compose.material.icons.rounded.Favorite
import androidx.compose.material.icons.rounded.Star
import androidx.compose.material.icons.rounded.VideoLibrary

/** Moves between screens. */
class Navigator(private val nav: NavHostController, private val context: Context) {
    fun scene(id: String) = nav.navigate("scene/$id")
    fun performer(id: String) = nav.navigate(if (SiteMode.storefront) "model/$id" else "performer/$id")
    fun studio(id: String) = nav.navigate("studio/$id")
    fun tag(id: String, name: String) = nav.navigate("tag/$id/${Uri.encode(name)}")
    fun playlist(id: String) = nav.navigate("playlist/$id")
    fun list(kind: SceneListKind) = nav.navigate("list/${kind.name}")
    fun downloads() = nav.navigate("downloads")
    fun back() = nav.popBackStack()

    /** Opens a website address such as /site/model/12 in the app. */
    fun siteLink(link: String) {
        val l = link.trim()
        if (l.startsWith("http://") || l.startsWith("https://")) {
            runCatching { context.startActivity(android.content.Intent(android.content.Intent.ACTION_VIEW, Uri.parse(l))) }
            return
        }
        val parts = l.substringBefore('?').trim('/').split('/')
        if (parts.firstOrNull() != "site") return
        val id = parts.getOrNull(2).orEmpty()
        when (parts.getOrNull(1)) {
            "scene" -> if (id.isNotBlank()) scene(id)
            "model" -> if (id.isNotBlank()) performer(id)
            "studio" -> if (id.isNotBlank()) studio(id)
            "category" -> if (id.isNotBlank()) tag(id, "")
            "scenes", "search", "categories", "channels" -> tab("scenes")
            "models" -> tab("models")
            "reels" -> tab("reels")
            "my-list" -> tab("mylist")
        }
    }

    fun tab(route: String) = nav.navigate(route) {
        popUpTo(nav.graph.findStartDestination().id) { saveState = true }
        launchSingleTop = true
        restoreState = true
    }

    /** Plays a scene full screen. queue plays on after it. */
    fun play(id: String, startAt: Double = 0.0, queue: List<String> = emptyList()) {
        context.startActivity(PlayerActivity.intent(context, id, startAt, queue))
    }
}

val LocalNav = staticCompositionLocalOf<Navigator> { error("no navigator") }

@Composable
fun StasharrRoot() {
    var signedIn by remember { mutableStateOf(app.session.isSetUp && !app.session.loggedOut) }
    val actions = remember {
        AppActions(signOut = {
            SiteMode.clear(app)
            val server = app.session.serverUrl
            app.session.clear()
            app.session.serverUrl = server
            app.session.loggedOut = true
            signedIn = false
        })
    }
    CompositionLocalProvider(LocalAppActions provides actions) {
        if (!signedIn) {
            LoginScreen(onDone = {
                app.session.loggedOut = false
                signedIn = true
            })
        } else {
            MainScaffold()
        }
    }
}

private data class Tab(val route: String, val label: String, val icon: ImageVector)

@Composable
private fun MainScaffold() {
    val nav = rememberNavController()
    val context = LocalContext.current
    val navigator = remember(nav) { Navigator(nav, context) }
    val me = rememberLoad(Unit) { app.repo.me() }
    // the site's look, for users the site front end is set up for
    androidx.compose.runtime.LaunchedEffect(Unit) {
        runCatching { app.repo.siteLook() }.onSuccess { SiteMode.apply(context, it) }
    }
    val canViewReels = me.value?.canViewReels ?: true
    val store = SiteMode.storefront

    val tabs = buildList {
        if (store) {
            add(Tab("home", "Home", Icons.Rounded.Home))
            add(Tab("scenes", "Scenes", Icons.Rounded.VideoLibrary))
            add(Tab("models", if (SiteMode.look?.femaleOnly != false) "Pornstars" else "Models", Icons.Rounded.Star))
            if (canViewReels) add(Tab("reels", "Reels", Icons.Rounded.SmartDisplay))
            add(Tab("mylist", "My list", Icons.Rounded.Favorite))
        } else {
            add(Tab("home", "Home", Icons.Rounded.Home))
            add(Tab("browse", "Browse", Icons.Rounded.Search))
            if (canViewReels) add(Tab("reels", "Reels", Icons.Rounded.SmartDisplay))
            add(Tab("playlists", "Playlists", Icons.AutoMirrored.Rounded.PlaylistPlay))
            add(Tab("profile", "You", Icons.Rounded.Person))
        }
    }

    // the age check, when the admin turned it on
    if (store && SiteMode.look?.ageGate == true && !SiteMode.adultOk) {
        AgeGate(onEnter = { SiteMode.confirmAdult(context) })
        return
    }

    val backStack by nav.currentBackStackEntryAsState()
    val current = backStack?.destination?.route

    AutoPreview.Controller()
    CompositionLocalProvider(LocalNav provides navigator, LocalOpenDownloads provides { navigator.downloads() }) {
        Scaffold(
            containerColor = Brand.Background,
            contentWindowInsets = WindowInsets(0),
            bottomBar = {
                NavigationBar(containerColor = Brand.Background.copy(alpha = 0.97f), tonalElevation = 0.dp) {
                    tabs.forEach { t ->
                        val selected = current == t.route
                        NavigationBarItem(
                            selected = selected,
                            onClick = { navigator.tab(t.route) },
                            icon = { Icon(t.icon, contentDescription = t.label) },
                            label = { Text(t.label) },
                            colors = NavigationBarItemDefaults.colors(
                                selectedIconColor = Color.White,
                                selectedTextColor = Color.White,
                                indicatorColor = Brand.Pink.copy(alpha = 0.85f),
                                unselectedIconColor = Brand.TextMuted,
                                unselectedTextColor = Brand.TextMuted,
                            ),
                        )
                    }
                }
            },
        ) { inner ->
            Box(Modifier.fillMaxSize().padding(bottom = inner.calculateBottomPadding())) {
                NavHost(
                    navController = nav,
                    startDestination = "home",
                    enterTransition = { fadeIn() },
                    exitTransition = { fadeOut() },
                ) {
                    composable("home") { if (store) StoreHome() else HomeScreen() }
                    composable("scenes") { StoreScenes() }
                    composable("models") { StoreModels() }
                    composable("mylist") { StoreMyList() }
                    composable("model/{id}") { StoreModel(it.arguments?.getString("id").orEmpty()) }
                    composable("browse") { BrowseScreen() }
                    composable("reels") { ReelsScreen() }
                    composable("playlists") { PlaylistsScreen() }
                    composable("profile") { ProfileScreen(me.value) }
                    composable("scene/{id}") { SceneScreen(it.arguments?.getString("id").orEmpty()) }
                    composable("performer/{id}") { PerformerScreen(it.arguments?.getString("id").orEmpty()) }
                    composable("studio/{id}") { StudioScreen(it.arguments?.getString("id").orEmpty()) }
                    composable("tag/{id}/{name}") {
                        TagScreen(it.arguments?.getString("id").orEmpty(), it.arguments?.getString("name").orEmpty())
                    }
                    composable("downloads") { DownloadsScreen() }
                    composable("playlist/{id}") { PlaylistScreen(it.arguments?.getString("id").orEmpty()) }
                    composable("list/{kind}") {
                        val kind = runCatching { SceneListKind.valueOf(it.arguments?.getString("kind").orEmpty()) }
                            .getOrDefault(SceneListKind.Recent)
                        SceneListScreen(kind)
                    }
                }
            }
        }
    }
}
