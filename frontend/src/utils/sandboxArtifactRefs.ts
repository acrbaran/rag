/**
 * Yanıt metnindeki "sandbox tarafından oluşturulan dosya" referanslarını artifact indirme akışına bağla.
 *
 * Model, sandbox içinde oluşturduğu dosyalara Markdown görsel sözdizimiyle başvurur (istem dosyaya şu biçimde yazılmasını belirtir:
 * `![açıklama](sandbox:dosya_adı)`); sunucu, veritabanına kaydetmeden önce bunu dosyanın kararlı tanıtıcısına dönüştürür
 * `resource://<handle>` — bilgi tabanı görselleri ve sohbet ekleriyle aynı referans biçimidir. Her iki yazım da
 * aynı `Message.Artifacts` öğesini gösterir:
 *
 *   - `resource://<handle>` — kaydetme sonrası yetkili biçimdir; geçmiş sohbetlerde okunan biçim budur;
 *   - `sandbox:<dosya_adı>`    — modelin özgün yazımıdır; bu turdaki akış çıktısı henüz dönüştürülmemişken kullanılır.
 *
 * Tanıtıcı biçimi bilgi tabanı görselleriyle tamamen aynı olduğundan, bir yanıtta hem "bulunan bilgi tabanı görseli"nin
 * hem de "beceri tarafından oluşturulan görsel"in bulunması olağandır. Bu nedenle tanıtıcı bu mesajın çıktı listesiyle eşleşmezse null döndürülmelidir,
 * "dosya kullanılamıyor" göstermek yerine varsayılan korumalı görsel işlemeye geri verilmelidir.
 *
 * Görsel çıktıları satır içinde gösterilir (yetkili olarak alındıktan sonra blob ile değiştirilir); diğer türler (HTML grafikler, CSV,
 * belgeler vb.) kart olarak işlenir ve tıklanınca sağdaki sandbox panelinin çıktı sayfasında önizlemeye verilir — metne
 * 1MB boyutunda kendinden kapsayıcı bir HTML iframe yerleştirmek hem yavaş hem de güvensizdir.
 *
 * Kullanıcının sildiği çıktılar listede mezar taşıdır (`deleted_at` içerir); soluk, tıklanamaz bir
 * kart olarak işlenir. Çağıran taraf mezar taşlarını filtreleyip iletmemelidir: filtrelenirse tanıtıcı bu yanıtın çıktılarıyla eşleşmez,
 * bilgi tabanı görseli sayılarak korumalı görsel işlemeye gider ve sonuçta yüklenemeyen kırık bir görsel olarak görünür.
 */

import { escapeHTML } from './security.ts';
import { renderArtifactFileIcon } from './artifactFileIcon';

/** Arka uçtaki artifactListItem / SSE publicArtifactViews ile uyumlu asgari alan kümesi.*/
export interface ArtifactRefMeta {
  index: number;
  file_name: string;
  file_type?: string;
  /** `resource://<handle>`. Arka uçta kaynak dizini etkin değilse boştur; bu durumda yalnızca dosya adına göre ayrıştırılabilir.*/
  handle?: string;
  /**
   * Geçmiş mesajlar `Message.Artifacts` öğesini doğrudan taşır; içindeki depolama başvurusuna `url` denir ve handle ile
   * eş anlamlıdır; ikisinden birini almak yeterlidir.
   */
  url?: string;
  /**
   * Kullanıcının bu dosyayı sildiği zaman. Mezar taşı girdisi, oluşturucuya iletilen listede **mutlaka** kalmalıdır: birincisi, dizin doğrudan
   * indirme adresidir; çıkarılması sonraki dosyaların tamamen kaymasına yol açar. İkincisi, bir tanıtıcının yalnızca eşleştiğinde bu
   * yanıta ait olduğu anlaşılır; eşleşmezse bilgi tabanı görseli sayılır, korumalı görsel oluşturma yoluna gider ve sonunda bozuk görsel olarak görünür.
   */
  deleted_at?: string | null;
}

export interface ArtifactRefContext {
  sessionId: string;
  messageId: string;
}

export interface ArtifactRefLabels {
  /** Kart alt başlığı, örneğin «Önizlemek için tıklayın».*/
  previewHint: string;
  /** Bu tur sona erdiği hâlde başvuru hiçbir çıktıyla eşleşmediğinde gösterilecek alt başlık, örneğin «Dosya kullanılamıyor».*/
  missingHint: string;
  /** Dosya kullanıcı tarafından silindiğinde gösterilecek alt başlık, örneğin «Dosya silindi».*/
  deletedHint: string;
}

