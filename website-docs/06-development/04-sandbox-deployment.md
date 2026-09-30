# Korumalı ortam dağıtımı ve sorun giderme

Beceri kullanım akışı için [Beceri dizini ve korumalı ortam](../03-features/22-skills-sandbox.md), alan tanımları için [Korumalı ortam ve beceri API](../04-api/02-api-sandbox-skills.md) belgelerine bakın. Bu belge; dağıtım, şablonlar, masaüstü ve arka uç entegrasyonu için işletim gereksinimlerini tamamlar.

## Arka uç ve şablonlar

Alan adlandırma yapılandırması Docker, CubeSandbox veya E2B seçebilir; bu belge bu üç arka uç türünün dağıtımını açıklar. Eski `local` ana makine işlem arka ucu kaldırılmıştır. Docker doğrudan Engine API kullanır; E2B kontrol düzlemi REST ve envd veri düzlemini kullanır; Cube, şablonları ve ağ politikalarını işlemek için özel bir bağdaştırıcıyı korur.

Standart imaj `docker/Dockerfile.sandbox` tarafından tanımlanır; Python 3.12, Node.js 20, Bash, jq ve `/workspace` içerir. Komutlar ve dosya işlemleri varsayılan olarak korumalı ortam içindeki root kullanılarak yürütülür; hesapla açıkça yürütme için UID 1000 olan `user` korunur. Oturumlar arası yalıtım kapsayıcı veya uzak korumalı ortam tarafından sağlanır; çalışma dizini sözleşmesi root için dosya izni kısıtlaması olarak yorumlanamaz.

| Derleme targetı | İmaj etiketi son eki | Amaç |
| --- | --- | --- |
| `sandbox` | yok | Docker oturum imajı; E2B CLI şablonu temel imajı |
| `cube` | `-cube` | envd içeren Cube CLI şablonu |
| `desktop` | `-desktop` | E2B grafik masaüstü şablonu temel imajı |
| `desktop-cube` | `-desktop-cube` | envd içeren Cube grafik masaüstü şablonu |

İmaj adı `rethra-sandbox:<sürüm><son ek>` biçimindedir. Şablon ID'si belirli bir kümeye veya hesaba aittir; başka dağıtımların ID'si doğrudan kopyalanmamalıdır. Üretim şablonları doğrulanmış uygulama sürümüne karşılık gelmelidir; derleme targetı ve yayın mimarisi için depodaki Dockerfile ve yayın iş akışı esas alınmalıdır.

## Docker dağıtımı

Docker arka ucu varsayılan olarak kapalıdır. Sistem yöneticisi bunu «Sistem Ayarları → Ağ Güvenliği» bölümünden etkinleştirir; henüz veritabanına kaydedilmemişse `RETHRA_SANDBOX_DOCKER_ENABLED=true` ile geri dönüş yapılabilir. Kapatıldıktan sonra mevcut yapılandırmalar yine görüntülenebilir ve silinebilir, ancak yeni oturum kapsayıcıları oluşturulmaz.

Her yapılandırma bir daemon'a bağlanır ve her oturum uzun süre çalışan bir kapsayıcı kullanır. Ana makineler arası zamanlama yoktur; farklı app kopyaları kendi bağımsız yerel daemon'larına bağlanıyorsa kapsayıcıları ve beceri snapshot'larını paylaşacakları varsayılamaz.

| Yapılandırma | Varsayılan değer veya gereksinim |
| --- | --- |
| daemon adresi | Boş bırakıldığında `DOCKER_HOST` veya geçerli Docker context algılanır; uzak kullanım için `tcp://host:2376` |
| TLS sertifika dizini | Uzak TCP için zorunludur; app'in okuyabildiği dizin `ca.pem`, `cert.pem`, `key.pem` içermelidir |
| CPU / bellek / PID | Varsayılan 2 çekirdek / 2 GiB / 512 işlem |
| Boşta TTL | Varsayılan 1800 saniye |
| Ağ modu | Yalnızca `bridge` veya `none`; `host`, `container:` ya da özel ağ adı kabul edilmez |
| OCI runtime | daemon'da kurulu `runsc` gibi bir runtime seçilebilir |

