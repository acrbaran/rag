/**
 * tdesign-icons-vue-next'in harici CDN'ye yönelik iconfont isteklerini engelle.
 *
 * Arka plan (issue #867 / #897):
 *   tdesign-icons-vue-next'in Icon / IconFont bileşenleri, onMounted sırasında şu yöntemle
 *   `checkScriptAndLoad` / `checkLinkAndLoad` aracılığıyla document içine şunu işaret eden bir düğüm ekler:
 *   `https://tdesign.gtimg.com/icon/<version>/fonts/index.(js|css)`
 *   <script> / <link>. Harici ağın olmadığı ortamlarda istek başarısız olur ve tüm simgeler render edilmez.
 *
 * Yaklaşım: Vue app bağlanmadan önce tdesign eşleştirme kurallarına uyan yer tutucu düğümler ekleyin,
 *   böylece `checkScriptAndLoad` / `checkLinkAndLoad` içindeki tekrar kontrolü eşleşir ve doğrudan döner,
 *   CDN'yi işaret eden gerçek düğümler artık eklenmez.
 *
 *   - Eşleştirme seçicisinin kaynağı: tdesign-icons-vue-next/esm/utils/check-url-and-load.js
 *       `.t-svg-js-stylesheet--unique-class[src="<url>"]`
 *       `.t-iconfont-stylesheet--unique-class[href="<url>"]`
 *   - Yer tutucu <script> / <link>, standart dışı type / rel kullanır; tarayıcı gerçekte ağ isteği başlatmaz.
 *
 * SVG sprite simgeleri ise index.html içindeki <script src="/tdesign-icons/.../index.js"></script> aracılığıyla
 * önceden yerel olarak kaydedilir; bu nedenle <t-icon name="..."> normal şekilde render edilmeye devam eder.
 */

const SVG_SCRIPT_CLASS = "t-svg-js-stylesheet--unique-class";
const ICONFONT_LINK_CLASS = "t-iconfont-stylesheet--unique-class";

// tdesign-icons-vue-next 0.4.x'in dahili sabit kodlanmış adresleriyle hizalanır; olası yükseltmelerle uyumluluk için birden fazla sürüm numarası birlikte bulunur.
const BLOCKED_ICON_VERSIONS = ["0.4.0", "0.4.1", "0.4.2", "0.4.3", "0.4.4"];

const BLOCKED_SCRIPT_URLS = BLOCKED_ICON_VERSIONS.map(
  (version) => `https://tdesign.gtimg.com/icon/${version}/fonts/index.js`,
);

const BLOCKED_LINK_URLS = BLOCKED_ICON_VERSIONS.map(
  (version) => `https://tdesign.gtimg.com/icon/${version}/fonts/index.css`,
);

let installed = false;

export function installTDesignIconOfflineGuard(): void {
  if (installed || typeof document === "undefined") return;
  installed = true;

  const body = document.body;
  if (!body) {
    document.addEventListener(
      "DOMContentLoaded",
      () => installTDesignIconOfflineGuard(),
      { once: true },
    );
    installed = false;
    return;
  }

  BLOCKED_SCRIPT_URLS.forEach((src) => {
    const exists = document.querySelector(
      `script.${SVG_SCRIPT_CLASS}[src="${src}"]`,
    );
    if (exists) return;
    const stub = document.createElement("script");
    stub.setAttribute("class", SVG_SCRIPT_CLASS);
    stub.setAttribute("src", src);
    // Standart dışı MIME türü, tarayıcının betiğin fetch/çalıştırma aşamasını atlamasını sağlar.
    stub.setAttribute("type", "text/no-load");
    stub.setAttribute("data-rethra-blocked-cdn", "tdesign-icons");
    body.appendChild(stub);
  });

  BLOCKED_LINK_URLS.forEach((href) => {
    const exists = document.querySelector(
      `link.${ICONFONT_LINK_CLASS}[href="${href}"]`,
    );
    if (exists) return;
    const stub = document.createElement("link");
    stub.setAttribute("class", ICONFONT_LINK_CLASS);
    stub.setAttribute("href", href);
    // rel="stylesheet" bildirilmezse tarayıcı stil sayfası isteği başlatmaz.
    stub.setAttribute("rel", "preload-blocked");
    stub.setAttribute("data-rethra-blocked-cdn", "tdesign-icons");
    document.head.appendChild(stub);
  });
}
