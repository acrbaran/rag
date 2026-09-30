#!/usr/bin/env bash
# firstboot.sh - rethra-firstboot.service tarafından yeni örneğin ilk açılışında otomatik olarak çalıştırılır.
# Görev: Rastgele anahtarları .env içine yaz -> container'ları başlat -> kimlik bilgilerini çıktıla -> tamamlandı olarak işaretle + kendini devre dışı bırak.
#
# İdempotentlik stratejisi:
#   1) Başta ${MARKER} işaretini "anahtarlar zaten oluşturuldu" olarak yaz,
#      ardından docker compose başarısız olup betik kesilse bile, yeniden başlatıldığında birim tarafından
#      ConditionPathExists=!${MARKER} ile engellenir; .env üzerine yazacak yeni anahtarlar yeniden oluşturulmaz.
#      (Eski anahtar zaten postgres veri birimine yazıldı, yeniden oluşturmak veritabanına kalıcı olarak giriş yapılamamasına yol açar)
#   2) Başarısız olduğunda unit failed olarak işaretlenir, kullanıcı docker compose up -d ile elle geri yükleyebilir;
#      Kimlik bilgileri hâlâ ${ENV_FILE} içinde bulunabilir.
set -euo pipefail

RETHRA_DIR="${RETHRA_DIR:-/opt/Rethra}"
ENV_FILE="${RETHRA_DIR}/.env"
ENV_TEMPLATE="${RETHRA_DIR}/.env.example"
CRED_FILE="/root/rethra-credentials.txt"
LOG_FILE="/var/log/rethra-firstboot.log"
MARKER="${RETHRA_DIR}/.firstboot.done"

# LOG_FILE dosyasını önceden aç, hata ayıklamayı kolaylaştırmak için stderr'i de systemd journal'a bir kopya olarak gönder
# (Yalnızca exec >> LOG_FILE kullanıldığında, betiğin başındaki hata stderr zaten yutulduğu için görünmez)
mkdir -p "$(dirname "${LOG_FILE}")"
exec > >(tee -a "${LOG_FILE}") 2>&1
echo "==== firstboot started at $(date -Iseconds) ===="

if [[ -f "${MARKER}" ]]; then
  echo "marker ${MARKER} exists, skip (already initialized)"
  exit 0
fi

# cleanup.sh artık .env dosyasını tutmaz; burada şablon .env.example dosyasından kopyalanıp ardından değiştirilir.
# Bu, firstboot öncesinde rethra.service için düz metin varsayılan parola içeren hiçbir .env bulunmamasını sağlar
# postgres veri birimini yanlış parolayla önceden başlatmasını engeller.
if [[ ! -f "${ENV_FILE}" ]]; then
  if [[ -f "${ENV_TEMPLATE}" ]]; then
    echo "creating ${ENV_FILE} from ${ENV_TEMPLATE}"
    cp "${ENV_TEMPLATE}" "${ENV_FILE}"
  else
    echo "ERROR: neither ${ENV_FILE} nor ${ENV_TEMPLATE} found"
    exit 1
  fi
fi

DOCKER_BIN="$(command -v docker || true)"
if [[ -z "${DOCKER_BIN}" ]]; then
  echo "ERROR: docker binary not found in PATH"
  exit 1
fi

# 32 baytlık güçlü rastgele dize oluştur (AES-256 key için kullanılır, tam olarak 32 bayt olmalıdır)
# Alt kabuğu `() ... ()` ile başlat ve pipefail'i kapat: head yalnızca N bayt okuduktan sonra stdin'i kapatır,
# tr SIGPIPE alır (çıkış kodu 141), `set -o pipefail` altında tüm boru hattı başarısız sayılır,
# üst düzey `set -e` tetiklenir ve firstboot.sh 8ms exit 1 ile sonlandırılır.
gen32() ( set +o pipefail; LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32 )
# Genel parola: 24 karakter, / + = içermez (URL / sed değiştirmelerinde sorun çıkmasını önlemek için)
genpw() ( set +o pipefail; LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 24 )

DB_PWD=$(genpw)
REDIS_PWD=$(genpw)
JWT=$(genpw)$(genpw)
SYS_AES=$(gen32)

