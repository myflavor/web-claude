<template>
  <div>
    <!-- Boot -->
    <section v-if="view === 'boot'" class="view view-boot">
      <div class="boot-spinner" aria-hidden="true" />
    </section>

    <!-- Sessions -->
    <section v-else-if="view === 'sessions'" class="view">
      <header class="miui-nav">
        <div class="nav-side nav-side-left">
          <span class="nav-side-placeholder" aria-hidden="true" />
        </div>
        <div class="nav-title">
          <h1>会话</h1>
          <p class="nav-sub">{{ projectsRoot || "本地项目" }}</p>
        </div>
        <div class="nav-side nav-side-right">
          <span class="desktop-actions">
            <button class="miui-btn miui-btn-primary miui-btn-sm" type="button" @click="openNewSession">
              新建
            </button>
            <button class="miui-btn miui-btn-text" type="button" @click="logout">退出</button>
          </span>
          <span class="mobile-only">
            <button class="miui-btn miui-btn-text" type="button" @click="logout">退出</button>
          </span>
        </div>
      </header>

      <div class="page-body">
        <div v-if="sessions.length" class="card-stack">
          <article v-for="s in sessions" :key="s.id" class="session-card">
            <div class="session-card-top">
              <div class="session-avatar" :class="{ dead: !s.alive }">
                {{ initialOf(sessionTitle(s)) }}
              </div>
              <div class="session-info">
                <h2 class="session-title">{{ sessionTitle(s) }}</h2>
                <p class="session-path">{{ pathOf(s) }}</p>
                <div class="session-meta">
                  <span class="chip" :class="s.alive ? 'ok' : 'bad'">
                    {{ s.alive ? "运行中" : "已退出" }}
                  </span>
                  <span class="chip">{{ s.clients }} 个连接</span>
                  <span class="chip">{{ fmtTime(s.createdAt) }}</span>
                </div>
              </div>
            </div>
            <div class="session-actions">
              <button class="miui-btn miui-btn-primary" type="button" @click="openTerminal(s)">
                进入
              </button>
              <button class="miui-btn miui-btn-secondary" type="button" @click="killSession(s)">
                结束
              </button>
            </div>
          </article>
        </div>
        <div v-else class="empty-state">
          <div class="empty-icon" aria-hidden="true">⌘</div>
          <p class="empty-title">暂无会话</p>
          <p class="empty-desc">选择项目目录后创建，或恢复历史对话</p>
        </div>
      </div>

      <div class="fab-bar mobile-only">
        <button class="miui-btn miui-btn-primary miui-btn-block" type="button" @click="openNewSession">
          新建会话
        </button>
      </div>
    </section>

    <!-- Picker -->
    <section v-else-if="view === 'picker'" id="view-picker" class="view">
      <header class="miui-nav">
        <div class="nav-side nav-side-left">
          <button class="miui-btn miui-btn-text nav-back" type="button" @click="backToSessions">
            返回
          </button>
        </div>
        <div class="nav-title">
          <h1>新建会话</h1>
          <p class="nav-sub">{{ displayPath(projectsRoot, pickerPath) }}</p>
        </div>
        <div class="nav-side nav-side-right">
          <span class="desktop-actions">
            <button class="miui-btn miui-btn-primary miui-btn-sm" type="button" @click="createSessionHere()">
              新建
            </button>
          </span>
          <span class="mobile-only">
            <span class="nav-side-placeholder" aria-hidden="true" />
          </span>
        </div>
      </header>

      <div class="page-body picker-body">
        <div class="tool-row">
          <button class="miui-btn miui-btn-secondary miui-btn-sm" type="button" @click="doMkdir">
            新建文件夹
          </button>
          <button class="miui-btn miui-btn-secondary miui-btn-sm" type="button" @click="doClone">
            Git Clone
          </button>
          <button
            class="miui-btn miui-btn-secondary miui-btn-sm"
            type="button"
            @click="createSessionHere({ continueLast: true })"
          >
            继续最近
          </button>
        </div>

        <div class="miui-seg" role="tablist" aria-label="新建会话内容">
          <button
            class="miui-seg-item"
            :class="{ active: pickerTab === 'dirs' }"
            type="button"
            role="tab"
            :aria-selected="pickerTab === 'dirs'"
            @click="pickerTab = 'dirs'"
          >
            目录
          </button>
          <button
            class="miui-seg-item"
            :class="{ active: pickerTab === 'history' }"
            type="button"
            role="tab"
            :aria-selected="pickerTab === 'history'"
            @click="switchHistory"
          >
            历史
          </button>
        </div>

        <div v-show="pickerTab === 'dirs'" class="picker-panel" role="tabpanel">
          <div class="miui-card list-card">
            <div class="miui-list">
              <button
                v-if="pickerParent !== undefined && pickerPath"
                type="button"
                class="miui-cell"
                @click="goParent"
              >
                <span class="cell-icon up">↑</span>
                <span class="cell-body">
                  <div class="cell-title">上级目录</div>
                </span>
                <span class="cell-chevron">›</span>
              </button>
              <button
                v-for="e in pickerDirs"
                :key="e.path"
                type="button"
                class="miui-cell"
                @click="enterDir(e.path)"
              >
                <span class="cell-icon">📁</span>
                <span class="cell-body">
                  <div class="cell-title">{{ e.name }}</div>
                  <div class="cell-sub">{{ displayPath(projectsRoot, e.path) }}</div>
                </span>
                <span class="cell-chevron">›</span>
              </button>
              <div v-if="!pickerDirs.length && !pickerPath" class="list-empty">
                空目录，可新建文件夹或 Git Clone
              </div>
            </div>
          </div>
        </div>

        <div v-show="pickerTab === 'history'" class="picker-panel" role="tabpanel">
          <div v-if="conversations.length" class="card-stack">
            <article
              v-for="c in conversations"
              :key="c.sessionId"
              class="session-card history-card"
            >
              <div class="session-card-top">
                <div class="session-avatar history">
                  {{ initialOf(convTitle(c)) }}
                </div>
                <div class="session-info">
                  <h2 class="session-title">{{ convTitle(c) }}</h2>
                  <div class="session-meta">
                    <span class="chip">{{ fmtTime(c.updatedAt) }}</span>
                  </div>
                </div>
              </div>
              <div class="session-actions">
                <button class="miui-btn miui-btn-primary" type="button" @click="resumeConversation(c)">
                  恢复
                </button>
              </div>
            </article>
          </div>
          <div v-else class="empty-state compact">
            <p class="empty-desc">此目录暂无历史对话</p>
          </div>
        </div>

        <div class="picker-footer">
          <div class="opts-card miui-card">
            <div class="miui-switch-row">
              <div class="miui-switch-text">
                <div class="miui-switch-title">跳过权限确认</div>
              </div>
              <label class="miui-switch">
                <input v-model="skipPerms" type="checkbox" />
                <span class="miui-switch-track" aria-hidden="true">
                  <span class="miui-switch-thumb" />
                </span>
              </label>
            </div>
            <label class="opts-field">
              <input
                v-model="extraArgs"
                type="text"
                placeholder="额外参数"
                autocomplete="off"
              />
            </label>
          </div>
        </div>
      </div>

      <div class="fab-bar mobile-only">
        <button class="miui-btn miui-btn-primary miui-btn-block" type="button" @click="createSessionHere()">
          在此目录新建会话
        </button>
      </div>
    </section>

    <!-- Terminal -->
    <section v-else-if="view === 'term'" class="view term-view">
      <header class="miui-nav term-nav">
        <div class="nav-side nav-side-left">
          <button class="miui-btn miui-btn-text nav-back" type="button" @click="leaveTerminal">
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
  </div>
