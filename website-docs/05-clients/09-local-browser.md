# Yerel tarayıcı bağlantısı ve dağıtımı

Rethra, BrowserSkill daemon ve [BrowserSkill](https://github.com/Tencent/BrowserSkill) tarayıcı uzantısı aracılığıyla kullanıcının bilgisayarındaki tarayıcıyı akıllı çıkarım sohbetine bağlar. Uzantı Chrome ve Microsoft Edge'i destekler (Chromium 125 ve üzeri tabanlı); diğer Chromium tarayıcılarla uyumluluk garanti edilmez. Tarayıcı kullanıcının bilgisayarında, daemon ise Rethra app tarafında çalışır; beceri sanal alanı gerekmez.

## Uzantıyı yükleme

Uzantı, **0.3.1 ve üzeri** sürüm gerektirir; aşağıdaki yöntemlerden birini seçerek yükleyin:

- Chrome: [Chrome Web Mağazası](https://chromewebstore.google.com/detail/hhcmgoofomhgciiibhipgmgkgnoenaoi)
- Edge: [Edge Eklentileri](https://microsoftedge.microsoft.com/addons/detail/browserskill/emacgiaaaiojkkpkddmmdfhmokgmnikg)
- Eşlik eden ZIP: Mağazaya erişemiyorsanız, "Tarayıcı bağlantısı" sayfasındaki "Manuel yükleme (yedek)" bölümünden indirin. Arşivi çıkardıktan sonra `chrome://extensions` içinde (Edge için `edge://extensions`) "Geliştirici modu"nu etkinleştirin ve "Paketlenmemiş uzantıyı yükle"yi seçin.

Bağlı uzantının sürümü 0.3.1'den düşükse, "Tarayıcı bağlantısı" sayfası yükseltme uyarısı gösterir.

## Eşleştirme ve kullanım

1. Kenar çubuğunda "Araç kutusu → Tarayıcı bağlantısı"nı açın. Eski "Ayarlar → Tarayıcı bağlantısı" bağlantısı otomatik olarak buraya yönlendirilir.
2. Sayfanın oluşturduğu tek kullanımlık eşleştirme bağlantısını kopyalayın, uzantıda "Bağlantı ayarları → Uzak bağlantı" bölümüne yapıştırıp kaydedin.
3. Akıllı çıkarım giriş alanında "Yerel tarayıcı"yı etkinleştirin, ardından görevi gönderin. Eşleştirmenin başarılı olması, her turda otomatik olarak etkinleştirileceği anlamına gelmez.
4. İlk çağrı ayrı bir görev penceresi oluşturur; sohbet önizlemesi sayfayı bulabilir, mevcut tarayıcı görevini duraklatabilir, sürdürebilir veya sonlandırabilir.

<Screenshot
  src="/screenshots/browser-connection.png"
  caption="Araç kutusu → Tarayıcı bağlantısı: eşleştirme bağlantısı, bağlantı durumu ve uzantı yükleme girişleri"
  hint="Eşleştirilmiş ve çevrimiçi durum: sekmedeki bağlantı durumu, cihaz bilgileri, eşleştirme bağlantısı alanı, Chrome/Edge mağaza girişleri ile "Manuel yükleme (yedek)" ve "Kenar çubuğunda bağlantı durumunu göster" anahtarları; kenar çubuğunda Araç kutusunun yanında yeşil bir durum noktası görünür." />

Bağlantı durumu "Tarayıcı bağlantısı" sekmesinde gösterilir (bağlı, çevrimdışı veya eşleştirilmemiş). Eşleştirildiğinde, kenar çubuğunda "Araç kutusu"nun sağındaki küçük tarayıcı simgesi de durumu bir noktayla belirtir: yeşil bağlı, turuncu çevrimdışı demektir. Gerekmiyorsa, "Tarayıcı bağlantısı" sayfasında "Kenar çubuğunda bağlantı durumunu göster" seçeneğini kapatabilirsiniz; bu ayar yalnızca mevcut tarayıcıda saklanır. Aynı sayfada, tercih edilen arama motorunu ve arama adresini belirtmek için "Tarayıcı arama komutu" da ayarlanabilir.

Yetkilendirmeler alan ve kullanıcı bazında saklanır; aynı alandaki birden fazla sohbet cihaz yetkilendirmesini paylaşır, ancak görevleri ayrıdır. Şu anda her alan + kullanıcı için bir cihaz yetkilendirmesi tutulur; yeni bir cihazı etkinleştirmek eskisinin yerini alır.

Eşleştirme bağlantısı beş dakika geçerlidir. Cihaz belirteci 90 gün geçerlidir ve 30 gün sonra otomatik olarak döndürülür; yeni bir bağlantı oluşturmak eski cihazın bağlantısını hemen kesmez, yetkilendirme ancak başarıyla etkinleştirildikten sonra değiştirilir. Cihazı iptal etmek, kullanıcıyı devre dışı bırakmak veya alan üyesini kaldırmak sonraki kullanımı engeller.

### İnsan katılımı ve görev kurtarma

Görev, kendisinin oluşturduğu sekmeleri yönetebilir; görev penceresinde tıklama veya tuş vuruşuyla açılan ve kaynak doğrulamasından geçen yeni sekmeler de yönetilebilir ve görev sona erdiğinde korunur. Ayrı açılır pencereler ve kullanıcının mevcut sekmeleri `tab_borrow` ile ödünç alınır; tarayıcı ayarlarına göre yetkilendirme onaylandıktan sonra kullanılır ve tamamlandığında `tab_return` ile iade edilir. Görev sona erdiğinde, görevin açıkça oluşturduğu sayfalar kapatılır ve ödünç alınan sayfalar iade edilir.

Bir sekmeyi görev penceresine elle sürüklemek yetkilendirme anlamına gelmez; yetkilendirilmemiş bir sayfa zaten görev penceresindeyse, kullanıcının önce onu normal pencereye taşıması, ardından ödünç vermesi gerekir. Ödünç alma onay istemi, normal penceredeki bir HTTP(S) sayfasında gösterilmelidir; uzantı ayar sayfaları, yeni sekme sayfaları ve bağımsız açılır pencereler bunu barındıramaz. Ödünç alma sonrasında özgün pencere kaybolursa, iade işlemi normal bir yedek pencere oluşturabilir; iade edilen sayfalar görev bittiğinde kapatılmaz.

Giriş, doğrulama kodu veya yetkilendirme adımları `request_help` tarafından başlatılır. Önizleme istemine göre tarayıcıya gidin; işlemi tamamladıktan sonra tarayıcıdaki yardım isteminde onaylayın. Ardından Agent sayfayı yeniden gözlemler ve devam eder. Sohbette yalnızca "lütfen giriş yap" demek insan devralma istemi oluşturmaz. Yardım bekleme süresi en fazla beş dakikadır; zaman aşımı veya iptal, mevcut durumu korur ve görevi duraklatır.

Duraklatma, mevcut tarayıcı çağrısını iptal eder ancak eşleştirmeyi kaldırmaz; Devam'a tıkladıktan sonra Agent bu turu zaten bitirmişse, devam komutunu ayrıca göndermek gerekir. Hata veya hizmet yeniden başlatması sonrasındaki görevler tıklama, gönderme gibi eylemleri otomatik olarak yeniden oynatmaz; kurtarma sonrasında önce sayfayı kontrol edin.

Normal başarılı görevler otomatik olarak sona erer; kullanıcı sayfanın korunmasını isterse, insan yardımı varsa veya görev bir hatayla kesilirse mevcut durum korunabilir. Önizlemede "Son görüntü korundu" yazması, artık yoklama yapılmadığını belirtir; tarayıcı kontrolünün bırakıldığı anlamına gelmez. Tarayıcı görevini sonlandırmak, tüm sohbeti durdurmakla aynı değildir.

## Tek örnekli dağıtım

Birlikte sunulan app imajı daemon, uzantı ZIP'i ve lisansı içerir ve dosya yollarını önceden ayarlar; birlikte sunulan frontend Nginx, WebSocket'i işler. Güncelleme sırasında app, frontend ve kullanıcının uzantıları birlikte güncellenir.

Yerel dağıtımda depo kök dizininde derleyin:

```bash
./scripts/build_browserskill.sh
```

Git, Node.js, Python 3, Rust/Cargo ve bir C derleyicisi gerekir; Linux'ta ayrıca CMake gerekir. Sabit kaynak kodu sürümü `scripts/browserskill-release.json` tarafından yönetilir, ek yama uygulanmadan doğrudan üst kaynak kodu derlenir ve çıktılar `artifacts/browserskill/` dizinine yazılır. İlgili işletim sistemi ve mimariye uygun daemon'u kullanın ve gerçek kurulum yoluna göre yapılandırın:

```dotenv
BROWSERSKILL_BINARY=/opt/rethra/browserskill/bsk
BROWSERSKILL_EXTENSION_PATH=/opt/rethra/browserskill/browser-skill-rethra-0.3.1.zip
BROWSERSKILL_MAX_CONNECTIONS=32
```

Varsayılan olarak kullanıcının geçerli sayfa origin'inden eşleme WSS adresi üretilir ve bağlantı noktası korunur. Yalnızca ayrı bir ağ geçidi alan adı veya özel yol ile dağıtım yapıldığında geçersiz kılma ayarlayın:

```dotenv
BROWSERSKILL_PUBLIC_URL=wss://rethra.example.com/api/v1/local-browser/extension
```

Yerelde localhost WS kullanılabilir; uzakta tarayıcının güvendiği bir WSS sertifikası gerekir. İç ağda güvenilir bir kurumsal CA kullanılabilir. `BROWSERSKILL_BINARY=` açıkça ayarlanırsa yetenek devre dışı bırakılabilir; Docker ortam değişkenlerini değiştirdikten sonra kapsayıcıları yeniden oluşturmak için `docker compose up -d app frontend` kullanın.

Birlikte sunulan uzantı ve daemon birlikte yükseltilmelidir; uzantının 0.3.1 veya üzeri olması gerekir (Chrome Web Mağazası, Edge Eklentileri ya da birlikte sunulan ZIP kullanılabilir). Özgün çıkarılmış dizinin üzerine yazıp yeniden yüklemek uzantı ID'sini korur; yeniden kurulum ID değişikliğine neden olursa yeniden eşleme gerekir. Dağıtım sırasında `BrowserSkill-LICENSE` dosyasını koruyun.

## Çoklu kopya ve giriş proxy'si

Tüm app'ler veritabanını paylaşır; cihaz yetkilendirmesi, bağlantı kiraları ve kesinti işaretleri veritabanına kalıcı olarak yazılır. Her app gerektiğinde paylaşılan daemon'u başlatır; diğer kopyalar, imzalı dahili RPC üzerinden görevleri uzantı bağlantısını tutan düğüme iletir.

```dotenv
# Her kopyada farklıdır; doğrudan o düğüme ulaşmalıdır, yük dengeleyici adresi yazılamaz
BROWSERSKILL_INTERNAL_URL=http://10.0.0.12:8080
# Tüm kopyalarda aynıdır; Secret üzerinden en az 32 karakterlik rastgele bir değer enjekte edin
BROWSERSKILL_CLUSTER_SECRET=<shared-random-secret>
```

Kubernetes, doğrudan bağlantı adreslerini oluşturmak için Pod IP'lerini kullanabilir. Paylaşılan veritabanı yerine her kopya için ayrı SQLite dosyaları kullanmayın. Dahili HTTP yalnızca güvenilir ve yalıtılmış ağlar için uygundur; aktarım gizliliği gerektiğinde HTTPS ve güvenilir sertifikalar kullanın.

Giriş proxy'si şunları yapmalıdır:

- `/api/v1/local-browser/extension` için WebSocket Upgrade'i ve `/authorize` POST isteğini iletmelidir.
- `/api/v1/local-browser/internal` yoluna genel girişten erişimi engellemeli, yalnızca app düğümlerinin birbirleriyle erişimine izin vermelidir. Birlikte sunulan frontend bu yolu zaten reddeder; app doğrudan açığa çıkarıldığında veya özel Ingress kullanıldığında da bunu uygulayın.
- Authorization, Sec-WebSocket-Protocol ve yetkilendirme isteği gövdesinin kaydedilmesini önlemelidir.

Bağlantı kirası 45 saniyedir ve her on saniyede yenilenir; düğüm çöktükten sonraki kurtarma, eski kiranın sona ermesinden ve uzantının yeniden bağlanma beklemesinden etkilenir. Hizmet geri geldikten sonra görev durumunu kontrol edin; sonucu bilinmeyen işlemleri otomatik olarak yeniden oynatmayın.

## Sorun giderme ve doğrulama

| Belirti | Kontrol yönü |
| --- | --- |
| Birlikte sunulan uzantı indirilemiyor veya yetenek kullanılamıyor | daemon ve uzantı dosya yolları, çalıştırma izni, hedef mimari ve app imajı sürümü |
| Uzantı sürümünün çok düşük olduğu bildiriliyor | Chrome Web Mağazası, Edge Eklentileri veya birlikte sunulan ZIP ile 0.3.1 ya da üzerine yükseltin; özgün çıkarılmış dizinin üzerine yazdıktan sonra yeniden yükleyin |
| Yetkilendirme başarılı ancak WSS başarısız | Genel ağ adresi/bağlantı noktası, güvenilir sertifika, dış proxy'nin WebSocket Upgrade'i |
| Cihaz bağlantısı normal ancak görev çalışmıyor | Bu turdaki anahtarlar, görevin duraklatılıp duraklatılmadığı, sekme ödünç alma veya insan yardımı bekleyip beklemediği |
| İnsan yardımı istemi yok | Çalıştırma kaydının `request_help` çağırıp çağırmadığı; uzantıda insan yardımının devre dışı olup olmadığı |
| Kopya değiştirildikten sonra başarısız oluyor | Veritabanının paylaşılıp paylaşılmadığı, düğüm doğrudan bağlantı adresinin doğruluğu, paylaşılan anahtar ve ağ politikası |
| Önizleme takılıyor | Sayfanın gizli olup olmadığı, görevin son görüntüyü koruyup korumadığı, uzantıdaki belirli CDP zaman aşımı; önizleme video değil, düşük kare hızlı ekran görüntüleridir |

Varsayılan 32, tek düğümdeki etkin cihazlar için koruma üst sınırıdır; aktarım kapasitesi garantisi değildir. Canlıya alırken hedef kümede eşleme, yeniden bağlanma, iptal, insan yardımı ve kopyalar arası yönlendirmeyi doğrulayın; daemon CPU, RSS, ekran görüntüsü trafiği, RPC gecikmesi ve yeniden bağlanma süresini gözlemleyin. Gerçek uzantı test giriş noktası `internal/browserskill/extension_test.go` olup ayrı bir test tarayıcısı profil dizini kullanır.
