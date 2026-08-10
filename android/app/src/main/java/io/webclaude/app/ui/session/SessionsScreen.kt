package io.webclaude.app.ui.session

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.navigation.NavHostController
import io.webclaude.app.net.ApiResult
import io.webclaude.app.net.CliSession
import io.webclaude.app.net.SessionClient
import io.webclaude.app.ui.Routes
import io.webclaude.app.ui.theme.MiuiTopBar
import io.webclaude.app.ui.theme.PrimaryBlue
import io.webclaude.app.ui.theme.TextPrimary
import io.webclaude.app.ui.theme.TextSecondary
import io.webclaude.app.ui.theme.TextTertiary
import kotlinx.coroutines.launch

@Composable
fun SessionsScreen(nav: NavHostController, client: SessionClient) {
    var sessions by remember { mutableStateOf<List<CliSession>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var projectsRoot by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        loading = true
        val m = client.me()
        if (m is ApiResult.Ok) projectsRoot = m.value
        val r = client.sessions()
        when (r) {
            is ApiResult.Ok -> { sessions = r.value; loading = false }
            is ApiResult.Err -> { loading = false; if (r.code == 401) nav.navigate(Routes.LOGIN) { popUpTo(0) } }
        }
    }

    Column(Modifier.fillMaxSize()) {
        MiuiTopBar(
            title = "会话",
            subtitle = projectsRoot.ifEmpty { "本地项目" },
            right = {
                Button(
                    onClick = { nav.navigate(Routes.NEW) },
                    colors = ButtonDefaults.buttonColors(containerColor = PrimaryBlue),
                    shape = RoundedCornerShape(999.dp),
                    contentPadding = androidx.compose.foundation.layout.PaddingValues(horizontal = 14.dp, vertical = 6.dp)
                ) {
                    Text("新建", color = Color.White, fontSize = 14.sp)
                }
            },
        )

        if (loading) {
            Column(Modifier.fillMaxSize(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = androidx.compose.foundation.layout.Arrangement.Center) {
                Text("加载中…", color = TextSecondary)
            }
        } else if (sessions.isEmpty()) {
            Column(Modifier.fillMaxSize(), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = androidx.compose.foundation.layout.Arrangement.Center) {
                Text("⌘", fontSize = 28.sp, color = TextTertiary)
                Text("暂无会话", color = TextSecondary, modifier = Modifier.padding(top = 12.dp))
            }
        } else {
            LazyColumn(Modifier.fillMaxSize().padding(horizontal = 14.dp, vertical = 6.dp)) {
                items(sessions, key = { it.id }) { s ->
                    SessionCard(s, onOpen = { nav.navigate(Routes.TERM.replace("{id}", s.id)) }, onKill = {
                        scope.launch { client.killSession(s.id); val r2 = client.sessions(); if (r2 is ApiResult.Ok) sessions = r2.value }
                    })
                }
            }
        }
    }
}

@Composable
private fun SessionCard(s: CliSession, onOpen: () -> Unit, onKill: () -> Unit) {
    androidx.compose.foundation.layout.Box(Modifier.fillMaxWidth().padding(bottom = 10.dp)) {
        Surface(shape = RoundedCornerShape(16.dp), color = MaterialTheme.colorScheme.surface, modifier = Modifier.fillMaxWidth()) {
            Column(Modifier.padding(14.dp)) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    // avatar
                    androidx.compose.foundation.layout.Box(
                        Modifier.size(42.dp).padding(end = 10.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        Surface(
                            shape = RoundedCornerShape(12.dp),
                            color = if (s.alive) PrimaryBlue else Color(0xFF636366),
                            modifier = Modifier.size(42.dp)
                        ) {}
                        Text(
                            s.title.ifBlank { s.cwdRel }.firstOrNull()?.uppercase() ?: "C",
                            color = Color.White, fontWeight = FontWeight.Bold, fontSize = 17.sp
                        )
                    }
                    Column(Modifier.weight(1f)) {
                        Text(s.title.ifBlank { s.cwdRel }, color = TextPrimary, fontSize = 17.sp, fontWeight = FontWeight.SemiBold, maxLines = 1)
                        Text(s.cwd, color = TextSecondary, fontSize = 13.sp, maxLines = 1)
                    }
                }
                Row(Modifier.padding(top = 10.dp)) {
                    Text(if (s.alive) "运行中" else "已退出", color = if (s.alive) Color(0xFF30D158) else Color(0xFFFF453A), fontSize = 12.sp, modifier = Modifier.padding(top = 6.dp))
                    Spacer(Modifier.weight(1f))
                    Button(onClick = onOpen, colors = ButtonDefaults.buttonColors(containerColor = PrimaryBlue), shape = RoundedCornerShape(999.dp)) {
                        Text("进入", color = Color.White)
                    }
                    TextButton(onClick = onKill, modifier = Modifier.padding(start = 4.dp)) {
                        Text("结束", color = Color(0xFFFF453A))
                    }
                }
            }
        }
    }
}