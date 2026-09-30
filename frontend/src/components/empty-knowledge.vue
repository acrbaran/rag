<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
</script>
<template>
    <div class="empty">
        <span class="empty-icon" aria-hidden="true">
            <t-icon name="file" />
        </span>
        <span class="empty-txt">{{ t('knowledgeBase.emptyKnowledgeDragDrop') }}</span>
        <span class="empty-type-txt">{{ t('knowledgeBase.pdfDocFormat') }}</span>
        <span class="empty-type-txt">{{ t('knowledgeBase.textMarkdownFormat') }}</span>
    </div>
</template>
<style scoped lang="less">
.empty {
    flex: 1;
    display: flex;
    flex-flow: column;
    justify-content: center;
    align-items: center;
    gap: 8px;
    padding: 40px 24px;
    text-align: center;
}

.empty-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: var(--app-radius-control);
    background: color-mix(in srgb, var(--td-text-color-primary) 5%, transparent);
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-xl);
}

.empty-txt {
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: var(--app-text-base);
    font-weight: 500;
    line-height: 20px;
}

.empty-type-txt {
    max-width: 320px;
    color: var(--td-text-color-secondary);
    font-family: var(--app-font-family);
    font-size: var(--app-text-sm);
    font-weight: 400;
    line-height: 18px;
}
</style>
