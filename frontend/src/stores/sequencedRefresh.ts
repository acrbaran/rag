/**
 * "Yenileme" anlamındaki istek sıralayıcısı (/auth/me gibi tekil okumalar için kullanılır).
 *
 * - `refresh()`: Sonuç, bu çağrıdan sonra gönderilen bir istekten gelmelidir. Zaten uçuş halinde bir istek varsa,
 *   onun arkasına yeni bir istek sıralanır; birden fazla eşzamanlı refresh aynı kuyruktaki isteği paylaşır.
 * - `share()`: Uçuş halinde bir istek varsa doğrudan onu yeniden kullanır, aksi hâlde yeni bir istek gönderir. İlk ekranda birden çok tüketici
 *   aynı veriye aynı anda ihtiyaç duyduğunda kullanılır.
 * - `invalidate()`: Uçuş halindeki isteği geçersiz kılar. `run` tarafından alınan `isCurrent()`, await sonrasında
 *   false döner; çağıran taraf buna göre veritabanına yazmayı atlar. Oturum kapatılırken kullanılır; geç gelen yanıtın oturumu geri yazmasını önler.
 */
export interface SequencedRefresh<T> {
  refresh: () => Promise<T>
  share: () => Promise<T>
  invalidate: () => void
  hasInFlightRequest: () => boolean
}

export function createSequencedRefresh<T>(
  run: (isCurrent: () => boolean) => Promise<T>,
): SequencedRefresh<T> {
  let revision = 0
  let inflight: Promise<T> | null = null
  let queued: Promise<T> | null = null

  const start = (): Promise<T> => {
    const requestRevision = revision
    const current = run(() => requestRevision === revision).finally(() => {
      if (inflight === current) inflight = null
    })
    inflight = current
    return current
  }

  const refresh = (): Promise<T> => {
    if (!inflight) return start()
    if (!queued) {
      const waitingFor = inflight
      queued = waitingFor.then(
        () => {
          queued = null
          return start()
        },
        () => {
          queued = null
          return start()
        },
      )
    }
    return queued
  }

  return {
    refresh,
    share: () => inflight ?? start(),
    invalidate: () => {
      revision += 1
    },
    hasInFlightRequest: () => inflight !== null,
  }
}
