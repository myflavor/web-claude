package io.webclaude.app.net

import okhttp3.Cookie
import okhttp3.CookieJar
import okhttp3.HttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import org.json.JSONArray
import org.json.JSONObject
import java.io.File
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.TimeUnit

data class CliSession(
    val id: String,
    val title: String,
    val cwd: String,
    val cwdRel: String,
    val createdAt: String,
    val alive: Boolean,
    val clients: Int,
)

data class FsEntry(val name: String, val path: String, val isDir: Boolean)

data class Conversation(
    val sessionId: String,
    val display: String,
    val updatedAt: String,
    val cwdRel: String,
)
object JSON_ { }

sealed class ApiResult<out T> {
    data class Ok<T>(val value: T) : ApiResult<T>()
    data class Err(val message: String, val code: Int = 0) : ApiResult<Nothing>()
}

/**
 * Blocking client for web-claude REST. Non-suspend so it can be called from
 * background threads/coroutines uniformly. Keeps a session cookie jar.
 */
class SessionClient(baseUrlIn: String? = null) {

    @Volatile
    var baseUrl: String = baseUrlIn ?: "http://<NAS>:3080"
        private set

    // Minimal in-memory cookie jar (avoids external okhttp JavaNetCookieJar dep).
    private val cookieStore = ConcurrentHashMap<String, MutableList<Cookie>>()
    private val jar = object : CookieJar {
        override fun saveFromResponse(url: HttpUrl, cookies: List<Cookie>) {
            cookieStore.computeIfAbsent(url.host) { mutableListOf() } += cookies
        }
        override fun loadForRequest(url: HttpUrl): List<Cookie> =
            cookieStore[url.host] ?: emptyList()
    }

    private val http = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(90, TimeUnit.SECONDS)
        .cookieJar(jar)
        .build()

    fun setBase(url: String) {
        baseUrl = url.trimEnd('/')
    }

    private fun body(r: Response): JSONObject =
        try { JSONObject(r.body?.string() ?: "{}") } catch (_: Exception) { JSONObject() }

    fun login(token: String): ApiResult<Unit> {
        val req = Request.Builder()
            .url("$baseUrl/api/login")
            .post(JSONObject().put("token", token).toString().toRequestBody("application/json".toMediaType()))
            .build()
        val r = exec(req)
        return if (r.ok) ApiResult.Ok(Unit) else ApiResult.Err(r.error)
    }

    fun me(): ApiResult<String> {
        val r = exec(Request.Builder().url("$baseUrl/api/me").get().build())
        return if (r.ok) ApiResult.Ok(r.json.optString("projectsRoot", "")) else ApiResult.Err(r.error)
    }

    fun sessions(): ApiResult<List<CliSession>> {
        val r = exec(Request.Builder().url("$baseUrl/api/sessions").get().build())
        if (!r.ok) return ApiResult.Err(r.error)
        return ApiResult.Ok(r.json.optJSONArray("sessions")?.let { arr ->
            (0 until arr.length()).map { i ->
                val o = arr.getJSONObject(i)
                CliSession(
                    o.optString("id"), o.optString("title"), o.optString("cwd"),
                    o.optString("cwdRel"), o.optString("createdAt"),
                    o.optBoolean("alive"), o.optInt("clients"),
                )
            }
        } ?: emptyList())
    }

    fun fs(path: String): ApiResult<FsList> {
        val url = "$baseUrl/api/fs?path=${java.net.URLEncoder.encode(path, "UTF-8")}"
        val r = exec(Request.Builder().url(url).get().build())
        if (!r.ok) return ApiResult.Err(r.error)
        val o = r.json
        val entries = o.optJSONArray("entries")?.let { arr ->
            (0 until arr.length()).map { i ->
                val e = arr.getJSONObject(i)
                FsEntry(e.optString("name"), e.optString("path"), e.optBoolean("isDir"))
            }
        } ?: emptyList()
        return ApiResult.Ok(FsList(o.optString("path", ""), o.optString("parent", ""), entries.sortedWith(compareBy({ !it.isDir }, { it.name.lowercase() }))))
    }