</template>

<script setup>
import { computed, inject, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { api } from "../api";
import {
  displayPath,
  folderTitle,
  fmtTime,
  initialOf,
  parseArgLine,
} from "../utils";

const router = useRouter();
const ui = inject("ui");

const view = ref("boot");
const projectsRoot = ref("");
const sessions = ref([]);
const pickerPath = ref("");
const pickerParent = ref("");
const pickerDirs = ref([]);
const pickerTab = ref("dirs");
const conversations = ref([]);
const skipPerms = ref(false);
const extraArgs = ref("");

const currentSession = ref(null);
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
  const s = currentSession.value;
  if (!s) return "会话";
  return s.title || folderTitle(projectsRoot.value, s.cwdRel, s.cwd);
});
const termCwd = computed(() => {
  const s = currentSession.value;
  if (!s) return "";
  return displayPath(projectsRoot.value, s.cwdRel || "") || s.cwd || "";
});

function sessionTitle(s) {
  return s.title || folderTitle(projectsRoot.value, s.cwdRel, s.cwd);
}
function pathOf(s) {
  return displayPath(projectsRoot.value, s.cwdRel || "") || s.cwd || "";
}
function convTitle(c) {
  return (c.display || "").trim() || "未命名对话";
}

function collectClaudeArgs() {
  const args = [];
  if (skipPerms.value) args.push("--dangerously-skip-permissions");
  if (extraArgs.value) args.push(...parseArgLine(extraArgs.value));
  return args;
}

