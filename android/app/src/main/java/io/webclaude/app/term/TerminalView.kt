package io.webclaude.app.term

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.gestures.detectDragGestures
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlin.math.roundToInt

/** State holder between the WS client and the Terminal emulator. */
class TerminalSession {
    val term = Terminal()
    private val _grid = MutableStateFlow(listOf<List<Terminal.Cell>>())
    val grid = _grid.asStateFlow()

    fun feed(bytes: ByteArray) {
        term.feed(bytes)
        val rows = term.rows
        val cols = term.cols
        _grid.value = (0 until rows).map { y ->
            (0 until cols).map { x -> term.visibleRow(y)[x] }
        }
    }

    fun feedText(s: String) = feed(s.toByteArray())
}

@Composable
fun TerminalView(session: TerminalSession, onResize: (cols: Int, rows: Int) -> Unit) {
    var cols by remember { mutableIntStateOf(session.term.cols) }
    var rows by remember { mutableIntStateOf(session.term.rows) }
    var scrollOffset by remember { mutableIntStateOf(0) } // rows scrolled up; 0 = bottom

    BoxWithConstraints(Modifier.fillMaxSize()) {
        val density = LocalDensity.current.density
        val maxWpx = maxWidth.value * density
        val maxHpx = maxHeight.value * density
        val cellWpx = 9f * density
        val cellHpx = 16f * density

        LaunchedEffect(maxWpx, maxHpx) {
            val c = (maxWpx / cellWpx).toInt().coerceIn(20, 300)
            val r = (maxHpx / cellHpx).toInt().coerceIn(10, 200)
            if (c != cols || r != rows) {
                cols = c; rows = r
                session.term.resize(c, r)
                onResize(c, r)
            }
        }

        val term = session.term
        val visibleCols = cols
        val visibleRows = rows
        val maxScroll = (term.lines - rows).coerceAtLeast(0)

        Canvas(
            Modifier
                .fillMaxSize()
                .pointerInput(visibleCols, visibleRows, maxScroll) {
                    var lastY = 0f
                    detectDragGestures(
                        onDragStart = { lastY = it.y },
                        onDrag = { change, _ ->
                            change.consume()
                            val dy = change.position.y - lastY
                            lastY = change.position.y
                            var off = scrollOffset + (dy / cellHpx).roundToInt()
                            off = off.coerceIn(0, maxScroll)
                            scrollOffset = off
                        },
                        onDragEnd = {},
                        onDragCancel = {}
                    )
                }
        ) {
            drawTerminal(term, visibleCols, visibleRows, scrollOffset, cellWpx, cellHpx)
        }
    }
}

private fun DrawScope.drawTerminal(
    term: Terminal,
    cols: Int,
    rows: Int,
    scrollOffset: Int,
    cellW: Float,
    cellH: Float,
) {
    drawRect(Color(0xFF0B0F14.toInt()))
    val pad = 1f
    for (y in 0 until rows) {
        val row = term.visibleRow(y + scrollOffset)
        for (x in 0 until cols) {
            val cell = row[x]
            val left = x * cellW + pad
            val top = y * cellH
            if (cell.bg != 0xFF0B0F14.toInt()) {
                drawRect(Color(cell.bg), topLeft = Offset(left, top), size = Size(cellW, cellH))
            }
            if (cell.ch != ' ') {
                drawRect(
                    Color(cell.fg),
                    topLeft = Offset(left + 1f, top + cellH / 4f),
                    size = Size(cellW - 1f, cellH / 2f)
                )
            }
        }
    }
}