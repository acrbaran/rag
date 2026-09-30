# Rethra web sitesi ve dokümantasyon

`website-docs/` web sitesini, dokümantasyonu, ortak stilleri ve tüm derleme/dağıtım betiklerini içerir; depo dışındaki dosyalara bağlı olmadan bağımsız olarak kopyalanıp derlenebilir. Web sitesi ve dokümantasyon tek bir alan adını, tek bir derlemeyi ve tek bir dağıtım çıktısını paylaşır.

Ürün, dağıtım, API ve geliştirme dokümantasyonu burada tek yerden yönetilir. Eski `docs/` dizinindeki geçerli içerik konulara göre birleştirildi, güncelliğini yitirmiş ve tekrarlanan metinler silindi. Taşıma kararları ve hâlâ tutulan mühendislik kaynakları için bkz. [Taşıma kaydı](MIGRATION.md); bu kayıt yalnızca bakımcılar içindir ve sitede yayımlanmaz.

- `/`: Ürün ana sayfası.
- `/docs/`: Doğrudan "Hızlı başlangıç" sayfasına gider.
- `/docs/…`: Kategori gezinmesi, arama ve kenar çubuğu korunmuş tam dokümantasyon.

Site yalnızca bu iki yolu kullanır: kök dizinde yalnızca `index.html`, `404.html` ve `docs/` bulunur. Web sitesinin betikleri, stilleri ve görselleri `/docs/_home/` altına yerleştirilmiştir; bu nedenle üst ağ geçidinin yalnızca `/` (tam eşleşme) ve `/docs/` önekini iletmesi yeterlidir. `build`, bu iki yol dışındaki kaynaklara başvuran sayfaları reddeder.

Üst çubuk 64px yüksekliğinde ve yatayda tam genişliktedir. İki tarafta da README'deki özgün logo kullanılır; renkler, yazı tipleri ve açık/koyu tema tercihi ortaktır. Logo, aynı sekmede web sitesine döner. Web sitesi ve dokümantasyon, bulundukları sayfaya uygun kendi gezinmelerini kullanır.

## Yayınlama yöntemi: statik dosyalar + Nginx

### 1. Dağıtım paketini hazırlama

Node.js 24 kullanın ve komutları deponun `website-docs/` dizininde çalıştırın. İlk derlemede veya bağımlılık kilit dosyası değiştiğinde önce bağımlılıkları kurun:

```bash
cd website-docs
npm run setup
npm run build
npm run package:site
```

`build` web sitesini derler, dokümantasyon bağlantılarını ve Mermaid'i denetler, dokümantasyonu derler ve birleştirilmiş kaynak yollarını doğrular. Hepsi geçerse `static-site/` güncellenir; `package:site` bunu `releases/` içine paketler.

### 2. Yükleme ve açma

Sunucuya yalnızca sıkıştırılmış paketi yüklemeniz yeterlidir. Sunucuda Node.js'e, Rethra backend'ine veya ayrı bir dokümantasyon servisine gerek yoktur.

Eski site dosyalarının karışmaması için her yayında yeni ve boş bir dizin kullanın. Aşağıdaki `20260915-1` örnek bir yayın numarasıdır; sonraki yayınlarda başka bir numara kullanın:

```bash
sudo mkdir -p /srv/www/rethra/releases/20260915-1
sudo tar -xzf rethra-site-v0.8.2.tar.gz -C /srv/www/rethra/releases/20260915-1
```

Açtıktan sonra dizinde doğrudan `index.html`, `404.html` ve `docs/` bulunmalıdır; ayrıca bir `static-site/` katmanı olmamalıdır.

### 3. Alan adı kök dizinini yapılandırma

[deploy/nginx.conf](deploy/nginx.conf) dosyasını şablon olarak kullanıp site yapılandırmasını ekleyin ve şunu değiştirin:

```nginx
server_name alan-adiniz;
root /srv/www/rethra/releases/20260915-1;
```

Alan adında zaten HTTPS yapılandırması varsa mevcut sertifika ve dinleme ayarlarını koruyun; şablondaki `root`, `index` ve `location` kurallarını mevcut site yapılandırmasına ekleyin. Alan adı bu sunucuyu göstermelidir.

Dokümantasyonun `.html` yönlendirme çözümleme kuralları korunmalıdır; bilinmeyen tüm yolları web sitesinin `index.html` dosyasına geri düşürmeyin. Mevcut derleme alan adının kök yoluna dağıtılır; doğrudan `/rethra/` gibi alt dizinlere yerleştirmek desteklenmez.

Yapılandırmayı denetleyip yeniden yükleyin:

```bash
sudo nginx -t
sudo nginx -s reload
```

### 4. Yayında olduğunu doğrulama

Şu yolları ziyaret edin:

- `/`: Web sitesi.
- `/docs/`: Hızlı başlangıç.
- `/docs/03-features/23-memory`: Uzun süreli bellek dokümanı; doğrudan açma ve yenileme sorunsuz çalışmalıdır.
- `/docs/not-found`: 404 döndürür.

Ardından aramanın, açık/koyu tema geçişinin ve Logo ile web sitesine dönüşün düzgün çalıştığını doğrulayın.

Sonraki güncellemelerde derleme, paketleme ve yeni dizine yükleme adımlarını tekrarlayın, ardından Nginx `root` değerini değiştirip yeniden yükleyin. Geri almak gerektiğinde `root` değerini önceki yayın dizinine döndürün. Yeni sürümün kararlı olduğu doğrulandıktan sonra eski yayın dizinlerini temizleyin.

## İsteğe bağlı: Docker ile dağıtım

