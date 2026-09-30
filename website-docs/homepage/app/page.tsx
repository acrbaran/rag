import { existsSync } from "node:fs";
import { join } from "node:path";
import { BrandLogo } from "./brand-logo";
import { Icon } from "./ui";
import { homeAssets } from "../../shared/header";
import { ProductGallery, type GalleryShot, type GallerySlide } from "./product-gallery";
import { Header, ProductVideo } from "./interactive";
import { clients, dataSources, modelProviders, IntegrationMark } from "./brands";
import s from "./home.module.css";

const repo = "https://github.com/acrbaran/rag";
const docs = "/docs/";
const guide = (path: string) => `${docs}${path}.html`;
const modes = [
  { number: "01", icon: "search", label: "RAG", title: "Cevaplar kaynaklarla doğrulanabilir", description: "Anlam ve anahtar kelime aramasını birleştirerek ilgili bilgileri bulun; cevaplar, açılıp özgün metinden doğrulanabilen kaynak alıntıları içerir.", tags: ["Karma arama", "Çok modlu ayrıştırma", "Özgün metin alıntıları"], link: "03-features/05-retrieval-engines" },
  { number: "02", icon: "agent", label: "Agent", title: "Görevleri bilgi ve araçlarla tamamlayın", description: "Ajan, göreve göre bilgi tabanında ve web'de arama yapar, MCP araçlarını ve becerileri çağırır; korumalı alanda dosya işler ve betik çalıştırır, bilgisayarınızdaki tarayıcıyı kullanabilir ve onayladığınız tercihleri oturumlar arasında hatırlar.", tags: ["Çok adımlı akıl yürütme", "Beceriler ve korumalı alan", "Yerel tarayıcı", "MCP araçları", "Uzun süreli bellek"], link: "03-features/07-agent" },
  { number: "03", icon: "wiki", label: "Wiki", title: "Belgeleri Wiki'ye dönüştürün", description: "Ham belgelerden birbirine bağlantılı Wiki sayfaları ve bilgi grafiği oluşturur; görüntüleme, düzenleme ve sürüm geri alma desteklenir.", tags: ["Otomatik düzenleme", "Bilgi grafiği", "Sürüm geri alma"], link: "03-features/14-wiki" },
];
const releaseExtras = ["Confluence / DingTalk belge veri kaynağı", "27 model sağlayıcısı kataloğu", "Bocha / Serply internet araması", "Japonca arayüz", "Yalnızca izin listeli çıkışlar"];