async function boot() {
  try {
    const me = await api("/api/me");
    projectsRoot.value = me.projectsRoot || "";
    view.value = "sessions";
    await loadSessions();
  } catch {
    router.replace("/login");
  }
}

async function logout() {
  try {
    await api("/api/logout", { method: "POST", body: {} });
  } catch {
    /* ignore */
  }
  router.replace("/login");
}

async function loadSessions() {
  const data = await api("/api/sessions");
  sessions.value = (data.sessions || [])
    .slice()
    .sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
}

async function killSession(s) {
  const title = sessionTitle(s);
  const ok = await ui.uiConfirm(`确定结束会话「${title}」？`, "结束会话", {
    danger: true,
    okText: "结束",
  });
  if (!ok) return;
  await api(`/api/sessions/${s.id}`, { method: "DELETE" });
  await loadSessions();
}

function openNewSession() {
  pickerPath.value = "";
  pickerTab.value = "dirs";
  view.value = "picker";
  loadPicker().catch((e) => {
    if (e.status !== 401) ui.uiAlert(e.message);
  });
}

function backToSessions() {
  view.value = "sessions";
  loadSessions().catch((e) => {
    if (e.status !== 401) ui.uiAlert(e.message);
  });
}

async function loadPicker() {
  const data = await api(`/api/fs?path=${encodeURIComponent(pickerPath.value)}`);
  pickerPath.value = data.path || "";
  pickerParent.value = data.parent;
  pickerDirs.value = (data.entries || []).filter((e) => e.isDir);
  loadConversations().catch(() => {});
}

function goParent() {
  pickerPath.value = pickerParent.value || "";
  loadPicker().catch((e) => {
    if (e.status !== 401) ui.uiAlert(e.message);
  });
}

function enterDir(path) {
  pickerPath.value = path;
  loadPicker().catch((e) => {
    if (e.status !== 401) ui.uiAlert(e.message);
  });
}

function switchHistory() {
  pickerTab.value = "history";
  loadConversations().catch(() => {});
}

async function loadConversations() {
  const data = await api(
    `/api/conversations?path=${encodeURIComponent(pickerPath.value || "")}`
  );
  conversations.value = data.conversations || [];
}

async function createSessionHere(extra = {}) {
  try {
    const s = await api("/api/sessions", {
      method: "POST",
      body: {
        path: pickerPath.value,
        ...extra,
        claudeArgs: collectClaudeArgs(),
      },
    });
    await openTerminal(s);
  } catch (e) {
    if (e.status !== 401) ui.uiAlert(e.message || "创建失败");
  }
}

async function doMkdir() {
  const vals = await ui.uiPrompt(
    [{ name: "name", placeholder: "文件夹名称", required: true }],
    "新建文件夹"
  );
  if (!vals) return;
  const name = String(vals.name || "").trim();
  if (!name) return;
  try {
    await api("/api/fs/mkdir", {
      method: "POST",
      body: { path: pickerPath.value, name },
    });
    await loadPicker();
  } catch (e) {
    if (e.status !== 401) ui.uiAlert(e.message || "创建失败");
  }
}

async function doClone() {
  const vals = await ui.uiPrompt(
    [
      { name: "url", placeholder: "Git 仓库地址（https:// 或 git@）", required: true },
      { name: "name", placeholder: "文件夹名称（可留空）" },
    ],
    "Git Clone"
  );
  if (!vals) return;
  const url = String(vals.url || "").trim();
  if (!url) return;
  const name = String(vals.name || "").trim();
  try {
    const s = await api("/api/sessions", {
      method: "POST",
      body: {
        path: pickerPath.value,
        cloneUrl: url,
        cloneName: name,
        claudeArgs: collectClaudeArgs(),
      },
    });
    await openTerminal(s);
  } catch (e) {
    if (e.status !== 401) ui.uiAlert(e.message || "Clone 失败");
  }
}

async function resumeConversation(c) {
  try {
    const s = await api("/api/sessions", {
      method: "POST",
      body: {
        path: c.cwdRel || pickerPath.value || "",
        resumeId: c.sessionId,
        claudeArgs: collectClaudeArgs(),
      },
    });
    await openTerminal(s);
  } catch (e) {
    if (e.status !== 401) ui.uiAlert(e.message || "恢复失败");
  }
}