Temiz kaynak koddan doğrudan derlenebilir; ana makineye Node.js kurmak veya statik dosyaları önceden üretmek gerekmez. Dockerfile iki aşama kullanır: Node.js 24 iki kilit dosyasına göre bağımlılıkları kurup web sitesini ve dokümantasyonu derler; son Nginx imajı yalnızca statik çıktıları ve servis yapılandırmasını içerir.

Deponun kök dizininde çalıştırın:

```bash
docker build -t rethra-site:0.8.2 website-docs
docker run -d --name rethra-site --restart unless-stopped -p 8080:80 rethra-site:0.8.2
```

`http://sunucu-adresi:8080/` adresini ziyaret edin. Alan adı ve HTTPS kullanıyorsanız mevcut ters proxy'nin bu porta yönlendirmesi yeterlidir. İmaj web sitesini, dokümantasyonu ve Nginx yönlendirme yapılandırmasını zaten içerir.

Konteynerdeki Nginx varsayılan olarak 80 portunu dinler; bu, yeniden derlemeden `WEBSITE_NGINX_PORT` ortam değişkeniyle değiştirilebilir. `--network host` kullanırken veya dağıtım platformu konteynerin belirli bir portu dinlemesini istediğinde şöyle ayarlayın:

```bash
docker run -d --name rethra-site --restart unless-stopped -e WEBSITE_NGINX_PORT=8080 -p 8080:8080 rethra-site:0.8.2
```

Yalnızca dış portu değiştirmek istiyorsanız `-p` parametresinin sol tarafındaki ana makine portunu değiştirmeniz yeterlidir, ör. `-p 9000:80`.

Yalnızca `website-docs/` dizinini kopyalayıp bu dizinde `docker build -t rethra-site:0.8.2 .` komutunu da çalıştırabilirsiniz. Derleme bağlamı `website-docs/` olmalıdır; ana makinedeki bağımlılıklar, eski derleme çıktıları ve dağıtım paketleri `.dockerignore` ile hariç tutulur.

Eski dokümantasyon imajından geçerken konteyner port eşlemesini veya ters proxy hedef portunu `8081` yerine `80` yapın (ana makine portunu istediğiniz gibi seçebilirsiniz, ör. `-p 8081:80`). Alan adının kök yolu `/` artık web sitesini, `/docs/` ise dokümantasyonu sunar; ters proxy tüm siteyi kapsamalı ve istek yolunu korumalıdır. Konteyner, Nginx resmî imajının varsayılan giriş noktasını kullanır; ek bir `docker-entrypoint.sh` gerekmez.

Derlemeden sonra Node.js 24 ve Docker kurulu bir makinede npm bağımlılıklarını kurmadan aşağıdaki denetimleri çalıştırabilirsiniz. Denetim geçici olarak bir konteyner başlatır ve otomatik temizler; iki sitenin yönlendirmelerini, statik kaynakları, 404'ü, sıkıştırmayı ve yanıt başlıklarını doğrular:

```bash
cd website-docs
npm run test:docker -- rethra-site:0.8.2
```

## Yerel geliştirme

Node.js 24 LTS kullanın (dizinde `.nvmrc` bulunur; nvm kullanıyorsanız `nvm use` çalıştırabilirsiniz). Kaynak kodu ilk kez aldıktan sonra bağımlılıkları kurun (dokümantasyon ve web sitesinin kilit dosyalarına göre ayrı ayrı):

```bash
cd website-docs
npm run setup
npm run build
npm run preview
```

Önizleme adresi: `http://127.0.0.1:3000/`. Kaynak kodu değiştirdikten sonra yeniden derleyip sayfayı yenileyin; başka bir port için `npm run preview -- --port 8080` kullanılabilir.

```bash
npm run check
npm run test:integration
```

Entegrasyon denetimi için Google Chrome kurulu olmalıdır; tema senkronizasyonunu, iki taraftaki gezinmeyi, aramayı, mobil denetimleri ve dokümanlara doğrudan erişim yollarını doğrular. Doğrulanacak site `SITE_TEST_URL` ile belirtilebilir.

Geliştirme sırasında sıcak yenileme için `npm run dev:homepage` (web sitesi) ve `npm run dev:docs` (dokümantasyon) ayrı ayrı kullanılabilir; site içi geçişlerin tamamı için birleşik derlemeden sonra `npm run preview` kullanın.

`npm run build:docs` yalnızca dokümantasyonu, `npm run build:homepage` yalnızca web sitesini derler; resmî yayında `npm run build` kullanılır ve çıktı ikisini birden içerir. `VERSION`, bu site derlemesinin kullandığı yayın sürümüdür.

## Proje dizinleri (website-docs'a göre)

- `homepage/`: Next.js web sitesi kaynak kodu, marka materyalleri ve bağımsız bağımlılık kilit dosyası.
- `01-getting-started/` ile `06-development/` arası: Dokümantasyon metinleri, özgün yollar değişmedi.
- `.vitepress/`: Dokümantasyon sitesi yapılandırması ve teması.
- `public/`: Dokümantasyon ekran görüntüleri.
- `shared/`: Marka değişkenleri, üst çubuk stilleri ve ortak simgeler.
- `scripts/`: Birleşik derleme, denetim, önizleme ve paketleme.
- `static-site/`: Derlemeyle üretilen tek dağıtım dizini.
- `deploy/`, `Dockerfile`: Nginx ve Docker dağıtım yapılandırması.
- `releases/`: Üretilen dağıtım paketleri.

Marka materyallerinin kaynakları için bkz. [homepage/BRAND-ASSETS.md](homepage/BRAND-ASSETS.md). Ürün videosu README'deki GitHub özgün ekini kullanır ve oynatmak için GitHub'a erişim gerekir; sayfalar, dokümantasyon ve ekran görüntüleri paketle birlikte dağıtılır.