app bir kapsayıcı içinde çalışıyorsa gerçek Docker socket'i bağlamalı veya uzak bir daemon'a bağlanmalıdır. Socket, ana makinedeki Docker'ı kontrol etme yetkisi verir; yalnızca güvenilir app'lere verilmelidir. Giriş betiği, yetkileri düşürmeden önce socket GID'sine göre appuser grubunu yapılandırır; socket'i `chmod 666` ile açmayın. Socket `root:root` ise ve yalnızca sahibi yazabiliyorsa, önce ana makinede root olmayan uygun grup izinlerini yapılandırın.

Öncelikle standart kalıpları kullanın. Özel kalıplar, bağdaştırıcının root, Bash, GNU `find -printf`, coreutils `timeout` ve yazılabilir çalışma alanı gereksinimlerini karşılamalıdır; oturum dosyası denetim noktaları ve geri alma ayrıca Git'e bağlıdır. HTTP isteğinin iptal edilmesi tek başına kapsayıcı işlemini sonlandırmaz; bağdaştırıcı, kapsayıcı içindeki `timeout` ile komut zaman aşımını denetler.

Boşta temizleme Create/Connect tarafından tetiklenir ve arka planda çalışır; kapsayıcı oluşturulurken kaydedilen TTL'ye göre karar verir; daemon'un kendi zamanlayıcısı değildir. Şu anda kesin bir yaşam süresi üst sınırı yoktur; komut çalıştırabilen betikler etkinlik işaretini sürekli güncelleyebilir, bu nedenle dağıtım tarafı uzun süre çalışan kapsayıcıları ayrıca izlemelidir.

### Beceri anlık görüntüleri ve disk

Beceri kurulumu, `docker commit` aracılığıyla `rethra-skill/` yerel kalıbını oluşturur ve sonraki oturumlar kurulum anlık görüntüsünden başlar. Dosya sistemini kaydeder, bellek durumunu kaydetmez; şu anda kiracılar arası paylaşılan birim veya Docker grafik masaüstü desteklenmemektedir.

Artımlı anlık görüntüler eski kalıp katmanlarını devralır; eski tag'in silinmesi diski mutlaka boşaltmaz. Becerinin kaldırılması, silme işaretli yeni bir katman ekler; özgün dosyalar üst katmanda kalabilir. Mevcut akış kalıpları otomatik olarak düzleştirmez veya daemon'lar arasında dağıtmaz. Katman sayısını ve diski izleyin, temizlemeden önce oturum ve anlık görüntü başvurularını doğrulayın; yeni bir temel kalıp gerektiğinde yeni bir yapılandırma oluşturup becerileri yeniden kurun.

## CubeSandbox / E2B entegrasyonu

1. Önce kullanılabilir bir denetim düzlemi, veri düzlemi ağ geçidi ve sandbox alan adı hazırlayın. Alan ayarlarında API, Proxy, domain ve kimlik bilgilerini doldurun; özel ağ/geri döngü uç noktaları özel ağ adresleri için açıkça izin gerektirir, bulut meta veri adreslerine yine izin verilmez.
2. "Bağlan ve devam et" seçeneğine tıklayın, denetim düzlemini doğruladıktan sonra şablon dizinini yükleyin. Bağlantının başarılı olması yalnızca denetim düzleminin kullanılabilir olduğunu kanıtlar.
3. Standart şablon eksikse açıkça oluşturun; masaüstü gerekliyse ayrıca bir masaüstü şablonu oluşturun. Seçmeden önce durumun `READY` olmasını bekleyin. Cube, envd içeren `-cube` varyantını kullanmalıdır; normal Docker kalıpları `:49983/health` sağlamaz.
4. "Tam doğrulama" işlemini çalıştırın; gerçekten bir sandbox oluşturur, yoklama çalıştırır ve yok eder; böylece veri düzlemi ile betik ortamının da kullanılabilir olduğunu doğrular.
5. Kaydettikten sonra aracıda bu yapılandırmayı seçin ve ek okuma, komut durumunun korunması ve çıktı indirmeyi doğrulayın.

