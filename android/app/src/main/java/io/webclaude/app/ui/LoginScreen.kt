package io.webclaude.app.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.Button
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.navigation.NavHostController
import io.webclaude.app.net.SessionClient
import io.webclaude.app.ui.session.SessionLogin
import io.webclaude.app.ui.theme.BgSurface2
import io.webclaude.app.ui.theme.PrimaryBlue
import io.webclaude.app.ui.theme.TextPrimary
import io.webclaude.app.ui.theme.TextSecondary

@Composable
fun LoginScreen(nav: NavHostController, client: SessionClient) {
    var url by remember { mutableStateOf("http://192.168.1.100:3080") }
    var token by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    Column(
        Modifier.fillMaxSize().imePadding().verticalScroll(rememberScrollState()).padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(14.dp)
    ) {
        // Hero
        Text(
            "Web Claude",
            fontSize = 32.sp,
            fontWeight = FontWeight.Bold,
            color = TextPrimary,
            modifier = Modifier.padding(top = 48.dp),
        )
        Text("随时随地使用 Claude Code", fontSize = 15.sp, color = TextSecondary)

        // Card
        androidx.compose.foundation.layout.Box(Modifier.fillMaxWidth().padding(top = 24.dp)) {
            androidx.compose.material3.Surface(
                shape = androidx.compose.foundation.shape.RoundedCornerShape(16.dp),
                color = MaterialTheme.colorScheme.surface,
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Text("服务地址", fontSize = 13.sp, color = TextSecondary, modifier = Modifier.padding(start = 4.dp))
                    OutlinedTextField(
                        value = url,
                        onValueChange = { url = it },
                        singleLine = true,
                        textStyle = androidx.compose.ui.text.TextStyle(color = TextPrimary, fontSize = 16.sp),
                        colors = OutlinedTextFieldDefaults.colors(
                            focusedBorderColor = PrimaryBlue,
                            unfocusedBorderColor = BgSurface2,
                            focusedContainerColor = BgSurface2,
                            unfocusedContainerColor = BgSurface2,
                        ),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Text("密码", fontSize = 13.sp, color = TextSecondary, modifier = Modifier.padding(start = 4.dp))
                    OutlinedTextField(
                        value = token,
                        onValueChange = { token = it },
                        singleLine = true,
                        visualTransformation = PasswordVisualTransformation(),
                        textStyle = androidx.compose.ui.text.TextStyle(color = TextPrimary, fontSize = 16.sp),
                        colors = OutlinedTextFieldDefaults.colors(
                            focusedBorderColor = PrimaryBlue,
                            unfocusedBorderColor = BgSurface2,
                            focusedContainerColor = BgSurface2,
                            unfocusedContainerColor = BgSurface2,
                        ),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    if (error != null) {
                        Text(error!!, color = MaterialTheme.colorScheme.error, fontSize = 13.sp, textAlign = androidx.compose.ui.text.style.TextAlign.Center, modifier = Modifier.fillMaxWidth())
                    }
                    Button(
                        onClick = {
                            loading = true; error = null
                            SessionLogin.login(nav, client, url, token) { msg, isErr ->
                                loading = false
                                if (isErr) error = msg
                            }
                        },
                        enabled = !loading,
                        colors = ButtonDefaults.buttonColors(containerColor = PrimaryBlue),
                        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                        shape = androidx.compose.foundation.shape.RoundedCornerShape(12.dp),
                    ) {
                        Text(if (loading) "登录中…" else "登录", fontSize = 17.sp, fontWeight = FontWeight.Medium, color = androidx.compose.ui.graphics.Color.White, modifier = Modifier.padding(vertical = 6.dp))
                    }
                }
            }
        }
    }
}