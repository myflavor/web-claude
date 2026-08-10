import { defineStore } from "pinia";
import { ref } from "vue";
import { api } from "./api";

export const useMainStore = defineStore("main", () => {
  // Auth / bootstrap state
  const booted = ref(false);
  const projectsRoot = ref("");

  // Sessions list
  const sessions = ref([]);

  // Picker navigation (dirs/history) + new-session options
  const pickerPath = ref("");
  const pickerParent = ref("");
  const pickerDirs = ref([]);
  const pickerTab = ref("dirs");
  const conversations = ref([]);
  const skipPerms = ref(false);
  const extraArgs = ref("");

  // Current terminal session (also surviving a picker detour)
  const currentSession = ref(null);

  async function boot() {
    try {
      const me = await api("/api/me");
      projectsRoot.value = me.projectsRoot || "";
      booted.value = true;
      return true;
    } catch {
      return false;
    }
  }

  async function loadSessions() {
    try {
      const data = await api("/api/sessions");
      sessions.value = []
        .concat(data.sessions || [])
        .sort((a, b) => new Date(b.createdAt) - new Date(a.createdAt));
    } catch (e) {
      if (e.status !== 401) throw e;
    }
    return sessions.value;
  }

  async function killSession(id) {
    await api(`/api/sessions/${id}`, { method: "DELETE" });
    await loadSessions();
  }

  async function loadPicker() {
    const data = await api(`/api/fs?path=${encodeURIComponent(pickerPath.value)}`);
    pickerPath.value = data.path || "";
    pickerParent.value = data.parent;
    pickerDirs.value = (data.entries || []).filter((e) => e.isDir);
    // History is fetched lazily on the History tab; don't scan it on dir nav.
  }

  function goParent() {
    pickerPath.value = pickerParent.value || "";
    loadPicker().catch(() => {});
  }

  function enterDir(path) {
    pickerPath.value = path;
    loadPicker().catch(() => {});
  }

  async function loadConversations() {
    const data = await api(
      `/api/conversations?path=${encodeURIComponent(pickerPath.value || "")}`
    );
    conversations.value = data.conversations || [];
  }

  async function createSession(extra = {}) {
    const res = await api("/api/sessions", {
      method: "POST",
      body: {
        path: pickerPath.value,
        ...extra,
        claudeArgs: extra.claudeArgs || [],
      },
    });
    currentSession.value = res;
    return res;
  }

  return {
    booted,
    projectsRoot,
    sessions,
    pickerPath,
    pickerParent,
    pickerDirs,
    pickerTab,
    conversations,
    skipPerms,
    extraArgs,
    currentSession,
    boot,
    loadSessions,
    killSession,
    loadPicker,
    goParent,
    enterDir,
    loadConversations,
    createSession,
  };
});