    fun mkdir(path: String, name: String): ApiResult<Unit> {
        val r = exec(Request.Builder().url("$baseUrl/api/fs/mkdir")
            .post(JSONObject().put("path", path).put("name", name).toString().toRequestBody("application/json".toMediaType())).build())
        return if (r.ok) ApiResult.Ok(Unit) else ApiResult.Err(r.error)
    }

    fun conversations(path: String): ApiResult<List<Conversation>> {
        val url = "$baseUrl/api/conversations?path=${java.net.URLEncoder.encode(path, "UTF-8")}"
        val r = exec(Request.Builder().url(url).get().build())
        if (!r.ok) return ApiResult.Err(r.error)
        return ApiResult.Ok(r.json.optJSONArray("conversations")?.let { arr ->
            (0 until arr.length()).map { i ->
                val o = arr.getJSONObject(i)
                Conversation(o.optString("sessionId"), o.optString("display"), o.optString("updatedAt"), o.optString("cwdRel"))
            }
        } ?: emptyList())
    }

    fun createSession(
        path: String,
        resumeId: String? = null,
        continueLast: Boolean = false,
        cloneUrl: String? = null,
        cloneName: String? = null,
        extraArgs: List<String> = emptyList(),
    ): ApiResult<CliSession> {
        val o = JSONObject()
            .put("path", path)
            .put("resumeId", resumeId ?: "")
            .put("continueLast", continueLast)
            .put("cloneUrl", cloneUrl ?: "")
            .put("cloneName", cloneName ?: "")
            .put("claudeArgs", JSONArray(extraArgs))
        val r = exec(Request.Builder().url("$baseUrl/api/sessions")
            .post(o.toString().toRequestBody("application/json".toMediaType())).build())
        if (!r.ok) return ApiResult.Err(r.error)
        val j = r.json
        return ApiResult.Ok(CliSession(j.optString("id"), j.optString("title"), j.optString("cwd"), j.optString("cwdRel"), j.optString("createdAt"), j.optBoolean("alive"), j.optInt("clients")))
    }

    fun killSession(id: String): ApiResult<Unit> {
        val r = exec(Request.Builder().url("$baseUrl/api/sessions/$id").delete().build())
        return if (r.ok) ApiResult.Ok(Unit) else ApiResult.Err(r.error)
    }

    fun upload(sessionId: String, path: String): ApiResult<String> {
        try {
            val f = File(path)
            require(f.exists()) { "file not found" }
            val boundary = "----wc" + System.currentTimeMillis()
            val header = "--$boundary\r\nContent-Disposition: form-data; name=\"file\"; filename=\"${f.name}\"\r\nContent-Type: application/octet-stream\r\n\r\n"
            val footer = "\r\n--$boundary--\r\n"
            val bytes = header.toByteArray() + f.readBytes() + footer.toByteArray()
            val req = Request.Builder()
                .url("$baseUrl/api/sessions/$sessionId/upload")
                .post(bytes.toRequestBody("multipart/form-data; boundary=$boundary".toMediaType()))
                .build()
            val r = exec(req)
            return if (r.ok) ApiResult.Ok(r.json.optString("path")) else ApiResult.Err(r.error)
        } catch (e: Exception) {
            return ApiResult.Err(e.message ?: "upload failed")
        }
    }

    private class Res(val ok: Boolean, val json: JSONObject, val error: String) {
        constructor(r: Response) : this(r.isSuccessful, bodyOf(r), if (r.isSuccessful) "" else try { JSONObject(r.body?.string() ?: "{}").optString("error") } catch (_: Exception) { r.message }, )
        companion object {
            private fun bodyOf(r: Response): JSONObject =
                try { JSONObject(r.body?.string() ?: "{}") } catch (_: Exception) { JSONObject() }
            fun cat(r: Response): Res = Res(r)
        }
    }

    private fun exec(req: Request): Res {
        return try {
            http.newCall(req).execute().use { r -> Res(r) }
        } catch (e: Exception) {
            Res(false, JSONObject(), e.message ?: "network error")
        }
    }
}

data class FsList(val cur: String, val parent: String, val entries: List<FsEntry>)