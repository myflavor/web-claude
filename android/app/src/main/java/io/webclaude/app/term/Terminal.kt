package io.webclaude.app.term

import java.util.ArrayDeque

/**
 * Minimal ANSI/VT terminal emulator: parses raw bytes into a character grid
 * with per-cell colors. Handles the escape sequences Claude/bash emit:
 * CSI CUP/ED/EL/SGR, OSC, DCS, LF/CR/TAB/BS/BEL. Scrollback as deque of rows.
 */
class Terminal {
    var cols = 80
        private set
    var rows = 24
        private set

    data class Cell(
        var ch: Char = ' ',
        var fg: Int = 0xFFE7ECF3.toInt(),
        var bg: Int = 0xFF0B0F14.toInt(),
        var bold: Boolean = false,
        var dim: Boolean = false,
        var invert: Boolean = false,
        var underline: Boolean = false,
    )

    private val screen = ArrayDeque<Array<Cell>>()
    private val maxScrollback = 5000
    private var scroll = 0

    var cursorX = 0
        private set
    var cursorY = 0
        private set

    private var fg = 0xFFE7ECF3.toInt()
    private var bg = 0xFF0B0F14.toInt()
    private var bold = false
    private var dim = false
    private var invert = false
    private var underline = false

    init { resetScreen() }

    private fun resetScreen() {
        screen.clear(); scroll = 0
        for (y in 0 until rows) screen.add(blankRow())
    }

    private fun blankRow(): Array<Cell> = Array(cols) { Cell() }

    val lines: Int get() = screen.size - scroll

    fun visibleRow(i: Int): Array<Cell> = screen.elementAt(scroll + i)

    // Count rows available above current viewport for scrollback.
    val scrollableLines: Int get() = (screen.size - rows).coerceAtLeast(0)

    // ---- input ----
    fun feed(bytes: ByteArray) {
        var i = 0
        val n = bytes.size
        val sb = StringBuilder()
        while (i < n) {
            val b = bytes[i++].toInt() and 0xFF
            when {
                b == 0x1b -> { flush(sb); if (i < n) { val r = parseEscape(bytes, i); i = maxOf(i, r.next) } }
                b == 0x0A -> { flush(sb); lineFeed() }
                b == 0x0D -> { flush(sb) }
                b == 0x09 -> { flush(sb); tab() }
                b == 0x08 -> { flush(sb); backspace() }
                b == 0x07 -> {}
                b in 0x00..0x06 || b in 0x0E..0x1F || b == 0x7F -> {}
                else -> sb.append(b.toChar())
            }
        }
        flush(sb)
        prune()
    }

    private fun flush(sb: StringBuilder) {
        if (sb.isEmpty()) return
        for (c in sb) putChar(c)
    }

    private fun putChar(c: Char) {
        if (cursorX >= cols) {
            cursorX = 0; cursorY++; if (cursorY >= rows) { scrollUp(); cursorY = rows - 1 }
        }
        val row = screen.elementAt(scroll + cursorY)
        row[cursorX] = Cell(c, fg, bg, bold, dim, invert, underline)
        cursorX++
    }

    private fun lineFeed() {
        cursorY++; if (cursorY >= rows) { scrollUp(); cursorY = rows - 1 }
    }

    private fun tab() { cursorX = ((cursorX / 8) + 1) * 8 }

    private fun backspace() { if (cursorX > 0) cursorX-- }

    private fun scrollUp() {
        screen.removeFirst()
        screen.addLast(blankRow())
    }

    private fun prune() {
        while (scroll > maxScrollback) {
            screen.removeFirst()
            scroll--
        }
    }

    private data class EscapeResult(val next: Int)