const productShotExists = (src: string) => existsSync(join(process.cwd(), "public", src));
type SlideSource = Omit<GallerySlide, "shots"> & ({ image: string; alt: string } | { shots: Omit<GalleryShot, "available">[] });
const withAvailability = (slide: SlideSource): GallerySlide => {
  const shots = "shots" in slide ? slide.shots : [{ image: slide.image, alt: slide.alt }];
  return { name: slide.name, icon: slide.icon, title: slide.title, description: slide.description, link: slide.link, external: slide.external, shots: shots.map(shot => ({ ...shot, available: productShotExists(`${homeAssets}/product/${shot.image}.png`) })) };
};
const releaseSlides = [
  { name: "Yerel tarayıcı", icon: "browser", shots: [
    { label: "Görevi yürüt", icon: "browser", image: "local-browser-task", alt: "Gerçek Rethra arayüzü: akıllı muhakeme sohbeti yerel tarayıcıyı kullanıyor; sohbette görev önizlemesi ile duraklatma, devam ettirme ve sonlandırma kontrolleri gösteriliyor" },
    { label: "Uzantıyı bağla", icon: "plug", image: "browser-connection", alt: "Gerçek Rethra arayüzü: araç kutusundaki tarayıcı bağlantı sayfasında BrowserSkill uzantısı bağlı ve ajanın gerçekleştirebileceği web işlemleri listeleniyor" },
  ], title: "Bilgisayarınızdaki tarayıcıyı kullanın", description: "Açık kaynak BrowserSkill uzantısı sayesinde ajan, Chrome veya Edge tarayıcınızda doğrudan web sayfalarını açar ve formları doldurur; oturum açma veya doğrulama kodu gerektiğinde kontrolü size bırakır.", link: guide("05-clients/09-local-browser"), external: { label: "BrowserSkill", href: "https://github.com/Tencent/BrowserSkill", logo: "browserskill.png" } },
  { name: "MCP Server", icon: "plug", image: "mcp-server-endpoint", title: "Bilgi tabanını diğer AI araçlarına yayımlayın", description: "Alan için bir MCP uç noktası oluşturun; Claude, Cursor ve diğer istemciler bağlandıktan sonra bilgi tabanında arama yapabilir ve soru sorabilir.", alt: "Gerçek Rethra arayüzü: uç nokta adresi ile Cursor ve Claude Desktop için mcpServers yapılandırmasını içeren MCP uç noktası bağlantı bilgileri", link: guide("03-features/08-mcp") },
  { name: "Sohbet kontrolü", icon: "branch", image: "chat-steer-queue", title: "Devam eden sohbeti istediğiniz zaman ayarlayın", description: "Yanıt sırasında ek istekler ekleyebilir, herhangi bir sorudan dallanabilir veya geri alabilirsiniz; oluşturulan dosyalar Çıktılar sayfasında toplanır.", alt: "Rethra gerçek arayüzü: Yanıt oluşturulurken giriş kutusunun üzerinde sırada bekleyen ek istekler", link: guide("03-features/18-chat-experience") },
].map(withAvailability);
const wikiSlides = [
  { name: "Bilgi grafiği", icon: "channels", image: "wiki-graph", title: "Bağlantıları takip ederek ilgili bilgileri görüntüleyin", description: "Bilgi grafiğinde sayfalar arasındaki ilişkileri görüntüleyin. İlgili içeriği okumak için bir öğeye tıklayın.", alt: "Wiki bilgi grafiği: Günlük gider geri ödeme öğeleri ile ilgili sayfalar arasındaki bağlantılar; sağ tarafta öğe ayrıntıları gösterilir" },
  { name: "Sayfa gezintisi", icon: "wiki", image: "wiki-browser", title: "Konulara göre düzenleyin, kaynakları koruyun", description: "Wiki etkinleştirildiğinde, bilgi bankası belgelerinden kişiler, ürünler ve kavramlar çıkarılır; kaynak alıntıları içeren sayfalar oluşturulur ve dizin üzerinden gezilebilir.", alt: "Wiki tarayıcısı: Konulara göre düzenlenmiş dizin, yıllık izin sayfası, ilişkili öğeler ve özgün belge alıntıları" },
  { name: "Sürüm geçmişi", icon: "history", image: "wiki-revision-history", title: "İstediğiniz zaman düzenleyin, değişiklikler geri izlenebilir", description: "Sayfaları doğrudan düzenleyebilir veya bakım için ajandan yardım isteyebilirsiniz. Sürüm farklarını görüntüleyin ve gerektiğinde geçmiş içeriğe geri dönün.", alt: "Wiki sürüm geçmişi: Yıllık izin sayfasının geçmiş sürüm listesi, içerik farkları ve geri alma seçeneği" },
].map(withAvailability);
const sandboxSlides = [
  { name: "Oturum düzeyinde sanal alan", icon: "sandbox", image: "skill-sandbox-chat", title: "Görevleri aynı sanal alanda yürütmeye devam edin", description: "Docker, E2B ve Cube desteklenir. Aynı oturumdaki çok turlu görevler tek bir çalışma alanını paylaşır; oluşturulan dosyalar önizlenebilir ve indirilebilir.", alt: "Rethra gerçek arayüzü: Ajan, bilgi bankasına dayanarak bir Word belgesi oluşturur ve sohbetin yanında çıktı önizlemesini açar", link: guide("03-features/22-skills-sandbox") },
  { name: "Grafik masaüstü ve terminal", icon: "monitor", shots: [
    { label: "Grafik masaüstü", icon: "monitor", image: "sandbox-desktop", alt: "Rethra gerçek arayüzü: Sohbetin sağındaki sanal alan panelinin masaüstü sekmesi; sanal alandaki XFCE grafik masaüstünü ve dosya yöneticisini gösterir" },
    { label: "Etkileşimli terminal", icon: "terminal", image: "sandbox-terminal", alt: "Rethra gerçek arayüzü: Sohbetin sağındaki sanal alan terminali, çalışma alanının output dizininde oluşturulan Word dosyalarını listeler" },
  ], title: "Masaüstünü ve terminali açın, her adımı net biçimde görün", description: "Sohbetin yanında grafik masaüstünü veya etkileşimli terminali açarak ajanın her adımını görüntüleyin; gerektiğinde kontrolü bizzat devralın.", link: guide("03-features/22-skills-sandbox") },
  { name: "Beceri kataloğu", icon: "skills", image: "skill-catalog", title: "Becerileri yükleyin, yönetin ve yeniden kullanın", description: "Becerileri ClawHub, SkillHub, Git veya ZIP üzerinden yükleyin; bunları çalışma alanında merkezi olarak yönetin ve yeniden kullanın.", alt: "Rethra gerçek arayüzü: Araç kutusundaki beceri yönetimi sayfası; docx, pptx ve pdf becerilerini ve yüklendikleri sanal alanları listeler", link: guide("03-features/22-skills-sandbox") },
].map(withAvailability);

