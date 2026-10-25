package app.stasharr.data

import android.content.Context
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import okhttp3.Cookie
import okhttp3.CookieJar
import okhttp3.FormBody
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import java.io.IOException
import java.util.concurrent.TimeUnit

/** Saved server address, login and session cookie. */
class Session(context: Context) {
    private val prefs = context.getSharedPreferences("stasharr", Context.MODE_PRIVATE)

    var serverUrl: String?
        get() = prefs.getString("server", null)
        set(v) = prefs.edit().putString("server", v).apply()
    var username: String?
        get() = prefs.getString("username", null)
        set(v) = prefs.edit().putString("username", v).apply()
    var password: String?
        get() = prefs.getString("password", null)
        set(v) = prefs.edit().putString("password", v).apply()
    var sessionCookie: String?
        get() = prefs.getString("cookie", null)
        set(v) = prefs.edit().putString("cookie", v).apply()

    /** True after signing out, until the user signs in again. */
    var loggedOut: Boolean
        get() = prefs.getBoolean("logged_out", false)
        set(v) = prefs.edit().putBoolean("logged_out", v).apply()

    val isSetUp: Boolean get() = !serverUrl.isNullOrBlank()

    fun clear() {
        prefs.edit().clear().apply()
    }
}

class ApiException(message: String, val unauthorized: Boolean = false) : IOException(message)

/** Keeps the server's session cookie, and remembers it between app starts. */
private class SessionCookieJar(private val session: Session) : CookieJar {
    override fun saveFromResponse(url: HttpUrl, cookies: List<Cookie>) {
        cookies.firstOrNull { it.name == "session" }?.let {
            session.sessionCookie = if (it.expiresAt < System.currentTimeMillis()) null else it.value
        }
    }

    override fun loadForRequest(url: HttpUrl): List<Cookie> {
        val value = session.sessionCookie ?: return emptyList()
        return listOf(
            Cookie.Builder().name("session").value(value).hostOnlyDomain(url.host).path("/").build()
        )
    }
}

class Api(private val session: Session) {
    val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        coerceInputValues = true
        isLenient = true
    }

    private val jar = SessionCookieJar(session)
    private val loginLock = Any()

    /** Logs in again when the session has expired, then retries the request once. */
    private val reauth = Interceptor { chain ->
        val request = chain.request()
        val response = chain.proceed(request)
        if (response.code != 401 || request.url.encodedPath.endsWith("/login")) {
            return@Interceptor response
        }
        val user = session.username
        val pass = session.password
        if (user.isNullOrBlank() || pass == null) return@Interceptor response
        response.close()
        synchronized(loginLock) { loginBlocking(user, pass) }
        chain.proceed(request.newBuilder().build())
    }

    private val plainClient: OkHttpClient = OkHttpClient.Builder()
        .cookieJar(jar)
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(60, TimeUnit.SECONDS)
        .build()

    /** Used for everything, including images and video, so they share the login. */
    val client: OkHttpClient = plainClient.newBuilder()
        .addInterceptor(reauth)
        .build()

    val baseUrl: String get() = session.serverUrl.orEmpty()

    private fun url(path: String): HttpUrl =
        (baseUrl.trimEnd('/') + path).toHttpUrlOrNull()
            ?: throw ApiException("That server address doesn't look right")

    private fun loginBlocking(user: String, pass: String) {
        val body = FormBody.Builder().add("username", user).add("password", pass).build()
        val req = Request.Builder().url(url("/login")).post(body).build()
        plainClient.newCall(req).execute().use { res ->
            when {
                res.code == 401 -> throw ApiException("Wrong username or password", unauthorized = true)
                !res.isSuccessful -> throw ApiException("Login failed (${res.code})")
            }
        }
    }

    suspend fun login(user: String, pass: String) = withContext(Dispatchers.IO) {
        loginBlocking(user, pass)
    }

    /** Runs a GraphQL query and returns its "data" object. */
    suspend fun gql(query: String, variables: JsonObject = JsonObject(emptyMap())): JsonObject =
        withContext(Dispatchers.IO) {
            val payload = buildJsonObject {
                put("query", query)
                put("variables", variables)
            }.toString()
            val req = Request.Builder()
                .url(url("/graphql"))
                .post(payload.toRequestBody("application/json".toMediaType()))
                .header("Accept", "application/json")
                .build()
            val res: Response = try {
                client.newCall(req).execute()
            } catch (e: ApiException) {
                throw e
            } catch (e: IOException) {
                throw ApiException("Can't reach the server. Check the address and that you're on the same network.")
            }
            res.use {
                if (it.code == 401) throw ApiException("Please log in", unauthorized = true)
                if (it.code == 403) throw ApiException("The server refused the connection (${it.code})")
                if (!it.isSuccessful) throw ApiException("Server error ${it.code}")
                val text = it.body?.string().orEmpty()
                val root = try {
                    json.parseToJsonElement(text).jsonObject
                } catch (e: Exception) {
                    throw ApiException("This doesn't look like a Stasharr server")
                }
                val data = root["data"]
                val errors = root["errors"]
                if ((data == null || data is JsonNull) && errors is JsonArray && errors.isNotEmpty()) {
                    val msg = errors.first().jsonObject["message"]?.jsonPrimitive?.content ?: "Request failed"
                    throw ApiException(msg)
                }
                (data as? JsonObject) ?: JsonObject(emptyMap())
            }
        }
}

internal fun JsonObject.arr(key: String): JsonArray = this[key]?.let { it as? JsonArray } ?: JsonArray(emptyList())
internal fun JsonObject.obj(key: String): JsonObject? = this[key] as? JsonObject
internal fun JsonObject.int(key: String): Int = this[key]?.jsonPrimitive?.content?.toIntOrNull() ?: 0
internal fun JsonArray.objects(): List<JsonObject> = mapNotNull { it as? JsonObject }
internal fun JsonObject.arrOf(key: String) = this[key]?.jsonArray
