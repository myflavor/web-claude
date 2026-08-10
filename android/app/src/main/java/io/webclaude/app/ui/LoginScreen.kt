package io.webclaude.app.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.NavHostController
import io.webclaude.app.net.SessionClient
import io.webclaude.app.ui.session.SessionLogin

@Composable
fun LoginScreen(nav: NavHostController, client: SessionClient) {
    var url by remember { mutableStateOf("http://192.168.1.100:3080") }
    var token by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    Column(
        Modifier.fillMaxSize().imePadding().verticalScroll(rememberScrollState()).padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        Text("Web Claude", style = MaterialTheme.typography.headlineLarge, color = MaterialTheme.colorScheme.primary)
        Text("服务地址", style = MaterialTheme.typography.labelMedium)
        OutlinedTextField(value = url, onValueChange = { url = it }, label = { Text("http://NAS:3080") }, singleLine = true)
        OutlinedTextField(value = token, onValueChange = { token = it }, label = { Text("密码") }, singleLine = true,
            visualTransformation = PasswordVisualTransformation())
        if (error != null) Text(error!!, color = MaterialTheme.colorScheme.error)
        Button(onClick = {
            loading = true; error = null
            SessionLogin.login(nav, client, url, token) { msg, isErr ->
                loading = false
                if (isErr) error = msg
            }
        }, enabled = !loading) {
            if (loading) CircularProgressIndicator(Modifier.size(20.dp), color = Color.White)
            else Text("登录")
        }
    }
}