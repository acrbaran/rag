#!/usr/bin/env bash
# cleanup.sh - Bulut imajı oluşturmadan önce özel verileri temizler.
# Uyarı: Bu betik SSH açık anahtarlarını siler, veritabanını ve günlükleri temizler, son olarak otomatik olarak kapanır.
# Çalıştırdıktan sonra doğrudan bulut konsolunda «İmaj oluştur / Anlık görüntü oluştur / AMI oluştur» seçeneğini kullanın, tekrar SSH ile bağlanmayın.
set -euo pipefail

RETHRA_DIR="${RETHRA_DIR:-/opt/Rethra}"

if [[ "${EUID}" -ne 0 ]]; then
  echo "[cleanup] Lütfen sudo veya root ile çalıştırın" >&2
  exit 1
fi

read -r -p "[cleanup] Bu işlem geri alınamaz, devam etmek istediğinizi onaylıyor musunuz? Devam etmek için YES yazın: " ans
if [[ "${ans}" != "YES" ]]; then
  echo "[cleanup] İptal edildi"
  exit 0
fi

echo "[cleanup] 1/8 Rethra konteynerleri durduruluyor"
COMPOSE_PROJECT=""
if [[ -d "${RETHRA_DIR}" ]]; then
  cd "${RETHRA_DIR}"
  # Gerçek project adını almak için öncelikle compose ls kullanın (varsayılan olarak dizin adının küçük harfli hâlidir, ör. rethra)
  COMPOSE_PROJECT="$(docker compose ls --format json 2>/dev/null \
    | grep -oE '"Name":"[^"]+"' | head -1 | cut -d'"' -f4 || true)"
  docker compose down -v --remove-orphans || true
fi

echo "[cleanup] 2/8 Rethra iş verilerini, ilk açılış işaretini ve günlükleri temizle"
if [[ -d "${RETHRA_DIR}" ]]; then
  rm -rf "${RETHRA_DIR}/data"/* "${RETHRA_DIR}/logs"/* 2>/dev/null || true
  # Burada .env dosyası kasıtlı olarak yeniden oluşturulmuyor: imajda .env eksik kalsın, firstboot öncesinde
  # başlatılan tüm docker compose işlemleri .env bulunamadığı için başarısız olur; böylece düz metin varsayılan parolaların
  # (postgres123!@# vb.) postgres veri birimini hatalı başlatması önlenir.
  # firstboot.sh kendisi .env.example dosyasından kopyalar ve anahtarları değiştirir.
  rm -f "${RETHRA_DIR}/.env" "${RETHRA_DIR}/.firstboot.done"
fi
rm -f /root/rethra-credentials.txt /var/log/rethra-firstboot.log

echo "[cleanup] 3/8 Kalan docker birimleri ve derleme önbelleği temizleniyor"
# Aynı ana makinedeki diğer postgres/redis birimlerinin yanlışlıkla etkilenmesini önlemek için compose project adı önekiyle sıkı eşleştirme yapın.
if [[ -n "${COMPOSE_PROJECT}" ]]; then
  docker volume ls -q --filter "label=com.docker.compose.project=${COMPOSE_PROJECT}" \
    | xargs -r docker volume rm -f || true
fi
# Not: Burada yalnızca "birimler / durdurulmuş kapsayıcılar / derleme önbelleği" temizlenir; imajlar kesinlikle temizlenmemelidir.
# Daha önce `docker system prune -af --volumes` kullanmak, prepare.sh tarafından önceden çekilen
# rethra-* gibi imajları da birlikte siler; bunun sonucunda imaj tabanlı oluşturulan yeni örnekler
# firstboot sırasında Docker Hub'dan yeniden birkaç GB imaj çekmek zorunda kalır; bu, ön kurulum amacına tamamen aykırıdır.
docker container prune -f      || true
docker builder    prune -af    || true
# Yalnızca bağlı olmayan dangling birimleri temizleyin (bu sırada compose down -v iş birimlerini zaten temizlemiştir)
docker volume     prune -f     || true

echo "[cleanup] 4/8 Sistem günlükleri temizleniyor"
journalctl --rotate || true
journalctl --vacuum-time=1s || true
find /var/log -type f \( -name '*.log' -o -name '*.gz' -o -name '*.[0-9]' \) -print0 \
  | xargs -0 -r truncate -s 0 || true
find /var/log -type f \( -name '*.gz' -o -name '*.[0-9]' \) -print0 \
  | xargs -0 -r rm -f || true

echo "[cleanup] 5/8 SSH geçmişini ve yetkili anahtarları temizle (sonrasında SSH ile bağlanılamaz)"
rm -f /root/.ssh/authorized_keys /root/.ssh/known_hosts /root/.bash_history
for d in /home/*; do
  [[ -d "$d" ]] || continue
  rm -f "$d/.ssh/authorized_keys" "$d/.ssh/known_hosts" "$d/.bash_history"
done
find / -xdev -type f \( -name 'id_rsa*' -o -name '*.pem' -o -name '*.key' \) \
  -not -path '/etc/ssl/*' -not -path '/usr/*' -not -path '/var/lib/docker/*' 2>/dev/null \
  | tee /tmp/cleanup-secrets-found.txt || true
echo "[cleanup]   ↑ Yukarıdakiler muhtemel kalıntı anahtar dosyalarıdır, gerekirse manuel olarak tekrar kontrol edin"

echo "[cleanup] 6/8 cloud-init / machine-id sıfırlanıyor (yeni örneğin yeni bir kimlik alması için)"
cloud-init clean --logs --seed 2>/dev/null || true
truncate -s 0 /etc/machine-id || true
rm -f /var/lib/dbus/machine-id || true

echo "[cleanup] 7/8 apt / tmp temizleniyor"
if command -v apt-get >/dev/null 2>&1; then
  apt-get clean
  rm -rf /var/lib/apt/lists/*
fi
rm -rf /tmp/* /var/tmp/* /root/.cache /home/*/.cache 2>/dev/null || true

echo "[cleanup] 8/8 disk eşitleniyor ve kapatılıyor"
history -c || true
sync
echo
echo "  Kapatılmak üzere. Kapatma tamamlandıktan sonra bulut konsolunda «imaj oluştur / anlık görüntü oluştur / AMI oluştur» işlemini gerçekleştirin."
echo
sleep 3
poweroff
