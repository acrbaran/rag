#!/usr/bin/env bash
# prepare.sh - Deploys the Rethra runtime on a clean Linux instance, for creating cloud image templates.
# No need to clone the entire Rethra repository; only downloads 4 runtime files (~100KB).
# Compatible with: distributions such as Ubuntu / Debian / CentOS / Rocky / TencentOS with systemd + Docker.
# Usage:  sudo bash prepare.sh
# Configurable environment variables:
#   RETHRA_REF              git ref to fetch (tag / branch / commit), default main
#   RETHRA_DIR              deployment directory, default /opt/Rethra
#   RETHRA_REPO             repository URL, default https://github.com/acrbaran/rag
#   RETHRA_GH_PROXY         GitHub acceleration prefix, default empty. Machines in mainland China can set
#                            https://gh-proxy.com/ or https://ghfast.top/
#                            (the actual download URL becomes ${RETHRA_GH_PROXY}${RETHRA_REPO}/archive/...)
#   DOCKER_INSTALL_MIRROR    Docker package mirror, default empty (uses get.docker.com).
#                            Set this when overseas CDNs are unreachable from machines in mainland China, for example:
#                              https://mirrors.tencent.com/docker-ce/linux/ubuntu
#                              https://mirrors.aliyun.com/docker-ce/linux/ubuntu
#                            apt + docker-ce resmi depo aynası kurulumu kullanılacak,
#                            docker-ce / containerd.io / docker-compose-plugin içerir,
#                            get.docker.com'a hiç erişmez. Yalnızca apt tabanlı dağıtımlar desteklenir.
#   DOCKER_REGISTRY_MIRROR   Docker Hub hızlandırıcısı, varsayılan boş. Tencent Cloud intranetinde ayarlanabilir
#                            https://mirror.ccs.tencentyun.com
#                            (/etc/docker/daemon.json dosyasına yazılır ve docker yeniden başlatılır)
#   PRUNE_OLD_IMAGES         Yükseltme sırasında dangling / eski sürüm tag imajlarının temizlenip temizlenmeyeceği,
#                            Varsayılan false. true olarak ayarlandığında yeni imajlar çekildikten sonra
#                            `docker image prune -af`, hiçbir container tarafından kullanılmayan imajları
#                            (eski RETHRA_VERSION değerindeki rethra-* dahil)
#                            tek seferde silerek bulut imajına eklenecek boyutu azaltır.
set -euo pipefail

RETHRA_REF="${RETHRA_REF:-main}"
RETHRA_DIR="${RETHRA_DIR:-/opt/Rethra}"
RETHRA_REPO="${RETHRA_REPO:-https://github.com/acrbaran/rag}"
RETHRA_GH_PROXY="${RETHRA_GH_PROXY:-}"
DOCKER_INSTALL_MIRROR="${DOCKER_INSTALL_MIRROR:-}"
DOCKER_REGISTRY_MIRROR="${DOCKER_REGISTRY_MIRROR:-}"
PRUNE_OLD_IMAGES="${PRUNE_OLD_IMAGES:-false}"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

if [[ "${EUID}" -ne 0 ]]; then
  echo "[prepare] Lütfen sudo veya root ile çalıştırın" >&2
  exit 1
fi

# docker-ce tam paketini (compose-plugin dahil) depo aynası üzerinden apt ile kurar.
# Çin ana karasındaki bulut sunucularının get.docker.com'a doğrudan bağlantısının RST aldığı durumlar içindir.
install_docker_via_apt_mirror() {
  local mirror="$1"
  if ! command -v apt-get >/dev/null 2>&1; then
    echo "[prepare] DOCKER_INSTALL_MIRROR şu anda yalnızca apt tabanlı dağıtımları destekler (Ubuntu/Debian)" >&2
    return 1
  fi
  apt-get update -y
  apt-get install -y ca-certificates curl gnupg lsb-release
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL "${mirror%/}/gpg" | gpg --dearmor --yes -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  local arch codename
  arch="$(dpkg --print-architecture)"
  codename="$(lsb_release -cs)"
  echo "deb [arch=${arch} signed-by=/etc/apt/keyrings/docker.gpg] ${mirror%/} ${codename} stable" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io \
                     docker-buildx-plugin docker-compose-plugin curl tar
}

echo "[prepare] 1/6 Docker ve bağımlılıkları kuruluyor"
if ! command -v docker >/dev/null 2>&1; then
  if [[ -n "${DOCKER_INSTALL_MIRROR}" ]]; then
    echo "[prepare]   docker-ce, ${DOCKER_INSTALL_MIRROR} üzerinden apt ile kuruluyor (get.docker.com atlanıyor)"
    install_docker_via_apt_mirror "${DOCKER_INSTALL_MIRROR}"
  else
    curl -fsSL https://get.docker.com | bash
  fi
fi
systemctl enable --now docker

