# Bulut imajı bakım betikleri

Betiklerin sınırları, imaj hazırlarken dikkat edilecekler ve ilk açılış sorun giderme bilgileri [website-docs geliştirme kılavuzunda](../../website-docs/06-development/01-dev-guide.md#cloud-image-scripts) tutulur. Normal kurulum için [Kurulum ve dağıtım](../../website-docs/01-getting-started/02-installation.md) sayfasını kullanın.

Bu dizinde `prepare.sh`, `cleanup.sh`, `firstboot.sh` ve bunların systemd birimleri bulunur. `cleanup.sh`, imaj hazırlama makinesindeki verileri, anahtarları, SSH yetkilerini ve önbelleği temizleyip makineyi kapatır; yalnızca bu iş için ayrılmış bir makinede kullanılmalıdır. Parametreler ve gerçek davranış için betiğin kendisi esas alınır.
