<template>
  <section class="view">
    <div v-if="!booted" class="view view-boot">
      <div class="boot-spinner" aria-hidden="true" />
    </div>

    <template v-else>
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
            <button class="miui-btn miui-btn-primary miui-btn-sm" type="button" @click="goNew">
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
              <button class="miui-btn miui-btn-secondary" type="button" @click="kill(s)">
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
        <button class="miui-btn miui-btn-primary miui-btn-block" type="button" @click="goNew">
          新建会话
        </button>
      </div>
    </template>
  </section>
</template>

<script setup>
import { computed, inject, onMounted } from "vue";
import { storeToRefs } from "pinia";
import { useRouter } from "vue-router";
import { useMainStore } from "../store";
import { folderTitle, displayPath, fmtTime, initialOf } from "../utils";
import { api } from "../api";

const router = useRouter();
const ui = inject("ui");
const store = useMainStore();

const { booted, projectsRoot, sessions } = storeToRefs(store);

const sessionTitle = (s) =>
  s.title || folderTitle(projectsRoot.value, s.cwdRel, s.cwd);
const pathOf = (s) =>
  displayPath(projectsRoot.value, s.cwdRel || "") || s.cwd || "";

function goNew() {
  store.pickerPath = "";
  store.pickerTab = "dirs";
  router.push("/new");
}

async function kill(s) {
  const ok = await ui.uiConfirm(
    `确定结束会话「${sessionTitle(s)}」？`,
    "结束会话",
    { danger: true, okText: "结束" }
  );
  if (!ok) return;
  try {
    await store.killSession(s.id);
  } catch (e) {
    if (e.status !== 401) ui.uiAlert(e.message || "结束失败");
  }
}

function openTerminal(s) {
  store.currentSession = s;
  router.push(`/session/${s.id}`);
}

async function logout() {
  try {
    await api("/api/logout", { method: "POST", body: {} });
  } catch {
    /* ignore */
  }
  router.replace("/login");
}

onMounted(async () => {
  await store.loadSessions();
});
</script>