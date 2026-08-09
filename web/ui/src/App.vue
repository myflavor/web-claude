<template>
  <router-view />
  <MiuiDialog ref="dialogRef" />
</template>

<script setup>
import { provide, ref } from "vue";
import MiuiDialog from "./components/MiuiDialog.vue";

const dialogRef = ref(null);

function uiAlert(message, title) {
  return dialogRef.value?.alert(message, title);
}
function uiConfirm(message, title, opts) {
  return dialogRef.value?.confirm(message, title, opts);
}
function uiPrompt(fields, title) {
  return dialogRef.value?.prompt(fields, title);
}

provide("ui", { uiAlert, uiConfirm, uiPrompt });
</script>