# Beceri kataloğu ve sandbox

Beceriler `SKILL.md` açıklaması, betikler, şablonlar ve belgelerden oluşur; sandbox betik yürütme ortamını sağlar. Beceriyi alan kataloğuna ekleyip sandbox'a kurduktan sonra, ajanda ilgili sandbox'ı ve beceriyi seçerek akıllı akıl yürütme sohbetlerinde kullanabilirsiniz.

## Becerileri yapılandırma ve kullanma {#becerileri-yapilandirma-ve-kullanma}

1. Alanın Admin/Owner rolündeki kullanıcı "Ayarlar → Sandbox yapılandırması" bölümünde bir yapılandırma oluşturur, Docker, CubeSandbox veya E2B'yi seçip bağlantı bilgilerini girer.
2. Sihirbazı izleyerek kümeye bağlanın ve şablon seçin; tüm zinciri doğrulamak gerekiyorsa "Tam doğrulama"yı çalıştırın. Tam doğrulama gerçekten bir sandbox oluşturur, yoklama çalıştırır ve temizler.
3. Kenar çubuğunda "Araç kutusu → Beceri yönetimi" bölümüne ZIP veya kaynak bağlantısı ekleyin, ardından kurulacak sandbox'ı seçin (eski "Ayarlar → Beceri yönetimi" bağlantısı otomatik olarak buraya yönlendirir). Kataloğa eklenmesi yalnızca kurulum paketinin kaydedildiğini gösterir; yürütme için kurulum durumunun hazır olması gerekir.
4. Ajan düzenleyicisinin "Beceriler" bölümünü açın, sandbox seçin ve beceri kapsamını tümü, belirli beceriler veya devre dışı olarak ayarlayın. Beceri yürütme akıllı akıl yürütme modunda kullanılır.
5. Sohbete girip soru sorun veya ajanın belirli bir beceriyi öncelikli kullanması için `@beceri` ipucunu verin. Dosya gerekiyorsa ek yükleyin; üretilen teslim dosyaları sohbetin sağındaki "Sandbox görselleştirme" panelinin "Çıktılar" sekmesinde önizlenip indirilebilir.

`@beceri`, ajanın yetkili diğer becerilere erişimini kaldırmaz. Sandbox seçilmemişse, alan betikleri kapatmışsa veya gereken arka uç yeteneği yoksa yürütme aracı kaydedilmez; istemle bu koşullar aşılamaz.

<Screenshot
  src="/screenshots/skill-catalog.png"
  caption="Alan beceri kataloğu: becerileri ve her sandbox'taki kurulum durumunu görüntüleme" />

## Kurulum ve güncellemeleri yönetme {#kurulum-ve-guncellemeleri-yonetme}

Alan kataloğu becerinin bir paketini saklar; her sandbox yapılandırmasının kendi kurulum kayıtları ve becerileri içeren imaj anlık görüntüsü vardır. Aynı beceri birden çok sandbox'a kurulabilir; kurulum ilerlemesi ve hata nedenleri ayrı ayrı kaydedilir.

| İşlem | Sonuç |
| --- | --- |
| Kataloğa ekleme | Paketi, adı, sürümü ve açıklamayı kaydeder; kurulum yapmaz |
| Kurulum | Seçilen sandbox'ta çalışma ortamını hazırlar, bağımlılıkları kurar ve yüklenebilirliği doğrular; başarılı olursa yeni anlık görüntü yayımlar |
| Yeniden deneme | O sandbox için kayıtlı paketi yeniden kurar, kurulum notu eklenebilir; aynı paket zaten hazırsa atlanabilir |
| Yükseltme | Katalog yeni sürümü kaydettikten sonra eski sürüm kurulumları yükseltilebilir olarak işaretlenir; yükseltme katalogdaki yeni sürümü seçilen sandbox'a kurar |
| Kurulumu durdurma | Devam eden kurulumu keser, durum failed olur; sonra yeniden denenebilir veya kaldırılabilir |
| Devre dışı bırakma | Kurulum kaydını korur, yalnızca ajanın bu beceriyi seçmesini engeller |
| Sandbox'tan kaldırma | O sandbox'ın beceri imajını günceller, katalogdaki kurulum paketini korur; diğer sandbox'lar kurmaya devam edebilir |
| Katalog girdisini silme | Hiçbir sandbox kurulumunun buna başvurmaması gerekir; tüm sandbox'lardan örtük olarak kaldırmaz |

