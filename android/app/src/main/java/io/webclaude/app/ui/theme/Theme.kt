package io.webclaude.app.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

// MIUIX / HyperOS-inspired palette (matches the Web UI).
val BgPage = Color(0xFF0A0A0A)
val BgSurface = Color(0xFF1C1C1E)
val BgSurface2 = Color(0xFF2C2C2E)
val BgSurface3 = Color(0xFF3A3A3C)
val Separator = Color(0x8A545458)
val PrimaryBlue = Color(0xFF0A84FF)
val PrimaryPress = Color(0xFF409CFF)
val TextPrimary = Color(0xFFFFFFFF)
val TextSecondary = Color(0x99EBEBF5) // ~60%
val TextTertiary = Color(0x66EBEBF5) // ~40%
val DangerRed = Color(0xFFFF453A)

private val MiuiDark = darkColorScheme(
    primary = PrimaryBlue,
    onPrimary = Color.White,
    background = BgPage,
    onBackground = TextPrimary,
    surface = BgSurface,
    onSurface = TextPrimary,
    surfaceVariant = BgSurface2,
    onSurfaceVariant = TextSecondary,
    secondary = TextSecondary,
    onSecondary = TextPrimary,
    error = DangerRed,
    onError = Color.White,
)

@Composable
fun AppTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = MiuiDark,
        content = content,
    )
}