`proxy_url`, kendi barındırılan veri düzlemi ağ geçidi için kullanılır: ağ geçidine bağlanırken sandbox Host yönlendirmesini korur ve joker karakterli DNS olmayan ortamlar için uygundur. Denetim düzlemine erişilebildiği hâlde çalıştırma başarısız oluyorsa Proxy, sandbox domain, gelen kimlik bilgileri ve uygulamadan ağ geçidine erişilebilirliği özellikle kontrol edin.

Cube guest DNS, şablon yapılandırmasına aittir. DNS/kalıp değiştirildikten sonra yeni ortama yansıması için şablon yeniden oluşturulmalıdır. Becerileri zaten kurulmuş yapılandırmalar temel kalıbı değiştiremez veya yeniden oluşturamaz; ayrıca CLI/masaüstü arasında geçiş yapamaz. Anlık görüntü ile temel kalıbın tutarsızlaşmasını önlemek için yeni bir yapılandırma oluşturup becerileri kurun.

Çoklu kopyalar, session→sandbox bağını ve masaüstü durumunu kaydetmek için paylaşılan Redis yapılandırmalıdır. Yalnızca tek örnekli geliştirmede bellek içi depolama kullanılabilir. Mevcut sandbox'lar arka uç yapılandırma değişikliklerini otomatik olarak uygulamaz; beceri güncellemelerinin `next_turn` / `new_session` stratejileri için bkz. [Beceri dizini ve sandbox](../03-features/22-skills-sandbox.md).

| Belirti | İnceleme yönü |
| --- | --- |
| Bağlantı doğrulaması başarısız | API'nin Dashboard adresi olarak yanlış girilip girilmediği; kimlik bilgileri, TLS ve özel ağ anahtarının doğru olup olmadığı |
| Bağlantı başarılı ancak çalıştırma başarısız | Proxy, sandbox domain, ağ geçidi yönlendirmesi ve gelen token |
| Şablon derlemesi başarısız | Kümenin döndürdüğü derleme hatasını inceleyin; kalıbın çekilip çekilemediği, mimarinin eşleşip eşleşmediği, Cube kalıbının envd içerip içermediği |
| Sandbox dış ağa çıkamıyor | Şablon ağ politikası, guest DNS, küme çıkış vekili; özel ağ denetim düzlemine izin vermek betiklerin dış ağa çıkmasına izin vermek değildir |
| Bağımlılık kurulumu başarısız | Varsayılan olarak çıkış reddedildiğinde yazılım kaynaklarına izin verilip verilmediği; beceri kurulumu ve oturum aynı ağ politikasını kullanır |
| Yeniden bağlanmadan sonra durum kayboluyor | Redis'in paylaşılıp paylaşılmadığı, TTL'nin dolup dolmadığı, beceri güncellemesinin yeniden oluşturmayı tetikleyip tetiklemediği |

## Etkileşimli terminal ve grafik masaüstü {#etkilesimli-terminal-ve-grafik-masaustu}

Sohbet kenar çubuğundaki terminal ve masaüstü, WebSocket aracılığıyla oturum sandbox'ına bağlanır; yalnızca Cube/E2B destekler, Docker arka ucu sağlamaz. Tarayıcılar WebSocket el sıkışmasında kimlik doğrulama başlığı taşıyamadığından, önce oturum açılmış POST ile iki dakika geçerli kısa süreli bir bilet alınır, ardından bilet el sıkışma query'sine eklenir:

| Yetenek | Bilet alma | WebSocket |
| --- | --- | --- |
| Terminal | `POST /api/v1/sessions/:session_id/sandbox/terminal-ticket` | `GET /api/v1/sessions/:id/sandbox/terminal?ticket=...` |
| Masaüstü | `POST /api/v1/sessions/:session_id/sandbox/desktop-ticket` | `GET /api/v1/sessions/:id/sandbox/desktop?ticket=...` |

