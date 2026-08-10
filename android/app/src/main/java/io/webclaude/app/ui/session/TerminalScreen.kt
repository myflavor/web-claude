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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.navigation.NavHostController
import io.webclaude.app.net.SessionClient
import io.webclaude.app.net.WsClient
import io.webclaude.app.term.TerminalSession
import io.webclaude.app.term.TerminalView
import io.webclaude.app.ui.theme.BgSurface2
import io.webclaude.app.ui.theme.MiuiTopBar
import io.webclaude.app.ui.theme.PrimaryBlue
import io.webclaude.app.ui.theme.TextPrimary

@Composable
fun TerminalScreen(
    nav: NavHostController,
    id: String,
    sessionId: androidx.compose.runtime.MutableState<String?>,
    wsHolder: WsClient,
    client: SessionClient,
) {
    val session = remember { TerminalSession() }
    var status by remember { mutableStateOf("连接中…") }
    var ws by remember { mutableStateOf<WsClient?>(null) }
    var input by remember { mutableStateOf("") }

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
        MiuiTopBar(
            title = "会话",
            subtitle = status,
            left = {
                TextButton(onClick = { ws?.close(); nav.popBackStack() }) { Text("返回", color = PrimaryBlue) }
            },
            right = {
                TextButton(onClick = { ws?.close(); nav.popBackStack() }) { Text("结束", color = Color(0xFFFF453A)) }
            },
        )
        TerminalView(session, onResize = { c, r -> ws?.sendResize(c, r) })
        HorizontalDivider(thickness = 0.5.dp, color = Color.Black.copy(alpha = 0.4f))
        Row(Modifier.fillMaxWidth().padding(6.dp)) {
            BasicTextField(
                value = input,
                onValueChange = { input = it },
                modifier = Modifier.weight(1f).heightIn(min = 42.dp).background(BgSurface2, RoundedCornerShape(10.dp)).padding(8.dp),
                singleLine = false,
                textStyle = TextStyle(color = TextPrimary, fontSize = 14.sp)
            )
            Spacer(Modifier.width(6.dp))
            Button(
                onClick = {
                    val text = input.trim()
                    if (text.isNotEmpty()) ws?.send(text.toByteArray())
                    input = ""
                    if (text.isNotEmpty()) ws?.send("\n".toByteArray())
                },
                colors = ButtonDefaults.buttonColors(containerColor = PrimaryBlue),
                shape = RoundedCornerShape(10.dp),
            ) { Text("发送", color = Color.White) }
        }
    }
}