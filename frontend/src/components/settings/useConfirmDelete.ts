import { DialogPlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'

interface ConfirmDeleteOptions {
  title?: string
  body: string
  confirmText?: string
  cancelText?: string
  onConfirm: () => Promise<void> | void
}

/**
 * `TDesign DialogPlugin.confirm` tabanli, birlestirilmis silme onay etkilesimi.
 * Dagınık `window.confirm` / `t-popconfirm` / ozel `Dialog` kullanimlarinin yerini alir.
 * Yikici islemler tutarli olarak `danger` temasi ve kirmizi onay dugmesi kullanir; yalnizca satir ici hafif islemlerde `t-popconfirm` kullanilir.
 */
export function useConfirmDelete() {
  const { t } = useI18n()

  return (opts: ConfirmDeleteOptions) => {
    const dialog = DialogPlugin.confirm({
      dialogClassName: 'confirm-delete-dialog',
      header: opts.title || (t('common.confirmDelete') as string),
      body: opts.body,
      confirmBtn: { content: opts.confirmText || (t('common.delete') as string), theme: 'danger' },
      cancelBtn: opts.cancelText || (t('common.cancel') as string),
      theme: 'danger',
      onConfirm: async () => {
        try {
          await opts.onConfirm()
        } finally {
          dialog.hide()
        }
      }
    })
    return dialog
  }
}