    private fun parseEscape(bytes: ByteArray, from: Int): EscapeResult {
        var i = from
        val first = bytes[i].toInt()
        if (first == '['.toInt()) {
            i++
            val params = StringBuilder()
            var cmd = 0
            while (i < bytes.size) {
                val c = bytes[i].toInt()
                i++
                if (c in 0x30..0x3F) { params.append(c.toChar()) }
                else { cmd = c; break }
            }
            execCsi(params.toString(), cmd)
            return EscapeResult(i)
        }
        if (first == ']'.toInt()) {
            // OSC — skip until BEL or ST
            var j = i + 1
            while (j < bytes.size) {
                val c = bytes[j].toInt()
                if (c == 0x07) { j++; break }
                if (c == 0x1b && j + 1 < bytes.size && bytes[j + 1].toInt() == '\\'.toInt()) { j += 2; break }
                j++
            }
            return EscapeResult(j)
        }
        if (first == 'P'.toInt() || first == '_'.toInt() || first == '^'.toInt()) {
            var j = i + 1
            while (j < bytes.size) {
                val c = bytes[j].toInt()
                if (c == 0x07) { j++; break }
                if (c == 0x1b && j + 1 < bytes.size && bytes[j + 1].toInt() == '\\'.toInt()) { j += 2; break }
                j++
            }
            return EscapeResult(j)
        }
        // single-char escape
        i++
        return EscapeResult(i)
    }

    private fun execCsi(params: String, cmd: Int) {
        fun read(s: String, idx: Int, def: Int): Int =
            if (s.isEmpty()) def
            else s.split(';').getOrNull(idx)?.takeIf { it.isNotEmpty() && it.all { ch -> ch in '0'..'9' } }?.toInt() ?: def

        when (cmd) {
            'A'.toInt() -> { val n = read(params, 0, 1); cursorY = maxOf(0, cursorY - n) }
            'B'.toInt() -> { val n = read(params, 0, 1); cursorY = minOf(rows - 1, cursorY + n) }
            'C'.toInt() -> { val n = read(params, 0, 1); cursorX = minOf(cols - 1, cursorX + n) }
            'D'.toInt() -> { val n = read(params, 0, 1); cursorX = maxOf(0, cursorX - n) }
            'G'.toInt() -> { val c = read(params, 0, 1); cursorX = (c - 1).coerceIn(0, cols - 1) }
            'H'.toInt(), 'f'.toInt() -> {
                val a = read(params, 0, 1); val b = read(params, 1, 1)
                cursorY = (a - 1).coerceIn(0, rows - 1); cursorX = (b - 1).coerceIn(0, cols - 1)
            }
            'J'.toInt() -> clearScreen(read(params, 0, 0))
            'K'.toInt() -> clearLine(read(params, 0, 0))
            'm'.toInt() -> sgr(params)
            'd'.toInt() -> { val a = read(params, 0, 1); cursorY = (a - 1).coerceIn(0, rows - 1) }
            'h'.toInt(), 'l'.toInt(), 'r'.toInt(), 's'.toInt(), 'u'.toInt() -> {}
        }
    }

    private fun clearScreen(mode: Int) {
        when (mode) {
            0 -> {
                clearRow(cursorY, cursorX, cols - 1)
                for (y in cursorY + 1 until rows) clearRow(y, 0, cols - 1)
            }
            2, 3 -> {
                cursorX = 0; cursorY = 0
                for (y in 0 until rows) clearRow(y, 0, cols - 1)
            }
        }
    }

    private fun clearLine(mode: Int) {
        when (mode) {
            0 -> clearRow(cursorY, cursorX, cols - 1)
            1 -> clearRow(cursorY, 0, cursorX)
            2 -> clearRow(cursorY, 0, cols - 1)
        }
    }

    private fun clearRow(y: Int, from: Int, to: Int) {
        val row = screen.elementAt(scroll + y)
        for (x in from..to) row[x] = Cell(' ', fg, bg, bold, dim, invert, underline)
    }

