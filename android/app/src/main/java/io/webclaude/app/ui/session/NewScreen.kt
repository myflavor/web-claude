package io.webclaude.app.ui.session

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
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
import io.webclaude.app.net.FsEntry
import io.webclaude.app.net.SessionClient
import io.webclaude.app.ui.Routes
import io.webclaude.app.ui.theme.BgSurface2
import io.webclaude.app.ui.theme.MiuiTopBar
import io.webclaude.app.ui.theme.PrimaryBlue
import io.webclaude.app.ui.theme.TextPrimary
import io.webclaude.app.ui.theme.TextSecondary
import kotlinx.coroutines.launch

@Composable
fun NewScreen(nav: NavHostController, client: SessionClient) {
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
                is ApiResult.Ok -> { path = r.value.cur; parent = r.value.parent; entries = r.value.entries }
                is ApiResult.Err -> loadError = r.message
            }
        }
    }
    LaunchedEffect(Unit) { reload("") }

    fun create(p: String, resumeId: String? = null, cloneUrl: String? = null, cloneName: String? = null, contLast: Boolean = false) {
        scope.launch {
            val args = buildList {
                if (skipPerms) add("--dangerously-skip-permissions")
                if (extraArgs.isNotBlank()) addAll(extraArgs.split(Regex("\\s+")))
            }
            val r = client.createSession(p, resumeId, contLast, cloneUrl, cloneName, args)
            when (r) {
                is ApiResult.Ok -> nav.navigate(Routes.TERM.replace("{id}", r.value.id)) { popUpTo(Routes.SESS) }
                is ApiResult.Err -> loadError = r.message
            }
        }
    }

    Column(Modifier.fillMaxSize()) {
        MiuiTopBar(
            title = "新建会话",
            subtitle = path.ifBlank { "/" },
            left = {
                TextButton(onClick = { nav.popBackStack() }) { Text("返回", color = PrimaryBlue) }
            },
            right = {
                Button(onClick = { create(path) }, colors = ButtonDefaults.buttonColors(containerColor = PrimaryBlue), shape = RoundedCornerShape(999.dp), contentPadding = PaddingValues(horizontal = 14.dp, vertical = 6.dp)) {
                    Text("新建", color = Color.White, fontSize = 14.sp)
                }
            },
        )

        loadError?.let { Text(it, color = MaterialTheme.colorScheme.error, fontSize = 13.sp, modifier = Modifier.padding(horizontal = 16.dp)) }

        // Tools row
        Row(Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 6.dp), horizontalArrangement = androidx.compose.foundation.layout.Arrangement.spacedBy(8.dp)) {
            OutlinedButtonCompact("新建文件夹") { if (mkdirName.isNotBlank()) { scope.launch { client.mkdir(path, mkdirName.trim()); reload(path) }; mkdirName = "" } }
            OutlinedButtonCompact("Git Clone") { if (cloneUrl.isNotBlank()) create(path, cloneUrl = cloneUrl.trim()) }
            OutlinedButtonCompact("继续最近") { create(path, contLast = true) }
        }

        LazyColumn(Modifier.weight(1f).fillMaxWidth().padding(horizontal = 14.dp)) {
            if (parent.isNotEmpty()) {
                item { DirCell(name = "↑ 上级", isDir = true) { reload(parent) } }
            }
            items(entries, key = { it.path }) { e ->
                DirCell(name = e.name, isDir = e.isDir) { if (e.isDir) reload(e.path) }
            }
        }

        // Options + actions
        androidx.compose.material3.HorizontalDivider(color = androidx.compose.ui.graphics.Color.Black.copy(alpha = 0.4f), thickness = 0.5.dp)
        Column(Modifier.fillMaxWidth().padding(14.dp)) {
            Text("额外参数", fontSize = 12.sp, color = TextSecondary)
            OutlinedTextField(
                value = extraArgs, onValueChange = { extraArgs = it },
                singleLine = true,
                textStyle = androidx.compose.ui.text.TextStyle(color = TextPrimary, fontSize = 15.sp),
                colors = OutlinedTextFieldDefaults.colors(focusedBorderColor = PrimaryBlue, unfocusedBorderColor = BgSurface2, focusedContainerColor = BgSurface2, unfocusedContainerColor = BgSurface2),
                modifier = Modifier.fillMaxWidth()
            )
            Text("新文件夹名", fontSize = 12.sp, color = TextSecondary, modifier = Modifier.padding(top = 8.dp))
            OutlinedTextField(
                value = mkdirName, onValueChange = { mkdirName = it },
                singleLine = true,
                textStyle = androidx.compose.ui.text.TextStyle(color = TextPrimary, fontSize = 15.sp),
                colors = OutlinedTextFieldDefaults.colors(focusedBorderColor = PrimaryBlue, unfocusedBorderColor = BgSurface2, focusedContainerColor = BgSurface2, unfocusedContainerColor = BgSurface2),
                modifier = Modifier.fillMaxWidth()
            )
            Text("Git URL", fontSize = 12.sp, color = TextSecondary, modifier = Modifier.padding(top = 8.dp))
            OutlinedTextField(
                value = cloneUrl, onValueChange = { cloneUrl = it },
                singleLine = true,
                textStyle = androidx.compose.ui.text.TextStyle(color = TextPrimary, fontSize = 15.sp),
                colors = OutlinedTextFieldDefaults.colors(focusedBorderColor = PrimaryBlue, unfocusedBorderColor = BgSurface2, focusedContainerColor = BgSurface2, unfocusedContainerColor = BgSurface2),
                modifier = Modifier.fillMaxWidth()
            )
        }
    }
}

@Composable
private fun OutlinedButtonCompact(text: String, onClick: () -> Unit) {
    Button(
        onClick = onClick,
        colors = ButtonDefaults.buttonColors(containerColor = BgSurface2),
        shape = RoundedCornerShape(999.dp),
        contentPadding = PaddingValues(horizontal = 12.dp, vertical = 5.dp),
    ) {
        Text(text, color = TextPrimary, fontSize = 13.sp)
    }
}

@Composable
private fun DirCell(name: String, isDir: Boolean, onClick: () -> Unit) {
    androidx.compose.foundation.layout.Box(Modifier.fillMaxWidth().padding(bottom = 8.dp)) {
        Surface(shape = RoundedCornerShape(12.dp), color = MaterialTheme.colorScheme.surface, modifier = Modifier.fillMaxWidth()) {
            androidx.compose.material3.Button(
                onClick = onClick,
                colors = ButtonDefaults.buttonColors(containerColor = androidx.compose.ui.graphics.Color.Transparent),
                elevation = null,
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(12.dp),
            ) {
                androidx.compose.foundation.layout.Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.CenterStart) {
                    Text(text = if (isDir) "📁  $name" else "🗎  $name", color = TextPrimary, fontSize = 16.sp, fontWeight = FontWeight.Normal)
                }
            }
        }
    }
}