export default function Home() {
  return <div className={s.site}>
    <a className={s.skipLink} href="#main">Ana içeriğe geç</a>
    <Header />
    <main id="main">
      <section className={`${s.shell} ${s.hero}`} aria-labelledby="hero-title">
        <a href="#release" className={s.releaseLink}><span>v0.8.2</span> Yerel tarayıcı, MCP Server ve sohbet dallandırma <Icon name="arrow" /></a>
        <div className={s.heroGrid}>
          <h1 id="hero-title">Yanıtları bulmanıza yardım eder,<br /><em>bilgiyi uygulamaya koyar.</em></h1>
          <div className={s.heroAside}>
            <p className={s.eyebrow}>OPEN SOURCE · RETHRA</p>
            <p className={s.heroDescription}>Açık kaynak, kurumsal düzeyde bilgi yönetimi çerçevesi.<br />Ekibinizin belgelerini bir araya getirir; bilgi soru-cevabı, görev yürütme ve Wiki düzenleme için kullanır.</p>
            <div className={s.actions}><a className={s.primary} href="#get-started">Başlayın <Icon name="arrow" /></a><a className={s.secondary} href={repo} target="_blank" rel="noreferrer"><Icon name="github" /> GitHub</a></div>
            <p className={s.heroNote}>RAG soru-cevap / Agent muhakemesi / Otomatik Wiki</p>
          </div>
        </div>
        <ProductVideo />
        <div className={s.trustBar}><span><Icon name="github" /> Açık kaynak · MIT License</span><span><Icon name="server" /> Şirket içi dağıtım desteği</span><span><Icon name="model" /> Model ve depolamayı özgürce seçin</span></div>
      </section>
      <section id="capabilities" className={`${s.shell} ${s.section}`} aria-labelledby="capabilities-title">
        <div className={s.sectionHeading}><div><p className={s.eyebrow}>01 / KNOWLEDGE AT WORK</p><h2 id="capabilities-title">Bilgi soru-cevabı, görev yürütme ve otomatik Wiki</h2></div><p>Bilgi sorgulamak için RAG, çok adımlı görevler için Agent,<br />bilgiyi düzenlemek için Wiki. Üç yetenek aynı bilgi tabanını paylaşır.</p></div>
        <div className={s.modeGrid}>{modes.map(mode => <article className={s.mode} key={mode.label}>
          <div className={s.modeTop}><Icon name={mode.icon} /><span>{mode.number} / {mode.label.toUpperCase()}</span></div>
          <h3>{mode.title}</h3><p>{mode.description}</p><ul className={s.tags}>{mode.tags.map(tag => <li key={tag}>{tag}</li>)}</ul>
          <a className={s.textLink} href={mode.label === "Wiki" ? "#wiki" : guide(mode.link)}>{mode.label} hakkında bilgi edinin <Icon name="arrow" /></a>
        </article>)}</div>
      </section>
      <section id="release" className={s.release} aria-labelledby="release-title"><div className={s.shell}>
        <div className={s.sectionHeading}><div><p className={s.eyebrow}>02 / INTRODUCING v0.8.2</p><h2 id="release-title">Ajan tarayıcıyı kullanabilir,<br />bilgi tabanı diğer AI araçlarına bağlanabilir.</h2></div><a className={s.textLink} href={guide("07-releases/v0.8.2")}>Sürüm notlarını görüntüle <Icon name="arrow" /></a></div>
        <ProductGallery id="release-gallery" label="v0.8.2" slides={releaseSlides} />
        <div className={s.releaseExtras}><span>Bu güncellemede ayrıca</span>{releaseExtras.map(item => <p key={item}>{item}</p>)}</div>
      </div></section>
      <section id="skills-sandbox" className={`${s.release} ${s.sandboxTopic}`} aria-labelledby="skills-sandbox-title"><div className={s.shell}>
        <div className={s.sectionHeading}><div><p className={s.eyebrow}>03 / SKILLS &amp; SANDBOX</p><h2 id="skills-sandbox-title">Ajan becerileri çalıştırabilir,<br />dosya da oluşturabilir.</h2></div><a className={s.textLink} href={guide("03-features/22-skills-sandbox")}>Beceriler ve sanal alan hakkında bilgi edinin <Icon name="arrow" /></a></div>
        <ProductGallery id="sandbox-gallery" label="Beceriler ve sanal alanlar" slides={sandboxSlides} />
      </div></section>
      <section id="wiki" className={`${s.shell} ${s.section} ${s.wiki}`} aria-labelledby="wiki-title">
        <div className={s.sectionHeading}>
          <div><p className={s.eyebrow}>04 / AUTOMATIC WIKI</p><h2 id="wiki-title">Belgeleri gezilebilir bir Wiki'ye dönüştürün.</h2></div>
          <a className={s.textLink} href={guide("03-features/14-wiki")}>Wiki'nin nasıl kullanıldığını öğrenin <Icon name="arrow" /></a>
        </div>
        <ProductGallery id="wiki-gallery" label="Wiki" slides={wikiSlides} />
      </section>
      <section id="ecosystem" className={`${s.shell} ${s.section}`} aria-labelledby="ecosystem-title">
        <div className={s.sectionHeading}><div><p className={s.eyebrow}>05 / INTEGRATIONS</p><h2 id="ecosystem-title">Veri kaynağı ve araç entegrasyonları</h2></div><p>Feishu, Confluence, GitLab gibi platformlardaki belgeleri eşitleyin;<br />IM, tarayıcı eklentisi, MCP veya API üzerinden sorgulayıp kullanın.</p></div>
        <div className={s.ecosystem}>
          <div className={s.ecosystemColumn}><Icon name="sources" /><h3>Belgeleri içe aktarın ve eşitleyin</h3><p>Dosya yükleyin, web sayfalarını içe aktarın veya harici veri kaynaklarına bağlanın.</p><div className={s.integrations}>{dataSources.map(item => <span key={item.name}><IntegrationMark item={item} /></span>)}</div><a className={s.textLink} href={guide("03-features/10-datasource")}>Veri kaynağı entegrasyonu <Icon name="arrow" /></a></div>
          <div className={s.ecosystemCore}><BrandLogo /><span>Ekip bilgi tabanı ve ajanlar</span><div>Anla · Ara · Akıl yürüt · Harekete geç</div></div>
          <div className={s.ecosystemColumn}><Icon name="channels" /><h3>Sık kullandığınız araçlardan erişin</h3><p>IM soru-cevap, tarayıcı eklentisi, MCP istemcileri ve geliştirici aracı entegrasyonları desteklenir.</p><div className={s.integrations}>{clients.map(item => <span key={item.name}><IntegrationMark item={item} /></span>)}</div><a className={s.textLink} href={guide("03-features/12-im-integration")}>İstemciler ve kanallar <Icon name="arrow" /></a></div>
        </div>
        <div className={s.models}><span>Modeli siz seçin · 27 yerleşik sağlayıcı <a className={s.textLink} href={guide("03-features/06-models")}>Tümünü görüntüle <Icon name="arrow" /></a></span>{modelProviders.map(item => <p key={item.name}><IntegrationMark item={item} /></p>)}</div>
      </section>
      <section id="enterprise" className={s.enterprise} aria-labelledby="enterprise-title"><div className={s.shell}>
        <div className={s.sectionHeading}><div><p className={s.eyebrow}>06 / BUILT FOR YOUR TEAM</p><h2 id="enterprise-title">Şirket içinde dağıtın,<br />izinleri ekibinizin ihtiyacına göre yönetin.</h2></div><p>Veri depolamayı ve üye izinlerini yapılandırın;<br />işlem kayıtlarını ve görev çalışma durumunu görüntüleyin.</p></div>
        <div className={s.enterpriseGrid}><article><Icon name="server" /><h3>Dağıtım ve depolama</h3><p>Docker, Kubernetes ve Helm desteklenir; ayrıca tek makinelik Lite sürümü de sunulur. Modeller, vektör veritabanı ve depolama arka ucu ihtiyaca göre değiştirilebilir; yerel çıkarım desteklenir.</p></article><article><Icon name="shield" /><h3>Alan ve kaynak izinleri</h3><p>Çoklu alan yalıtımı ve dört seviyeli rol matrisi. API anahtarlarının kapsamı yetenek ve bilgi tabanına göre sınırlandırılır; OIDC kimlik entegrasyonu desteklenir ve hizmetin yalnızca izin listesindeki alan adlarına erişmesi sağlanabilir.</p></article><article><Icon name="trace" /><h3>Denetim ve çalışma izleme</h3><p>Alan denetim günlükleri, çalışma zamanı görev kuyruğu paneli ve Langfuse izleme, ekibinizin sorunları bulmasına ve çalışma durumunu yönetmesine yardımcı olur.</p></article></div>
      </div></section>
      <section id="get-started" className={`${s.shell} ${s.closing}`} aria-labelledby="closing-title">
        <div className={s.sectionHeading}><div><p className={s.eyebrow}>GET STARTED</p><h2 id="closing-title">Size uygun kullanım yolunu seçin.</h2></div></div>
        <div className={s.startGrid}>
          <article className={s.startCard}>
            <div className={s.startLabel}><Icon name="terminal" /><span>Tek makinede deneyin</span></div>
            <h3>Rethra Lite</h3>
            <p>SQLite ve bellek içi kuyrukla tek bir ikili dosya olarak çalıştırın; Redis veya vektör veritabanı gerekmez.</p>
            <a className={s.textLink} href={`${guide("01-getting-started/02-installation")}#lite-modu`}>Lite modunu görüntüle <Icon name="arrow" /></a>
          </article>
          <article className={s.startCard}>
            <div className={s.startLabel}><BrandLogo /><span>Kendiniz dağıtın</span></div>
            <h3>Kendi ortamınıza dağıtın</h3>
            <p>Docker veya Kubernetes ile dağıtın; modelleri, depolamayı ve ağı kendiniz yapılandırın.</p>
            <a className={s.textLink} href={guide("01-getting-started/02-installation")}>Dağıtım belgelerini görüntüle <Icon name="arrow" /></a>
          </article>
        </div>
      </section>
    </main>
    <footer className={`${s.shell} ${s.footer}`}><a className={s.brand} href="/" aria-label="Rethra ana sayfası"><BrandLogo /></a><p>Açık kaynak · MIT License</p><nav aria-label="Alt bilgi gezintisi"><a href={docs}>Dokümantasyon</a><a href={repo} target="_blank" rel="noreferrer">GitHub <Icon name="external" /></a><a href={`${repo}/blob/main/CHANGELOG.md`} target="_blank" rel="noreferrer">Değişiklik günlüğü</a></nav></footer>
  </div>;
}
