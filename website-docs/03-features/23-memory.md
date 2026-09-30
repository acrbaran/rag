# Oturumlar arası uzun süreli bellek

Uzun süreli bellek, çağıranın profilini, ifade tercihlerini, kalıcı olguları ve uzun vadeli ilgi alanlarını sonraki oturumlarda kullanılmak üzere saklar. Veriler çalışma alanına ve çağıran kimliğine göre ayrılır; farklı alanlar, IM kullanıcıları veya gömülü ziyaretçiler kendi kişisel belleklerini ayrı tutar.

## Uzun süreli belleği açma {#uzun-sureli-bellegi-acma}

1. Alanın Admin/Owner'ı "Ayarlar → Uzun süreli bellek" bölümünden alan anahtarını açar; varsayılan olarak kapalıdır.
2. "Yalnızca açıkça hatırla" veya "Otomatik çıkar" seçilir; otomatik modda çıkarım modeli belirtilebilir, boş bırakılırsa sohbette kullanılan model kullanılır.
3. Kişisel bellek yönetiminde kendi belleğinizi kapatabilirsiniz; alan açık olsa bile kullanımı zorunlu değildir.
4. Sohbette hatırlanmasını istediğiniz tercihi açıkça belirtin veya bellek yönetiminden elle ekleyin; ardından başka bir oturumda uygulanıp uygulanmadığını kontrol edin.
5. Onay bekleyen girdileri düzenli olarak inceleyin; çıkarımları onaylayın, düzenleyin veya reddedin. Geçerliliğini yitirmiş öğeler silinebilir.

| Mod | Davranış |
| --- | --- |
| `explicit_only` | Yalnızca açıkça hatırlanması istenen içeriği kaydeder, arka planda LLM damıtması çalıştırmaz |
| `auto` | Ek olarak arka planda sohbetten çıkarım yapar; gecikme ve en kısa aralığa göre birleştirerek işler |

Ajanın `memory_enabled=false` ayarı bellek okuma/yazmayı ayrıca devre dışı bırakabilir; belirtilmezse alan ayarını devralır. IM/Embed, bağlı ajanın bellek tercihlerini kullanır; kişisel bellek yönetimi arayüzlerini kullanmak için API Key'in full-access olması gerekir. Özellik; alan, kişisel ve geçerli istek anahtarlarının birlikte izin vermesine bağlıdır.

## Belleği yönetme

| İşlem | Amaç |
| --- | --- |
| Ekle / düzenle | Kişisel profil, tercih, olgu, iş veya ilgi alanlarını doğrudan yönetir; elle düzenlenen girdilerin üzerine arka plan çıkarımı artık yazmaz |
| Bekleyeni onayla | Çıkarım girdisi pending durumundan kullanılabilir duruma geçer; onaylanmadan istem metnine eklenmez. Çıkarım mevcut bir belleği değiştirmek içinse onaylanana kadar eski girdi geçerli kalır ve onay sırasında birlikte değiştirilir; çıkarım geçerliliğini yitirmişse onay başarısız olur ve liste yenilenmelidir |
| Reddet / sil | Hatalı veya artık gerekmeyen belleği geri alır; reddetme bir bastırma kaydı bırakarak tekrar çıkarılmasını azaltır |
| Konuyu yükselt | Sık tartışılan bir konuyu hemen uzun vadeli ilgi alanına çevirir; konu takibi de durdurulabilir |
| Belge tercihleri | Sık başvurulan belgeleri gösterir; artık gerekmeyen kişiselleştirilmiş arama tercihleri kaldırılabilir |
| Hemen düzenle | Benzer girdileri birleştirir, süresi dolan işleri arşivler; arka plandaki düzenlemeyi beklemek gerekmez |
| Dışa aktar / temizle | Kendi belleğinizi JSON olarak indirir veya kendi bellek verilerinizi temizler |

