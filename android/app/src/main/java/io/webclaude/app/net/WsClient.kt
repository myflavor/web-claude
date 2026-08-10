package io.webclaude.app.net

import org.java_websocket.client.WebSocketClient
import org.java_websocket.handshake.ServerHandshake
import java.net.URI
import java.nio.ByteBuffer
import java.nio.charset.StandardCharsets
import kotlin.concurrent.thread

/**
 * Terminal WebSocket client. Sends raw bytes to the PTY; receives raw bytes
 * from the ring buffer. Also supports {type:resize/input} JSON control messages.
 */
class WsClient(
    private val onMessage: (ByteArray) -> Unit = {},
    private val onState: (String) -> Unit = {},
) {
    @Volatile
    var baseUrl: String = "http://<NAS>:3080"

    private var sock: WebSocketClient? = null
    @Volatile
    var open: Boolean = false

    fun connect(sessionId: String) {
        try {
            sock?.close()
        } catch (_: Exception) {}
        val uri = URI("$baseUrl/api/sessions/$sessionId/ws")
        val client = object : WebSocketClient(uri) {
            override fun onOpen(h: ServerHandshake) {
                open = true
                onState("connected")
            }
            override fun onMessage(bytes: ByteBuffer) {
                val arr = ByteArray(bytes!!.remaining())
                bytes.get(arr)
                onMessage(arr)
            }
            override fun onMessage(message: String) {
                onMessage(message.toByteArray(StandardCharsets.UTF_8))
            }
            override fun onClose(code: Int, reason: String?, remote: Boolean) {
                open = false
                onState("closed")
            }
            override fun onError(ex: Exception) {
                open = false
                onState("error")
            }
        }
        this.sock = client
        client.connect()
    }

    fun send(bytes: ByteArray) {
        val s = sock ?: return
        if (!s.isOpen) return
        s.send(bytes)
    }

    fun sendResize(cols: Int, rows: Int) {
        val msg = "{\"type\":\"resize\",\"cols\":$cols,\"rows\":$rows}"
        val s = sock ?: return
        if (s.isOpen) s.send(msg)
    }

    fun close() {
        try { sock?.close() } catch (_: Exception) {}
        open = false
    }
}