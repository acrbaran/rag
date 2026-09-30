# Feishu bulut sürücüsü entegrasyonu

`feishu_drive` / `lark_drive`, belirtilen klasördeki belgeleri ve dosyaları bilgi tabanına eşitler. Wiki alanları için `feishu` / `lark` bağlayıcısını kullanın; genel eşitleme akışı için bkz. [veri kaynağı eşitleme](10-datasource.md).

## Uygulama kimliği ve klasör izinleri

İlgili bölgedeki kurumsal özel uygulamanın App ID ve App Secret bilgilerini kullanın; Feishu ve Lark kimlik bilgileri birlikte kullanılamaz. Gerçekte kullanılan arayüzlere göre okuma izinleri açılmalıdır: dizin listeleme ve dosya indirme, bulut belgesi dışa aktarma ve blocks modunda docx içerik okuma. İzin adları, alternatif izinler ve başvuru gereksinimleri için Feishu açık platformundaki [klasör listeleme arayüzü](https://open.feishu.cn/document/server-docs/docs/drive-v1/folder/list) ve [dışa aktarma arayüzü](https://open.feishu.cn/document/server-docs/docs/drive-v1/export_task/create) gibi arayüz açıklamaları esas alınmalıdır.

İzinleri yapılandırdıktan sonra uygulama sürümünü yayımlayın ve hedef klasörü uygulamanın erişebileceği kapsama yetkilendirin. Uygulamanın bulunduğu grup üzerinden paylaşım yapılıyorsa uygulama, klasör yöneticisi ve grup arasındaki yetkilendirme ilişkisini doğrulayın. API izni ve dosya erişim izni iki ayrı denetim katmanıdır: tenant access token alınabilmesi, herhangi bir klasörün okunabileceği anlamına gelmez.

Rethra içindeki bağlantı testi uygulama kimlik bilgilerini doğrular; klasör erişimi ise kaynak ağacı yüklendiğinde doğrulanır. Bağlantı başarılı olduğu halde dizin 403 hatası veriyorsa önce paylaşım kapsamını ve izin yayımlama durumunu kontrol edin.

## Bilgi tabanında yapılandırma

1. Bilgi tabanı ayarları → Veri kaynakları → Yeni oluştur seçeneğine gidin, ardından «Feishu bulut sürücüsü» veya «Lark Drive» seçin.
2. App ID ve App Secret bilgilerini girin, bağlantı testini çalıştırın.
3. Belirli klasörün `folder_token` değerini veya tam `/drive/folder/<token>` bağlantısını girin, kaynak ağacını yükleyin ve eşitleme kapsamını seçin. Bulut alanı kök dizinini kullanmak için boş bırakılamaz.
4. Tam artımlı eşitlemeyi, eşitleme planını, çakışma stratejisini ve eşitleme ile silmeyi seçin; kaydettikten sonra önce az sayıda dosyayı elle eşitleyin, günlükleri ve ayrıştırma sonuçlarını kontrol edin.

Artımlı eşitleme, önceki imleci geçerli dosya ağacıyla karşılaştırır: yalnızca seçili kaynaklar eksiksiz listelendiğinde kaybolan dosyalar silinmiş olarak işaretlenir; alt dizin listeleme başarısız olduğunda algılama ertelenir. Eşitleme ile silme etkinleştirildiğinde, buna göre yalnızca **geçerli veri kaynağına karşılık gelen bilgi öğeleri** silinir; devre dışı olduğunda korunurlar. İlk eşitleme veya önceki imleç taşımayan tam eşitleme, geçmiş silmeleri bu yolla algılayamaz; silme başarısızlıklarının tek bir tam eşitlemeyle mutlaka tamamlanacağı da garanti edilmez. Eşitleme günlüklerini ve kalan öğeleri kontrol edin.

Belgenin kendi güncellenmesi, blocks modunda artık mevcut olmayan ek alt öğelerini de temizler. Bu, içerik güncellemesinin bir parçasıdır ve yukarıdaki kaynak belge eşitleme ile silme anahtarından etkilenmez.

## Dosyalar ve ayrıştırma modları

Normal dosyalar indirildikten sonra dosya türüne göre ayrıştırma akışına girer; tablolar ve çok boyutlu tablolar xlsx olarak dışa aktarılır, klasörler özyinelemeli olarak taranır, kısayollar hedef dosyaya çözümlenir. Desteklenmeyen bulut belgesi türleri atlanır; atlama nedenini eşitleme günlüklerinde kontrol edin.

Yeni docx sürümünde iki yol vardır; bunlar **app hizmeti**ndeki `FEISHU_DOCX_PARSE_MODE` tarafından kontrol edilir ve hem Wiki hem de bulut sürücüsü bağlayıcılarını etkiler:

| Mod | Yol | Uygunluk ve kısıtlamalar |
| --- | --- | --- |
| Boş veya `export` (varsayılan) | Docx'i eşzamansız olarak dışa aktarır, ardından belge ayrıştırmasına gönderir | Görseller üst belgeyle birlikte ayrıştırılır; dışa aktarma ve ayrıştırma daha uzun sürer, gömülü file block ekleri dışa aktarmayla korunmaz |
| `blocks` | blocks API ile Markdown'a dönüştürür; API başarısız olduğunda veya gövde boş olduğunda dışa aktarmaya geri döner | Ayrıştırılabilir ekleri bağımsız öğeler olarak korur; çok modlu eşitleme etkin olduğunda görseller bağımsız öğeler olur, aramanın onları otomatik olarak gövde metniyle ilişkilendireceği varsayılamaz |

blocks modu gerektiğinde `.env` içinde ayarlayın ve app'i yeniden oluşturun:

```dotenv
FEISHU_DOCX_PARSE_MODE=blocks
```

```bash
docker compose up -d app
```

Mod değişikliği sonraki alımları etkiler; depolanmış içeriği otomatik olarak başka bir biçime dönüştürmez. Görsel içeriğin aranabilir olup olmadığı ayrıca nesne depolama, OCR/çok modlu işleme ve dizin durumuna bağlıdır; bkz. [Belge ayrıştırma](03-document-parsing.md).

## Sık sorulan sorular

| Belirti | İşlem |
| --- | --- |
| Token geçersiz veya kimlik bilgisi testi başarısız | Feishu/Lark bölgesini, App ID/Secret'i ve özel proxy adresini kontrol edin |
| Klasör yüklenemedi | Belirli bir klasör bağlantısı kullanın; API izinlerini ve klasör paylaşım yetkisini kontrol edin |
| Bulut belgesi başarısız ancak normal dosyalar çalışıyor | Dışa aktarma iznini kontrol edin; blocks modu ayrıca docx okuma izni gerektirir |
| Senkronizasyon sayısı dizindeki dosya sayısından az | Desteklenmeyen türleri, alt dizin yetkilerini, başarısız/atlanan istatistiklerini kontrol edin |
| Görseller metinle birlikte getirilemiyor | Ayrıştırma modunu ve çok modlu durumu doğrulayın; blocks içindeki bağımsız görseller ile export iç satır içi görsellerin anlamları farklıdır |
| Belge değişikliğinden sonra kısa süreyle aranamaz | Güncelleme eski bilgiyi silebilir ve yeniden depolayabilir; yeni sürümün ayrıştırılması ve dizinlenmesinin tamamlanmasını bekleyin |

Uygulama referansı: `internal/datasource/connector/feishu/drive/`, `core/shared.go` ve `internal/application/service/datasource_service.go`.
