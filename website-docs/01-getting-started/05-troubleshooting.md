# Sık Sorulan Sorular ve Yükseltme Sorun Giderme

Önce "Ayarlar → Sistem Bilgileri" bölümünden uygulama sürümünü, veritabanı geçiş durumunu ve bağımlılık bağlantı durumunu doğrulayın; ardından ilgili hizmet günlüklerini inceleyin. Aşağıdaki Compose komutları depo kök dizininde çalıştırılır; yerel geliştirme bağımlılık hizmetleri `docker compose -f docker-compose.dev.yml` kullanır, app günlükleri ana makine terminalinden veya günlük dosyasından görüntülenir ve geliştirme Compose dosyasında `app` hizmeti yoktur.

```bash
docker compose ps
docker compose logs --tail=200 app
docker compose logs --tail=200 docreader
docker compose logs --tail=200 postgres redis
```

`.env` dosyasını değiştirdikten sonra ilgili kapsayıcıyı yeniden oluşturmak için `docker compose up -d <hizmet_adı>` kullanın; `docker compose restart`, kapsayıcı ortam değişkenlerini yeniden yüklemez.

## Belirtiye göre tespit

| Belirti | Kontrol sırası ve açıklama |
| --- | --- |
| Yükleme başarısız, ayrıştırma sürekli işleniyor | Belge ayrıntılarındaki hata nedeni ve ayrıştırma izlemesi → ayrıştırma motoru bağlantısı → arka plan görev kuyruğu; bkz. [belge ayrıştırma](../03-features/03-document-parsing.md) ve [eşzamansız görevler](../02-architecture/05-async-tasks.md) |
| Görseller yerel makinede görüntüleniyor, diğer cihazlarda açılmıyor | Dönen adresin `localhost`, kapsayıcı hizmet adı veya erişilemeyen nesne depolama adresi içerip içermediğini kontrol edin; üçüncü taraf istemciler `resource://` aldığında bunu [dosya erişimi](../03-features/21-file-access.md) uyarınca çözümler, doğrudan görsel URL'si olarak kullanamaz |
| Yapılandırma kaydedildikten sonra eski değer geri geliyor | Proxy/tarayıcı önbelleğinin yanıtı değiştirip değiştirmediğini, başka bir ortama bağlanılıp bağlanılmadığını veya alan değiştirilip değiştirilmediğini kontrol edin; yerleşik modeller için ayrıca YAML başlangıç eşitlemesini inceleyin; bkz. [model yönetimi](../03-features/06-models.md) |
| Yükseltmeden sonra yetersiz izin uyarısı | Geçerli alan rolünü, kaynak sahipliğini ve API Key yetki kapsamını doğrulayın; bkz. [kimlik doğrulama ve yetkilendirme](../03-features/01-tenant-auth.md). Platform yöneticisi ile alan sahibi aynı kavram değildir |
| Agent, modelin hazır olmadığını bildiriyor | Bu ajanın başvurduğu modelin hâlâ mevcut ve yapılandırmasının eksiksiz olduğunu doğrulayın, model bağlantı testini çalıştırın; bkz. [model yönetimi](../03-features/06-models.md) |
| Özel ağ veri kaynağı, model veya vektör veritabanı bağlantısı reddediliyor | SSRF doğrulamasını ve bağlantı noktası politikasını kontrol edin; [yapılandırma başvurusu](04-configuration.md) uyarınca yalnızca gerekli hedeflere izin verin |
| Hata "yalnızca izin listeli çıkışlara izin verilir" diyor | Dağıtımda `SSRF_DNS_WHITELIST_ONLY` etkinleştirilmiştir; izin listesinde olmayan ana makineler DNS çözümlemesinden önce reddedilir. Hedef ana makineyi `SSRF_WHITELIST` veya `SSRF_WHITELIST_EXTRA` içine ekledikten sonra app / docreader kapsayıcılarını yeniden oluşturun; bkz. [yapılandırma ayrıntıları](04-configuration.md) |
| Giriş arayüzü HTML 404 döndürüyor | `SEARXNG_PORT` değerinin `APP_PORT` (varsayılan 8080) ile aynı olup olmadığını kontrol edin. Linux üzerinde SearXNG, `127.0.0.1:8080` adresine bağlandıktan sonra `localhost:8080` istekleri önce SearXNG'ye gider. SearXNG'nin 8888'i veya başka boş bir bağlantı noktasını kullanmasını sağlayın |
| MinIO imajı çekilirken `pull access denied` veya `401 UNAUTHORIZED` uyarısı | MinIO, Docker Hub ve quay.io için anonim çekmeyi resmî olarak kapattı. Güncel Compose ve Helm, topluluk tarafından oluşturulan `pgsty/minio` imajını kullanır; özel düzenleme dosyalarında imaj adresi de eşzamanlı değiştirilmelidir |
| Yükseltmeden sonra gömülü sayfa veya imzalı bağlantı geçersiz (gömülü sayfada `embed session signing key is not configured` uyarısı, başlangıç günlüğünde `[startup-env] no usable signing key` var) | İmzalama anahtarı `SYSTEM_SIGNING_KEY` değerinden alınır; ayarlanmamışsa `SYSTEM_AES_KEY` değerine geri dönülür. Örnek anahtarlar ve uzunluğu 16'dan az anahtarlar imzalama yapamaz. `openssl rand -hex 32` ile oluşturup `SYSTEM_SIGNING_KEY` olarak yapılandırın; **bunun için `SYSTEM_AES_KEY` değerini değiştirmeyin**, aksi halde kaydedilmiş kimlik bilgileri çözülemez. v0.8.2, gömülü oturumların imzalama algoritmasını değiştirdiğinden yükseltme öncesi oluşturulan tüm gömülü oturumlar geçersizdir ve ziyaretçilerin yeniden girmesi gerekir; çoklu kopya dağıtımları aynı anahtarı kullanmalıdır |
| Yükseltmeden sonra DingTalk robotu artık yanıt vermiyor | v0.8.2'den itibaren DingTalk yalnızca Stream modunu destekler; yükseltme geçişi `webhook` kanalını `websocket` olarak değiştirir. DingTalk geliştirici panelinde bu uygulama için Stream modunu etkinleştirin; bkz. [IM entegrasyonu](../03-features/12-im-integration.md) |
| Arka plan kuyruğu sürekli birikiyor | En eski görevin bekleme süresi, etkin worker sayısı ve alt hizmet kota bilgilerini birlikte değerlendirerek tespit edin; worker sayısını artırmak model sağlayıcısı kotasını artırmaz, bkz. [kapasite tahmini](../02-architecture/05-async-tasks.md#capacity-planning) |
| Beceri dizine eklendi ancak çalıştırılamıyor | Dizin kaydı ve sandbox kurulumu iki ayrı adımdır; kurulum kaydını, ajanın sandbox seçimini ve beceri kapsamını kontrol edin; bkz. [beceri dizini ve sandbox](../03-features/22-skills-sandbox.md) |
| Yükseltmeden sonra Local sandbox bulunamıyor | `local` arka ucu kaldırıldı; Docker, CubeSandbox veya E2B'yi yeniden yapılandırın; bkz. [sandbox dağıtımı ve sorun giderme](../06-development/04-sandbox-deployment.md) |
| Yerel tarayıcı eşleştirilmiş ancak web görevi yürütülmüyor | Akıllı çıkarım giriş kutusunda yerel tarayıcı etkinleştirilmelidir; duraklatılmış görevler açıkça sürdürülmelidir, bkz. [yerel tarayıcı](../05-clients/09-local-browser.md) |
| Gömülü sayfa yüklenemiyor veya 403 dönüyor | İzin listesine gerçek ana makine Origin değerini girin; CSP'yi, ters proxy'yi ve güvenlik modu exchange işleminin Origin değerini kontrol edin; bkz. [gömülü kanal](../03-features/13-embed-channel.md) |
| Feishu uygulama testi başarılı ancak klasör yüklenemiyor | Bağlantı testi yalnızca uygulama kimliğini doğrular, klasör için ayrıca yetkilendirme gerekir; bkz. [Feishu bulut sürücüsü bağlantısı](../03-features/24-feishu-drive.md) |

## Veritabanı geçişi başarısız {#database-migrations}

Uygulama varsayılan olarak başlatılırken geçişleri çalıştırır. Başarısızlıktan sonra yine de başlatılabilir; bu nedenle "sayfanın açılması", tablo yapısının başarıyla yükseltildiği anlamına gelmez. Sistem bilgisindeki geçiş sürümü, dirty işareti ve hata ile app başlatma günlükleri, inceleme için başlangıç noktalarıdır.

**Başarısız geçişin tamamen geri alındığını varsaymayın.** Gerçek durum, SQL'in işlem sınırlarına ve başarısızlık konumuna bağlıdır. Kurtarmadan önce günlükleri saklayın, veritabanını yedekleyin, başarısız sürüme karşılık gelen geçiş dosyasını ve gerçek schema'yı doğrulayın; hatayı atlamak için veri birimini silmeyin veya sürüm numarasını doğrudan değiştirmeyin.

PostgreSQL `migrations/versioned/` kullanır, SQLite ise `migrations/sqlite/` kullanır. Aşağıdaki Make komutları PostgreSQL geçiş betiklerini çağırır; Lite'ın SQLite veritabanı, ilgili sürücü ve geçiş diziniyle işlenmelidir, PostgreSQL DSN uygulanamaz.

```bash
make migrate-version
```

Hedef veritabanında salt okunur denetim de yapabilirsiniz:

```sql
SELECT version, dirty FROM schema_migrations;
-- Aşağıdaki iki komut yalnızca PostgreSQL içindir
SELECT version();
SELECT extname, extversion FROM pg_extension;
```

### Eksik eklenti veya yetersiz izin

`gin_trgm_ops` bulunmaması genellikle `pg_trgm` ile ilişkilidir, `vector` türünün bulunmaması pgvector ile ilişkilidir, BM25 işlevi ParadeDB'nin `pg_search` eklentisine bağlıdır. Önce uygulamanın gerçekten bağlandığı veritabanını doğrulayın, ardından veritabanı yöneticisinden eklenti paketlerini, veritabanındaki eklentileri ve nesne sahipliğini denetlemesini isteyin.

```sql
SELECT name, default_version, installed_version
FROM pg_available_extensions
WHERE name IN ('pg_trgm', 'vector', 'pg_search');
```

Eklenti dosyalarının kurulu olmaması ile yetersiz SQL izinleri farklı sorunlardır; `CREATE EXTENSION IF NOT EXISTS` işletim sistemi paketlerini otomatik olarak kurmaz ve mevcut eklentileri yükseltmez. Yalnızca mevcut dağıtımın gerçekten ihtiyaç duyduğu eklentileri oluşturun; harici arama motoru dağıtımını yanlışlıkla PostgreSQL aramasına dönüştürmekten kaçının.

ParadeDB mevcut veritabanlarının yansı ve eklenti sürümü yükseltmeleri için bkz. [ParadeDB Yükseltme](06-paradedb-upgrade.md).

### Dirty durumu

`AUTO_RECOVER_DIRTY` varsayılan olarak etkindir; başlatma sırasında geçiş sürümünü sıfırlamayı ve yeniden çalıştırmayı dener. Bu bir yeniden deneme mekanizmasıdır; eksik eklentileri, yetersiz disk alanını veya elle yapılan değişikliklerin neden olduğu schema farklarını düzeltemez.

Elle kurtarma sırasında önce uygulama yazmalarını durdurun ve başarısız geçişin kısmi değişiklik bırakıp bırakmadığını inceleyin. Yalnızca veritabanının önceki başarılı sürüme uygun olduğunu ve başarısız geçişin güvenle yeniden çalıştırılabileceğini doğruladıktan sonra `force` kullanın. Bu işlem **yalnızca geçiş sürüm işaretini değiştirir, geri alma SQL'ini çalıştırmaz**. Örneğin, önceki başarılı sürümün 98 olduğu doğrulandıktan sonra:

```bash
make migrate-force version=98
make migrate-up
```

Buradaki 98 bir örnektir; gerçekten doğrulanmış sürümle değiştirilmelidir, hata numarasını mekanik olarak bir azaltmayın. İlk geçiş başarısız olursa başlatma durumunu ayrıca denetleyin. Kurtarmadan sonra app'i yeniden başlatın, sistem bilgisinin artık hata göstermediğini doğrulayın ve etkilenen işlevleri test edin.

### Eşzamanlı dizin oluşturmanın kesintiye uğraması

Geçiş `000106`, `messages` için `CREATE INDEX CONCURRENTLY` ile dizin oluşturur ve oluşturma sırasında yazmaları engellemez. Oluşturma kesintiye uğrarsa (işlem yeniden başlatma, zaman aşımı, yetersiz disk alanı), INVALID bir `idx_messages_session_created_id` kalır; `IF NOT EXISTS` bunu yeniden oluşturmaz. Önce bu dizini doğrulayıp silin, ardından yukarıdaki şekilde geçiş durumunu kurtardıktan sonra yeniden çalıştırın:

```sql
SELECT indexrelid::regclass, indisvalid FROM pg_index
WHERE indexrelid = 'idx_messages_session_created_id'::regclass;
DROP INDEX CONCURRENTLY IF EXISTS idx_messages_session_created_id;
```

### Yetersiz disk alanı ve Schema farkları

Dizin oluşturma ek geçici alan gerektirir. `No space left on device` ile karşılaşırsanız veritabanı veri birimi ve geçici dizin kapasitesini denetleyin, alan temizlendikten sonra gerçek geçiş durumuna göre kurtarma yapın.

Sütun türü, dizin veya kısıtlamalar geçiş beklentisiyle uyuşmadığında, başarısız dosyayı ve önceki başarılı sürümü karşılaştırarak denetleyin; üretim veritabanında hata veren SQL'i doğrudan tekrar tekrar çalıştırmayın. Yazma işlemlerini yeniden üretmeniz gerekiyorsa, önce yedekten geri yüklenen yalıtılmış bir veritabanında doğrulayın.

## Sorun bildirirken sağlanacak bilgiler

Uygulama sürümünü/commit numarasını, başarısızlık zamanını, tam hatayı, veritabanı türü ve sürümünü, geçiş sürümü/dirty durumunu ve ilgili varsayılan dışı yapılandırmaları sağlayın. Önce günlüklerdeki Token, parola, bağlantı dizesi ve belge içeriğini gizleyin. Veritabanı geçiş uygulaması ve geliştirme yöntemi için bkz. [Veritabanı ve Geçişler](../06-development/02-database-schema.md).
