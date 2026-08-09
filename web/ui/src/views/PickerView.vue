<template>
  <section id="view-picker" class="view">
    <header class="miui-nav">
      <div class="nav-side nav-side-left">
        <button class="miui-btn miui-btn-text nav-back" type="button" @click="back">
          返回
        </button>
      </div>
      <div class="nav-title">
        <h1>新建会话</h1>
        <p class="nav-sub">{{ displayPath(projectsRoot, pickerPath) }}</p>
      </div>
      <div class="nav-side nav-side-right">
        <span class="desktop-actions">
          <button class="miui-btn miui-btn-primary miui-btn-sm" type="button" @click="createNow()">
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
          @click="createNow({ continueLast: true })"
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
          @click="pickerTab = 'dirs'; loadConversations().catch(()=>{})"
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
              <button class="miui-btn miui-btn-primary" type="button" @click="resume(c)">
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
      <button class="miui-btn miui-btn-primary miui-btn-block" type="button" @click="createNow()">
        在此目录新建会话
      </button>
    </div>
  </section>
</template>

<script setup>
import { computed, inject, onMounted } from "vue";
import { storeToRefs } from "pinia";
import { useRouter } from "vue-router";
import { useMainStore } from "../store";
import { displayPath, fmtTime, initialOf, parseArgLine } from "../utils";
import { api } from "../api";

const router = useRouter();
const ui = inject("ui");
const store = useMainStore();

const {
  projectsRoot,
  pickerPath,
  pickerParent,
  pickerDirs,
  pickerTab,
  conversations,
  skipPerms,
  extraArgs,
} = storeToRefs(store);

function collectClaudeArgs() {
  const args = [];
  if (skipPerms.value) args.push("--dangerously-skip-permissions");
  if (extraArgs.value) args.push(...parseArgLine(extraArgs.value));
  return args;
}

function back() {
  router.push("/");
}

async function createNow(extra = {}) {
  try {
    const s = await store.createSession({ ...extra, claudeArgs: collectClaudeArgs() });
    router.replace(`/session/${s.id}`);
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
    await store.loadPicker();
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
    const s = await store.createSession({
      cloneUrl: url,
      cloneName: name,
      claudeArgs: collectClaudeArgs(),
    });
    router.replace(`/session/${s.id}`);
  } catch (e) {
    if (e.status !== 401) ui.uiAlert(e.message || "Clone 失败");
  }
}

async function resume(c) {
  try {
    const s = await store.createSession({
      path: c.cwdRel || pickerPath.value || "",
      resumeId: c.sessionId,
      claudeArgs: collectClaudeArgs(),
    });
    router.replace(`/session/${s.id}`);
  } catch (e) {
    if (e.status !== 401) ui.uiAlert(e.message || "恢复失败");
  }
}

function switchHistory() {
  pickerTab.value = "history";
  store.loadConversations().catch(() => {});
}

function convTitle(c) {
  return (c.display || "").trim() || "未命名对话";
}

onMounted(() => {
  store.loadPicker().catch(() => {});
});
</script>