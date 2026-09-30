# Depolama arka uçları (Storage Backends)

Depolama arka uçları özgün dosyaları, ayrıştırılmış görselleri ve dışa aktarma çıktılarını saklar. Bir alan birden çok örnek kaydedebilir, varsayılan örneği belirleyebilir ve her bilgi tabanı için depolama konumunu ayrıca seçebilir.

Çoklu örnekli depolama şu senaryolar için uygundur:

- Farklı ekiplerin veya projelerin belgelerini ayrı depolama kovalarında tutup kullanım ve yetkileri ayrı yönetmek;
- Veri saklama gereksinimlerine göre kovanın bulunduğu bölgeyi seçmek;
- Bulut nesne depolamaya geçerken yeni bilgi tabanlarının yeni arka ucu kullanması, mevcut bilgi tabanlarının ise özgün dosyalara erişmeye devam etmesi.

<Screenshot
  src="/screenshots/settings-storage-backends.png"
  caption="Depolama arka ucu ayarları: çoklu örnek listesi, varsayılan örnek ve bağlantı testi"
  hint="Kayıtlı depolama arka ucu kartlarını (provider, durum, varsayılan işareti) ve oluşturma/düzenleme formunu, 'Bağlantıyı test et' sonucuyla birlikte gösterir." />

## Depolama arka ucunu kaydetme ve seçme {#depolama-arka-ucunu-kaydetme-ve-secme}

Alan Admin'i "Ayarlar → Depolama" bölümünden örnekleri kaydedip yönetebilir:

