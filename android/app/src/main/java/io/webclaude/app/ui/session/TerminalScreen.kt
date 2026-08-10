package io.webclaude.app.ui.session

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material3.Button
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.navigation.NavHostController
import io.webclaude.app.net.SessionClient
import io.webclaude.app.net.WsClient
import io.webclaude.app.term.TerminalSession
import io.webclaude.app.term.TerminalView

@Composable
fun TerminalScreen(
    nav: NavHostController,
    id: String,
    sessionId: androidx.compose.runtime.MutableState<String?>,
    wsHolder: WsClient,
) {
    val session = remember { TerminalSession() }
    var status by remember { mutableStateOf("连接中…") }
    var ws by remember { mutableStateOf<WsClient?>(null) }
    var input by remember { mutableStateOf("") }
    val client = remember { SessionClient() }

    LaunchedEffect(id) {
        sessionId.value = id
        val w = WsClient(
            onMessage = { bytes -> session.feed(bytes) },
            onState = { s -> status = s }
        )
        w.baseUrl = client.baseUrl
        w.connect(id)
        ws = w
    }

    Column(Modifier.fillMaxSize()) {
        Row(Modifier.fillMaxWidth().statusBarsPadding().padding(8.dp)) {
            Button(onClick = { ws?.close(); nav.popBackStack() }) { Text("返回") }
            Spacer(Modifier.width(8.dp))
            Text(status, modifier = Modifier.padding(top = 12.dp), color = MaterialTheme.colorScheme.secondary)
        }
        TerminalView(session, onResize = { c, r -> ws?.sendResize(c, r) })
        HorizontalDivider()
        Row(Modifier.fillMaxWidth().padding(6.dp)) {
            BasicTextField(
                value = input,
                onValueChange = { input = it },
                modifier = Modifier.weight(1f).heightIn(min = 40.dp).background(MaterialTheme.colorScheme.surfaceVariant).padding(8.dp),
                singleLine = false,
                textStyle = androidx.compose.ui.text.TextStyle(color = Color.White, fontSize = 14.sp)
            )
            Spacer(Modifier.width(4.dp))
            Button(onClick = {
                val text = input.ifEmpty { "" }
                if (text.isNotEmpty()) ws?.send(text.toByteArray())
                input = ""
                ws?.send("\n".toByteArray())
            }) { Text("发送") }
        }
    }
}