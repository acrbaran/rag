# Rethra Lite ile standart sürüm arasındaki farklar

Lite, yerelde hızlıca kullanmak ve dağıtımı olabildiğince basit tutmak isteyenler içindir; standart sürüm çok alanlı işbirliği ve tam kurumsal yetenekler içindir. Başlıca farklar şunlardır.


| Boyut | Lite | Standart sürüm |
| --------- | ----------------------------------------------- | --------------------------------- |
| **Paylaşılan alan** | Paylaşılan alan yoktur (üye daveti, üyeler arası bilgi tabanı ve ajan paylaşımı vb.) | Paylaşılan alan ve alan izolasyonlu arama gibi işbirliği yetenekleri sunar |
| **Alan ve hesap** | Tek alan; kutudan çıktığı gibi çalışır, **kayıt gerekmez** | Çok alan; genellikle kayıt, giriş ve organizasyon yönetimi gerekir |
| **Belge ayrıştırma** | Yalnızca yerleşik **Simple** türü ayrıştırma motoru; **Cloud** gibi yollarla başka ayrıştırma yetenekleri bağlanabilir | Birden çok ayrıştırma motoru yapılandırılabilir (yüksek doğruluklu olanlar dahil), tam belge işleme hattıyla entegredir |
| **Dağıtım biçimi** | **Tek uygulama, sıfır bağımlılık** (ayrı veritabanı, mesaj kuyruğu gibi harici servis yığınlarına bağlı değildir) | Genellikle Docker Compose gibi çok servisli dağıtım; daha fazla bağımlılık ve bileşen |
| **Veri sahipliği** | Veriler **tamamen yerelde** saklanır ve işlenir | Şirket içi dağıtımda veriler yerelde de olabilir; dağıtım biçiminize bağlıdır |
| **Ağ erişimi** | **Varsayılan olarak yalnızca yerel makineden erişim**; gerektiğinde yapılandırılarak **genel ağa açılıp açılmayacağı seçilebilir** | Adres ve ağ geçidi dağıtım ve güvenlik politikasına göre bağlanır |


Çok ekipli işbirliğine, karmaşık ayrıştırma hatlarına ve çok servisli mimariye ihtiyacınız yoksa Lite, bireyler veya küçük ekipler için yerel makinede sıfır bağımlılıkla denemeye daha uygundur; paylaşılan alan, çoklu alan ve tam ayrıştırma motoru seçenekleri gerektiğinde standart sürümü kullanın.