Bellek girdileri profil, tercih, olgu, iş ve ilgi alanı olarak sınıflandırılır. Aynı konudaki yeni bir olgu eskisinin yerini alabilir. Kişisel anahtarı kapatmak kullanımı duraklatır; silme veya temizleme verileri kaldırır. Arayüzdeki kategori ve durum değerleri için bkz. [Bellek API'si](../04-api/02-api-memory.md).

## Yanıtlara ve aramaya etkisi

Sistem profil, tercih ve ilgi alanlarını uzunluğu sınırlı kalıcı bağlam olarak kullanır; ilgili olgu ve işleri o anki soruya göre geri çağırır. Vektör modeli yapılandırıldığında anlamsal geri çağırma, o kimliğin tüm belleklerini soruyla ilgisine göre sıralar; girdilerin önem derecesi bu sıralamayı etkilemez. PostgreSQL'de pgvector etkinse sıralama veritabanında yapılır, diğer ortamlarda servis içinde hesaplanır. Akıllı akıl yürütme ayrıca belleği ve geçmiş sohbetleri kendisi arayabilir. Bellek yalnızca sorunun anlaşılmasını ve kaynak seçimini etkiler; bilgi tabanı erişim yetkilerini genişletmez.

Arama kişiselleştirmesi açıldığında sistem, soruyu anlamak için konulardan, sıralamaya yardımcı olmak için de sık başvurulan belgelerden yararlanır. Sohbet zaman çizelgesi bellekle ilgili adımları gösterir. Arka plan görevleri oturum bazında çıkarım imlecini ve bekleyen durumu kaydeder; kısa sürede art arda sorulan sorular, toplu işlemin kesilmesi veya servisin yeniden başlaması mesaj kaybına yol açmaz. Bir mesaj bölümünün model çıktısı sürekli ayrıştırılamıyorsa en fazla üç tur yeniden denenir, ardından o bölüm atlanır ve sonraki mesajlara geçilir; atlanan mesajlar için otomatik olarak yeniden çıkarım yapılmaz.

## Belleğin çalışmaması sorununu giderme {#bellegin-calismamasi-sorununu-giderme}

| Belirti | Kontrol edilecekler |
| --- | --- |
| Bellek kullanılmıyor | Alan anahtarı, kişisel anahtar, ajan anahtarı; alanın veya kimliğin değişip değişmediğini kontrol edin |
| Otomatik çıkarım görünmüyor | write_mode, çıkarım gecikmesi/en kısa aralık, model bağlantısı ve arka plan görevleri |
| Çıkarım yanıtı etkilemiyor | Hâlâ pending durumunda mı; yalnızca onaylandıktan sonra kullanılır |
| Belirli bir belgeye artık başvurulmasın isteniyor | Belge tercihlerinden kaldırın; bilgi tabanı yetkileri ayrıca kontrol edilir |
| Hemen düzenle hiçbir girdiyi birleştirmedi | Dönen skipped nedenini inceleyin; girdi sayısı az olabilir veya aday yoktur |

## Alan yapılandırması

Alan yöneticileri bellek anahtarını, çıkarım modunu ve modeli ayarlar sayfasından yönetebilir. Yapılandırma `memory_config` içinde saklanır, `GET/PUT /tenants/kv/memory-config` ile okunur ve güncellenir; alanların tamamı için bkz. [Bellek API'si](../04-api/02-api-memory.md).

| Ad | Tür | Varsayılan | Açıklama |
| --- | --- | --- | --- |
| `enabled` | bool | `false` | Alan anahtarı |
| `write_mode` | string | `explicit_only` | `explicit_only` / `auto`; diğer değerler `explicit_only` olarak işlenir |
| `extract_model_id` | string | boş | Çıkarım modeli; boşsa sohbet modeli kullanılır |
| `extract_delay_seconds` | int | `90` | Yanıt tamamlandıktan sonra çıkarımı geciktirir, kısa sürede gelen çok turlu mesajları birleştirir; aralık 5–3600 |
| `extract_min_interval_seconds` | int | `300` | Aynı kişi için iki çıkarım arasındaki en kısa süre; çağrı sıklığını denetler, aralık içindeki mesajlar sonraya ertelenir; en fazla 86400 |
| `extract_instructions` | string | boş | Alana özgü çıkarım kuralları, en fazla 1000 karakter |
| `max_items` | int | `200` | Kimlik başına etkin bellek üst sınırı, en fazla 2000; 0 varsayılanı kullanır |
| `interest_threshold` | int | `3` | Bir konunun ilgi alanına dönüşmesi için kaç oturumda geçmesi gerektiği, en fazla 20 |
| `embedding_model_id` | string | boş | Bellek geri çağırmada kullanılan vektör modeli; boşsa yalnızca sözcüksel eşleştirme yapılır |
| `vector_recall` | bool | belirtilmezse açık | Vektör modeli yapılandırıldığında anlamsal geri çağırmanın etkin olup olmadığı |
| `retrieval_conditioning` | bool | belirtilmezse açık | Belleğin soru anlama ve belge sıralamasını etkilemesine izin verir |

Çıkarım modeli arka planda özetleme için, vektör modeli ise o anki soruyla eşleştirme için kullanılır. Bellek özelliği ilgili model çağrılarını artırır; bir model bellek yapılandırmasında kullanılıyorsa, model silinirken bağımlılık ayrıntıları gösterilir.

## Uygulama başvurusu

- `internal/application/service/memory/`: kapsam, çıkarım, geri çağırma, konular, belge tercihleri, düzenleme
- `internal/handler/memory.go`, `internal/router/routes_memory.go`: kişisel API
- `internal/types/memory.go`: modeller, bütçe, durumlar ve yapılandırma
- `frontend/src/views/settings/MemorySettings.vue`, `MemoryWorkspaceSettings.vue`
