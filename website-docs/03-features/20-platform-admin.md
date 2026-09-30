# Platform yönetimi ve sistem yöneticileri

Sistem yöneticileri, tüm Rethra dağıtımının genel ayarlarından, görev kuyruğundan, platform API Key'lerinden ve alanlar arası denetimden sorumludur. Alan Owner'ı tek bir çalışma alanından sorumludur; iki kimlik ayrı ayrı atanır. Alan rolleri için bkz. [Kiracı, kullanıcı ve kimlik doğrulama yetkilendirmesi](01-tenant-auth.md).

İki kimliğin yönetim kapsamları aşağıdaki gibidir:

| | Alan Owner'ı | Sistem yöneticisi |
| --- | --- | --- |
| Etki alanı | Tek çalışma alanı | Tüm dağıtım |
| Atama yöntemi | Alan oluşturulurken atanır veya önceki Owner tarafından devredilir | Mevcut bir sistem yöneticisi tarafından atanır; ilk yönetici ortam değişkeni aracılığıyla başlatılır |
| Yönetilen içerik | Alan üyeleri, modeller, bilgi tabanları, entegrasyonlar, alan denetimi | Genel sistem ayarları, görev kuyruğu, platform API Key'leri, alanlar arası denetim, kullanıcı parolalarını sıfırlama |
| Otomatik olarak eklenir mi | — | **Hayır**: Bir alanda Owner olmak sistem yöneticisi olmak anlamına gelmez; tersi de geçerlidir |

Alanlar arası veri erişimi `CanAccessAllTenants` tarafından ayrı olarak kontrol edilir. Sistem yöneticisi kimliği, diğer alanların bilgi tabanı içeriklerine erişim iznini otomatik olarak vermez.

<Screenshot
  src="/screenshots/settings-system-admin.png"
  caption="Platform konsolu: sistem ayarları, görev kuyruğu, platform API Anahtarı ve sistem denetim günlükleri"
  hint="Sistem yöneticisi olarak ‘Ayarlar’ı açın; kenar çubuğunun altında yalnızca sistem yöneticilerinin görebildiği dört bölümü gösterin. Ana içerik sistem ayarları sayfası olabilir." />

## İlk sistem yöneticisini belirleme {#ilk-sistem-yoneticisini-belirleme}

Yeni bir kurulumda önce ilk sistem yöneticisini ayarlamanız gerekir:

1. Önce normal süreçle bir hesap kaydedin;
2. app hizmeti için `RETHRA_BOOTSTRAP_SYSTEM_ADMIN_EMAIL=<bu_hesabın_e-postası>` yapılandırmasını ekleyin ve yeniden başlatın;
3. Başlatma sırasında, yalnızca kurulumda henüz sistem yöneticisi yoksa bu e-postaya karşılık gelen kullanıcı sistem yöneticisi yapılır.

Önyükleme süreci yalnızca mevcut hesapların yetkisini yükseltir. E-posta henüz kaydedilmemişse bir uyarı günlüğe yazılır ve sonraki yeniden başlatmalarda tekrar denetlenir; hatalı yapılandırma hizmetin başlatılmasını engellemez. Kurulumda zaten bir sistem yöneticisi bulunduğunda, bu değişken artık yetki vermez.

Daha sonra arayüzden sistem yöneticileri eklenebilir veya kaldırılabilir. Kendi yetkinizi ya da son sistem yöneticisinin yetkisini kaldıramazsınız. İlgili uç noktalar `POST /system/admin/promote` ve `/revoke` şeklindedir; yönetici olmayan birinin yetkisini tekrar kaldırma isteği 200 döndürür ve denetim kaydı, yetki değişikliği olmadığını `changed=false` ile belirtir.

## Platform konsolunu kullanma {#platform-konsolunu-kullanma}

Sistem yöneticileri, ‘Ayarlar’ kenar çubuğunda aşağıdaki yönetim bölümlerini görebilir:

