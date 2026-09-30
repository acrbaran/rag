<script setup lang="ts">
import { computed } from 'vue';

interface KnowledgeItem {
  id: string;
  file_name?: string;
  title?: string;
  type?: string;
  parse_status?: string;
}

const props = defineProps<{
  item: KnowledgeItem;
  canDownload: boolean;
  canMutateKnowledge: boolean;
  traceVisible: boolean;
  /** Whether the knowledge base has a folder structure to file documents into. */
  foldersAvailable?: boolean;
}>();

const emit = defineEmits<{
  (e: 'download'): void;
  (e: 'edit'): void;
  (e: 'view-trace'): void;
  (e: 'reparse'): void;
  (e: 'cancel-parse'): void;
  (e: 'move'): void;
  (e: 'move-folder'): void;
  (e: 'batch-manage'): void;
  (e: 'delete'): void;
}>();


const CANCELABLE_PARSE_STATUSES = new Set(['pending', 'processing', 'finalizing']);

const isParseInFlight = computed(() =>
  CANCELABLE_PARSE_STATUSES.has(String(props.item.parse_status ?? ''))
);

const fileName = computed(() => props.item.file_name || props.item.title || props.item.id);
</script>

<template>
  <!-- Orijinal belgeyi indir -->
  <div
    v-if="canDownload && (item.type === 'file' || item.type === 'manual')"
    class="card-menu-item"
    @click.stop="emit('download')"
  >
    <t-icon class="icon" name="download" />
    <span>{{ $t('common.download') }}</span>
  </div>

  <!-- Belgeyi düzenle -->
  <div v-if="item.type === 'manual'" class="card-menu-item" @click.stop="emit('edit')">
    <t-icon class="icon" name="edit" />
    <span>{{ $t('knowledgeBase.editDocument') }}</span>
  </div>

  <!-- İşleme sürecini görüntüle -->
  <div v-if="traceVisible" class="card-menu-item" @click.stop="emit('view-trace')">
    <t-icon class="icon" name="chart-bar" />
    <span>{{ $t('knowledgeStages.viewTrace') }}</span>
  </div>

  <!-- Bilgiyi yeniden oluştur (in-flight: popconfirm yok, yalnızca emits) -->
  <div v-if="isParseInFlight" class="card-menu-item" @click.stop="emit('reparse')">
    <t-icon class="icon" name="refresh" />
    <span>{{ $t('knowledgeBase.rebuildDocument') }}</span>
  </div>

  <!-- Bilgiyi yeniden oluştur (normal: popconfirm ile) -->
  <t-popconfirm v-else theme="warning"
    :content="$t('knowledgeBase.rebuildConfirm', { fileName })"
    :confirm-btn="{ content: $t('common.confirm'), theme: 'primary' }"
    :cancel-btn="{ content: $t('common.cancel') }" placement="left"
    @confirm="emit('reparse')">
    <div class="card-menu-item" @click.stop>
      <t-icon class="icon" name="refresh" />
      <span>{{ $t('knowledgeBase.rebuildDocument') }}</span>
    </div>
  </t-popconfirm>

  <!-- Ayrıştırmayı iptal et -->
  <t-popconfirm v-if="isParseInFlight" theme="warning"
    :content="$t('knowledgeBase.cancelParseConfirmBody', { title: fileName })"
    :confirm-btn="{ content: $t('knowledgeBase.cancelParse'), theme: 'danger' }"
    :cancel-btn="{ content: $t('common.cancel') }" placement="left"
    @confirm="emit('cancel-parse')">
    <div class="card-menu-item danger" @click.stop>
      <t-icon class="icon" name="close-circle" />
      <span>{{ $t('knowledgeBase.cancelParse') }}</span>
    </div>
  </t-popconfirm>

  <!-- Dizine taşı -->
  <div v-if="canMutateKnowledge" class="card-menu-item" @click.stop="emit('move-folder')">
    <t-icon class="icon" name="folder" />
    <span>{{ $t('knowledgeBase.moveToFolder.action') }}</span>
  </div>

  <!-- Başka bilgi tabanına taşı -->
  <div v-if="canMutateKnowledge" class="card-menu-item" @click.stop="emit('move')">
    <t-icon class="icon" name="swap" />
    <span>{{ $t('knowledgeBase.moveDocument') }}</span>
  </div>

  <!-- Toplu yönetim -->
  <div v-if="canMutateKnowledge || canDownload" class="card-menu-item" @click.stop="emit('batch-manage')">
    <t-icon class="icon" name="queue" />
    <span>{{ $t('menu.batchManage') }}</span>
  </div>

  <!-- Belgeyi sil -->
  <t-popconfirm theme="warning"
    :content="$t('knowledgeBase.confirmDeleteDocument', { fileName })"
    :confirm-btn="{ content: $t('knowledgeBase.confirmDelete'), theme: 'danger' }"
    :cancel-btn="{ content: $t('common.cancel') }" placement="left"
    @confirm="emit('delete')">
    <div class="card-menu-item danger" @click.stop>
      <t-icon class="icon" name="delete" />
      <span>{{ $t('knowledgeBase.deleteDocument') }}</span>
    </div>
  </t-popconfirm>
</template>