Kurulum sayfası yüzde, aşama ve logları gösterir; ayrıntılı çalışma süreci kurulum kaydında görülebilir. İlerleme çekmecesini kapatmak veya ilerleme akışının kopması kurulumu durdurmaz. Canlı ilerleme yoksa beceri durumunu yenileyebilirsiniz; ayrıntılı olay loglarının süresinin dolması beceri paketinin kaybolduğu anlamına gelmez.

Kurulumu yerleşik kurulum ajanı yapar: `SKILL.md` dosyasını okuyup bağımlılıkları kurar ve becerinin ihtiyaç duyduğu harici komut satırı araçları gibi çalışma ön koşullarını kontrol eder. Gerekli bir komut eksikse veya çözülmemiş bir ön koşul varsa (ör. ayrıca dağıtılması gereken bir hizmet) kurulum başarısız sayılır ve neden belirtilir; yalnızca bağımlılık listesi olmadığı için hazır olarak işaretlenmez. Kurulum sürerken yöneticiler "Kurulum süreci" bölümüne kurulum notu ekleyebilir, örneğin kurulması gereken CLI, kurulum belgesi veya ortam kısıtları; kurulum bittikten sonra da notla yeniden kurulum yapılabilir.

Yükseltme veya yeniden deneme başarısız olursa sandbox bir önceki hazır sürümü sunmaya devam eder ve ajan bunu kullanabilir; yeni sürüm ancak başarılı kurulumdan sonra devreye girer.

`skill_rollout` imaj güncellemesini denetler: varsayılan `next_turn`, mevcut oturumlarda bir sonraki turda sandbox'ı yeniden oluşturur; `new_session` yeni imajı yalnızca bundan sonra sandbox oluşturan oturumlara uygular. Sandbox yeniden oluşturulunca eski örneğin geçici çalışma durumu kaybolur; teslim edilecek dosyalar `/workspace/output` dizinine yazılmalı ve sistem tarafından toplanmalıdır.

### Desteklenen kaynaklar {#desteklenen-kaynaklar}

| Girdi | Açıklama |
| --- | --- |
| ZIP dosyası | Beceri dizinini dışa aktarıp yükleyin |
| `@owner/slug`, `@owner/slug@1.2.0` | ClawHub'da yazar/sürüm belirtir |
| `slug`, `slug@1.2.0` | ClawHub slug |
| ClawHub, SkillHub, kendi barındırdığınız SkillHub sayfaları | İlgili kaynak üzerinden çözümlenir |
| GitHub/GitLab depo veya dizin URL'si | İlgili beceri paketini alır |
| `https://skills.sh/owner/repo/slug`, ClawHub'ın skills.sh sayfaları | Kurulum çözümleyicisiyle deponun belirli sürümüne ve dizinine çözümlenir |
| Doğrudan ZIP veya SKILL.md URL'si | Paketi veya giriş dosyasını indirir |

Kaynaklar anonim olarak okunabilir olmalıdır; indirme sırasında kullanıcının özel depo kimlik bilgileri gönderilmez. Özel beceriler önce ZIP olarak dışa aktarılabilir. `owner/slug` belirsizdir; `@owner/slug` veya tam URL kullanın.

Beceri paketi sınırı normal belgelerden bağımsızdır: `MAX_SKILL_BUNDLE_SIZE_MB` varsayılan olarak 256 MiB'dir, ayarlanmamışsa en az `MAX_FILE_SIZE_MB` kadardır, en fazla 512 MiB olabilir. GitHub indirmeleri yalnızca beceri alt dizinine göre değil, tüm depo arşivine göre hesaplanır. Değiştirdikten sonra app ve frontend'i yeniden başlatın ki uygulama ve Nginx sınırları eşleşsin.