| Bölüm | İşlev | Uç nokta |
| --- | --- | --- |
| Sistem ayarları | Genel çalışma zamanı anahtarları (kayıt modu, alan politikası, eşzamanlılık, SSRF izin listesi vb.); ayar öğesinin yürürlük kurallarına göre uygulanır, aşağıdaki ayar tablosuna bakın | `GET/PUT/DELETE /system/admin/settings[/:key]` |
| Model kataloğu | Katalog modellerini ve kaynaklarını görüntüleme, model düzenleme, ekleme veya JSON ile toplu değiştirme (kaydedince geçerli olur), sürüm geçmişinden geri yükleme; ayrıntılar için bkz. [Model kataloğu yönetimi](06-models.md#sistem-yoneticileri-model-katalogunu-yonetir) | `/system/admin/model-catalog*` |
| Görev kuyruğu | Asynq kuyruklarının gerçek zamanlı birikimini görüntüleme, tek tek görevleri yeniden deneme/arşivleme/silme, arşivlenmiş görevleri toplu temizleme; Lite modunda `available=false` döner | `/system/admin/runtime/queues*` |
| Platform API Anahtarı | Kontrol düzlemi otomasyonu için platform kapsamlı Anahtar; yetenekler: `system_tenants_read/manage`, `system_settings_read/manage`, `system_runtime_read/manage`, `system_audit_read` | `/system/admin/api-keys` |
| Sistem denetim günlüğü | `tenant_id = 0` olan platform düzeyindeki olaylar (ayar değiştirme, yönetici yetkisi verme/kaldırma, kuyruk işlemleri vb.). Alan düzeyindeki denetim uç noktaları tenant'a göre filtrelenir ve bu satırları göstermez | `GET /system/admin/audit-log` |

Sistem yöneticileri ayrıca aşağıdaki işlemleri gerçekleştirebilir:

- **Kullanıcı parolasını sıfırlama** (`POST /system/admin/users/reset-password`): Hedef kullanıcının yerel parolasını değiştirir ve tüm oturumlarını geçersiz kılar. **Kendi parolanızı sıfırlayamazsınız** — kullanıcıların kendi parolasını değiştirmesi hâlâ eski parolanın verilmesini gerektirir;
- **Varsayılan depolama kotasını toplu uygulama** (`POST /system/admin/tenants/apply-default-storage-quota`): Geçerli varsayılan kotayı mevcut tüm alanlara yazar. Bu işlem, mevcut alanların kota verilerini günceller.

### Kullanıcı oluşturma

v0.8.2'den itibaren sistem yöneticileri, ‘Ayarlar → Sistem ayarları → Hesaplar ve erişim’ bölümünde ‘Kullanıcı oluştur’a tıklayarak yerel hesap açabilir: Kullanıcı adını (2–50 karakter) ve e-postayı girin; ‘Rastgele parola otomatik oluştur’ seçeneğini açık tutun veya kapattıktan sonra parola politikasına uygun bir parolayı elle girin. Bu özellik, açık kayıt kapatıldıktan sonra yöneticilerin hesapları merkezi olarak açtığı senaryolar için uygundur. Alan ataması `auth.default_tenant_mode` kuralını izler: `create_personal` kişisel alan oluşturur, `tenantless` kullanıcının bir alana katılmasını bekler.

<Screenshot
  src="/screenshots/system-admin-create-user.png"
  caption="Sistem yöneticisi kullanıcı oluşturur: hesap bilgilerini girme ve tek kullanımlık parola gösterimi"
  hint="‘Ayarlar → Sistem ayarları → Hesaplar ve erişim’ altında ‘Kullanıcı oluştur’ iletişim kutusunu açın; kullanıcı adı, e-posta ve ‘Rastgele parola otomatik oluştur’ anahtarını gösterin. İsteğe bağlı olarak, oluşturma başarıyla tamamlandıktan sonra tek kullanımlık parolayı ve ‘Hesap bilgilerini kopyala’ düğmesini gösteren sonuç sayfasını da ekleyin." />

Otomatik oluşturulan parola yalnızca bu oluşturma sonucunda gösterilir; hesap bilgilerinin tamamı hemen kopyalanmalıdır. Onaylamadan önce iletişim kutusu kapatılamaz ve kapatıldıktan sonra açık metin parola tekrar sorgulanamaz. Her oluşturma işlemi sistem denetim günlüğüne yazılır (`system.user_created`). Yinelenen kimlik mevcut kullanıcıyı döndürür ve parolayı değiştirmez; e-posta ve kullanıcı adı farklı kullanıcılara işaret ediyorsa oluşturma reddedilir. Uç noktalar için [Sistem API'sine](../04-api/02-api-system.md) bakın.

Kullanıcı oluşturma ile ilk yönetici yönlendirmesi farklı işlemlerdir: bootstrap hâlâ yalnızca mevcut kullanıcıyı yükseltir, hesap oluşturmaktan sorumlu değildir.

## Çalışma zamanı ayarları başvurusu {#calisma-zamani-ayarlari-basvurusu}

Desteklenen çalışma zamanı ayarları konsolda değiştirilebilir. Veritabanındaki ayar değerleri ortam değişkenlerinden önceliklidir; çoğu hemen geçerli olur. Yeni kaynakları etkileyen veya çalışma zamanının yeniden oluşturulmasını gerektiren ayarlar için tablodaki açıklamalar esas alınır.

| Anahtar | Tür | Varsayılan | Geçerlilik zamanı |
| --- | --- | --- | --- |
| `auth.registration_mode` | `self_serve` / `invite_only` | `self_serve` | Hemen |
| `auth.default_tenant_mode` | `create_personal` / `tenantless` | `create_personal` | Yalnızca bundan sonra kaydolan yeni kullanıcıları etkiler |
| `tenant.self_service_creation_enabled` | bool | `true` | Hemen |
| `tenant.max_owned_per_user` | int | `10` (0 = yerleşik varsayılanı kullan, negatif sayı = kotayı devre dışı bırak) | Her alan oluşturulduğunda okunur |
| `tenant.default_storage_quota_gb` | int | `10` | **Yalnızca yeni alan oluşturulurken okunur**, mevcut alanlara geri yazılmaz |
| `tenant.auto_create_api_key` | bool | `false` | Her alan oluşturulduğunda okunur |
| `ssrf.whitelist` | Dize listesi | Boş | Hemen (`SSRF_WHITELIST_EXTRA` hâlâ yalnızca dağıtımı yapan tarafça yönetilir, burada geçersiz kılınmaz) |
| `asynq.core/postprocess/enrichment/maintenance/shared/wiki_concurrency` | int | [Eşzamansız görev sistemi](../02-architecture/05-async-tasks.md) bölümüne bakın | Her worker havuzu yeniden yapılandırılır |
| `model.max_concurrency` | int | `32` | Hemen |

Aşağıdaki anahtarlar da veritabanı önceliği kuralını izler:

| Anahtar | Varsayılan | Etki |
| --- | --- | --- |
| `auth.complex_password_enabled` | false | Yeni kayıtlar/yeni parolalar büyük ve küçük harf, sayı ve özel karakter gerektirir |
| `tenant.auto_accept_invitation` | false | E-posta davetiyle kayıtlı kullanıcılar için üyelik ilişkisini doğrudan yazar |
| `sandbox.docker_enabled` | false | Docker sanal alanı yapılandırmasını ve örnek oluşturmayı etkinleştirir |

`tenant.auto_create_api_key` varsayılan olarak kapalıdır. Eski kayıt yanıtına bağlı entegrasyonlar, yeni alanların otomatik olarak tam erişimli bir Key oluşturup düz metin olarak döndürmesini sağlamak için bu uyumluluk anahtarını etkinleştirebilir.

::: tip Çalışma zamanı ayarlarını sıfırlama
Konsolda ayar kaydedildikten sonra ilgili ortam değişkenini değiştirmek veritabanı değerini geçersiz kılmaz. Bir yapılandırma etkili olmadığında önce çalışma zamanı ayarlarını kontrol edin; sıfırlama (`DELETE /system/admin/settings/:key`) veritabanı geçersiz kılma değerini siler ve ortam değişkeninin veya yerleşik varsayılanın yeniden kullanılmasını sağlar.
:::

## İlgili belgeler {#ilgili-belgeler}

- Alan içindeki dört seviyeli roller ve API Key: [Kiracı, kullanıcı ve kimlik doğrulama/yetkilendirme](01-tenant-auth.md)
- Kuyruk topolojisi ve worker havuzları: [Eşzamansız görev sistemi](../02-architecture/05-async-tasks.md)
- Denetim günlükleri ve izleme: [Gözlemlenebilirlik ve denetim](16-observability.md)
- Arayüz listesi: [Sistem ve Platform Yönetimi API](../04-api/02-api-system.md)

## Uygulama referansı

- `cmd/server/bootstrap.go`: İlk sistem yöneticisinin başlangıç kurulumu.
- `frontend/src/config/settingsAccess.ts`: Platform ayarları bölümü.
- `internal/application/service/system_setting.go`: Çalışma zamanı ayar kayıt defteri.