    private fun sgr(params: String) {
        val codes = params.split(';').filter { it.isNotEmpty() }
        if (codes.isEmpty()) { resetAttrs(); return }
        var i = 0
        while (i < codes.size) {
            val code = codes[i].toIntOrNull() ?: 0
            when {
                code == 0 -> resetAttrs()
                code == 1 -> { bold = true }
                code == 2 -> { dim = true }
                code == 4 -> { underline = true }
                code == 7 -> { invert = true }
                code == 22 -> { bold = false; dim = false }
                code == 24 -> { underline = false }
                code == 27 -> { invert = false }
                code in 30..37 -> { fg = ansiColor(code - 30, false) }
                code == 38 -> {
                    if (i + 1 < codes.size && codes[i + 1] == "5" && i + 2 < codes.size) {
                        fg = palette256(codes[i + 2].toIntOrNull() ?: 0); i += 2
                    } else if (i + 1 < codes.size && codes[i + 1] == "2" && i + 4 < codes.size) {
                        val r = codes[i + 2].toIntOrNull() ?: 0
                        val g = codes[i + 3].toIntOrNull() ?: 0
                        val b = codes[i + 4].toIntOrNull() ?: 0
                        fg = 0xFF000000.toInt() or (r shl 16) or (g shl 8) or b
                        i += 4
                    }
                }
                code == 39 -> { fg = 0xFFE7ECF3.toInt() }
                code in 40..47 -> { bg = ansiColor(code - 40, true) }
                code == 48 -> {
                    if (i + 1 < codes.size && codes[i + 1] == "5" && i + 2 < codes.size) {
                        bg = palette256(codes[i + 2].toIntOrNull() ?: 0); i += 2
                    } else if (i + 1 < codes.size && codes[i + 1] == "2" && i + 4 < codes.size) {
                        val r = codes[i + 2].toIntOrNull() ?: 0
                        val g = codes[i + 3].toIntOrNull() ?: 0
                        val b = codes[i + 4].toIntOrNull() ?: 0
                        bg = 0xFF000000.toInt() or (r shl 16) or (g shl 8) or b
                        i += 4
                    }
                }
                code == 49 -> { bg = 0xFF0B0F14.toInt() }
                code in 90..97 -> { fg = ansiColor(code - 90, true) }
                code in 100..107 -> { bg = ansiColor(code - 100, true) }
            }
            i++
        }
    }

    private fun resetAttrs() {
        fg = 0xFFE7ECF3.toInt(); bg = 0xFF0B0F14.toInt()
        bold = false; dim = false; invert = false; underline = false
    }

    private fun ansiColor(base: Int, bright: Boolean): Int {
        val list = arrayOf(
            0x000000, 0xcc0000, 0x4e9a06, 0xc4a000, 0x3465a4, 0x75507b, 0x06989a, 0xd3d7cf,
            0x555753, 0xef2929, 0x8ae234, 0xfce94f, 0x729fcf, 0xad7fa8, 0x34e2e2, 0xeeeeec,
        )
        val idx = if (bright && base < 8) base + 8 else base
        return 0xFF000000.toInt() or list[idx.coerceIn(0, 15)]
    }

    private fun palette256(n: Int): Int {
        if (n < 16) return ansiColor(n % 8, n >= 8)
        if (n < 231) {
            val v = n - 16
            val r = v / 36; val g = (v % 36) / 6; val b = v % 6
            fun cv(x: Int) = if (x == 0) 0 else 55 + x * 40
            return 0xFF000000.toInt() or (cv(r) shl 16) or (cv(g) shl 8) or cv(b)
        }
        val v = 8 + (n - 231) * 10
        return 0xFF000000.toInt() or (v shl 16) or (v shl 8) or v
    }

    fun resize(newCols: Int, newRows: Int) {
        var c = if (newCols <= 0) 80 else newCols
        var r = if (newRows <= 0) 24 else newRows
        if (c == cols && r == rows) return
        val oldCols = cols; val oldRows = rows
        cols = c; rows = r
        val kept = screen.toList()
        screen.clear(); scroll = 0
        for (y in 0 until rows) {
            val src = kept.getOrNull(kept.size - oldRows + y) ?: blankRow()
            val dst = Array(cols) { x -> src.getOrNull(x) ?: Cell() }
            screen.add(dst)
        }
        cursorX = cursorX.coerceIn(0, cols - 1); cursorY = cursorY.coerceIn(0, rows - 1)
    }
}