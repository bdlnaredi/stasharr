package app.stasharr.player

import android.content.Context
import androidx.annotation.OptIn
import androidx.media3.common.AudioAttributes
import androidx.media3.common.C
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultDataSource
import androidx.media3.datasource.okhttp.OkHttpDataSource
import androidx.media3.exoplayer.DefaultRenderersFactory
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.mediacodec.MediaCodecSelector
import androidx.media3.exoplayer.mediacodec.MediaCodecUtil
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.ui.AspectRatioFrameLayout
import app.stasharr.app

/** How video is decoded. */
enum class Decoder(val label: String, val description: String) {
    Hardware("Phone – hardware", "Fastest and easiest on the battery. Best for most videos."),
    Software("Phone – software", "Slower, but can play some files the hardware decoder can't."),
    Server("Server transcode", "Your server converts the video while it plays. Uses the server's power; works for any file."),
}

/** How the picture fills the screen. */
enum class Aspect(val label: String, val resizeMode: Int) {
    Fit("Fit", AspectRatioFrameLayout.RESIZE_MODE_FIT),
    Fill("Crop to fill", AspectRatioFrameLayout.RESIZE_MODE_ZOOM),
    Stretch("Stretch", AspectRatioFrameLayout.RESIZE_MODE_FILL),
}

/** Player choices remembered between videos. */
class PlayerPrefs(context: Context) {
    private val prefs = context.getSharedPreferences("player", Context.MODE_PRIVATE)

    var decoder: Decoder
        get() = runCatching { Decoder.valueOf(prefs.getString("decoder", null) ?: "") }.getOrDefault(Decoder.Hardware)
        set(v) = prefs.edit().putString("decoder", v.name).apply()

    var aspect: Aspect
        get() = runCatching { Aspect.valueOf(prefs.getString("aspect", null) ?: "") }.getOrDefault(Aspect.Fit)
        set(v) = prefs.edit().putString("aspect", v.name).apply()
}

/** An ExoPlayer that uses the app's login for every request. */
@OptIn(UnstableApi::class)
fun newPlayer(context: Context, decoder: Decoder = Decoder.Hardware): ExoPlayer {
    val http = OkHttpDataSource.Factory(app.api.client)
    val data = DefaultDataSource.Factory(context, http)

    // order the phone's decoders: hardware first, or software first
    val selector = MediaCodecSelector { mimeType, secure, tunneling ->
        val all = MediaCodecUtil.getDecoderInfos(mimeType, secure, tunneling)
        when (decoder) {
            Decoder.Software -> all.sortedByDescending { it.softwareOnly }
            else -> all.sortedByDescending { it.hardwareAccelerated }
        }
    }
    val renderers = DefaultRenderersFactory(context)
        .setMediaCodecSelector(selector)
        // if the first decoder fails, try the next one
        .setEnableDecoderFallback(true)

    return ExoPlayer.Builder(context, renderers)
        .setMediaSourceFactory(DefaultMediaSourceFactory(data))
        .setAudioAttributes(
            AudioAttributes.Builder()
                .setUsage(C.USAGE_MEDIA)
                .setContentType(C.AUDIO_CONTENT_TYPE_MOVIE)
                .build(),
            true,
        )
        .setSeekBackIncrementMs(10_000)
        .setSeekForwardIncrementMs(10_000)
        .build()
}
