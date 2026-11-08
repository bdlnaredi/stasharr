package app.stasharr

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import app.stasharr.alerts.FavoriteAlertsState
import app.stasharr.ui.StasharrRoot
import app.stasharr.ui.theme.StasharrTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.dark(android.graphics.Color.TRANSPARENT),
            navigationBarStyle = SystemBarStyle.dark(android.graphics.Color.TRANSPARENT),
        )
        super.onCreate(savedInstanceState)
        openFrom(intent)
        setContent {
            StasharrTheme {
                StasharrRoot()
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        openFrom(intent)
    }

    // a notification about a new scene opens that scene
    private fun openFrom(intent: Intent?) {
        intent?.getStringExtra(FavoriteAlertsState.EXTRA_SCENE)?.let {
            FavoriteAlertsState.openScene = it
            intent.removeExtra(FavoriteAlertsState.EXTRA_SCENE)
        }
    }
}
