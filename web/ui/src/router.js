import { createRouter, createWebHistory } from "vue-router";
import LoginView from "./views/LoginView.vue";
import MainView from "./views/MainView.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/login", name: "login", component: LoginView, meta: { public: true } },
    { path: "/:pathMatch(.*)*", name: "main", component: MainView },
  ],
});

export default router;
