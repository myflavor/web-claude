import { createRouter, createWebHistory } from "vue-router";
import LoginView from "./views/LoginView.vue";
import SessionsView from "./views/SessionsView.vue";
import PickerView from "./views/PickerView.vue";
import TerminalView from "./views/TerminalView.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/login", name: "login", component: LoginView, meta: { public: true } },
    { path: "/", name: "sessions", component: SessionsView },
    { path: "/new", name: "picker", component: PickerView },
    { path: "/session/:id", name: "terminal", component: TerminalView },
    { path: "/:pathMatch(.*)*", redirect: "/" },
  ],
});

export default router;