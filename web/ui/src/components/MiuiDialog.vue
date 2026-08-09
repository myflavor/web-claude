<template>
  <div
    v-if="open"
    class="dialog-root"
    aria-hidden="false"
  >
    <div class="dialog-mask" @click="onDismiss" />
    <div class="dialog-sheet" role="dialog" aria-modal="true" aria-labelledby="dialog-title">
      <h2 id="dialog-title" class="dialog-title">{{ title }}</h2>
      <p v-if="message" class="dialog-message">{{ message }}</p>
      <div v-if="fields.length" class="dialog-fields">
        <input
          v-for="(f, i) in fields"
          :key="f.name"
          v-model="values[f.name]"
          :type="f.type || 'text'"
          :name="f.name"
          :placeholder="f.placeholder || ''"
          :autocomplete="f.autocomplete || 'off'"
          :ref="(el) => setInputRef(el, i)"
          @keydown.enter.prevent="onOk"
        />
      </div>
      <div class="dialog-actions">
        <button
          v-if="showCancel"
          class="miui-btn miui-btn-secondary"
          type="button"
          @click="onCancel"
        >
          {{ cancelText }}
        </button>
        <button
          class="miui-btn miui-btn-primary"
          :class="{ danger: danger }"
          type="button"
          @click="onOk"
        >
          {{ okText }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { nextTick, onMounted, onUnmounted, reactive, ref } from "vue";

const open = ref(false);
const title = ref("");
const message = ref("");
const fields = ref([]);
const values = reactive({});
const okText = ref("确定");
const cancelText = ref("取消");
const showCancel = ref(true);
const danger = ref(false);
const inputEls = ref([]);
let resolveFn = null;

function setInputRef(el, i) {
  if (el) inputEls.value[i] = el;
}

function reset() {
  open.value = false;
  title.value = "";
  message.value = "";
  fields.value = [];
  Object.keys(values).forEach((k) => delete values[k]);
  okText.value = "确定";
  cancelText.value = "取消";
  showCancel.value = true;
  danger.value = false;
  inputEls.value = [];
}

function finish(result) {
  const r = resolveFn;
  resolveFn = null;
  reset();
  if (r) r(result);
}

function onOk() {
  if (fields.value.length) {
    const vals = {};
    for (const f of fields.value) {
      vals[f.name] = values[f.name] ?? "";
      if (f.required && !String(vals[f.name] || "").trim()) {
        inputEls.value[0]?.focus?.();
        return;
      }
    }
    finish(vals);
    return;
  }
  finish(true);
}

function onCancel() {
  finish(fields.value.length ? null : null);
}

function onDismiss() {
  finish(null);
}

function openDialog(opts) {
  return new Promise((resolve) => {
    resolveFn = resolve;
    title.value = opts.title || "";
    message.value = opts.message || "";
    fields.value = opts.fields || [];
    Object.keys(values).forEach((k) => delete values[k]);
    for (const f of fields.value) {
      values[f.name] = f.value || "";
    }
    okText.value = opts.okText || "确定";
    cancelText.value = opts.cancelText === false ? "" : opts.cancelText || "取消";
    showCancel.value = opts.cancelText !== false;
    danger.value = !!opts.danger;
    open.value = true;
    nextTick(() => {
      const first = inputEls.value[0];
      if (first) first.focus();
    });
  });
}

function alert(msg, t) {
  return openDialog({
    title: t || "提示",
    message: msg,
    okText: "知道了",
    cancelText: false,
  }).then(() => {});
}

function confirm(msg, t, opts = {}) {
  return openDialog({
    title: t || "确认",
    message: msg,
    okText: opts.okText || "确定",
    cancelText: opts.cancelText || "取消",
    danger: !!opts.danger,
  }).then((v) => !!v);
}

function prompt(fieldList, t) {
  const list = (fieldList || []).map((f, i) => ({
    name: f.name || `f${i}`,
    placeholder: f.placeholder || "",
    value: f.value || "",
    type: f.type || "text",
    required: !!f.required,
  }));
  return openDialog({
    title: t || "输入",
    fields: list,
    okText: "确定",
    cancelText: "取消",
  });
}

function onKey(e) {
  if (e.key === "Escape" && open.value) onDismiss();
}

onMounted(() => document.addEventListener("keydown", onKey));
onUnmounted(() => document.removeEventListener("keydown", onKey));

defineExpose({ alert, confirm, prompt, openDialog });
</script>