if ! docker compose version >/dev/null 2>&1; then
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -y
    apt-get install -y docker-compose-plugin curl tar
  elif command -v yum >/dev/null 2>&1; then
    yum install -y docker-compose-plugin curl tar
  fi
fi

# İsteğe bağlı: registry-1.docker.io'ya doğrudan bağlantının zaman aşımına uğradığı durumları çözmek için Docker Hub hızlandırıcısını yapılandırın
# (tipik: Çin ana karasındaki bulut sunucuları / kısıtlı intranet ortamları). daemon.json yalnızca kullanıcı açıkça ilettiğinde değiştirilir.
if [[ -n "${DOCKER_REGISTRY_MIRROR}" ]]; then
  echo "[prepare] 1.5/6 Docker Hub hızlandırıcısı yapılandırılıyor: ${DOCKER_REGISTRY_MIRROR}"
  mkdir -p /etc/docker
  # Mevcut daemon.json, kullanıcının diğer ayarlarının üzerine yazılmasını önlemek için python ile birleştirilir
  if [[ -s /etc/docker/daemon.json ]] && command -v python3 >/dev/null 2>&1; then
    python3 - "$DOCKER_REGISTRY_MIRROR" <<'PY'
import json, sys, pathlib
p = pathlib.Path("/etc/docker/daemon.json")
mirror = sys.argv[1]
try:
    cfg = json.loads(p.read_text())
except Exception:
    cfg = {}
mirrors = cfg.get("registry-mirrors") or []
if mirror not in mirrors:
    mirrors.insert(0, mirror)
cfg["registry-mirrors"] = mirrors
p.write_text(json.dumps(cfg, indent=2) + "\n")
PY
  else
    cat >/etc/docker/daemon.json <<EOF
{
  "registry-mirrors": ["${DOCKER_REGISTRY_MIRROR}"]
}
EOF
  fi
  systemctl restart docker
  # docker daemon'a yeniden başlatılması için biraz süre verin; aşağıdaki docker compose pull komutunun hemen EOF almasını önleyin
  for _ in 1 2 3 4 5; do
    docker info >/dev/null 2>&1 && break
    sleep 1
  done
fi

echo "[prepare] 2/6 Rethra çalışma zamanı dosyaları çekiliyor (ref=${RETHRA_REF})"
# Yalnızca gerçekten gereken 4 dosyayı indirir, tüm depoyu clone etmez (~MB düzeyi -> ~KB düzeyi)
mkdir -p "${RETHRA_DIR}/config"

tmp=$(mktemp -d)
trap 'rm -rf "${tmp}"' EXIT

tarball_url="${RETHRA_GH_PROXY}${RETHRA_REPO}/archive/${RETHRA_REF}.tar.gz"
echo "[prepare]   tarball: ${tarball_url}"
curl -fsSL "${tarball_url}" -o "${tmp}/repo.tar.gz"
# Yalnızca gereken yolları açar, belirgin ölçüde hızlandırır ve alan tasarrufu sağlar
tar -xzf "${tmp}/repo.tar.gz" -C "${tmp}" \
  --wildcards \
  '*/docker-compose.yml' \
  '*/.env.example' \
  '*/config/config.yaml'
src=$(find "${tmp}" -maxdepth 1 -mindepth 1 -type d -name 'Rethra-*' | head -1)
if [[ -z "${src}" ]]; then
  echo "Arşiv açma başarısız, Rethra-* dizini bulunamadı" >&2
  exit 1
fi

cp    "${src}/docker-compose.yml" "${RETHRA_DIR}/"
cp    "${src}/.env.example"       "${RETHRA_DIR}/"
cp    "${src}/config/config.yaml" "${RETHRA_DIR}/config/"

# firstboot / yükseltme sırasında başvuru için meta verileri kaydeder
cat >"${RETHRA_DIR}/.cloud-image-meta" <<EOF
RETHRA_REF=${RETHRA_REF}
RETHRA_REPO=${RETHRA_REPO}
PREPARED_AT=$(date -Iseconds)
EOF

echo "[prepare] 3/6 .env hazırlanıyor (varsayılan değerler, firstboot bunları rastgele anahtarlarla değiştirecek)"
cd "${RETHRA_DIR}"
[[ -f .env ]] || cp .env.example .env
sed -i 's/^GIN_MODE=.*/GIN_MODE=release/' .env || true

# Yerel olarak oluşturulan imaj tag'inin ref ile tutarlı olması için RETHRA_VERSION ile RETHRA_REF değerlerini hizalar.
# Koşulsuz olarak üzerine yazar; .env içinde önceki prepare işleminden kalan eski sürüm numarasını önler.
# İmaj tag adlandırma kuralı：
#   - Değişken tag：main（sürekli olarak en güncel derlemeyi gösterir）
#   - Sabit release tag：v öneki + semver（örn. v0.7.2、v0.5.2）
# Bu nedenle burada v kaldırılmaz ve latest'e eşlenmez。
RETHRA_VERSION_VAL="${RETHRA_REF}"
if grep -qE '^RETHRA_VERSION=' .env; then
  sed -i "s|^RETHRA_VERSION=.*|RETHRA_VERSION=${RETHRA_VERSION_VAL}|" .env
