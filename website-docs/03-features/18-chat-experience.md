# Oturum ve diyalog deneyimi

Oturumda, bilgi tabanı veya yüklenen ekler hakkında sorular sorabilir, yanıt kaynaklarını görüntüleyebilir ve ajanın oluşturduğu dosyaları önizleyebilirsiniz. Yanıt oluşturulurken ek gereksinimler eklemeye devam edebilirsiniz; mevcut diyaloglarda herhangi bir mesajdan dallanabilir veya geri alabilirsiniz. Soru dizini ve geçmiş araması diyalogları gözden geçirmek için kullanılır; Markdown olarak kopyalama ise eksiksiz soru-cevap kaydını saklamanızı sağlar.

## Yanıtları ve kaynakları görüntüleme {#yanitlari-ve-kaynaklari-goruntuleme}

Soru sorulduktan sonra ilerleme çubuğu; ek ayrıştırma, arama, araç çağrıları ve yanıt oluşturma gibi aşamaları gösterir. Akıllı çıkarım modu ayrıca daraltılabilir düşünme sürecini, komut çıktısını ve dosya yazma ilerlemesini gösterir.

Yanıt metnindeki kaynak üst simgeleri, özgün metin parçalarını bulmak için kullanılır; sağdaki kaynak paneli, Wiki aracının döndürdüğü içerikler dahil bu turda kullanılan arama kaynaklarını listeler. Ajanın kaynak üst simgesi ayarı kapatıldıktan sonra da kaynakları kaynak panelinden görüntüleyebilirsiniz. Yanıt tamamlandığında sistem takip sorusu önerileri sunabilir; ayrıntılar için bkz. [Ajan motoru](07-agent.md).

Bilgi tabanı kaynak üst simgesine tıkladığınızda, kaynak paneli doğrudan alıntılanan özgün dosyayı açar, alıntının bulunduğu konuma gider ve vurgular:

| Özgün format | Konumlandırma ve vurgulama |
| --- | --- |
| PDF (metin tabanlı) | İlgili sayfaya gider, alıntılanan paragrafı çerçeveler |
| PDF (taranmış belge) | İlgili sayfaya gider; MinerU / PaddleOCR-VL ile ayrıştırıldığında paragrafı çerçeveler |
| Word | Alıntılanan paragrafa veya tabloya kaydırır |
| PPT | Alıntılanan slayta kaydırır ve metni vurgular |
| Excel / CSV | Alıntılanan satırı vurgular |
| Markdown / TXT / EPUB | Alıntılanan metne kaydırır (EPUB önce bölümü konumlandırır) |
| Ses | Alıntılanan parçanın zamanından itibaren oynatır (ses tanıma modeli, whisper-1 gibi bölüm zamanlarını döndürmelidir) |
| Web sayfası içe aktarma | Alıntılanan parçayı gösterir ve parçaya göre özgün web sayfasını açmak için bir giriş sunar |

Vurgulama alanı, yanıtta alıntılanan cümleye göre en ilgili paragrafa daraltılır. Bu özellik kullanıma sunulmadan önce bilgi tabanına eklenen belgelerde konum bilgisi yoktur; özgün metinde metin aranır, bulunamazsa uyarı gösterilir ve belgenin başında kalınır; yeniden ayrıştırma sonrasında kesin konumlandırma yapılabilir. Elle girilen bilgi ve FAQ için özgün dosya yoktur; kaynağa tıklandığında yine kaynak listesi gösterilir. Panelin üst kısmındaki «Tüm kaynaklar» listeye döner; listedeki belge kartlarındaki «Özgün metni görüntüle» düğmesi de özgün metni açabilir. Gömülü kanallardaki ziyaretçilerin özgün dosyaları okuma yetkisi yoktur, yalnızca kaynak listesi gösterilir.

<Screenshot
  src="/screenshots/chat-references-drawer.png"
  caption="Diyalog sayfası: yanıt, kaynak üst simgeleri ve sağdaki kaynak paneli"
  hint="Kaynaklı bir yanıt turunu, genişletilmiş kaynak panelini (kaynak başlığı ve parçası dahil) ve üstteki oturum işlem çubuğunu gösterir." />

