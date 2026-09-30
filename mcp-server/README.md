# Rethra MCP Server

> **⚠️ Kullanımdan kaldırıldı (Deprecated)**
>
> Rethra artık yerleşik bir MCP Server içeriyor: "Ayarlar → Yayınlama ve Entegrasyon → MCP Server" bölümünden alan için bir uç nokta oluşturmanız yeterli. Birden çok uç nokta desteklenir, her uç nokta için bilgi tabanı kapsamı ve araçlar seçilebilir; istemciler Streamable HTTP ile doğrudan `/mcp/<endpoint_id>` adresine bağlanır, bu dizindeki Python sürecini ayrıca dağıtmaya gerek yoktur. Bu dizin yalnızca eski dağıtımlarla uyumluluk için tutulmaktadır ve sonraki sürümlerde kaldırılacaktır.
>
> Rethra now ships a built-in MCP server: create endpoints under "Settings → Publish & Integrations → MCP Server", each with its own token, knowledge-base scope and tool list, and connect clients to `/mcp/<endpoint_id>` over Streamable HTTP. This Python package is kept only for existing deployments and will be removed in a future release.

Bu, Rethra bilgi yönetimi API'sine erişim sağlayan bir Model Context Protocol (MCP) sunucusudur.

## Hızlı başlangıç

> Doğrudan [MCP yapılandırma açıklamasına](./MCP_CONFIG.md) bakmanız önerilir; aşağıdaki adımları uygulamanız gerekmez.

### 1. Bağımlılıkları yükleyin
```bash
pip install -r requirements.txt
```

### 2. Ortam değişkenlerini yapılandırın
```bash
# Linux/macOS
export RETHRA_BASE_URL="http://localhost:8080/api/v1"
export RETHRA_API_KEY="your_api_key_here"

# Windows PowerShell
$env:RETHRA_BASE_URL="http://localhost:8080/api/v1"
$env:RETHRA_API_KEY="your_api_key_here"

# Windows CMD
set RETHRA_BASE_URL=http://localhost:8080/api/v1
set RETHRA_API_KEY=your_api_key_here
```

### 3. Sunucuyu çalıştırın

**Önerilen yol - ana giriş noktasını kullanın:**
```bash
python main.py
```

**Diğer çalıştırma yolları:**
```bash
# Orijinal başlatma betiğini kullanın
python run_server.py

# Kolaylık betiğini kullanın
python run.py

# Sunucu modülünü doğrudan çalıştırın
python rethra_mcp_server.py

# Python modülü olarak çalıştırın
python -m rethra_mcp_server
```

### 4. Komut satırı seçenekleri
```bash
python main.py --help                 # yardım bilgisini gösterir
python main.py --check-only           # yalnızca ortam yapılandırmasını denetler
python main.py --verbose              # ayrıntılı logları etkinleştirir
python main.py --version              # sürüm bilgisini gösterir
```

## Python paketi olarak kurulum

### PyPI'dan kurulum

```bash
pip install rethra-mcp
# veya uvx ile doğrudan çalıştırın (önceden kurulum gerekmez)
uvx --from rethra-mcp rethra-mcp-server
```

> Resmi PyPI paket adı **`rethra-mcp`** (acrbaran/rag tarafından bakımı yapılır, Trusted Publishing ile yayımlanır).
> Kurulumdan sonra komut satırı giriş noktaları yine `rethra-mcp-server` / `rethra-server` olarak kalır.

### Geliştirme modunda kurulum
```bash
pip install -e .
```

Kurulumdan sonra komut satırı aracı kullanılabilir:
```bash
rethra-mcp-server
# veya
rethra-server
```

### Üretim modunda kurulum
```bash
pip install .
```

### Dağıtım paketi oluşturma
```bash
# setuptools kullanarak
python setup.py sdist bdist_wheel

# modern derleme araçlarıyla
pip install build
python -m build
```

## Modül testi

Modülün düzgün çalıştığını doğrulamak için test betiğini çalıştırın:
```bash
python test_module.py
```

## Özellikler

Bu MCP sunucusu şu araçları sunar:

### Alan yönetimi
- `create_tenant` - yeni alan oluşturur
- `list_tenants` - tüm alanları listeler

### Bilgi tabanı yönetimi
- `create_knowledge_base` - bilgi tabanı oluşturur
- `list_knowledge_bases` - bilgi tabanlarını listeler
- `get_knowledge_base` - bilgi tabanı ayrıntılarını getirir
- `delete_knowledge_base` - bilgi tabanını siler
- `hybrid_search` - hibrit arama

### Bilgi yönetimi
- `create_knowledge_from_file` - yerel dosyadan bilgi oluşturur
- `create_knowledge_from_url` - URL'den bilgi oluşturur
- `create_knowledge_from_text` - metinden bilgi oluşturur
- `update_knowledge_from_text` - elle yazılmış Markdown bilgisini günceller; yeniden indeksleyebilir veya taslak olarak kaydedebilir
- `list_knowledge` - bilgileri listeler
- `get_knowledge` - bilgi ayrıntılarını getirir
- `delete_knowledge` - bilgiyi siler

### Model yönetimi
- `create_model` - model oluşturur
- `list_models` - modelleri listeler
- `get_model` - model ayrıntılarını getirir

### Oturum yönetimi
- `create_session` - sohbet oturumu oluşturur
- `get_session` - oturum ayrıntılarını getirir
- `list_sessions` - oturumları listeler
- `delete_session` - oturumu siler

### Sohbet işlevleri
- `chat` - sohbet mesajı gönderir
- `agent_chat` - çok adımlı arama ve araç çağrısı için ajanı çağırır

Her iki sohbet aracı da olayları SSE boş satır sınırlarına göre birleştirir; bir olay birden çok `data:` satırı içerebilir.
Bağlantı kapandığında henüz boş satıra ulaşmamış olay atılır; tek bir olayın veri tamponu en fazla 8 MiB olabilir,
sınır aşılırsa hata döndürülür ve yanıt bağlantısı kapatılır.

### Parça yönetimi
- `list_chunks` - bilgi parçalarını listeler
- `delete_chunk` - bilgi parçasını siler

## Sorun giderme

İçe aktarma hatası alırsanız şunlardan emin olun:
1. Gerekli tüm bağımlılık paketleri kurulu
2. Python sürümü uyumlu (3.10+ önerilir)
3. Dosya adı çakışması yok (dosya adı olarak `mcp.py` kullanmaktan kaçının)

## Çalışma örneği

<img width="950" height="2063" alt="118d078426f42f3d4983c13386085d7f" src="https://github.com/user-attachments/assets/09111ec8-0489-415c-969d-aa3835778e14" />

### Local upload directory boundary

All transports, including stdio, restrict local file uploads to the working
directory by default. Set `MCP_ALLOWED_UPLOAD_DIRS` to a comma-separated list of
trusted absolute directories when additional roots are needed. Starting in a
filesystem root requires an explicit directory configuration. Paths and symbolic
links resolving outside the selected roots are rejected.
