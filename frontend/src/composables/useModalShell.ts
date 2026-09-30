import { onBeforeUnmount, onMounted } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'

/**
 * Tam ekran "ayar benzeri" acilir pencereler (`Settings` / `AgentEditor` / `OrganizationSettings` / `KnowledgeBaseEditor`).
 * Ortak kabuk katmanı etkileşimleri:
 *   - Esc ile kapatma (üstte TDesign modal / çekmece / katman varsa yanıt vermez; önce onların kapanmasına izin verir)
 *   - Maske tıklamasıyla kapatma
 *   - Kaydedilmemiş değişiklikler varsa önce ikinci bir onay göster
 *
 * Kullanım:
 *   const shell = useModalShell({
 *     visible: () => props.visible,
 *     close: () => emit('update:visible', false),
 *     snapshot: () => formData.value,   // İsteğe bağlı: dirty karşılaştırmasına katılan veri
 *   })
 *   // Veri yükleme tamamlandıktan / kaydetme başarılı olduktan sonra: shell.markClean()
 *   // Şablon: @click.self="shell.requestClose", kapatma düğmesi @click="shell.requestClose"
 */
export interface ModalShellOptions {
  visible: () => boolean
  close: () => void
  /** Dirty karşılaştırmasına katılan veri; verilmezse kaydedilmemiş değişiklik uyarısı gösterilmez*/
  snapshot?: () => unknown
  /** true döndüğünde Esc atlanır (örneğin bileşen içindeki özel çizilmiş katman açıksa)*/
  ignoreEscape?: () => boolean
}

function serialize(value: unknown): string {
  try {
    return JSON.stringify(value ?? null)
  } catch {
    return ''
  }
}

/** Sayfada açık durumda TDesign modal / çekmece / katman olup olmadığı*/
function hasOpenTDesignOverlay(): boolean {
  if (typeof document === 'undefined') return false
  const dialogCtx = document.querySelector<HTMLElement>('.t-dialog__ctx')
  if (dialogCtx && dialogCtx.style.display !== 'none') return true
  if (document.querySelector('.t-drawer--open')) return true
  const popups = document.querySelectorAll<HTMLElement>('.t-popup')
  for (const popup of popups) {
    if (popup.style.display !== 'none') return true
  }
  return false
}

export function useModalShell(options: ModalShellOptions) {
  const { t } = useI18n()
  let cleanSnapshot = serialize(options.snapshot?.())
  let confirming = false

  const markClean = () => {
    cleanSnapshot = serialize(options.snapshot?.())
  }

  const isDirty = () => {
    if (!options.snapshot) return false
    return serialize(options.snapshot()) !== cleanSnapshot
  }

  // Template handlers pass the DOM event as the first argument, so only call
  // afterClose when it really is a callback.
  const requestClose = (afterClose?: unknown) => {
    if (!options.visible() || confirming) return
    const close = () => {
      options.close()
      if (typeof afterClose === 'function') afterClose()
    }
    if (!isDirty()) {
      close()
      return
    }
    confirming = true
    const dialog = DialogPlugin.confirm({
      header: t('common.unsavedChanges.title'),
      body: t('common.unsavedChanges.body'),
      theme: 'warning',
      confirmBtn: { content: t('common.unsavedChanges.discard'), theme: 'danger' },
      cancelBtn: t('common.unsavedChanges.keepEditing'),
      onConfirm: () => {
        confirming = false
        dialog.destroy()
        close()
      },
      onClose: () => {
        confirming = false
        dialog.destroy()
      },
    })
  }

  const onKeydown = (event: KeyboardEvent) => {
    if (event.key !== 'Escape' || !options.visible()) return
    if (options.ignoreEscape?.()) return
    if (hasOpenTDesignOverlay()) return
    requestClose()
  }

  onMounted(() => window.addEventListener('keydown', onKeydown))
  onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

  return { requestClose, markClean, isDirty }
}