Hızlı soru-cevap aramayı tamamladıktan sonra modelin içerik döndürmesini beklerken “Modele bağlanılıyor ve yanıt oluşturuluyor...” gösterilir; arama yapılmayan soru-cevaplarda “Yanıt hazırlanıyor...” gösterilir. Bekleme 60 saniyeyi aştığında, uyarı “Model yanıtı yavaş, bekleniyor...” olarak değişir. Bu uyarı mevcut bekleme durumunu yansıtır; yanıt ve araç yürütme sonuçları gerçek döndürülen içeriğe bağlıdır.

Yanıt, modelin tek seferlik çıktı sınırına ulaştığı için kesildiğinde, yanıt araç çubuğunda bir uyarı simgesi görünür ve yukarıdaki içeriğin kesilmeden önce oluşturulan içerik olduğu belirtilir. Tam yanıt gerektiğinde soru kapsamını daraltabilir veya ajan yapılandırmasında en büyük çıktı uzunluğunu artırabilirsiniz.

## Düşünme yoğunluğunu ayarlama

Seçilen model kademeli düşünmeyi desteklediğinde, giriş kutusunun altındaki model seçiminin yanında «Düşünme yoğunluğu» gösterilir; kapalı, otomatik, çok düşük, düşük, orta, yüksek, çok yüksek ve en yüksek seçenekleri arasından modelin desteklediği kademe seçilebilir. Varsayılan olarak ajan yapılandırması kullanılır; seçim değiştirildikten sonra yalnızca mevcut oturum için geçerlidir, ajanı değiştirmez ve oturum yeniden açıldığında son seçim geri yüklenir. Bu kademeyi desteklemeyen bir modele geçildiğinde, otomatik seçimi geçersiz olur.

## Yanıt sırasında ek gereksinimler

Akıllı çıkarım modu yanıt üretirken, doğrudan giriş kutusuna yazmaya devam edebilirsiniz:

| İşlem | Etki |
| --- | --- |
| Enter veya gönder düğmesine tıklama | Kuyruğa alınır, mevcut yanıt bittikten sonra sonraki tur sorusu olarak gönderilir |
| ⌘ Enter (Mac) veya Alt + Enter | Mevcut göreve hemen eklenir; ajan sonraki adımda okur ve ayarlar |

Kuyruktaki mesajlar giriş kutusunun üzerinde gösterilir; hemen eklemek için "Mevcut göreve ekle" seçeneğine, geri çekmek için "Kaldır" seçeneğine tıklayabilirsiniz. Ajanın zaten aldığı mesajlar geri çekilemez. Giriş boş olduğunda, gönder düğmesi üretimi durdurmak için kullanılır; durdurulduktan sonra kuyruktaki mesajlar da atılır. Eklenen mesajlar ek veya resim içeremez, @ ile bahsetme kullanılabilir. Her turda en fazla 10 mesaj kuyruğa alınabilir. Hızlı soru-cevap eklemeyi desteklemez; yanıt bitene kadar beklemek gerekir.

<Screenshot
  src="/screenshots/chat-steer-queue.png"
  caption="Yanıt üretimi sırasında giriş kutusunun üzerindeki kuyruktaki mesajlar"
  hint="Akıllı çıkarım yanıtı sürerken, giriş kutusunun üzerinde bir veya iki kuyruktaki mesaj gösterilir; her birinde "Mevcut göreve ekle" ve "Kaldır" düğmeleri bulunur. Giriş kutusunda yazılmakta olan metin vardır ve gönder düğmesinde “Tamamlandığında gönder · Enter” ipucu görünür." />

## Dallandırma ve geri alma

Kullanıcı mesajının altında ve yanıt araç çubuğunda iki işlem bulunur; yanıt üretilirken ikisi de kullanılamaz:

