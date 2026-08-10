package io.webclaude.app.ui.session

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.navigation.NavHostController
import io.webclaude.app.net.CliSession
import io.webclaude.app.net.SessionClient
import io.webclaude.app.ui.Routes
import kotlinx.coroutines.launch

@Composable
fun SessionsScreen(nav: NavHostController, client: SessionClient) {
    var sessions by remember { mutableStateOf<List<CliSession>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    val scope = rememberCoroutineScope()
    var projectsRoot by remember { mutableStateOf("") }

    LaunchedEffect(Unit) {
        loading = true
        val m = client.me()
        if (m is io.webclaude.app.net.ApiResult.Ok) projectsRoot = m.value
        val r = client.sessions()
        when (r) {
            is io.webclaude.app.net.ApiResult.Ok -> { sessions = r.value; loading = false }
            is io.webclaude.app.net.ApiResult.Err -> { loading = false; if (r.code==401) nav.navigate(Routes.LOGIN){popUpTo(0)} }
        }
    }

    Column(Modifier.fillMaxSize().padding(12.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text("会话", style = MaterialTheme.typography.headlineSmall)
            Spacer(Modifier.weight(1f))
            Button(onClick = { nav.navigate(Routes.NEW) }) { Text("新建") }
        }
        Spacer(Modifier.height(8.dp))
        if (loading) {
            Text("加载中…")
        } else if (sessions.isEmpty()) {
            Text("暂无会话")
        } else {
            LazyColumn {
                items(sessions, key = { it.id }) { s ->
                    Card(Modifier.fillMaxWidth().padding(bottom = 8.dp), colors = CardDefaults.cardColors(MaterialTheme.colorScheme.surfaceVariant)) {
                        Column(Modifier.padding(12.dp)) {
                            Text(s.title.ifBlank { s.cwdRel }, style = MaterialTheme.typography.titleMedium)
                            Text(s.cwd, style = MaterialTheme.typography.bodySmall)
                            Row {
                                Text(if (s.alive) "运行中" else "已退出", color = if (s.alive) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.error)
                                Spacer(Modifier.weight(1f))
                                Button(onClick = { nav.navigate(Routes.TERM.replace("{id}", s.id)) }) { Text("进入") }
                                OutlinedButton(onClick = {
                                    scope.launch { client.killSession(s.id); val r2=client.sessions(); if (r2 is io.webclaude.app.net.ApiResult.Ok) sessions = r2.value }
                                }, modifier = Modifier.padding(start=4.dp)) { Text("结束") }
                            }
                        }
                    }
                }
            }
        }
    }
}