function ensureTerm() {
  if (term) return;
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
  });
  fitAddon = new FitAddon();
  term.loadAddon(fitAddon);
  term.loadAddon(new WebLinksAddon());
  term.open(termEl.value);
  term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(new TextEncoder().encode(data));
    }
  });
}

function sendResize() {
  if (!ws || ws.readyState !== WebSocket.OPEN || !term) return;
  ws.send(
    JSON.stringify({
      type: "resize",
      cols: term.cols,
      rows: term.rows,
    })
  );
}

function connectWS(sessionId) {
  if (ws) {
    try {
      ws.close();
    } catch {
      /* ignore */
    }
    ws = null;
  }
  const proto = location.protocol === "https:" ? "wss" : "ws";
  const url = `${proto}://${location.host}/api/sessions/${sessionId}/ws`;
  termStatus.value = "连接中…";
  ws = new WebSocket(url);
  ws.binaryType = "arraybuffer";

  ws.onopen = () => {
    termStatus.value = "";
    try {
      fitAddon.fit();
    } catch {
      /* ignore */
    }
    sendResize();
  };
  ws.onmessage = (ev) => {
    if (!term) return;
    if (ev.data instanceof ArrayBuffer) {
      term.write(new Uint8Array(ev.data));
    } else {
      term.write(ev.data);
    }
  };
  ws.onclose = () => {
    if (shouldReconnect && currentSession.value) {
      termStatus.value = "重新连接中…";
      clearTimeout(reconnectTimer);
      reconnectTimer = setTimeout(() => connectWS(currentSession.value.id), 1200);
    } else {
      termStatus.value = "";
    }
  };
  ws.onerror = () => {};
}

async function openTerminal(s) {
  currentSession.value = s;
  shouldReconnect = true;
  view.value = "term";
  await nextTick();
  ensureTerm();
  term.reset();
  try {
    fitAddon.fit();
  } catch {
    /* ignore */
  }
  connectWS(s.id);
  setTimeout(() => term?.focus(), 50);
}

function leaveTerminal() {
  shouldReconnect = false;
  clearTimeout(reconnectTimer);
  if (ws) {
    try {
      ws.close();
    } catch {
      /* ignore */
    }
    ws = null;
  }
  currentSession.value = null;
  view.value = "sessions";
  loadSessions().catch((e) => {
    if (e.status !== 401) ui.uiAlert(e.message);
  });
}

async function killCurrent() {
  if (!currentSession.value) return;
  const ok = await ui.uiConfirm("确定结束并删除该会话？", "结束会话", {
    danger: true,
    okText: "结束",
  });
  if (!ok) return;
  shouldReconnect = false;
  clearTimeout(reconnectTimer);
  if (ws) {
    try {
      ws.close();
    } catch {
      /* ignore */
    }
    ws = null;
  }
  try {
    await api(`/api/sessions/${currentSession.value.id}`, { method: "DELETE" });
  } catch (e) {
    if (e.status !== 401) ui.uiAlert(e.message);
    return;
  }
  currentSession.value = null;
  view.value = "sessions";
  await loadSessions().catch(() => {});
}

async function onUpload(e) {
  const file = e.target.files && e.target.files[0];
  e.target.value = "";
  if (!file || !currentSession.value) return;
  const fd = new FormData();
  fd.append("file", file);
  termStatus.value = `正在上传 ${file.name}…`;
  try {
    const res = await api(`/api/sessions/${currentSession.value.id}/upload`, {
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
  if (!currentSession.value || view.value !== "term") return;
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
        const res = await api(`/api/sessions/${currentSession.value.id}/upload`, {
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
  if (view.value !== "term" || !fitAddon) return;
  try {
    fitAddon.fit();
    sendResize();
  } catch {
    /* ignore */
  }
}

watch(view, async (v) => {
  if (v === "term") {
    await nextTick();
    onWinResize();
  }
});

onMounted(() => {
  boot();
  window.addEventListener("resize", onWinResize);
  document.addEventListener("paste", onPaste);
});

onUnmounted(() => {
  window.removeEventListener("resize", onWinResize);
  document.removeEventListener("paste", onPaste);
  shouldReconnect = false;
  clearTimeout(reconnectTimer);
  if (ws) {
    try {
      ws.close();
    } catch {
      /* ignore */
    }
  }
  if (term) {
    term.dispose();
    term = null;
  }
  if (resizeObserver) {
    resizeObserver.disconnect();
  }
});
</script>