- **Dallandır**: Mevcut konuşmayı temel alarak yeni bir oturum oluşturur, özgün oturum değişmeden kalır. Kullanıcı mesajından dallandırırken, bu sorudan önceki konuşma kopyalanır ve soru yeniden giriş kutusuna yerleştirilir; düzenlenip tekrar gönderilebilir. Yanıttan dallandırırken, bu yanıta kadarki konuşma kopyalanır. Yeni oturumun başlığı “Özgün başlık (dal)” olur ve kenar çubuğunda dallandırma simgesiyle işaretlenir.
- **Geri al**: Mevcut oturumda seçili mesajdan sonraki konuşmayı siler; işlem geri alınamaz. Kullanıcı mesajından geri alırken, sorunun kendisi de silinir ve yeniden giriş kutusuna yerleştirilir. Silinen turlarda oluşturulan dosyalar, takip sorusu önerileri ve geçmiş arama dizinleri de aynı anda temizlenir.

Ajan bir korumalı alana bağlı olduğunda, sistem her yanıt turu bittikten sonra çalışma alanı denetim noktası kaydeder. Dallandırma, dallanma noktasına karşılık gelen çalışma alanı dosyalarını da getirir; bunlar getirilemediğinde (örneğin korumalı alan geri alındığında veya değiştirildiğinde) dallandırma yine başarılı olur ve yeni oturum tamamen yeni bir korumalı alan kullanır. Geri alma, çalışma alanını da korunan son turun durumuna döndürür; mevcut korumalı alanda ilgili denetim noktası bulunamazsa, dosyalar ile konuşmanın tutarsız olmasını önlemek için geri alma reddedilir. Korumalı alan olmadığında yalnızca konuşma geri alınır, dosyalar değiştirilmez.

## Geçici eklerle soru sorma

Konuşmaya dosya yükleyerek dosya içeriği hakkında soru sorabilirsiniz. Ekler yalnızca mevcut oturumda kullanılır; bilgi tabanı listesine veya vektör dizinine eklenmez. Uzun süreli arama ve paylaşım gerektiren materyaller bilgi tabanına yüklenmelidir.

Yüklemeden sonra sistem dosyayı eşzamansız olarak ayrıştırır ve yükleniyor, işleniyor ve hazır durumlarını gösterir. Soru sorulduğunda ayrıştırması henüz tamamlanmamış ekler beklemeye devam eder; varsayılan en fazla bekleme süresi 60 saniyedir. Taranmış belgeler ve görüntü tabanlı belgelerdeki metin, görsel model aracılığıyla tanınabilir.

Geçici eklerin saklama süresi yükleme anından itibaren hesaplanır ve varsayılan olarak 24 saattir; süre dolduğunda arka plan görevi tarafından temizlenir. Yönetici saklama süresini, ayrıştırma bekleme süresini ve OCR parametrelerini ayarlayabilir; ayrıntılı yapılandırma için belgenin sonundaki “Yapılandırma ve arayüz başvurusu” bölümüne bakın.

Ajan; yüklenmesine izin verilen dosya türleri, görsel anlama anahtarı ve dosya türüne göre seçilen ayrıştırma motorlarıyla yapılandırılabilir. İlgili açıklamalar için [Ajan motorları](07-agent.md) bölümüne bakın.

## Oluşturulan dosyaları önizleme ve indirme

Korumalı alana bağlı ajanlar ekleri işleyebilir ve dosya oluşturabilir. Yanıt tamamlandıktan sonra yanıt araç çubuğundaki "Bu sefer oluşturulan dosyaları görüntüle" seçeneğine veya konuşma sayfasının sağ üst köşesindeki korumalı alan düğmesine tıklayarak "Korumalı alan görselleştirmesi" panelindeki "Çıktılar" sekmesinden görüntüleyebilirsiniz; bu sekmede mevcut tur veya tüm dosyalar arasında geçiş yapabilir ve dosya adına göre arama yapabilirsiniz. Desteklenen dosya türleri doğrudan önizlenebilir, diğer dosyalar indirildikten sonra açılabilir.

