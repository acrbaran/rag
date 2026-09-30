# migrations/mysql — bağlanmamış MySQL tablo oluşturma betiği

⚠️ **Buradaki SQL hiçbir kod veya betik tarafından çalıştırılmaz. MySQL şu anda Rethra'nın ana veritabanı seçeneği değildir.**

## Mevcut durum

`internal/container/container.go` içindeki `initDatabase()` fonksiyonunda `DB_DRIVER` switch'inin yalnızca iki dalı vardır:

| `DB_DRIVER` | Taşıma kaynağı |
| --- | --- |
| `postgres` | `migrations/versioned/` (golang-migrate artımlı taşımalar) |
| `sqlite` | `migrations/sqlite/` (golang-migrate artımlı taşımalar, Lite modu) |

Diğer tüm değerler `unsupported database driver` döndürür; bu yüzden `DB_DRIVER=mysql` yapılandırması servisin başlamamasına yol açar.

Bu dizindeki `00-init-db.sql`, 10 çekirdek tabloyu (`tenants`, `models`, `knowledge_bases`, `knowledges`, `sessions`, `messages`, `message_suggestion_sets`, `message_suggestion_events`, `chunks`, `chunk_revisions`) kapsayan tek seferlik bir tablo oluşturma betiğidir. Biri bu temel tabloları değiştirdiğinde yan etki olarak güncellenir, ancak **hiçbir Go kodu, Makefile hedefi veya compose yapılandırması ona başvurmaz**.

## MySQL'i gerçekten desteklemek için eksikler

Yalnızca bu tablo oluşturma betiği yeterli değildir; yalnızca başlangıç şemasına karşılık gelir ve `migrations/versioned/` ile (Postgres tarafında 100'den fazla artımlı taşıma var) eşdeğer bir artımlı taşıma kümesi yoktur. Tam bağlantı için en azından şunlar gerekir:

1. `initDatabase()` içine `case "mysql"` eklemek, `gorm.io/driver/mysql` ve golang-migrate'in MySQL DSN'ini bağlamak;
2. `migrations/mysql/` için artımlı taşıma dizisi oluşturmak ve `versioned/` şema evrimiyle eşzamanlı tutmak;
3. Postgres'e özgü kullanımların (`JSONB`, dizi türleri, `ON CONFLICT`, ParadeDB BM25 indeksleri vb.) MySQL tarafındaki eşdeğer uygulamasını veya düşürülmüş çözümünü ele almak;
4. Vektör aramanın nasıl yapılacağını netleştirmek — MySQL'in kendisi vektör indeksi sunmaz, harici bir vektör deposuna (`RETRIEVE_DRIVER`) dayanmak gerekir.

İlgili tartışma için bkz. upstream #1418. Yukarıdaki çalışmalar tamamlanmadan belgelerde veya yapılandırma yorumlarında `DB_DRIVER=mysql` desteklendiğini iddia etmeyin.
