<script setup lang="ts">
import { watch } from "vue"

const props = defineProps(['reviewImg', 'reviewUrl'])
const emit = defineEmits(['closePreImg'])
const close = () => {
    emit('closePreImg')
}
</script>
<template>
    <!-- Mount only while open: without a trigger slot TImageViewer renders an
         inline thumbnail trigger, which shows an empty broken-image box (with
         Her mesajın yanında TDesign'ın varsayılan zh_CN "Görüntü gösterilemiyor/önizlenemiyor" metni. -->
    <t-image-viewer v-if="reviewImg" :visible="reviewImg" closeOnOverlay closeOnEscKeydown @close="close"
        :images="reviewUrl ? [reviewUrl] : []">
    </t-image-viewer>
</template>
<style scoped lang="less"></style>
