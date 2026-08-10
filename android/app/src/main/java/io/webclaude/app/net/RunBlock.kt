package io.webclaude.app.net

import android.os.Handler
import android.os.Looper
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

/** Bridge suspend API to background threads; block the calling (worker) thread for a value. */
object RunBlock {
    fun <T> run(block: suspend () -> T): T {
        val ref = java.util.concurrent.atomic.AtomicReference<T?>(null)
        val errRef = java.util.concurrent.atomic.AtomicReference<Throwable?>(null)
        val latch = java.util.concurrent.CountDownLatch(1)
        CoroutineScope(Dispatchers.IO).launch {
            try {
                ref.set(block())
            } catch (e: Throwable) {
                errRef.set(e)
            }
            latch.countDown()
        }
        latch.await()
        errRef.get()?.let { throw it }
        return ref.get()!!
    }

    fun main(block: () -> Unit) {
        Handler(Looper.getMainLooper()).post(block)
    }
}