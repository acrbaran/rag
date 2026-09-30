# Rethra MCP Server kurulum ve kullanım kılavuzu

## Hızlı başlangıç

### 1. Bağımlılıkları yükleyin
```bash
pip install -r requirements.txt
```

### 2. Ortam değişkenlerini ayarlayın
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

Sunucuyu çalıştırmanın birkaç yolu vardır:

#### Yol 1: Ana giriş noktasını kullanın (önerilir)
```bash
python main.py
```

#### Yol 2: Orijinal başlatma betiğini kullanın
```bash
python run_server.py
```

#### Yol 3: Sunucu modülünü doğrudan çalıştırın
```bash
python rethra_mcp_server.py
```

#### Yol 4: Python modülü olarak çalıştırın
```bash
python -m rethra_mcp_server
```

## Python paketi olarak kurulum

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
# Kaynak dağıtım paketi ve wheel oluşturun
python setup.py sdist bdist_wheel

# veya build aracını kullanın
pip install build
python -m build
```

## Komut satırı seçenekleri

Ana giriş noktası `main.py` şu seçenekleri destekler:

```bash
python main.py --help                 # yardım bilgisini gösterir
python main.py --check-only           # yalnızca ortam yapılandırmasını denetler
python main.py --verbose              # ayrıntılı logları etkinleştirir
python main.py --version              # sürüm bilgisini gösterir
```

## Ortam denetimi

Ortam yapılandırmasını denetlemek için şu komutu çalıştırın:
```bash
python main.py --check-only
```

Bu komut şunları gösterir:
- Rethra API temel URL yapılandırması
- API anahtarının ayarlanma durumu
- Bağımlılık paketlerinin kurulum durumu

## Sorun giderme

### 1. İçe aktarma hataları
`ImportError` alırsanız şunlardan emin olun:
- Tüm bağımlılıklar kurulu: `pip install -r requirements.txt`
- Python sürümü uyumlu (3.10+ önerilir)
- Dosya adı çakışması yok

### 2. Bağlantı hataları
Rethra API'ye bağlanamıyorsanız:
- `RETHRA_BASE_URL` değerinin doğru olduğunu kontrol edin
- Rethra hizmetinin çalıştığını doğrulayın
- Ağ bağlantısını doğrulayın

### 3. Kimlik doğrulama hataları
Kimlik doğrulama sorunu yaşarsanız:
- `RETHRA_API_KEY` değerinin ayarlı olduğunu kontrol edin
- API anahtarının geçerli olduğunu doğrulayın
- İzin ayarlarını doğrulayın

## Geliştirme modu

### Proje yapısı
```
Rethra/mcp-server/
├── __init__.py              # paket başlatma dosyası
├── main.py                  # ana giriş noktası
├── run_server.py           # orijinal başlatma betiği
├── rethra_mcp_server.py   # MCP sunucu uygulaması
├── requirements.txt        # bağımlılık listesi
├── setup.py               # kurulum betiği
├── pyproject.toml         # proje meta verisi (PyPI: rethra-mcp)
├── MANIFEST.in            # dahil edilecek dosyalar listesi
├── LICENSE                # lisans
├── README.md              # proje açıklaması
└── INSTALL.md             # kurulum kılavuzu
```

### Yeni özellik ekleme
1. `RethraClient` sınıfına yeni API yöntemini ekleyin
2. `@mcp.tool()` dekoratörüyle yeni bir araç fonksiyonu kaydedin: parametreleri tür açıklamalarıyla belirtin (şema otomatik üretilir), açıklamayı docstring'e yazın, fonksiyon gövdesinde yukarıda eklediğiniz istemci yöntemini çağırın
3. Belgeleri ve testleri güncelleyin

### Test
```bash
# Temel testleri çalıştırın
python check_imports.py

# Ortam yapılandırmasını test edin
python main.py --check-only

# Sunucu başlatmayı test edin
python main.py --verbose
```

## Dağıtım

### Docker ile dağıtım
`Dockerfile` oluşturun:
```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY . .
RUN pip install -e .

ENV RETHRA_BASE_URL=http://localhost:8080/api/v1
EXPOSE 8000

CMD ["rethra-mcp-server"]
```

### Sistem hizmeti
`/etc/systemd/system/rethra-mcp.service` systemd hizmet dosyasını oluşturun:
```ini
[Unit]
Description=Rethra MCP Server
After=network.target

[Service]
Type=simple
User=rethra
WorkingDirectory=/opt/rethra-mcp
Environment=RETHRA_BASE_URL=http://localhost:8080/api/v1
Environment=RETHRA_API_KEY=your_api_key
ExecStart=/usr/local/bin/rethra-mcp-server
Restart=always

[Install]
WantedBy=multi-user.target
```

Hizmeti etkinleştirin:
```bash
sudo systemctl enable rethra-mcp
sudo systemctl start rethra-mcp
```

## Destek

Sorunla karşılaşırsanız:
1. Log çıktısına bakın
2. Ortam yapılandırmasını kontrol edin
3. Sorun giderme bölümüne başvurun
4. Proje deposunda Issue açın: https://github.com/acrbaran/rag/issues