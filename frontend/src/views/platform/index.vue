<template>
    <div class="main" ref="dropzone" :style="{ '--sidebar-width': `${uiStore.sidebarDisplayWidth}px` }">
        <Menu></Menu>
        <div v-if="isRouterAlive" class="platform-route-outlet">
            <RouterView />
        </div>
        <div class="upload-mask" v-show="ismask">
            <UploadMask></UploadMask>
        </div>
        <!-- Tüm platform alt rotaları tarafından kullanılan genel ayarlar modalı-->
        <Settings />
        <!-- platform rotaları boyunca etkin kalan genel komut paneli (⌘K)-->
        <GlobalCommandPalette />
        <!-- Bilgi tabanı dosya yükleme ilerleme katmanı: yükleme kuyruğu store içinde tutulur, sayfa değiştirildiğinde kesilmez-->
        <UploadTasksPanel />
        <!-- Maskeli başlangıç rehberi: ilk girişte otomatik açılır, kullanıcı menüsünün üst kısmında takma adın yanındaki yardım düğmesinden yeniden açılabilir-->
        <NewUserGuide />
    </div>
</template>
<script setup lang="ts">
import Menu from '@/components/menu.vue'
import { ref, onMounted, onUnmounted, nextTick, provide, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router'
import UploadMask from '@/components/upload-mask.vue'
import Settings from '@/views/settings/Settings.vue'
import GlobalCommandPalette from '@/components/GlobalCommandPalette.vue'
import UploadTasksPanel from '@/components/upload-tasks/UploadTasksPanel.vue'
import NewUserGuide from '@/components/NewUserGuide.vue'
import { useCommandPaletteStore } from '@/stores/commandPalette'
import { useChatResourcesStore } from '@/stores/chatResources'
import { useUIStore } from '@/stores/ui'
import { getKnowledgeBaseById } from '@/api/knowledge-base/index'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { collectDroppedFiles } from './collectDroppedFiles'

const route = useRoute();
const router = useRouter();
const commandPaletteStore = useCommandPaletteStore();
const uiStore = useUIStore();
let ismask = ref(false)
const { t } = useI18n();

const isRouterAlive = ref(true)
const reloadApp = () => {
    isRouterAlive.value = false
    nextTick(() => {
        isRouterAlive.value = true
    })
}
provide('app:reload', reloadApp)

// Alt öğelerin dragleave tetiklemesi sorununu çözmek için sürükleme giriş/çıkış sayacını izlemek amacıyla kullanılır
let dragCounter = 0;

// Geçerli bilgi tabanı ID'sini al
const getCurrentKbId = (): string | null => {
    return (route.params as any)?.kbId as string || null
}

const CHAT_DROP_ROUTE_NAMES = new Set(['chat', 'globalCreatChat', 'kbCreatChat']);

const isChatDropRoute = () => {
    return CHAT_DROP_ROUTE_NAMES.has(String(route.name || ''));
}

// Bilgi tabanı başlatma durumunu kontrol et
const checkKnowledgeBaseInitialization = async (): Promise<boolean> => {
    const currentKbId = getCurrentKbId();
    
    if (!currentKbId) {
        MessagePlugin.error(t('knowledgeBase.missingId'));
        return false;
    }
    
    try {
        const kbResponse = await getKnowledgeBaseById(currentKbId);
        const kb = kbResponse.data;
        
        if (!kb.summary_model_id) {
            MessagePlugin.warning(t('knowledgeBase.notInitialized'));
            return false;
        }
        const strategy = kb.indexing_strategy;
        const needsEmbedding = !strategy || strategy.vector_enabled || strategy.keyword_enabled;
        if (needsEmbedding && !kb.embedding_model_id) {
            MessagePlugin.warning(t('knowledgeBase.notInitialized'));
            return false;
        }
        return true;
    } catch (error) {
        MessagePlugin.error(t('knowledgeBase.getInfoFailed'));
        return false;
    }
}


// isFileDrag distinguishes an OS file drag (the only thing the global upload
// drop zone cares about) from an in-app element drag such as the wiki
// folder/page drag-and-drop. Element drags carry only "text/*" types, never
// "Files", so we bail out and let the originating component handle the drop.
const isFileDrag = (event: DragEvent): boolean => {
    const types = event.dataTransfer?.types
    if (!types) return false
    return Array.from(types).includes('Files')
}

const shouldHandleGlobalFileDrag = (event: DragEvent): boolean => {
    if (!isFileDrag(event)) return false;
    // Keep the browser from opening dropped files, even outside upload pages.
    event.preventDefault();
    // Settings and its teleported skill drawers own their uploads. This runs
    // in document capture, before a local drop handler can stop propagation.
    const enabled = !uiStore.showSettingsModal && (
        isChatDropRoute() || (route.name === 'knowledgeBaseDetail' && !!getCurrentKbId())
    );
    if (!enabled) {
        dragCounter = 0;
        ismask.value = false;
    }
    return enabled;
}

// Genel sürükle-bırak olay işleme
const handleGlobalDragEnter = (event: DragEvent) => {
    if (!shouldHandleGlobalFileDrag(event)) return;
    dragCounter++;
    if (event.dataTransfer) {
        event.dataTransfer.effectAllowed = 'all';
    }
    ismask.value = true;
}

const handleGlobalDragOver = (event: DragEvent) => {
    if (!shouldHandleGlobalFileDrag(event)) return;
    if (event.dataTransfer) {
        event.dataTransfer.dropEffect = 'copy';
    }
}

const handleGlobalDragLeave = (event: DragEvent) => {
    if (!shouldHandleGlobalFileDrag(event)) return;
    dragCounter--;
    if (dragCounter === 0) {
        ismask.value = false;
    }
}

const handleGlobalDrop = async (event: DragEvent) => {
    if (!shouldHandleGlobalFileDrag(event)) return;
    dragCounter = 0;
    ismask.value = false;

    const droppedFiles = await collectDroppedFiles(event);
    if (droppedFiles.length === 0) {
        MessagePlugin.warning(t('knowledgeBase.dragFileNotText'));
        return;
    }

    if (isChatDropRoute()) {
        event.stopPropagation();
        window.dispatchEvent(new CustomEvent('rethra:chat-file-drop', {
            detail: { files: droppedFiles }
        }));
        return;
    }
    
    const isInitialized = await checkKnowledgeBaseInitialization();
    if (!isInitialized) {
        return;
    }

    window.dispatchEvent(new CustomEvent('rethra:knowledge-file-drop', {
        detail: { kbId: getCurrentKbId(), files: droppedFiles }
    }));
}

// Bileşen bağlandığında genel olay dinleyicilerini ekle
onMounted(() => {
    document.addEventListener('dragenter', handleGlobalDragEnter, true);
    document.addEventListener('dragover', handleGlobalDragOver, true);
    document.addEventListener('dragleave', handleGlobalDragLeave, true);
    document.addEventListener('drop', handleGlobalDrop, true);
    // Eski yollar gibi URL sorgu parametreleri aracılığıyla genel komut panelinin açılmasını destekler
    // /platform/knowledge-search?q=foo yönlendirmesinden sonra ?cmdk=foo korunur
    maybeOpenCmdkFromRoute()
    // Arka planda sohbet giriş çubuğu kaynaklarını önceden getir; creatChat / chat girildiğinde önbelleği yeniden kullan
    void useChatResourcesStore().prefetchChatInput()
});

// SPA içi geçişlerdeki ?cmdk= parametresiyle uyumlu olmak için rota değişikliklerini dinle
watch(() => route.query.cmdk, () => {
    maybeOpenCmdkFromRoute()
})

function maybeOpenCmdkFromRoute() {
    if (!('cmdk' in route.query)) return
    const q = String(route.query.cmdk ?? '')
    commandPaletteStore.openPalette(q)
    // Geri dönme/yenileme sırasında tekrar tekrar tetiklenmesini önlemek için query'yi temizle
    const newQuery = { ...route.query }
    delete (newQuery as any).cmdk
    router.replace({ path: route.path, query: newQuery, hash: route.hash })
}

// Bileşen kaldırıldığında genel olay dinleyicilerini kaldır
onUnmounted(() => {
    document.removeEventListener('dragenter', handleGlobalDragEnter, true);
    document.removeEventListener('dragover', handleGlobalDragOver, true);
    document.removeEventListener('dragleave', handleGlobalDragLeave, true);
    document.removeEventListener('drop', handleGlobalDrop, true);
    dragCounter = 0;
});
</script>
<style lang="less">
.main {
    display: flex;
    align-items: stretch;
    width: 100%;
    height: 100%;
    min-width: 600px;
    min-height: 0;
    /* Sol menü ile sağ içerik alanını görsel olarak bütünleştirmek için tam sayfa arka planını birleştir*/
    background: var(--td-bg-color-container);
}

/* Sağ rota alanı: kalan genişliği ve sütunun tüm yüksekliğini kaplar; iç flex kaydırma için min-height:0 değerini alt sayfalara iletir*/
.platform-route-outlet {
    flex: 1;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.upload-mask {
    background-color: var(--app-overlay);
    backdrop-filter: blur(2px);
    position: fixed;
    width: 100%;
    height: 100%;
    z-index: 999;
    display: flex;
    justify-content: center;
    align-items: center;
}

img {
    -webkit-user-drag: none;
    -khtml-user-drag: none;
    -moz-user-drag: none;
    -o-user-drag: none;
    user-drag: none;
}
</style>