1. Yeni bir arka uç oluşturun, provider seçin (`local` / `minio` / `cos` / `tos` / `s3` / `oss` / `ks3` / `obs`; [Belge içe alma akışındaki](../02-architecture/03-document-pipeline.md) depolama provider'larıyla aynıdır) ve bağlantı parametrelerini girin (aşağıdaki tabloya bakın). Seçilebilir liste `STORAGE_ALLOW_LIST` ortam değişkeniyle sınırlanır; boş bırakılırsa hepsi seçilebilir;
2. **Bağlantıyı test edin**: test depolamaya gerçekten okuma/yazma yaparak uç noktayı, kovayı ve kimlik bilgilerini doğrular;
3. Örneği alanın varsayılanı yapın. Yeni bilgi tabanı bir örnek belirtmezse bu varsayılan kullanılır;
4. Ayrıca belirtmek gerekirse bilgi tabanı düzenleme penceresindeki "Depolama" sekmesinden örnek seçin.

Bilgi tabanı boşken depolama arka ucu değiştirilebilir; dosya varsa arayüz seçimi devre dışı bırakır ve taşıma yapılmasını önerir. Dosya yolları içe alma sırasındaki arka uca bağlıdır; doğrudan değiştirmek özgün dosyalara erişimi etkiler. Değiştirmek gerekiyorsa yeni bir bilgi tabanı oluşturup içeriği taşıyın.

## Bağlantı parametreleri {#baglanti-parametreleri}

Tüm provider'lar ortak bir yapılandırma alanı kümesi kullanır; `access_key_id` / `secret_access_key` şifreli saklanır ve API yanıtlarında maskelenir.

| Ad | Tür | Varsayılan | Açıklama |
| --- | --- | --- | --- |
| `endpoint` | string | boş | Servis adresi. `minio` (`mode=remote`), `tos`, `s3`, `oss`, `ks3`, `obs` için zorunludur; `cos` için gerekmez. Kaydederken SSRF denetimi yapılır; iç ağ adresleri `SSRF_WHITELIST`'e eklenmelidir |
| `region` | string | boş | Bölge. `cos`, `tos`, `s3`, `oss`, `ks3`, `obs` için zorunludur |
| `access_key_id` / `secret_access_key` | string | boş | Erişim anahtarı; COS için SecretId / SecretKey'e karşılık gelir. `local` ve `mode=docker` MinIO için gerekmez |
| `bucket_name` | string | boş | Depolama kovası; `local` dışında zorunludur |
| `path_prefix` | string | boş | Nesne anahtarı öneki; göreli yol olmalı, `/` ile başlamamalı veya `..` içermemelidir |
| `mode` | string | `remote` | Yalnızca MinIO: `docker` dağıtımla gelen MinIO'yu kullanır (adres ve anahtarlar `MINIO_ENDPOINT` gibi ortam değişkenlerinden okunur), `remote` harici bir MinIO'ya bağlanır |
| `use_ssl` | bool | false | MinIO, S3, OBS için HTTPS kullanılıp kullanılmayacağı |
| `force_path_style` | bool | false | Yalnızca S3: path-style adresleme kullanır; S3 uyumlu servislerin çoğunda açılması gerekir |
| `app_id` | string | boş | Yalnızca COS: Tencent Cloud AppID |
| `temp_bucket_name` / `temp_region` | string | boş | Yalnızca COS, TOS, OSS: geçici dosyalar için kullanılan kova ve bölge; OSS'de ayrıca `use_temp_bucket=true` ile etkinleştirilmelidir |

- **OBS**: endpoint bir alan adıysa sanal ana makine adreslemesi (`<bucket>.<endpoint>`) kullanılır; Huawei Cloud 2023-12-30'dan itibaren alan adı endpoint'lerine yapılan path-style istekleri reddeder. Endpoint bir IP ise yine path-style kullanılır. Proxy alan adı yapılandırılmamışsa dosya URL'si `<scheme>://<bucket>.<endpoint-host>/<key>` biçimindedir.
- **Aktarım zaman aşımı**: S3, COS, KS3, OBS ve OSS'de tek bir yükleme/indirme artık 30 saniyelik genel zaman aşımıyla sınırlı değildir; tek aktarım en fazla 30 dakika sürebilir. Bağlantı kurma, TLS el sıkışması ve yanıt başlığını bekleme için ayrı zaman aşımları vardır; karşı taraf yanıt vermezse hızlıca başarısız olur.
- **KS3 yönlendirmesi**: Yönlendirme izlenirken her adımda SSRF denetimi yapılır; istek imzası artık yönlendirme hedefi ana makineye iletilmez.

## Vektör depolamadan farkı

Dosya depolama ve vektör depolama sırasıyla özgün dosyaları ve arama indekslerini yönetir:

| | Depolama arka ucu (Storage Backend) | Vektör deposu (Vector Store) |
| --- | --- | --- |
| Ne saklar | Özgün dosyalar, görseller, dışa aktarma çıktıları | Vektörler ve arama indeksleri |
| Nerede yapılandırılır | "Ayarlar → Depolama" | "Ayarlar → Vektör deposu" |
| Bilgi tabanı alanı | `storage_backend_id` | `vector_store_id` |
| İlgili bölüm | Bu sayfa | [Arama motorları ve vektör depolama](05-retrieval-engines.md) |

## Arayüz başvurusu {#arayuz-basvurusu}

| Yöntem | Yol | Yetki |
| --- | --- | --- |
| GET | `/storage-backends/types` | Viewer+; `STORAGE_ALLOW_LIST` ile izin verilen provider adlarının listesini döndürür |
| GET | `/storage-backends`, `/storage-backends/:id` | Viewer+ |
| POST | `/storage-backends` | Admin+ |
| PUT / DELETE | `/storage-backends/:id` | Admin+ |
| POST | `/storage-backends/test` | Admin+; kaydedilmemiş parametrelerle bağlantıyı dener |
| POST | `/storage-backends/:id/test` | Admin+; kaydedilmiş örneği test eder |
| PUT | `/storage-backends/:id/default` | Admin+; alanın varsayılanı yapar |

API Key için `manage_storage_backends` yeteneği veya full-access gerekir.

## Veri modeli ve uyumluluk kuralları {#veri-modeli-ve-uyumluluk-kurallari}

`storage_backends` tablosunun (`tenant_id` ile ayrılır, geçici silme) temel alanları:

| Alan | Açıklama |
| --- | --- |
| `name` | Alan içinde benzersizdir (geçici silme altında kısmi benzersiz indeks) |
| `provider` | Depolama türü |
| `config` | JSONB, şifrelenmiş anahtarları içerir |
| `source` | `user` (arayüz veya API ile kaydedilen) / `env` (ortam değişkeni yapılandırmasından üretilen) |
| `status` | `active` / `disabled` |
| `legacy_alias` | Aşağıya bakın |

`legacy_alias`, ortam değişkenleriyle yapılandırılmış eski depolamalarla uyumluluk için kullanılır. Yükseltme sırasında bir takma ad kaydı oluşturulur; böylece mevcut dosya yolları çözülmeye devam eder ve veri taşımak gerekmez. Aynı alanda ve aynı provider için yalnızca bir takma ad kaydına izin verilir; elle kaydedilen örnekler ayrı saklanır.