# Çakışmayı önlemek için sed ayıracı olarak | kullan; yalnızca KEY= ile başlayan satırları değiştir
replace() {
  local key="$1" val="$2"
  if grep -qE "^${key}=" "${ENV_FILE}"; then
    sed -i "s|^${key}=.*|${key}=${val}|" "${ENV_FILE}"
  else
    echo "${key}=${val}" >>"${ENV_FILE}"
  fi
}

replace DB_PASSWORD     "${DB_PWD}"
replace REDIS_PASSWORD  "${REDIS_PWD}"
replace JWT_SECRET      "${JWT}"
replace SYSTEM_AES_KEY  "${SYS_AES}"
replace GIN_MODE        "release"

# prepare.sh aşamasında .cloud-image-meta içine kaydedilen RETHRA_REF değerini .env içindeki
# RETHRA_VERSION değerine geri yükle; aksi halde docker compose varsayılan :latest değerine döner ve görüntü sürümü
# prepare sırasında çekilen sürümle tutarsız olur.
META_FILE="${RETHRA_DIR}/.cloud-image-meta"
if [[ -f "${META_FILE}" ]]; then
  META_REF=$(grep -E '^RETHRA_REF=' "${META_FILE}" | tail -1 | cut -d= -f2- || true)
  if [[ -n "${META_REF}" ]]; then
    replace RETHRA_VERSION "${META_REF}"
    echo "restored RETHRA_VERSION=${META_REF} from ${META_FILE}"
  fi
fi

# Kritik: .env değiştirildikten hemen sonra marker yaz.
# Bundan sonra docker compose up başarısız olsa bile, yeniden başlatma .env dosyasını tekrar yazmaz,
# postgres tarafından zaten kalıcılaştırılmış parola ile tutarsızlığı önler.
umask 077
touch "${MARKER}"
chmod 0600 "${MARKER}"

echo "env updated, marker written, starting docker compose..."
cd "${RETHRA_DIR}"
"${DOCKER_BIN}" compose up -d

# Genel IP alınmaya çalışılır, başarısız olursa özel ağ kullanılır
PUB_IP=$(curl -fsS --max-time 5 https://ifconfig.me 2>/dev/null \
  || curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null \
  || hostname -I | awk '{print $1}')

cat >"${CRED_FILE}" <<INFO
========================================
  Rethra örneğinin ilk kurulumu tamamlandı
  Oluşturulma zamanı: $(date -Iseconds)
========================================

Erişim adresi : http://${PUB_IP}

Kayıttan sonra yeni kayıtları kapatmak için ${ENV_FILE} dosyasını düzenleyin:
    DISABLE_REGISTRATION=true
Ardından çalıştırın:  cd ${RETHRA_DIR} && docker compose up -d

Aşağıdaki rastgele kimlik bilgileri ${ENV_FILE} dosyasına yazıldı, güvenle saklayın (yalnızca root okuyabilir):
  DB_PASSWORD     = ${DB_PWD}
  REDIS_PASSWORD  = ${REDIS_PWD}
  JWT_SECRET      = ${JWT}
  SYSTEM_AES_KEY  = ${SYS_AES}

Dikkat:
  - Bu dosya yalnızca ilk açılıştaki anahtarları gösterir; sonrasında ${ENV_FILE} esas alınır.
  - 5432 / 6379 / 9000 gibi altyapı portlarını asla doğrudan dışarıya açmayın.
  - Dışarıya yalnızca 80 / 443 üzerinden hizmet verin; gerekirse ters vekil sunucu + HTTPS yapılandırın.

INFO

echo "credentials written to ${CRED_FILE}"

# Yalnızca unit'i durdur, unit dosyasını silme (aksi halde şu anda çalışan oneshot systemd tarafından failed olarak işaretlenebilir).
# Sonraki yeniden başlatmada, rethra-firstboot.service ConditionPathExists=!${MARKER} aracılığıyla otomatik olarak atlanır.
systemctl disable rethra-firstboot.service || true

echo "==== firstboot finished at $(date -Iseconds) ===="