const RESOURCE_HANDLE_RE = /^resource:\/\/([A-Za-z0-9_-]{22})$/;
const SANDBOX_NAME_RE = /^sandbox:(?:\/\/)?(.+)$/i;
const IMAGE_EXTENSIONS = new Set(['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'avif']);
const TRANSPARENT_PIXEL =
  'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==';

type ArtifactRef = { kind: 'handle'; handle: string } | { kind: 'name'; name: string };

function parseArtifactRef(href: string): ArtifactRef | null {
  const trimmed = (href || '').trim();
  if (!trimmed) return null;

  const handleMatch = trimmed.match(RESOURCE_HANDLE_RE);
  if (handleMatch) {
    return { kind: 'handle', handle: handleMatch[1] };
  }

  const nameMatch = trimmed.match(SANDBOX_NAME_RE);
  if (!nameMatch) return null;
  let name = nameMatch[1].trim();
  try {
    name = decodeURIComponent(name);
  } catch {
    // Olduğu gibi koruyun: dosya adında yalın `%` bulunması `decodeURIComponent` işlevinin hata fırlatmasına neden olur.
  }
  // Dizin öneki bilgi taşımaz; çıktılar dosya adına göre dizinlenir.
  name = name.split('/').pop() || '';
  return name ? { kind: 'name', name } : null;
}

/**
 * Bu bağlantı hedefinin bir sanal alan çıktısı başvurusu olma olasılığı var mı?
 *
 * Tanıtıcı biçimi bilgi tabanı görselleriyle aynı şekildedir; bu nedenle burada true olması yalnızca «çıktı çözümleyicisine bir kez denemesi için vermeye değer» anlamına gelir,
 * bu mesajda gerçekten bu dosyanın bulunduğu anlamına gelmez.
 */
export function isArtifactRefHref(href: string): boolean {
  return parseArtifactRef(href) !== null;
}

function artifactHandle(artifact: ArtifactRefMeta): string {
  const raw = (artifact.handle || artifact.url || '').trim();
  return raw.match(RESOURCE_HANDLE_RE)?.[1] || '';
}

const CODE_SPAN_OR_FENCE_RE = /(```[\s\S]*?```|~~~[\s\S]*?~~~|`[^`\n]*`)/g;

/**
 * Bir Markdown bağlantı hedefini tara; parantez içindeki özgün metni ve sağ parantezin konumunu döndür.
 *
 * Beceri tarafından oluşturulan dosya adlarında sıkça parantez ve boşluk bulunduğundan, normal ifade yerine parantez eşleştirmesi kullanılır
 * (`Tencent Holdings(00700) Islem Hacmi_838ccc.html`). marked, hedefi ilk boşlukta keser;
 * böylece başvuru hiçbir zaman çıktıyla eşleşmez; derinliğe göre tarama ise gerçekten kapanan bağlantının doğru sağ parantezini kesin olarak bulur.
 */
function scanLinkDestination(text: string, openIndex: number): { inner: string; end: number } | null {
  let depth = 1;
  for (let i = openIndex + 1; i < text.length; i += 1) {
    const ch = text[i];
    if (ch === '\n') return null;
    if (ch === '(') depth += 1;
    else if (ch === ')') {
      depth -= 1;
      if (depth === 0) return { inner: text.slice(openIndex + 1, i), end: i };
    }
  }
  return null;
}

/** Hedefteki isteğe bağlı başlık bölümünü (`dest "title"`) ayır.*/
function splitDestinationTitle(inner: string): { destination: string; title: string } {
  const match = inner.match(/^([\s\S]*?)(\s+(?:"[^"]*"|'[^']*'))$/);
  if (!match) return { destination: inner.trim(), title: '' };
  return { destination: match[1].trim(), title: match[2] };
}

/** ` ](` öncesinde aynı satırda kapanmış bir `[...]` bulunmalıdır; aksi hâlde bu bir bağlantı değildir.*/
function hasLinkLabelBefore(text: string, closeBracketIndex: number): boolean {
  for (let i = closeBracketIndex - 1; i >= 0; i -= 1) {
    const ch = text[i];
    if (ch === '\n') return false;
    if (ch === '[') return true;
  }
  return false;
}

function normalizeSegment(segment: string): string {
  if (!segment.includes('](')) return segment;

  let out = '';
  let cursor = 0;
  while (cursor < segment.length) {
    const relative = segment.slice(cursor).indexOf('](');
    if (relative < 0) break;
    const closeBracket = cursor + relative;
    const open = closeBracket + 1;

    if (!hasLinkLabelBefore(segment, closeBracket)) {
      out += segment.slice(cursor, open + 1);
      cursor = open + 1;
      continue;
    }
    const scanned = scanLinkDestination(segment, open);
    if (!scanned) {
      out += segment.slice(cursor, open + 1);
      cursor = open + 1;
      continue;
    }

    const { destination, title } = splitDestinationTitle(scanned.inner);
    const ref = parseArtifactRef(destination);
    if (!ref || ref.kind !== 'name') {
      out += segment.slice(cursor, scanned.end + 1);
      cursor = scanned.end + 1;
      continue;
    }

    // Percent-encode so marked sees a single whitespace-free token;
    // parseArtifactRef decodes it again on the way out.
    out += `${segment.slice(cursor, open + 1)}sandbox:${encodeURIComponent(ref.name)}${title})`;
    cursor = scanned.end + 1;
  }
  return out + segment.slice(cursor);
}

/**
 * marked ayrıştırmadan önce, `sandbox:` başvurularının hedefini boşluk içermeyen tek bir token olacak şekilde düzenle.
 *
 * Bu adım yapılmazsa, boşluk içeren dosya adı marked tarafından boşlukta kesilir ve hedefin yalnızca ilk yarısı kalır,
 * ikinci yarı ise gövde metni olarak sızar; bu tam olarak «kart adının kesilmesi + kuyruğun yalın metne dönüşmesi» olgusudur.
 * Kod bloğu içindeki örnekleri olduğu gibi koru.
 */
export function normalizeSandboxArtifactRefs(markdown: string): string {
  if (!markdown || !markdown.includes('](')) return markdown;
  if (!/\]\(\s*sandbox:/i.test(markdown)) return markdown;

  const parts = markdown.split(CODE_SPAN_OR_FENCE_RE);
  for (let i = 0; i < parts.length; i += 2) {
    parts[i] = normalizeSegment(parts[i]);
  }
  return parts.join('');
}

/** Başvuruyu somut bir çıktıya çözümle; çözümlenemezse (henüz tamamen toplanmadıysa/dosya adı eşleşmiyorsa) null döndür.*/
export function resolveArtifactRef(
  href: string,
  artifacts: ArtifactRefMeta[] | undefined | null,
): ArtifactRefMeta | null {
  const ref = parseArtifactRef(href);
  if (!ref || !artifacts?.length) return null;

  if (ref.kind === 'handle') {
    return artifacts.find((item) => artifactHandle(item) === ref.handle) || null;
  }
  return artifacts.find((item) => (item.file_name || '').trim() === ref.name) || null;
}

function fileExtension(fileName: string): string {
  const base = (fileName || '').trim().toLowerCase();
  const dot = base.lastIndexOf('.');
  return dot > 0 ? base.slice(dot + 1) : '';
}

/**
 * Görsel olarak satır içi oluşturulup oluşturulmayacağı. SVG özellikle hariç tutulur: çalıştırılabilir içeriktir; kart + önizleme için sanal alan yolundan geçer.
 */
function rendersAsImage(artifact: ArtifactRefMeta): boolean {
  const ext = fileExtension(artifact.file_name);
  if (ext) return IMAGE_EXTENSIONS.has(ext);
  const type = (artifact.file_type || '').toLowerCase();
  return type.startsWith('image/') && !type.includes('svg');
}

// Blob URL'leri (oturum, mesaj, indis) bazında önbelleğe al ve window üzerine bağla: Vite sıcak güncellemesi modülü değiştirir
// ancak belgeyi yeniden oluşturmaz; modül düzeyindeki Map, yüklenmiş görsellerin yer tutucu görsele dönmesine neden olur.
type ArtifactBlobState = { blobByKey: Map<string, string>; inflight: Map<string, Promise<string | null>> };

const artifactBlobState: ArtifactBlobState = (() => {
  const fresh = (): ArtifactBlobState => ({ blobByKey: new Map(), inflight: new Map() });
  if (typeof window === 'undefined') return fresh();
  const scope = window as typeof window & { __rethraArtifactBlobCacheV1__?: ArtifactBlobState };
  scope.__rethraArtifactBlobCacheV1__ ||= fresh();
  return scope.__rethraArtifactBlobCacheV1__;
})();

function blobCacheKey(ctx: ArtifactRefContext, index: number): string {
  return `${ctx.sessionId}\u0000${ctx.messageId}\u0000${index}`;
}

// chatMarkdownRenderer'ın akışlı görsel iskeletiyle aynı sınıf adı; stiller yeniden kullanılır.
const STREAMING_PLACEHOLDER =
  '<span class="streaming-image-loading"><span class="streaming-image-loading__skeleton"></span></span>';

/**
 * Kartın üç durumu:
 *   - `ready`   —— normal, önizlemeyi açmak için tıklanabilir;
 *   - `pending` —— bu tur sona erdi ancak başvuru hiçbir çıktıyla eşleşmiyor (model var olmayan bir dosyaya başvurdu);
 *   - `deleted` —— dosya gerçekten oluşturulmuştu, ancak kullanıcı tarafından silindi ve baytları geri kazanıldı.
 * Son iki durum tıklanamaz, ancak ayırt edilebilmelidir: biri «hiç yoktu», diğeri «onu sen sildin».
 */
type ArtifactCardVariant = 'ready' | 'pending' | 'deleted';

function renderCard(
  fileName: string,
  hint: string,
  index: number | null,
  variant: ArtifactCardVariant,
): string {
  const safeName = escapeHTML(fileName);
  const safeHint = escapeHTML(hint);
  // Kart satır içi bir öğe olmalıdır: marked görseli <p> içine sarar; blok düzeyindeki öğeler HTML ayrıştırıcısı tarafından
  // paragrafın dışına taşınır ve ana metin yapısını bozar.
  const interactive = variant === 'ready' && index !== null
    ? ` data-artifact-index="${index}" role="button" tabindex="0"`
    : '';
  const state = variant === 'ready' ? '' : ` artifact-ref-card--${variant}`;
  return (
    `<span class="artifact-ref-card${state}"${interactive} title="${safeName}">`
    + `<span class="artifact-ref-card__icon" aria-hidden="true">${renderArtifactFileIcon(fileName)}</span>`
    + '<span class="artifact-ref-card__text">'
    + `<span class="artifact-ref-card__name">${safeName}</span>`
    + `<span class="artifact-ref-card__hint">${safeHint}</span>`
    + '</span></span>'
  );
}

function renderImage(
  artifact: ArtifactRefMeta,
  alt: string,
  ctx: ArtifactRefContext | null,
): string {
  const safeAlt = escapeHTML(alt || artifact.file_name || '');
  // Daha önce çekildiyse doğrudan blob ver: akışlı yeniden oluşturma <img>'yi yeniden kurar; aksi halde her karede yer tutucuya geri döner.
  const cached = ctx ? artifactBlobState.blobByKey.get(blobCacheKey(ctx, artifact.index)) : undefined;
  const src = cached || TRANSPARENT_PIXEL;
  const loading = cached ? '' : ' data-img-loading="1"';
  return (
    `<img class="markdown-image artifact-ref-image" src="${src}" alt="${safeAlt}"`
    + ` data-artifact-index="${artifact.index}"${loading}>`
  );
}

/**
 * Bir Markdown görseli/bağlantı hedefini işle.
 *
 * null döndürmek, bunun bir korumalı alan çıktı başvurusu olmadığını belirtir; çağıran varsayılan işlemeye geri dönmelidir (normal görseller,
 * `resource://` korumalı görseller, dış bağlantılar vb. etkilenmez).
 * Boş dize döndürmek, hedefin boş olduğunu belirtir; çağıran artık kırık görsel çizmemelidir.
 */
export function renderArtifactReference(args: {
  href: string;
  alt?: string;
  artifacts?: ArtifactRefMeta[] | null;
  labels: ArtifactRefLabels;
  context?: ArtifactRefContext | null;
  /** Bu turdaki yanıt hâlâ oluşturuluyor. Çıktılar ancak bu tur bittiğinde toplanır; şu anda ayrıştırılamaması normaldir.*/
  streaming?: boolean;
}): string | null {
  const href = (args.href || '').trim();
  if (!href) return '';
  const ref = parseArtifactRef(href);
  if (!ref) return null;

  const artifact = resolveArtifactRef(href, args.artifacts);
  if (!artifact) {
    // Tanıtıcı bu mesajın çıktılarıyla eşleşmiyorsa, bunun başka bir korumalı dosya olduğu anlaşılır (bilgi tabanı arama görseli,
    // ek görseli...). Varsayılan işlemeye geri ver; hydrateProtectedFileImages yetkilendirmeyle çeksin.
    if (ref.kind === 'handle') return null;
    // Çıktı listesi ancak bu tur bittiğinde complete olayıyla gelir; akış sırasında mutlaka ayrıştırılamaz.
    // Bu durumda kart yerine iskelet ekranı göster: hem yarım dosya adının anlık görünmesini önler hem de «oluşturuluyor»
    // durumunun aslında sona ermiş bir yanıtta kalmasını engeller.
    if (args.streaming) return STREAMING_PLACEHOLDER;
    // Bu tur bittikten sonra da eşleşmiyorsa, modelin var olmayan bir dosyaya başvurduğu anlaşılır. Durumu olduğu gibi belirt,
    // «oluşturuluyor» göstermeye devam etme.
    const fallbackName = ref.name || (args.alt || '').trim();
    if (!fallbackName) return '';
    return renderCard(fallbackName, args.labels.missingHint, null, 'pending');
  }

  // Kullanıcının sildiği dosyanın baytları zaten geri kazanıldı; indirme 404 döner. Görsel türü çıktılar da karta indirgenir:
  // renderImage yoluyla yalnızca çekme başarısız olur ve sonsuza dek yüklenemeyen kırık bir görsel kalır.
  if (artifact.deleted_at) {
    const name = artifact.file_name || (args.alt || '').trim();
    if (!name) return '';
    return renderCard(name, args.labels.deletedHint, null, 'deleted');
  }

  if (rendersAsImage(artifact)) {
    return renderImage(artifact, args.alt || '', args.context ?? null);
  }
  return renderCard(artifact.file_name, args.labels.previewHint, artifact.index, 'ready');
}

/**
 * Bir Markdown bağlantısını (`[metin](sandbox:dosya)` / `[metin](resource://<handle>)`) işle.
 *
 * Model çıktıya görsel yerine düz bağlantıyla da başvurabilir. Varsayılan `<a href="resource://…">`
 * tarayıcıda açılamaz (akış sırasındaki `sandbox:` biçiminin href'i ise temizleyicide silinir), bu yüzden
 * eşleşen bağlantı, tıklanınca kimlik doğrulamalı indirmeyi başlatan bir öğeye dönüştürülür. `href`
 * bilerek verilmez: temizleyici href taşıyan bağlantılara `target="_blank"` ekler.
 *
 * null döndürmek varsayılan bağlantı işlemeye geri dönülmesi gerektiğini belirtir.
 */
export function renderArtifactLink(args: {
  href: string;
  /** Bağlantı etiketinin zaten işlenmiş (güvenli) HTML'i. */
  label: string;
  artifacts?: ArtifactRefMeta[] | null;
  labels: ArtifactRefLabels;
}): string | null {
  const ref = parseArtifactRef(args.href);
  if (!ref) {
    // Model sık sık öneksiz dosya adı yazar (`[rapor.docx](rapor.docx)`); sunucu düz bağlantıları
    // yeniden yazmaz. Yalnızca bu mesajın bir çıktısıyla birebir eşleşen göreli yollar ele alınır.
    const bare = bareArtifactName(args.href);
    const match = bare
      ? args.artifacts?.find((item) => (item.file_name || '').trim() === bare && !item.deleted_at)
      : null;
    return match ? renderArtifactDownloadLink(match, args.label) : null;
  }

  const artifact = resolveArtifactRef(args.href, args.artifacts);
  if (!artifact) {
    // Bu mesaja ait olmayan tanıtıcı: başka bir korumalı kaynak, varsayılana bırak.
    if (ref.kind === 'handle') return null;
    // Çözülemeyen `sandbox:` başvurusu hiçbir yere gitmez; kırık bağlantı yerine düz metin göster.
    return `<span class="artifact-ref-link artifact-ref-link--pending">${args.label}</span>`;
  }
  if (artifact.deleted_at) {
    const hint = escapeHTML(args.labels.deletedHint);
    return `<span class="artifact-ref-link artifact-ref-link--deleted" title="${hint}">${args.label}</span>`;
  }
  return renderArtifactDownloadLink(artifact, args.label);
}

function renderArtifactDownloadLink(artifact: ArtifactRefMeta, label: string): string {
  const safeName = escapeHTML(artifact.file_name || '');
  return (
    `<a class="artifact-ref-link" data-artifact-index="${artifact.index}" role="button" tabindex="0"`
    + ` title="${safeName}">${label}</a>`
  );
}

/** Şemasız göreli bir bağlantı hedefinin dosya adını döndür (`./output/a.docx` → `a.docx`). */
function bareArtifactName(href: string): string {
  const trimmed = (href || '').trim();
  if (!trimmed || trimmed.startsWith('#') || trimmed.startsWith('//')) return '';
  if (/^[A-Za-z][A-Za-z0-9+.-]*:/.test(trimmed)) return '';
  let name = trimmed.split(/[?#]/)[0];
  try {
    name = decodeURIComponent(name);
  } catch {
    // Yalın `%` içeren adları olduğu gibi bırak.
  }
  return name.split('/').pop()?.trim() || '';
}

/** İndirme bağlantısından etkinleştirilen çıktı dizinini al; `null` olayın bağlantıyla ilgili olmadığını gösterir. */
export function artifactDownloadIndexFromEventTarget(target: EventTarget | null): number | null {
  if (!(target instanceof Element)) return null;
  const link = target.closest('a.artifact-ref-link[data-artifact-index]');
  if (!link) return null;
  const index = Number(link.getAttribute('data-artifact-index'));
  return Number.isInteger(index) && index >= 0 ? index : null;
}

/** Çıktıyı kimlik doğrulamayla çekip tarayıcı indirmesi olarak kaydet. */
export async function downloadArtifactFile(
  ctx: ArtifactRefContext,
  index: number,
  fileName: string,
): Promise<void> {
  const { downloadArtifact } = await import('@/api/chat');
  const blob = await downloadArtifact(ctx.sessionId, ctx.messageId, index);
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = fileName || 'artifact';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

async function loadArtifactBlobURL(ctx: ArtifactRefContext, index: number): Promise<string | null> {
  const key = blobCacheKey(ctx, index);
  const cached = artifactBlobState.blobByKey.get(key);
  if (cached) return cached;

  let task = artifactBlobState.inflight.get(key);
  if (!task) {
    task = (async () => {
      try {
        // Imported lazily so parsing/rendering stays free of the axios
        // transport — those parts run in plain Node during unit tests.
        const { downloadArtifact } = await import('@/api/chat');
        const blob = await downloadArtifact(ctx.sessionId, ctx.messageId, index);
        const blobURL = URL.createObjectURL(blob);
        artifactBlobState.blobByKey.set(key, blobURL);
        return blobURL;
      } catch (error) {
        console.warn('[sandboxArtifactRefs] artifact image load failed:', error);
        return null;
      } finally {
        artifactBlobState.inflight.delete(key);
      }
    })();
    artifactBlobState.inflight.set(key, task);
  }
  return task;
}

/**
 * Ana metindeki ürün görsellerini kimlik doğrulamalı olarak çekilen blob'larla değiştir.
 *
 * `hydrateProtectedFileImages` gibi bu da idempotenttir: daha önce değiştirilmiş öğeler `authHydrated` işareti taşır,
 * Aynı dosyaya yönelik eşzamanlı istekler aynı Promise'i paylaşır.
 */
export async function hydrateArtifactImages(
  root: ParentNode | null | undefined,
  ctx: ArtifactRefContext | null | undefined,
): Promise<void> {
  if (!root || !ctx?.sessionId || !ctx?.messageId) return;

  const images = Array.from(
    root.querySelectorAll<HTMLImageElement>('img.artifact-ref-image[data-artifact-index]'),
  ).filter((img) => img.dataset.authHydrated !== '1');
  if (!images.length) return;

  await Promise.all(images.map(async (img) => {
    const index = Number(img.getAttribute('data-artifact-index'));
    if (!Number.isInteger(index) || index < 0) return;
    img.dataset.authHydrated = '1';

    const blobURL = await loadArtifactBlobURL(ctx, index);
    if (!blobURL) {
      img.dataset.authHydrated = '0';
      return;
    }
    img.src = blobURL;
    img.removeAttribute('data-img-loading');
  }));
}

/**
 * Tıklama/klavye olayından etkinleştirilen ürün kartı dizinini al; `null` döndürülmesi olayın kartla ilgili olmadığını gösterir.
 */
export function artifactIndexFromEventTarget(target: EventTarget | null): number | null {
  if (!(target instanceof Element)) return null;
  const card = target.closest('.artifact-ref-card[data-artifact-index]');
  if (!card) return null;
  const index = Number(card.getAttribute('data-artifact-index'));
  return Number.isInteger(index) && index >= 0 ? index : null;
}
