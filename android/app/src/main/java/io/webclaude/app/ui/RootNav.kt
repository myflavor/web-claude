package io.webclaude.app.ui

import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.setValue
import androidx.navigation.NavHostController
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import io.webclaude.app.net.SessionClient
import io.webclaude.app.net.WsClient
import io.webclaude.app.ui.session.NewScreen
import io.webclaude.app.ui.session.SessionsScreen
import io.webclaude.app.ui.session.TerminalScreen

object Routes {
    const val LOGIN = "login"
    const val SESS = "sessions"
    const val NEW = "new"
    const val TERM = "term/{id}"
    fun term(id: String) = "term/$id"
}

@Composable
fun RootNav() {
    val nav = rememberNavController()
    val client = remember { SessionClient() }
    val wsHolder = remember { WsClient() }
    val sessionId = remember { mutableStateOf<String?>(null) }

    NavHost(nav, startDestination = Routes.LOGIN) {
        composable(Routes.LOGIN) {
            LoginScreen(nav = nav, client = client)
        }
        composable(Routes.SESS) {
            SessionsScreen(nav = nav, client = client)
        }
        composable(Routes.NEW) {
            NewScreen(nav = nav, client = client)
        }
        composable(
            Routes.TERM,
            arguments = listOf(navArgument("id") { type = NavType.StringType })
        ) {
            TerminalScreen(
                nav = nav,
                id = it.arguments?.getString("id") ?: "",
                sessionId = sessionId,
                wsHolder = wsHolder,
                client = client,
            )
        }
    }
}