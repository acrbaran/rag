/// <reference types="vite/client" />
// Bu dosyayı yapılandırmak, "@/views/login/index.vue" modülünün veya ilgili tür bildirimlerinin bulunamaması hatasını çözer. ts(2307)
// Bu kod, TypeScript'e `.vue` ile biten tüm dosyaların `import` ifadesiyle içe aktarılabilen Vue bileşenleri olduğunu bildirir. Bu genellikle modülün tanınamaması sorununu çözer.
declare module '*.vue' {
    import { Component } from 'vue'; const component: Component; export default component;
}

declare const __FRONTEND_VERSION__: string;
declare const __FRONTEND_COMMIT__: string;