Kenar çubuğundaki "Çıktılar" sayfası, mevcut kullanıcının tüm web oturumlarında oluşturulan dosyaları bir araya getirir. Belgeler, elektronik tablolar, sunumlar, görseller, web sayfaları ve veriler kategorilerine göre filtrelenebilir veya dosya adına göre aranabilir; ayrıca önizleme, indirme ve ilgili oturuma gitme işlemleri yapılabilir. Aynı oturumda aynı yoldaki aynı adlı dosya birden çok kez oluşturulduğunda yalnızca en yeni sürüm gösterilir ve sürüm sayısı belirtilir. Dağıtımda korumalı alan özelliği etkin değilse bu sayfa gösterilmez.

<Screenshot
  src="/screenshots/artifacts-library.png"
  caption=""Çıktılar" sayfası: çeşitli oturumlarda oluşturulan dosyaları bir araya getirir"
  hint="Kenar çubuğunda "Çıktılar" seçilidir. Sayfanın üstünde tür sekmeleri (Tümü/Belgeler/Elektronik tablolar/Sunumlar/Görseller/Web sayfaları/Veriler) ve dosya adı arama kutusu bulunur. Liste bugün/dün/son 7 gün şeklinde gruplandırılır; en az bir satırda “N sürüm” gösterilir ve satır içinde önizleme, indirme, silme işlemleri görünür." />

Ajanın indirilebilir dosya oluşturması gerektiğinde, çıktılar `/workspace/output` konumuna kaydedilmelidir; sistem bunları yanıtın sonunda toplar ve kalıcı olarak saklar. Yüklenen ekler geçici olarak `/workspace/input` konumunda tutulur; betikler ve ara dosyalar `/workspace` altındaki diğer dizinlere yerleştirilebilir. Giriş ekleri ve çıktı dosyaları ayrı yönetilir; giriş ekini silmek, daha önce toplanmış çıktı dosyalarını otomatik olarak silmez.

Artık gerekli olmayan dosyalar "Çıktılar" sekmesinden veya "Çıktılar" sayfasından silinebilir. Silme işlemi depolamadaki baytları geri kazanır ve geri alınamaz; bu nedenle önce onay istenir. Yalnızca oturumun sahibi silebilir; paylaşılan ajan üzerinden edinilen salt okunur erişim yalnızca indirmeye izin verir. "Çıktılar" sekmesi her oluşturmayı ayrı ayrı listeler ve tıklanan sürüm silinir; "Çıktılar" sayfasında bir satır bir dosyayı temsil eder ve silme işlemi o dosyanın bu oturumdaki tüm geçmiş sürümlerini de siler. Aynı dosya bilgi tabanına kaydedilmişse, sonraki yanıtlarda yeniden kullanılmışsa veya konuşma dallandırmasıyla kopyalanmışsa, bu konumlar etkilenmez. Silindikten sonra korumalı alandaki aynı adlı dosya da yeniden toplanmaz.