Giriş proxy'si bu iki yol için WebSocket Upgrade'i iletmeli, okuma zaman aşımını gevşetmeli ve erişim günlüklerine ticket query'sini yazmamalıdır. Standart frontend Nginx, `^/api/v1/sessions/[^/]+/sandbox/(terminal|desktop)$` için query içermeyen bir günlük biçimi yapılandırmıştır; özel Ingress kullanıyorsanız bunu kendiniz ayarlamalısınız. Terminal bağlandıktan sonra sunucu yaklaşık dakikada bir oturum açma durumunu, alan üyeliğini ve oturum sahipliğini yeniden denetler; oturumu kapatmak veya alandan çıkarılmak terminal bağlantısını keser. Arayüz parametreleri için bkz. [Terminal API](../04-api/02-api-sandbox-skills.md#oturum-etkilesimli-terminali) ve [Masaüstü API](../04-api/02-api-chat.md#sandbox-desktop).

Terminal ve masaüstü, işlem yapılmadığında yapılandırılmış `terminal_idle_disconnect_sec` süresi sonunda bağlantıyı keser (varsayılan 900 saniye, en az 60 saniye, en fazla 24 saat); ardından sandbox sağlayıcının TTL'sine göre duraklatılır. Terminal, klavye girdisi ve PTY çıktısını etkinlik olarak sayar; masaüstü ise klavye ve fareyi etkinlik olarak sayar.

### Grafik masaüstü

Yalnızca Cube/E2B masaüstü şablonları sohbet kenar çubuğu masaüstünü destekler. İlk açılışta masaüstü işlemi arka uç tarafından başlatılır; yapılandırmada masaüstü şablonu seçilmeli ve `desktop_enabled` ayarlanmalıdır. Beceri yüklendikten sonra temel imaj yerinde değiştirilemez.

```text
Tarayıcı noVNC → Rethra bilet aktarıcısı → sağlayıcı ağ geçidi → websockify :6080 → yerel x11vnc :5900
```

Tarayıcı sandbox API Key'ini, gelen token'ı veya websockify parolasını tutmaz. Masaüstü bileti yalnızca bir kez kullanılabilir; tam tanım için [Oturum API](../04-api/02-api-chat.md#sandbox-desktop) belgesine bakın. Cube masaüstü şablonunun `exposedPorts` ayarı yalnızca envd'nin 49983 bağlantı noktasını açar, **6080'i ana makine NAT'ına eklemeyin**; masaüstü ağ geçidi ve Rethra aktarması üzerinden geçmelidir.

Her oturumda aynı anda yalnızca bir masaüstü aktarmasına izin verilir; çoklu kopya yuvaları Redis tarafından koordine edilir. Sandbox yeniden oluşturulduktan sonra bağlantının kesildiğini bildirmek için `SANDBOX_REBUILT` kullanılır; yeni masaüstü, eski geçici dosyaları koruyan eski örnek olarak kabul edilemez. Boşta olma tespiti RFB klavye ve fare etkinliğini kullanır; ekran görüntüsü istekleri kullanıcı işlemi sayılmaz.

## Geliştirme doğrulaması

Birim testleri gerçek bir daemon gerektirmez:

```bash
go test ./internal/sandbox -run 'TestDocker' -count=1
```

Gerçek Docker doğrulaması önce standart imajın oluşturulmasını gerektirir ve test kapsayıcıları oluşturur:

```bash
docker build -f docker/Dockerfile.sandbox --target sandbox -t rethra-sandbox:dev .
DOCKER_INTEGRATION_IMAGE=rethra-sandbox:dev \
go test -tags=docker_integration ./internal/sandbox \
  -run '^TestDocker.*Integration' -count=1 -v -timeout=15m
```

Uzak bağlantı tutarlılık testleri `internal/sandbox/cube_integration_test.go` ve `e2b_compatible_integration_test.go` içinde bulunur; çalıştırmadan önce dosyaların başındaki yönergelere göre test kümesi kimlik bilgilerini yapılandırın. Doğrulama kapsamı oturum durumunun korunmasını, Shell yeniden kullanımını, eklerin geçici olarak saklanmasını, çıktıların toplanmasını ve zaman aşımını içermelidir; yalnızca Health testi yapılmamalıdır. Test tamamlandıktan sonra test örneklerinin temizlendiğini doğrulayın.
