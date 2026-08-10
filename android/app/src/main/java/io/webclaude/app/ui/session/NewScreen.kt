package io.webclaude.app.ui.session

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.navigation.NavHostController
import io.webclaude.app.net.FsEntry
import io.webclaude.app.net.SessionClient
import io.webclaude.app.ui.Routes
import kotlinx.coroutines.launch

@Composable
fun NewScreen(nav: NavHostController, client: SessionClient, wsHolder: Any?) {
    var path by remember { mutableStateOf("") }
    var parent by remember { mutableStateOf("") }
    var entries by remember { mutableStateOf<List<FsEntry>>(emptyList()) }
    var extraArgs by remember { mutableStateOf("") }
    var skipPerms by remember { mutableStateOf(false) }
    var mkdirName by remember { mutableStateOf("") }
    var cloneUrl by remember { mutableStateOf("") }
    var loadError by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun reload(p: String) {
        loadError = null
        scope.launch {
            val r = client.fs(p)
            when (r) {
                is io.webclaude.app.net.ApiResult.Ok -> { path = r.value.cur; parent = r.value.parent; entries = r.value.entries }
                is io.webclaude.app.net.ApiResult.Err -> loadError = r.message
            }
        }
    }
    LaunchedEffect(Unit) { reload("") }

    fun create(p: String, resumeId: String? = null, cloneUrl_: String? = null, cloneName: String? = null, contLast: Boolean = false) {
        scope.launch {
            val args = buildList {
                if (skipPerms) add("--dangerously-skip-permissions")
                if (extraArgs.isNotBlank()) addAll(extraArgs.split(Regex("\\s+")))
            }
            val r = client.createSession(p, resumeId, contLast, cloneUrl_, cloneName, args)
            when (r) {
                is io.webclaude.app.net.ApiResult.Ok -> nav.navigate(Routes.TERM.replace("{id}", r.value.id)) { popUpTo(Routes.SESS) }
                is io.webclaude.app.net.ApiResult.Err -> loadError = r.message
            }
        }
    }

    Column(Modifier.fillMaxSize().padding(12.dp)) {
        Row(verticalAlignment = androidx.compose.ui.Alignment.CenterVertically) {
            Text("新建会话", style = MaterialTheme.typography.headlineSmall)
            Spacer(Modifier.weight(1f))
            Button(onClick = { nav.popBackStack() }) { Text("返回") }
        }
        Text(path.ifBlank { "/" }, style = MaterialTheme.typography.bodySmall)
        loadError?.let { Text(it, color = MaterialTheme.colorScheme.error) }

        LazyColumn(Modifier.weight(1f)) {
            if (parent.isNotEmpty()) {
                item {
                    Card(Modifier.fillMaxWidth().padding(vertical=4.dp)) {
                        Button(onClick = { reload(parent) }, modifier = Modifier.fillMaxWidth()) { Text("↑ 上级") }
                    }
                }
            }
            items(entries, key = { it.path }) { e ->
                androidx.compose.material3.Card(
                    onClick = { if (e.isDir) reload(e.path) },
                    modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)
                ) {
                    Row(Modifier.padding(12.dp)) {
                        Text(if (e.isDir) "📁" else "🗎")
                        Spacer(Modifier.padding(4.dp))
                        Text(e.name, style = MaterialTheme.typography.bodyLarge)
                    }
                }
            }
        }

        HorizontalDivider()
        // options
        Column(Modifier.fillMaxWidth().padding(vertical=6.dp)) {
            Text("额外参数（如 --dangerously-skip-permissions）", style = MaterialTheme.typography.labelMedium)
            OutlinedTextField(value = extraArgs, onValueChange = { extraArgs = it }, modifier = Modifier.fillMaxWidth(), singleLine = true)
        }
        Row(Modifier.fillMaxWidth()) {
            Button(onClick = { create(path) }, Modifier.weight(1f)) { Text("在此目录新建") }
            Spacer(Modifier.padding(4.dp))
            Button(onClick = { create(path, contLast = true) }, Modifier.weight(1f)) { Text("继续最近") }
        }
        Row(Modifier.fillMaxWidth()) {
            OutlinedTextField(value = mkdirName, onValueChange = { mkdirName = it }, label = { Text("新文件夹名") }, modifier = Modifier.weight(1f))
            Spacer(Modifier.padding(4.dp))
            Button(onClick = {
                if (mkdirName.isNotBlank()) { scope.launch { client.mkdir(path, mkdirName.trim()); reload(path) }; mkdirName = "" }
            }, Modifier.weight(1f)) { Text("新建文件夹") }
        }
        Row(Modifier.fillMaxWidth()) {
            OutlinedTextField(value = cloneUrl, onValueChange = { cloneUrl = it }, label = { Text("Git URL") }, modifier = Modifier.weight(1f), singleLine = true)
            Spacer(Modifier.padding(4.dp))
            Button(onClick = { if (cloneUrl.isNotBlank()) create(path, cloneUrl_ = cloneUrl.trim()) }, Modifier.weight(1f)) { Text("Clone") }
        }
    }
}

object RoutesLocal {
    const val NEW = "new"
}