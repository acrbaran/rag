import { createVersionedRequestCoordinator } from './versionedRequest'

/**
 * Alan düzeyindeki kaynaklar için okuma ilkesi. TTL yoktur:
 *
 * - `ensure()`: Aynı kaynak için aynı anda yalnızca bir istek devam eder; eşzamanlı çağrılar onu paylaşır. Devam eden yoksa
 *   yeni bir istek gönderilir. Dönen Promise, en güncel veri store'a yazıldıktan sonra resolve olur; bu nedenle
 *   `await ensure()` sonrasında okunan değer, eski anlık görüntü değil mutlaka bu isteğin sonucudur.
 * - `ensure(true)`: Yazma işleminden sonra çağrılır. O anda bir istek devam ediyorsa, bittikten sonra bir kez daha istek gönderir,
 *   böylece yazma işleminden sonraki verinin alınmasını garanti eder.
 * - `invalidate()`: Anlık görüntüyü atar ve devam eden eski yanıtı geçersiz kılar; sonraki `ensure()` mutlaka yeni bir istek gönderir.
 * - `markLoaded()`: Çağıran taraf en güncel listeyi başka bir API ile alıp store'a geri yazmıştır (ör. ayarlar sayfası
 *   `replaceModels`); uçuş halindeki eski isteği geçersiz kılarak eski veriyi geri yazmasını önler.
 *
 * Eski anlık görüntü, sayfanın hemen render edebilmesi için store'un ref'inde kalır; verinin güncelliği yalnızca "az önce istek gönderilip gönderilmediği"
 * ve açıkça geçersiz kılınıp kılınmadığına göre belirlenir; artık 60s boyunca her durumda önbellekten okuma penceresi yoktur.
 */
export interface CachedResource {
  ensure: (force?: boolean) => Promise<void>
  invalidate: () => void
  markLoaded: () => void
  isLoaded: () => boolean
  hasInFlightRequest: () => boolean
}

export function createCachedResource<T>(
  request: () => Promise<T>,
  apply: (value: T) => void,
): CachedResource {
  let loaded = false
  // invalidate sonrasında eski bir istek hâlâ uçuş halindeyse, sonraki ensure bunun ardından yeniden gönderilmelidir,
  // aksi hâlde çağıran taraf geçersiz kılınmış bir sonuç Promise'ini await eder.
  let needsFreshRequest = false

  const coordinator = createVersionedRequestCoordinator(request, (value) => {
    apply(value)
    loaded = true
  })

  return {
    ensure(force = false) {
      const mustForce = force || needsFreshRequest
      needsFreshRequest = false
      return coordinator.fetch(mustForce)
    },
    invalidate() {
      loaded = false
      coordinator.invalidate()
      needsFreshRequest = coordinator.hasInFlightRequest()
    },
    markLoaded() {
      coordinator.invalidate()
      needsFreshRequest = false
      loaded = true
    },
    isLoaded: () => loaded,
    hasInFlightRequest: () => coordinator.hasInFlightRequest(),
  }
}

/**
 * Key'e göre önbelleğe alınan tekil kaynaklar (bilgi bankası ayrıntıları, agent'ın görebildiği bilgi bankaları vb.).
 * Bu tür veriler her ensure çağrısında yeniden çekilmez: Bir kez alındıktan sonra açıkça geçersiz kılınana kadar kullanılmaya devam edilir.
 * Çağıran taraf döngü içinde her öğe için ensure çağırır (ör. @ bahsetme listesine count eklemek için); her seferinde API çağrısı yapılamaz.
 */
export function createKeyedSnapshotCache<T>(load: (key: string) => Promise<T>) {
  const snapshots = new Map<string, T>()
  const inflight = new Map<string, { promise: Promise<T>; revision: number }>()
  // Her key için bir nesil vardır; invalidate / force bunu ilerletir. İstek döndüğünde nesil değişmişse,
  // arada bir geçersiz kılma gerçekleştiği veya daha yeni bir istek olduğu anlaşılır; bu yanıt yalnızca çağıran tarafa verilir, anlık görüntüye yazılmaz.
  const revisions = new Map<string, number>()

  const bump = (key: string) => {
    const next = (revisions.get(key) ?? 0) + 1
    revisions.set(key, next)
    return next
  }

  return {
    async ensure(key: string, force = false): Promise<T> {
      if (!force && snapshots.has(key)) return snapshots.get(key) as T
      const existing = inflight.get(key)
      if (existing && !force) return existing.promise
      const revision = force ? bump(key) : (revisions.get(key) ?? 0)
      const promise = (async () => {
        try {
          const value = await load(key)
          const stillCurrent = (revisions.get(key) ?? 0) === revision
          // Yükleme hatası null ile ifade edildiğinde anlık görüntüye yazılmaz; sonraki okumada yeniden denenir.
          if (stillCurrent && value != null) snapshots.set(key, value)
          return value
        } finally {
          if (inflight.get(key)?.revision === revision) inflight.delete(key)
        }
      })()
      inflight.set(key, { promise, revision })
      return promise
    },
    invalidate(key?: string) {
      if (key === undefined) {
        for (const k of new Set([...snapshots.keys(), ...inflight.keys()])) bump(k)
        snapshots.clear()
        inflight.clear()
        return
      }
      bump(key)
      snapshots.delete(key)
      inflight.delete(key)
    },
  }
}