else
  echo "RETHRA_VERSION=${RETHRA_VERSION_VAL}" >>.env
fi
echo "[prepare]   -> RETHRA_VERSION=${RETHRA_VERSION_VAL}"

echo "[prepare] 4/6 Varsayılan 5 kalıcı konteyner oluşturuluyor ve başlatılıyor (frontend/app/docreader/postgres/redis)"
docker compose pull --ignore-buildable
docker compose up --build -d

# sandbox imajını önceden oluştur (Agent Skills çalışma zamanı app tarafından ihtiyaç halinde docker run ile çalıştırılır, sürekli açık değildir)
echo "[prepare] 4.5/6 sandbox imajı oluşturuluyor (Agent Skills için, kalıcı değil)"
docker compose --profile full build sandbox || true

# Diğer vektör veritabanları / gözlemlenebilirlik bileşenleri (qdrant, milvus, weaviate, doris, neo4j, langfuse-*, minio, dex)
# Önceden çekilmez, 5-15GB alan tasarrufu sağlar. Kullanıcının etkinleştirmesi gerekirse:
#   cd /opt/Rethra && docker compose --profile <name> up -d

# Yükseltme senaryosu: Eski sürüm tag'lerine ait rethra-* imajlarını temizle.
# Varsayılan olarak kapalıdır, geri alma yolu korunur; boyutu küçültmek için imaj oluşturulmadan önce açıkça etkinleştirin.
#
# Dikkat: `docker image prune -af` kullanmayın!
# sandbox imajı compose içinde yalnızca pull edilir, up edilmez (Agent Skills app tarafından ihtiyaç halinde docker run ile çalıştırılır),
# Hiçbir container ona başvurmaz; `prune -a` uygulanırsa mevcut sürümün sandbox'ı da silinir,
# Bu ise prepare.sh 4.5 adımındaki sandbox'ı önceden çekme amacına aykırıdır.
# Burada tag'e göre kesin karşılaştırma yapılır; yalnızca rethra-* deposu altında, tag'i mevcut
# RETHRA_VERSION değerine eşit olmayan imajlar silinir; altyapı imajlarına (paradedb / redis) dokunulmaz.
if [[ "${PRUNE_OLD_IMAGES,,}" == "true" || "${PRUNE_OLD_IMAGES}" == "1" ]]; then
  echo "[prepare] 4.6/6 rethra-* deposu altındaki eski sürüm imajları temizleniyor (PRUNE_OLD_IMAGES=true, keep=${RETHRA_VERSION_VAL})"
  docker image ls --format '{{.Repository}}:{{.Tag}}' \
    | grep -E '^rethra-' \
    | grep -vE ":${RETHRA_VERSION_VAL}\$" \
    | xargs -r docker rmi -f 2>/dev/null || true
fi

echo "[prepare] 5/6 systemd birimleri kuruluyor"
# docker ikili dosya yolunu algıla; farklı dağıtımlarda /usr/bin veya /usr/local/bin altında olabilir
DOCKER_BIN="$(command -v docker)"
if [[ -z "${DOCKER_BIN}" ]]; then
  echo "[prepare] docker ikili dosyası bulunamadı" >&2
  exit 1
fi
echo "[prepare]   docker binary: ${DOCKER_BIN}"

install -m 0644 "${SCRIPT_DIR}/systemd/rethra.service"           /etc/systemd/system/rethra.service
install -m 0644 "${SCRIPT_DIR}/systemd/rethra-firstboot.service" /etc/systemd/system/rethra-firstboot.service
install -m 0755 "${SCRIPT_DIR}/firstboot.sh"                      /usr/local/sbin/rethra-firstboot.sh

# systemd birimindeki docker yol şablonunu gerçek yolla değiştir
sed -i "s|@DOCKER_BIN@|${DOCKER_BIN}|g" /etc/systemd/system/rethra.service

systemctl daemon-reload
systemctl enable rethra.service
systemctl enable rethra-firstboot.service

echo "[prepare] 6/6 Tamamlandı"
echo
echo "  Rethra çalışma zamanı ${RETHRA_DIR} konumuna dağıtıldı"
echo "    docker-compose.yml / config/config.yaml / .env"
echo "  Sürüm: ${RETHRA_REF}  (${RETHRA_DIR}/.cloud-image-meta dosyasına bakın)"
echo
echo "  Özellikleri doğrulamak için tarayıcıda  http://<sunucunun-genel-IP-adresi>  adresini açın"
echo
echo "  Doğrulama başarılı olduktan sonra temizleme işlemini yürütün ve imajı oluşturun:"
echo "      sudo bash ${SCRIPT_DIR}/cleanup.sh"
