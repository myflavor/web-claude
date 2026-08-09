<template>
  <section class="view term-view">
    <header class="miui-nav term-nav">
      <div class="nav-side nav-side-left">
        <button class="miui-btn miui-btn-text nav-back" type="button" @click="leave">
          返回
        </button>
      </div>
      <div class="nav-title">
        <h1>{{ termTitle }}</h1>
        <p class="nav-sub">{{ termCwd }}</p>
      </div>
      <div class="nav-side nav-side-right">
        <label class="miui-btn miui-btn-text file-btn">
          上传
          <input ref="fileInput" type="file" hidden @change="onUpload" />
        </label>
        <button class="miui-btn miui-btn-text danger-text" type="button" @click="killCurrent">
          结束
        </button>
      </div>
    </header>
    <div ref="termEl" class="terminal" />
    <p class="term-status" :class="{ empty: !termStatus }">{{ termStatus }}</p>
  </section>
</template>

<script setup>
import { computed, inject, nextTick, onMounted, onUnmounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { useMainStore } from "../store";
import { displayPath, folderTitle } from "../utils";
import { api } from "../api";

const route = useRoute();
const router = useRouter();
const ui = inject("ui");
const store = useMainStore();

const sessionId = computed(() => route.params.id);
const currentSession = computed(
  () => store.currentSession && store.currentSession.id === sessionId.value
    ? store.currentSession
    : { id: sessionId.value }
);

const termEl = ref(null);
const termStatus = ref("");
const fileInput = ref(null);

let term = null;
let fitAddon = null;
let ws = null;
let reconnectTimer = null;
let shouldReconnect = false;
let resizeObserver = null;

const termTitle = computed(() => {
  const s = store.currentSession;
  if (!s) return "会话";
  return s.title || folderTitle(store.projectsRoot, s.cwdRel, s.cwd);
});
const termCwd = computed(() => {
  const s = store.currentSession;
  if (!s) return "";
  return displayPath(store.projectsRoot, s.cwdRel || "") || s.cwd || "";
});

function disposeTerm() {
  if (resizeObserver) {
    try {
      resizeObserver.disconnect();
    } catch {
      /* ignore */
    }
    resizeObserver = null;
  }
  if (term) {
    try {
      term.dispose();
    } catch {
      /* ignore */
    }
    term = null;
  }
  fitAddon = null;
}

function ensureTerm() {
  if (!termEl.value) return false;
  // xterm can only open once per instance; recreate if missing or detached.
  if (term) {
    const host = term.element?.parentElement;
    if (host === termEl.value) return true;
    disposeTerm();
  }
  term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace',
    theme: {
      background: "#0b0f14",
      foreground: "#e7ecf3",
      cursor: "#0a84ff",
      selectionBackground: "rgba(10,132,255,0.35)",
    },
    scrollback: 5000,
    allowProposedApi: true,
    smoothScrollDuration: 0,
    scrollOnUserInput: true,
  });
  fitAddon = new FitAddon();
  term.loadAddon(fitAddon);
  term.loadAddon(new WebLinksAddon());
  termEl.value.innerHTML = "";
  term.open(termEl.value);
  // Mobile: native-feeling vertical scroll over text.
  try {
    const vp = term.element?.querySelector(".xterm-viewport");
    if (vp) {
      vp.style.touchAction = "pan-y";
      vp.style.webkitOverflowScrolling = "touch";
      vp.style.overscrollBehavior = "contain";
    }
    const screen = term.element?.querySelector(".xterm-screen");
    if (screen) {
      screen.style.touchAction = "pan-y";
    }
  } catch {
    /* ignore */
  }

  // xterm's own touchstart/touchmove handlers preventDefault and hand-roll
  // scrollTop with no momentum, hijacking native scrolling. Intercept in the
  // capture phase so browser scrolls .xterm-viewport natively (inertial).
  try {
    const host = termEl.value;
    const blocker = (e) => {
      e.stopImmediatePropagation();
    };
    host.addEventListener("touchstart", blocker, { capture: true, passive: true });
    host.addEventListener("touchmove", blocker, { capture: true, passive: true });
    host.addEventListener("touchcancel", blocker, { capture: true, passive: true });
  } catch {
    /* ignore */
  }

  // Browser key path ≠ native TTY. Map common shortcuts browsers swallow.
  term.attachCustomKeyEventHandler((ev) => {
    if (ev.type !== "keydown") return true;
    const ctrl = ev.ctrlKey || ev.metaKey;
    if (!ctrl) return true;
    if (ev.key === "Enter") {
      ev.preventDefault();
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(new TextEncoder().encode("\n"));
      }
      return false;
    }
    if (ev.key === "v" || ev.key === "c" || ev.key === "x" || ev.key === "a") {
      return true;
    }
    return true;
  });

  term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(new TextEncoder().encode(data));
    }
  });
  if (typeof ResizeObserver !== "undefined") {
    resizeObserver = new ResizeObserver(() => {
      fitAndResize();
    });
    resizeObserver.observe(termEl.value);
  }
  return true;
}