Dosya okuma, betik yürütme ve dizin kuralları için [Beceriler ve korumalı alan](22-skills-sandbox.md); listeleme, indirme ve silme arayüzleri için [Oturum ve sohbet API'si](../04-api/02-api-chat.md) bölümüne bakın.

## Yanıtları bilgi tabanına kaydetme

Yanıt araç çubuğundaki "Bilgi tabanına ekle", sağ çekmecede bir Markdown düzenleyicisi açar; soru ve yanıt önceden doldurulur, başvurulan belgeler ise sonunda "Referans kaynaklar" altında bir arada listelenir. Hedef bilgi tabanını seçtikten sonra (yalnızca belge türü bilgi tabanları), taslağı geçici olarak kaydedebilir veya depoya yayımlayabilirsiniz; yayımlandıktan sonra dizinleme başlar. Düzenleyici; düzenleme, bölünmüş ekran ve önizleme olmak üzere üç görünümü ve tam ekranı destekler. Araç çubuğu yaygın biçimlendirme ile tablo, kod bloğu, görsel ve benzeri ekleme işlemlerini sunar; Cmd/Ctrl + S geçici kaydeder, Cmd/Ctrl + Enter depoya yayımlar.

<Screenshot
  src="/screenshots/chat-save-answer-editor.png"
  caption="Yanıtları bilgi tabanına kaydetmek için Markdown düzenleyicisi"
  hint="Sohbet sayfasının sağ çekmecesi bölünmüş ekran görünümündedir: üstte başlık giriş kutusu ve hedef bilgi tabanı seçicisi, ortada solda Markdown kaynak kodu ve sağda önizleme, altta ise 'Geçerli durum: Taslak', kelime sayısı ve 'Taslağı geçici kaydet' ile 'Depoya yayımla' düğmeleri gösterilir." />

## Sohbetleri gözden geçirme ve dışa aktarma

Uzun sohbetlerin sağındaki soru dizini, doğrudan belirli bir soruya gitmenizi sağlar; üzerine gelindiğinde tur numarası, soru ve yanıt özeti gösterilir. Tarih değiştiğinde veya aralık 5 dakikayı aştığında iletiler arasına zaman ayırıcı eklenir; kullanıcı iletisinin altında gönderim zamanı gösterilir, bugünün iletilerinde yalnızca saat, daha eski iletilerde ise tarih de yer alır. Kenar çubuğu, bu tarayıcıda hâlâ üretilmekte olan oturumları dönen simgeyle işaretler ve API oturumları için çağıranı belirtir.

Oturum işlemleri menüsündeki "Markdown olarak kopyala", tüm sohbeti panoya kopyalar; içerik oturum başlığını, kimliğini, dışa aktarma zamanını ve tur bazında soru-cevapları (ekler ve alıntı listeleri dahil) içerir. Kopyalama tarayıcıda gerçekleştirilir ve sohbet kayıtlarını arşivlemek veya paylaşmak için kullanılabilir.

Kenar çubuğundaki "Ara", geçmiş iletileri oturumlar arasında bulmanızı sağlar. Alan yöneticisi "Sistem ayarları → Alan → İleti yönetimi" bölümünde ileti dizinlemeyi etkinleştirip bir Embedding modeli seçtiğinde, arama anlamsal eşleştirmeyi de kullanır; etkin değilse anahtar sözcüklere göre eşleştirir.

## Kanal oturumlarını görüntüleme

IM, web yerleştirmesi ve API çağrılarıyla oluşturulan oturumlar sırasıyla IM kimliği, ziyaretçi veya API Key bazında yalıtılır. Normal oturum listesi varsayılan olarak yalnızca geçerli çağıranın kendi oturumlarını gösterir.

Alan Admin veya Owner kullanıcıları, ilgili kanal oturumlarını IM, yerleştirme ve API grupları üzerinden görüntüleyebilir; gruplar, oturum bulunduğunda gösterilir. Yöneticiler bu giriş noktasını kanallardaki soru-cevap kayıtlarını denetlemek için salt okunur görüntüleme amacıyla kullanır. API Key oturumlarının yazma işlemleri, özgün oturum sahipliği kısıtlamalarına tabi olmaya devam eder.

## Oturumlar arası belleği kullanma

Alan yöneticisi "Sistem ayarları → Alan → Uzun süreli bellek" bölümünde etkinleştirdikten sonra üyeler, ifade tercihlerini yönetebilir, onay bekleyen öğeleri doğrulayabilir ve bunları sonraki oturumlarda "Sistem ayarları → Hesap → Belleğim" bölümünden kullanabilir. Bellek, alan ve çağıran kimliğine göre yalıtılır.

Kişisel belleği kapatmak kullanımı duraklatır; silme veya temizleme ise verileri kaldırmak için kullanılır. Otomatik çıkarma, konu yönetimi ve hemen düzenleme açıklamaları için [Uzun süreli bellek](23-memory.md) bölümüne bakın.

## Yapılandırma ve arayüz başvurusu

Aşağıdaki parametreler yöneticiler ve entegrasyon geliştiricileri içindir. Günlük sohbet işlemleri için yukarıda belirtilen arayüz girişlerini kullanmanız yeterlidir.

### Ek yapılandırması

| Yapılandırma öğesi | Varsayılan değer | İşlev |
| --- | --- | --- |
| `RETHRA_CHAT_ATTACHMENT_WAIT_TIMEOUT_SEC` | 60 saniye | Soru sorulurken ekin ayrıştırmayı tamamlaması için bekleme üst sınırı |
| `RETHRA_CHAT_ATTACHMENT_TTL_HOURS` | 24 saat | Yükleme anından itibaren hesaplanan ek saklama süresi |
| `RETHRA_CHAT_ATTACHMENT_OCR_CONCURRENCY` | 8 | OCR eşzamanlılık sayısı |
| `RETHRA_CHAT_ATTACHMENT_OCR_MAX_PAGES` | 8 | OCR sayfa üst sınırı |

Aracı yapılandırmasındaki `supported_file_types`, yüklenmesine izin verilen türleri sınırlar; `attachment_image_understanding` görsel anlamayı denetler, `chat_parser_engine_rules` ise ek ayrıştırma motorunu belirtir.

Ek API'si `/api/v1/sessions` önekini kullanır:

| İşlem | Yöntem ve yol |
| --- | --- |
| Yükleme | `POST /:session_id/attachments` |
| Liste | `GET /:id/attachments` |
| Ayrıntılar | `GET /:id/attachments/:attachment_id` |
| Önizleme | `GET /:id/attachments/:attachment_id/preview` |
| Silme | `DELETE /:id/attachments/:attachment_id` |

Tam parametreler ve izinler için [Oturum ve Sohbet API](../04-api/02-api-chat.md) bölümüne bakın.

### Oturum işlemi arayüzleri

| İşlem | Yöntem ve yol |
| --- | --- |
| Dallandırma | `POST /api/v1/sessions/:session_id/fork` |
| Geri alma | `POST /api/v1/sessions/:session_id/rewind` |
| Çalışırken mesaj ekleme | `POST /api/v1/sessions/:session_id/steer`, ayrıca listeleme, geri çekme ve anında ekleme olarak değiştirme seçenekleri bulunur |
| Bu turdaki düşünme yoğunluğu | Sohbet isteğindeki `reasoning_effort` alanı |

İstek gövdesi, dönüş kodları ve sınırlar için [Oturum ve Sohbet API](../04-api/02-api-chat.md) bölümüne bakın.

### Alıntılar ve geçmiş sorguları

Aracının `citation_enabled` ayarı, yanıt metnindeki alıntı üst simgelerini denetler; belirtilmezse varsayılan olarak etkindir. Alıntı paneli bu alandan etkilenmez.

Geçmiş araması `POST /api/v1/messages/search` kullanır, dizin istatistikleri `GET /api/v1/messages/chat-history-stats` kullanır; her ikisi de Viewer+ gerektirir. API Key için `message_history` yeteneği veya full-access gerekir. Oturum geçmişini okumak için `GET /api/v1/messages/:session_id/load` kullanılır ve Viewer+ gerektirir; API Key için `chat` yeteneği gerekir ve oturum sahipliği doğrulamasına tabidir.

Oturum listesindeki `source` boş veya `web` olduğunda kendi oturumlarınız sorgulanır; `api`, `im` ya da `embed` alan görünümlerinin açıkça sorgulanması Admin+ gerektirir, yetersiz izin durumunda 403 döner.

## Uygulama referansı

- `frontend/src/views/chat/components/`: yanıt, ilerleme ve araç süreci gösterimi.
- `frontend/src/utils/sessionMarkdown.ts`: Markdown olarak kopyalama.
- `internal/application/service/session.go`: oturum listesi ve kanal erişim kuralları.
- `internal/application/service/session_fork.go`, `session_rewind.go`, `workspace_checkpointer.go`: dallandırma, geri alma ve çalışma alanı denetim noktaları.
- `internal/handler/session/steer.go`: çalışırken mesaj ekleme.
- `internal/application/service/temporary_document.go`: geçici ek ayrıştırma, süre sonu ve temizleme.
- Geçici ekler `temporary_documents` tablosunda saklanır; tablo yapısı için bkz. [Veritabanı ve geçişler](../06-development/02-database-schema.md).