## Sandbox arka ucunu seçme {#sandbox-arka-ucunu-secme}

| Arka uç | Doldurulacak alanlar | Çalışma biçimi |
| --- | --- | --- |
| Docker | İmaj; isteğe bağlı daemon adresi, TLS sertifika dizini, CPU/bellek/PID sınırları, ağ modu, runtime, boşta TTL | Her oturum için uzun ömürlü bir konteyner |
| CubeSandbox | Kontrol düzlemi adresi, veri düzlemi vekili, sandbox alan adı, şablon; kümeye göre API Key | Oturum düzeyinde uzak sandbox |
| E2B | API Key, şablon; kendi barındırıyorsanız API adresi, sandbox alan adı ve veri düzlemi vekili | E2B Cloud veya E2B uyumlu kontrol düzlemi |

`local` ana makine süreci arka ucu kaldırıldı: yerel makinede hiçbir yalıtım olmadan doğrudan çalışıyordu. Mevcut yapılandırmanın tüm alanları için bkz. [Sandbox ve beceri API'si](../04-api/02-api-sandbox-skills.md).

Docker arka ucu varsayılan olarak kapalıdır. Sistem yöneticisi "Sistem ayarları → Ağ güvenliği" bölümünden etkinleştirir; veritabanında kayıt yoksa `RETHRA_SANDBOX_DOCKER_ENABLED=true` geri dönüş olarak kullanılır. Yerel bağlantı için gerçek Docker socket'inin app'e bağlanması da gerekir; bu, app'e ana makinedeki Docker'ı yönetme yetkisi verir. Uzak TCP daemon için `ca.pem`, `cert.pem` ve `key.pem` içeren bir TLS sertifika dizini yapılandırılmalıdır. Docker ağı yalnızca `bridge` veya `none` kabul eder; isteğe bağlı olarak `runsc` gibi kurulu OCI runtime'ları seçilebilir.

Kendi barındırdığınız E2B/Cube için `proxy_url` veri düzlemi ağ geçidini gösterir: Rethra ağ geçidine bağlanır ama sandbox Host'unu korur; bu, joker alan adı DNS'i olmayan kümeler içindir. `allow_private_endpoints` özel ağ/loopback küme adreslerine bağlanmaya izin verir, link-local/bulut meta veri adreslerine yine izin vermez; sandbox içindeki betiklerin internete çıkıp çıkamayacağı ayrı bir ayardır.

Betikler varsayılan olarak sandbox içindeki `root` hesabıyla çalışır; şablondaki `user` hesabı açıkça seçilmelidir. Yürütme yalıtımını konteyner veya uzak sandbox sağlar; `/workspace` yalnızca çalışma dizini kuralıdır, root komutlarının dosya erişimini kısıtlamaz.

### Ağ politikası

Cube/E2B'nin `config.network` ayarı sohbet sandbox'larında, beceri kurulumunda ve tam doğrulamada birlikte kullanılır:

- Varsayılan olarak giden trafiğe izin verilir; `deny_egress_by_default=true` varsayılanı reddetmeye çevirir, ardından `allow_out` ile IP, CIDR veya alan adlarına izin verilir.
- `deny_out` IPv4/CIDR kabul eder; alan adı izin kuralları varsayılan reddetme ile birlikte kullanılmalıdır.
- Cube'un `cube_rules` ayarı Host/SNI, yöntem ve yola göre kural tanımlamaya, sıralamaya, denetim yapılandırmaya ve HTTPS başlık enjeksiyonuna izin verir; E2B'nin `e2b_host_rules` ayarı izin verilen alan adlarının istek başlığı enjeksiyonunu yapılandırır.
- Enjekte edilen kimlik bilgileri şifreli saklanır ve yanıtlarda maskelenir. Gelen trafik her zaman kimlik bilgisi gerektirir; eski `allow_public_inbound` alanı anonim gelen trafiğe izin vermez.
- Docker, çıkışı `docker.network_mode` ile denetler; Cube/E2B'nin ayrıntılı kuralları birebir uygulanamaz. Çıkış varsayılan olarak reddedildiğinde bağımlılık kurulumunun ihtiyaç duyduğu kaynak sunuculara da açıkça izin verilmelidir.

Mevcut örnekler politika değişikliğiyle yeni yapılandırmayı otomatik almaz; yeni sandbox oluşturduktan veya yeniden oluşturduktan sonra doğrulayın. Arka uç kimliğini değiştirmeden veya yapılandırmayı silmeden önce sistem çalışan/duraklatılmış örnekleri ve ilişkili ajanları sorgular; kullanımda ise işlem reddedilir ve ayarlar sayfası kullanım bilgisini gösterir.

## Ortam değişkenleri ve kimlik bilgileri

Kişisel değişkenler "Ayarlar → Sandbox anahtarları" bölümündedir; alan düzeyindeki beceri değişkenleri "Araç kutusu → Beceri yönetimi" altındaki beceri kartında yapılandırılır. Liste yalnızca değişken bildirimini, ayarlanıp ayarlanmadığını ve kaynağını gösterir; gizli değerleri geri göstermez.

| Katman | İşlev |
| --- | --- |
| Alan sandbox'ı `config.env_vars` | Bu yapılandırmayla oluşturulan sandbox'a enjekte edilir, betikler kullanır |
| Alan beceri değişkenleri | Yönetici, becerinin bildirdiği değişkenler için varsayılan değer girer |
| Kişisel sandbox değişkenleri | Kullanıcı bu sandbox yapılandırmasında komut çalıştırırken kullanılır |
| Kişisel beceri değişkenleri | Kullanıcı bu beceriyi çalıştırırken kullanılır |

Beceri değişkenleri çözümlenirken **kişisel beceri değeri > kişisel sandbox değeri > alan beceri değeri** önceliği geçerlidir; yalnızca ayarlanmamış adlar alt katmana düşer. Alan sandbox ortamı çalışma ortamının parçasıdır; betiklerin okumaması gereken gizli bilgiler buraya konmamalıdır. Kişisel geçersiz kılma silinince alt katmandaki değer yeniden kullanılır; beceriyi kapatmak kişisel kimlik bilgilerini silmez.

Kişisel beceri değişkenleri yalnızca becerinin bildirdiği adları kullanabilir; bunlar becerinin ihtiyaç duyduğu `RETHRA_*` kimlik bilgilerini içerebilir. Kişisel sandbox değişkenleri `RETHRA_*`, `PATH` gibi ayrılmış adları kabul etmez. Sistemin yürütülen komutlardan tespit ettiği değişkenler yalnızca ayarlanmamış kişisel değerleri tamamlar, mevcut kişisel veya alan yapılandırmasının üzerine yazmaz. Alanlar ve örnekler için bkz. [Kişisel değişken API'si](../04-api/02-api-sandbox-skills.md#kisisel-ortam-degiskenleri).

## Dosya üretme ve indirme {#dosya-uretme-ve-indirme}

`read_file(path="skill://<name>/SKILL.md")` açıklamayı okur; beceriyle gelen betikler `shell_exec(skill_name=..., command=...)` ile çalıştırılır. Komuttaki `$RETHRA_SKILL_DIR` becerinin gerçek kurulum dizinini gösterir; `skill://` bir okuma adresidir, doğrudan shell yolu olarak kullanılamaz.

Ekler `/workspace/input` dizinine geçici olarak konur, çalışma betikleri `/workspace` altında, indirilebilir çıktılar `/workspace/output` altında tutulur. Yeni dosya oluşturmak/devam ettirmek için `write_sandbox_file`, kısmi değiştirme için `edit_sandbox_file`, sayfalı okuma için `read_file` kullanın. Dosya aracı sınırları, çıktı bütçesi ve yeniden oluşturma davranışı için bkz. [Agent motoru](07-agent.md); teslim girişleri için bkz. [Oturum ve sohbet deneyimi](18-chat-experience.md).

<Screenshot
  src="/screenshots/skill-sandbox-chat.png"
  caption="Sandbox Word dosyası ürettikten sonra sohbette önizleme ve indirme" />

Komut çalışırken sohbetteki araç kartı çıktının son birkaç satırını canlı gösterir; böylece uzun komutların ilerlemesi izlenebilir. Bu ara çıktılar model bağlamına yazılmaz.

### Sandbox görselleştirme paneli

Uzak sandbox kullanan sohbetlerde sağdaki "Sandbox görselleştirme" panelinde üç sekme bulunur:

| Sekme | İçerik | Kullanım koşulu |
| --- | --- | --- |
| Çıktılar | Bu turda veya tüm turlarda üretilen indirilebilir dosyalar; dosya adına göre aranabilir | Tüm uzak sandbox'lar |
| Terminal | Bu oturumun sandbox'ına bağlanan etkileşimli Shell (xterm); sayfa yenilendikten sonra hâlâ çalışan terminale yeniden bağlanılabilir | Cube, E2B; Docker desteklenmez |
| Masaüstü | Tarayıcı üzerinden sandbox içindeki XFCE grafik masaüstünü kullanma; her oturumda aynı anda tek bağlantıya izin verilir | Cube, E2B masaüstü şablonları ve yapılandırmada `desktop_enabled` açık |

Paneli açmak sandbox oluşturmaz veya uyandırmaz: sandbox çalışıyorsa terminal ve masaüstü doğrudan bağlanır; sandbox duraklatılmışsa veya henüz oluşturulmamışsa "Terminali başlat", "Masaüstüne bağlan" veya "Oluştur ve başlat" düğmesine basmak gerekir; uyandırılan veya yeni oluşturulan sandbox alan yapılandırmasına göre ücretlendirilir. Terminal veya masaüstü uzun süre işlem görmezse bağlantı otomatik kesilir, sandbox ardından sağlayıcının TTL'ine göre duraklatılır; kesilme süresi sandbox yapılandırmasındaki "Terminal / masaüstü boşta kesilme (saniye)" ayarıyla belirlenir, varsayılan 900 saniye, aralık 60 saniye ile 24 saat arasıdır. Beceri güncellemesi sandbox'ın yeniden oluşturulmasını tetiklerse eski terminal ve masaüstündeki kaydedilmemiş içerik kaybolur.

<Screenshot
  src="/screenshots/sandbox-panel-terminal.png"
  caption="Sandbox görselleştirme paneli: sohbetin yanında çıktıları görüntüleme, terminal ve masaüstünü kullanma"
  hint="Cube veya E2B sandbox kullanan bir sohbeti gösterin; sağdaki 'Sandbox görselleştirme' paneli 'Terminal' sekmesine geçmiş olsun, terminalde renkli user@host istemi ve bir komut çıktısı görünsün; panelin üstünde 'Çıktılar / Terminal / Masaüstü' sekmeleri görünsün." />

## Dağıtım ve sorun giderme

Docker socket/TLS, şablon sürümleri, uzak ağ geçidi, çok kopyalı Redis, beceri anlık görüntülerinin disk kullanımı, terminal ve masaüstü aktarımı gereksinimleri için bkz. [Sandbox dağıtımı ve sorun giderme](../06-development/04-sandbox-deployment.md).

## Uygulama başvurusu

- `internal/handler/sandbox_config.go`, `sandbox_skill.go`, `skill_catalog.go`, `me_env_var.go`
- `internal/application/service/tenant_skill_install.go`, `tenant_skill_runtime_verify.go`, `tenant_skill_steer.go`, `user_env_resolver.go`
- `internal/types/tenant.go`, `sandbox_network_policy.go`
- `internal/sandbox/remote_client.go`, `docker_remote_client.go`, `gateway_transport.go`, `terminal.go`
