# Belgeler website-docs'a taşındı

Ürün, dağıtım, API ve geliştirme belgeleri tek yerde, [website-docs](../website-docs/README.md) içinde tutulur. Taşınan, yinelenen ve güncelliğini yitiren eski el yazımı belgeler silindi; geçmiş içeriğe Git geçmişinden bakılabilir.

- [Sık sorulan sorular ve yükseltme sorunlarını giderme](../website-docs/01-getting-started/05-troubleshooting.md)
- [API başvurusu](../website-docs/04-api/01-api-overview.md)
- [Geliştirme kılavuzu](../website-docs/06-development/01-dev-guide.md)
- [Taşıma eşleme tablosu ve kalan bağımlılıklar](../website-docs/MIGRATION.md)

Bu dizinde yalnızca aşağıdaki mühendislik kaynakları ve bu giriş açıklaması tutulur:

- `docs.go`, `swagger.json`, `swagger.yaml` ve sözleşme testleri: backend derlemesine ve testlerine katılır; üretilen dosyalar hâlâ `make docs` ile güncellenir.
- `LITE.md`: Lite sürüm paketi tarafından çevrimdışı README olarak kopyalanır.
- `images/logo.png`: Helm chart ikonu ve tanıtım videosu logosu tarafından kullanılır.
- `poc/docker-sandbox/`: Bağımsız Go deney modülü; kaynak kodu ve çalıştırma açıklaması korunur, güncel ürün kılavuzu değildir.

Bu kaynakların hâlâ derleme, yayın veya geçmiş deney amaçları vardır; dizin toptan silinemez. Yeni ürün açıklamalarını buraya koymayın.
