import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import router from "./router";
import TDesign from "tdesign-vue-next";
// Bileşen kütüphanesinden az sayıda genel stil değişkeni eklenir
import "tdesign-vue-next/dist/tdesign.css";
import "@/assets/theme/theme.css";
import "@/assets/theme/tdesign-overrides.less";
import "@/assets/theme/rethra.less";
import "@/assets/dropdown-menu.less";
import "@/components/css/chat-hljs-dark.less";
import "@/components/css/chat-rethra.less";
import "@/assets/theme/tables.css";
import "@/components/announcements/announcementUi.less";
// vue-virtual-scroller ships its own tiny stylesheet — required for
// RecycleScroller/DynamicScroller to size their viewport correctly.
// Without it the scroller computes 0 height and renders no items.
import "vue-virtual-scroller/dist/vue-virtual-scroller.css";
import i18n from "./i18n";
import { initTheme } from "@/composables/useTheme";
import { initFont } from "@/composables/useFont";
import { installTDesignIconOfflineGuard } from "@/utils/tdesign-icon-offline";
import { installAutofillGuard } from "@/utils/disable-autofill";
import { useAuthStore } from "@/stores/auth";

// Vue bileşeni bağlanmadan önce çalıştırılmalıdır; böylece tdesign-icons çalışma zamanında tdesign.gtimg.com isteği yapmaz
installTDesignIconOfflineGuard();

initTheme();
initFont();

async function bootstrap() {
  const app = createApp(App);

  // Genel hata işleme: işlenmemiş bileşen hatalarını yakalayarak beyaz ekranı önler
  app.config.errorHandler = (err, instance, info) => {
    console.error("[Rethra] Unhandled Vue error:", err, "\nComponent:", instance, "\nInfo:", info);
  };

  app.use(TDesign);
  const pinia = createPinia();
  app.use(pinia);

  // Capabilities (can_create_tenant, auto_accept_invitation) are not cached
  // in localStorage — reconcile once before first paint when a session exists.
  const authStore = useAuthStore();
  if (localStorage.getItem("rethra_token")) {
    try {
      await authStore.refreshFromAuthMe();
    } catch {
      // best-effort; capabilities stay at defaults until the next refresh
    }
  }

  app.use(router);
  app.use(i18n);

  // İlk ekran rotası (gezinme korumaları ve Lite otomatik girişi dahil) tamamlandıktan sonra bağlanır; böylece önce varsayılan sayfanın görünüp sonra yönlendirilmesi önlenir
  await router.isReady();
  app.mount("#app");
  installAutofillGuard();
}

bootstrap();
