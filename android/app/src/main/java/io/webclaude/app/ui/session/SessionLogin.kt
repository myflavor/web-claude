package io.webclaude.app.ui.session

import android.os.Handler
import android.os.Looper
import io.webclaude.app.net.ApiResult
import io.webclaude.app.net.SessionClient
import io.webclaude.app.ui.Routes
import androidx.navigation.NavHostController

object SessionLogin {
    fun login(
        nav: NavHostController,
        client: SessionClient,
        url: String,
        token: String,
        done: (String, Boolean) -> Unit,
    ) {
        val main = Handler(Looper.getMainLooper())
        Thread {
            client.setBase(url)
            val r = client.login(token)
            if (r is ApiResult.Ok) {
                client.me()
                main.post {
                    done("", false)
                    nav.navigate(Routes.SESS) { popUpTo(Routes.LOGIN) }
                }
            } else {
                val msg = (r as? ApiResult.Err)?.message ?: "登录失败"
                main.post { done(msg, true) }
            }
        }.start()
    }
}