function fitAndResize() {
  if (!term || !fitAddon || !termEl.value) return;
  const { clientWidth, clientHeight } = termEl.value;
  if (clientWidth < 20 || clientHeight < 20) return;
  try {
    fitAddon.fit();
  } catch {
    /* ignore */
  }
  sendResize();
}

function sendResize() {
  if (!ws || ws.readyState !== WebSocket.OPEN || !term) return;
  if (!term.cols || !term.rows) return;
  ws.send(
    JSON.stringify({
      type: "resize",
      cols: term.cols,
      rows: term.rows,
    })
  );
}

function connectWS(sid) {
  if (ws) {
    try {
      ws.onclose = null;
      ws.close();
    } catch {
      /* ignore */
    }
    ws = null;
  }
  const proto = location.protocol === "https:" ? "wss" : "ws";
  const url = `${proto}://${location.host}/api/sessions/${sid}/ws`;
  termStatus.value = "连接中…";
  const socket = new WebSocket(url);
  ws = socket;
  socket.binaryType = "arraybuffer";

  socket.onopen = () => {
    if (ws !== socket) return;
    termStatus.value = "";
    fitAndResize();
    requestAnimationFrame(() => fitAndResize());
  };
  socket.onmessage = (ev) => {
    if (!term || ws !== socket) return;
    if (ev.data instanceof ArrayBuffer) {
      term.write(new Uint8Array(ev.data));
    } else {
      term.write(ev.data);
    }
  };
  socket.onclose = () => {
    if (ws === socket) ws = null;
    if (shouldReconnect && store.currentSession) {
      termStatus.value = "重新连接中…";
      clearTimeout(reconnectTimer);
      reconnectTimer = setTimeout(() => connectWS(store.currentSession.id), 1200);
    } else if (!termStatus.value) {
      termStatus.value = "";
    }
  };
  socket.onerror = () => {
    if (ws === socket) {
      termStatus.value = "连接失败";
    }
  };
}

async function open() {
  const sid = sessionId.value;
  if (!sid) {
    ui.uiAlert("会话无效");
    router.replace("/");
    return;
  }
  shouldReconnect = true;
  termStatus.value = "连接中…";
  await nextTick();
  let ok = ensureTerm();
  if (!ok) {
    await nextTick();
    ok = ensureTerm();
  }
  if (!ok || !term) {
    termStatus.value = "终端初始化失败";
    ui.uiAlert("终端初始化失败，请返回重试");
    return;
  }
  term.reset();
  await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)));
  fitAndResize();
  connectWS(sid);
  setTimeout(() => {
    fitAndResize();
    term?.focus();
  }, 50);
}

function leave() {
  shouldReconnect = false;
  clearTimeout(reconnectTimer);
  unloadWs();
  disposeTerm();
  router.push("/");
}

function killCurrent() {
  const sid = sessionId.value;
  if (!sid) return;
  ui
    .uiConfirm("确定结束并删除该会话？", "结束会话", { danger: true, okText: "结束" })
    .then(async (ok) => {
      if (!ok) return;
      shouldReconnect = false;
      clearTimeout(reconnectTimer);
      unloadWs();
      try {
        await store.killSession(sid);
      } catch (e) {
        if (e.status !== 401) ui.uiAlert(e.message || "结束失败");
        return;
      }
      disposeTerm();
      router.replace("/");
    });
}

function unloadWs() {
  if (ws) {
    try {
      ws.onclose = null;
      ws.close();
    } catch {
      /* ignore */
    }
    ws = null;
  }
}

async function onUpload(e) {
  const file = e.target.files && e.target.files[0];
  e.target.value = "";
  if (!file || !sessionId.value) return;
  const fd = new FormData();
  fd.append("file", file);
  termStatus.value = `正在上传 ${file.name}…`;
  try {
    const res = await api(`/api/sessions/${sessionId.value}/upload`, {
      method: "POST",
      body: fd,
    });
    termStatus.value = `已上传：${res.path}`;
  } catch (err) {
    if (err.status === 401) return;
    termStatus.value = `上传失败：${err.message}`;
    ui.uiAlert(err.message || "上传失败");
  }
}

async function onPaste(e) {
  if (!sessionId.value) return;
  const items = e.clipboardData && e.clipboardData.items;
  if (!items) return;
  for (const item of items) {
    if (item.type && item.type.startsWith("image/")) {
      e.preventDefault();
      const file = item.getAsFile();
      if (!file) return;
      const fd = new FormData();
      fd.append("file", file, `paste-${Date.now()}.png`);
      termStatus.value = "正在上传图片…";
      try {
        const res = await api(`/api/sessions/${sessionId.value}/upload`, {
          method: "POST",
          body: fd,
        });
        termStatus.value = `图片已上传：${res.path}`;
      } catch (err) {
        if (err.status === 401) return;
        termStatus.value = `图片上传失败：${err.message}`;
      }
      return;
    }
  }
}

function onWinResize() {
  fitAndResize();
}

onMounted(() => {
  open();
  window.addEventListener("resize", onWinResize);
  document.addEventListener("paste", onPaste);
});

onUnmounted(() => {
  window.removeEventListener("resize", onWinResize);
  document.removeEventListener("paste", onPaste);
  shouldReconnect = false;
  clearTimeout(reconnectTimer);
  unloadWs();
  disposeTerm();
});
</script>