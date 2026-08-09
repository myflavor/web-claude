<template>
  <section class="view view-login">
    <div class="login-wrap">
      <div class="login-hero">
        <div class="app-icon" aria-hidden="true">W</div>
        <h1>Web Claude</h1>
        <p class="subtitle">随时随地使用 Claude Code</p>
      </div>
      <div class="miui-card login-card">
        <label class="miui-field">
          <input
            v-model="token"
            type="password"
            autocomplete="current-password"
            placeholder="请输入密码"
            @keydown.enter="login"
          />
        </label>
        <button
          class="miui-btn miui-btn-primary miui-btn-block"
          type="button"
          :disabled="loading"
          @click="login"
        >
          登录
        </button>
        <p v-if="error" class="form-error">{{ error }}</p>
      </div>
    </div>
  </section>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { api } from "../api";

const router = useRouter();
const token = ref("");
const error = ref("");
const loading = ref(false);

onMounted(async () => {
  try {
    await api("/api/me", { skipAuthRedirect: true });
    router.replace("/");
  } catch {
    // stay on login
  }
});

async function login() {
  error.value = "";
  loading.value = true;
  try {
    await api("/api/login", {
      method: "POST",
      body: { token: token.value.trim() },
      skipAuthRedirect: true,
    });
    router.replace("/");
  } catch (e) {
    error.value = e.message || "登录失败，请检查密码";
  } finally {
    loading.value = false;
  }
}
</script>
