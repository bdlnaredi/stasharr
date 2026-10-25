package app.stasharr

import android.app.Application
import app.stasharr.data.Api
import app.stasharr.data.Repository
import app.stasharr.data.Session
import coil.ImageLoader
import coil.ImageLoaderFactory
import coil.disk.DiskCache
import coil.memory.MemoryCache
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob

class StasharrApp : Application(), ImageLoaderFactory {
    lateinit var session: Session
        private set
    lateinit var api: Api
        private set
    lateinit var repo: Repository
        private set

    /** For work that must finish even when a screen closes, like saving the play position. */
    val appScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onCreate() {
        super.onCreate()
        instance = this
        session = Session(this)
        api = Api(session)
        repo = Repository(api)
    }

    // images use the same connection and login as everything else
    override fun newImageLoader(): ImageLoader = ImageLoader.Builder(this)
        .okHttpClient(api.client)
        .crossfade(true)
        .memoryCache { MemoryCache.Builder(this).maxSizePercent(0.25).build() }
        .diskCache {
            DiskCache.Builder()
                .directory(cacheDir.resolve("images"))
                .maxSizeBytes(256L * 1024 * 1024)
                .build()
        }
        .respectCacheHeaders(false)
        .build()

    companion object {
        lateinit var instance: StasharrApp
            private set
    }
}

val app: StasharrApp get() = StasharrApp.instance
