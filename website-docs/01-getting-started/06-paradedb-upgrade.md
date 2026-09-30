# ParadeDB mevcut veritabanı yükseltmesi

Deponun üretim ve geliştirme Compose yapılandırmaları `paradedb/paradedb:v0.22.6-pg17` kullanır. Bu belge **0.22.2 → 0.22.6, PostgreSQL ana sürümü 17 olarak kalacak şekilde** yükseltmeyi ele alır; PostgreSQL ana sürümleri arası yükseltmeler için geçerli değildir ve Helm içindeki diğer eski sürümlerin doğrudan uygulanabileceği anlamına gelmez.

## Yükseltme adımları

Tüm komutları depo kök dizininde çalıştırın. Geliştirme ortamındaki veritabanı komutları `docker compose -f docker-compose.dev.yml` kullanır ve geliştirme arka ucu ana makinede el ile durdurulur; geliştirme Compose yapılandırmasında `app` hizmeti yoktur. Mevcut veri birimini koruyun; **`down -v` çalıştırmayın, birimleri silmeyin veya PG18 imajına geçmeyin**.

1. app ve veritabanına yazan diğer bileşenleri durdurun; Langfuse etkinse web/worker bileşenlerini de durdurun. Yerelde çalışan geliştirme arka ucu da durdurulmalıdır.

   ```bash
   # Standart dağıtım; geliştirme ortamında ana makinedeki backend sürecini durdurun
   docker compose stop app
   # Yalnızca Langfuse etkinse çalıştırın
   docker compose stop langfuse-web langfuse-worker
   ```

2. Langfuse veritabanı dahil tüm veritabanlarını ve rolleri yedekleyin, ardından geri yükleme yapılabildiğini doğrulayın. Yedekleri veritabanı veri biriminin dışında saklayın.

   ```bash
   umask 077
   docker compose exec -T postgres sh -c 'pg_dumpall -U "$POSTGRES_USER"' > paradedb-before-upgrade.sql
   ```

   Eski imaj mevcut CPU üzerinde başlatılamıyorsa, önce durdurulmuş durumdaki veri biriminin anlık görüntüsünü kaydedin; uyumlu bir makinede mantıksal yedek oluşturun veya anlık görüntünün geri yüklenebildiğini doğrulayın, ardından tek veri kopyasını değiştirin.

3. Güncellenmiş Compose yapılandırmasını kullanın ve yalnızca veritabanı kapsayıcısını değiştirin:

   ```bash
   docker compose pull postgres
   docker compose up -d --no-deps --wait postgres
   ```

4. Uzantının SQL yükseltmesini tamamlayın. Yalnızca imajı değiştirmek, mevcut veritabanlarındaki `pg_extension` sürümünü güncellemez.

   ```bash
   docker compose exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1' <<'SQL'
   ALTER EXTENSION pg_search UPDATE TO '0.22.6';
   SELECT extname, extversion FROM pg_extension WHERE extname IN ('pg_search', 'vector');
   SELECT * FROM paradedb.version_info();
   SQL
   ```

   **`pg_search` yüklü diğer veritabanlarında** da aynı işlemi uygulayın; buna uzantının yüklenmiş olduğu `postgres`, şablon veritabanları veya Langfuse veritabanı dahildir. Yükseltme amacıyla, aslında gerekmeyen veritabanlarına uzantı yüklemeyin. Katalog ve `paradedb.version_info()` ikisi de 0.22.6 bildirmelidir.

5. Önceden çalışan yazma hizmetlerini geri başlatın, veritabanı geçiş durumunu doğrulayın ve temsil niteliğinde anahtar sözcük ile vektör aramaları yapın. Bu yama yükseltmesi, tüm dizinlerin özel olarak yeniden oluşturulmasını veya belgelerin yeniden içe aktarılmasını gerektirmez; doğrulama tamamlanana kadar yedekleri saklayın.

## Otomatik geçişin kapsamı

`000099` geçişi uzantı yükseltmesini yalnızca Rethra veritabanında gerçekleştirir: yüklü sürüm 0.22.2–0.22.5 olmalı ve sunucu 0.22.6 uzantı paketini sağlamalıdır. `app.skip_embedding` ayarına uyar, eksik uzantıları yüklemez ve diğer sürüm hatlarını işlemez.

Bu geçiş, veritabanı imajı değiştirilmeden **önce** zaten çalıştırılmışsa, yeni imaj yüklendikten sonra otomatik olarak yeniden çalışmaz; yukarıdaki SQL'i el ile çalıştırmanız gerekir. `000099` içermeyen eski uygulamalar da el ile yükseltilmelidir. Diğer veritabanlarının uzantıları Rethra geçişi tarafından yönetilemez.

## Geri alma ve yeniden üretim doğrulaması

Geri alma, yükseltme öncesi yedek/snapshot verisinin bağımsız bir veri birimine geri yüklenmesini ve eski imajla birlikte kullanılmasını gerektirir. Yalnızca imaj etiketini eski sürüme çevirmek, eklenti SQL değişikliklerini geri almaz; `000099` için down dosyası da eklentiyi düşürmeye çalışmaz.

Depo, yalıtılmış doğrulama betiği sağlar:

```bash
docker pull paradedb/paradedb:v0.22.2-pg17
docker pull paradedb/paradedb:v0.22.6-pg17
python3 scripts/test_paradedb_upgrade.py
```

Betik, geçici kapsayıcılar ve birimler kullanır, bağlantı noktası eşlemesi yapmaz; aynı veri birimindeki yükseltmeyi, tablo içeriğini, anahtar sözcük/vektör aramasını, geçiş idempotensini ve yeniden başlatma sonrası sonuçları doğrular; ayrıca yedekleri ve günlükleri çıktılar. Bu senaryo sabit test verilerini doğrular; dağıtım sırasında yine kendi verilerinizi ve arama yükünüzü denetlemelisiniz.
