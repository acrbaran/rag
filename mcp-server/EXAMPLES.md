# Rethra MCP Server kullanım örnekleri

Bu belge Rethra MCP Server için ayrıntılı kullanım örnekleri sunar.

## Temel kullanım

### 1. Sunucuyu başlatma

```bash
# Önerilen yol - ana giriş noktasını kullanın
python main.py

# Ortam yapılandırmasını kontrol et
python main.py --check-only

# Ayrıntılı günlüğü etkinleştir
python main.py --verbose
```

### 2. Ortam yapılandırması örneği

```bash
# Ortam değişkenlerini ayarla
export RETHRA_BASE_URL="http://localhost:8080/api/v1"
export RETHRA_API_KEY="your_api_key_here"

# ya da .env dosyasında ayarla
echo "RETHRA_BASE_URL=http://localhost:8080/api/v1" > .env
echo "RETHRA_API_KEY=your_api_key_here" >> .env
```

## MCP araçları kullanım örnekleri

Aşağıda çeşitli MCP araçlarının kullanım örnekleri yer alır:

### Alan yönetimi

#### Alan oluşturma
```json
{
  "tool": "create_tenant",
  "arguments": {
    "name": "Şirketim",
    "description": "Şirket bilgi yönetim sistemi",
    "business": "technology",
    "retriever_engines": {
      "engines": [
        {"retriever_type": "keywords", "retriever_engine_type": "postgres"},
        {"retriever_type": "vector", "retriever_engine_type": "postgres"}
      ]
    }
  }
}
```

#### Tüm alanları listeleme
```json
{
  "tool": "list_tenants",
  "arguments": {}
}
```

### Bilgi tabanı yönetimi

#### Bilgi tabanı oluşturma
```json
{
  "tool": "create_knowledge_base",
  "arguments": {
    "name": "Ürün belgeleri",
    "description": "Ürünle ilgili belgeler ve materyaller",
    "embedding_model_id": "text-embedding-ada-002",
    "summary_model_id": "gpt-3.5-turbo"
  }
}
```

#### Bilgi tabanlarını listeleme
```json
{
  "tool": "list_knowledge_bases",
  "arguments": {}
}
```

#### Bilgi tabanı ayrıntılarını alma
```json
{
  "tool": "get_knowledge_base",
  "arguments": {
    "kb_id": "kb_123456"
  }
}
```

#### Hibrit arama
```json
{
  "tool": "hybrid_search",
  "arguments": {
    "kb_id": "kb_123456",
    "query": "API nasıl kullanılır",
    "vector_threshold": 0.7,
    "keyword_threshold": 0.5,
    "match_count": 10
  }
}
```

### Bilgi yönetimi

#### URL'den bilgi oluşturma
```json
{
  "tool": "create_knowledge_from_url",
  "arguments": {
    "kb_id": "kb_123456",
    "url": "https://docs.example.com/api-guide",
    "enable_multimodel": true
  }
}
```

#### Metinden bilgi oluşturma
```json
{
  "tool": "create_knowledge_from_text",
  "arguments": {
    "kb_id": "my-knowledge-base",
    "title": "Dikkat mekanizması özeti",
    "content": "# Dikkat mekanizması\n\nDikkat mekanizması, modelin dizileri işlerken ağırlıkları dinamik olarak dağıtmasını sağlar...",
    "status": "publish"
  }
}
```

#### Elle girilen bilgiyi güncelleme
```json
{
  "tool": "update_knowledge_from_text",
  "arguments": {
    "knowledge_id": "know_789012",
    "title": "Dikkat mekanizması özeti (gözden geçirilmiş)",
    "content": "# Dikkat mekanizması\n\nBurada gözden geçirilmiş Markdown içeriği yer alır...",
    "status": "publish"
  }
}
```

`title` atlanırsa ya da boş dize verilirse özgün başlık korunur; `status` değeri `draft` yapılırsa içerik yeniden dizinlenmeden yalnızca kaydedilir.

#### Bilgileri listeleme
```json
{
  "tool": "list_knowledge",
  "arguments": {
    "kb_id": "kb_123456",
    "page": 1,
    "page_size": 20
  }
}
```

#### Bilgi ayrıntılarını alma
```json
{
  "tool": "get_knowledge",
  "arguments": {
    "knowledge_id": "know_789012"
  }
}
```

### Model yönetimi

#### Model oluşturma
```json
{
  "tool": "create_model",
  "arguments": {
    "name": "GPT-4 Chat Model",
    "type": "KnowledgeQA",
    "source": "openai",
    "description": "Bilgi soru-cevap için OpenAI GPT-4 modeli",
    "base_url": "https://api.openai.com/v1",
    "api_key": "sk-...",
    "is_default": true
  }
}
```

#### Modelleri listeleme
```json
{
  "tool": "list_models",
  "arguments": {}
}
```

### Oturum yönetimi

#### Sohbet oturumu oluşturma
```json
{
  "tool": "create_session",
  "arguments": {
    "kb_id": "kb_123456",
    "max_rounds": 10,
    "enable_rewrite": true,
    "fallback_response": "Üzgünüm, bu soruyu yanıtlayamıyorum.",
    "summary_model_id": "gpt-3.5-turbo"
  }
}
```

#### Oturum ayrıntılarını alma
```json
{
  "tool": "get_session",
  "arguments": {
    "session_id": "sess_345678"
  }
}
```

#### Oturumları listeleme
```json
{
  "tool": "list_sessions",
  "arguments": {
    "page": 1,
    "page_size": 10
  }
}
```

### Sohbet özellikleri

#### Sohbet mesajı gönderme
```json
{
  "tool": "chat",
  "arguments": {
    "session_id": "sess_345678",
    "query": "Lütfen ürünün temel özelliklerini anlat"
  }
}
```

### Parça yönetimi

#### Bilgi parçalarını listeleme
```json
{
  "tool": "list_chunks",
  "arguments": {
    "knowledge_id": "know_789012",
    "page": 1,
    "page_size": 50
  }
}
```

#### Bilgi parçasını silme
```json
{
  "tool": "delete_chunk",
  "arguments": {
    "knowledge_id": "know_789012",
    "chunk_id": "chunk_456789"
  }
}
```

## Tam iş akışı örneği

### Senaryo: eksiksiz bir bilgi soru-cevap sistemi kurmak

```bash
# 1. Sunucuyu başlat
python main.py --verbose

# 2. MCP istemcisinde şu adımları uygula:
```

#### Adım 1: Alan oluşturma
```json
{
  "tool": "create_tenant",
  "arguments": {
    "name": "Teknik belge merkezi",
    "description": "Şirket teknik belgeleri bilgi yönetimi",
    "business": "technology"
  }
}
```

#### Adım 2: Bilgi tabanı oluşturma
```json
{
  "tool": "create_knowledge_base",
  "arguments": {
    "name": "API belgeleri",
    "description": "API ile ilgili tüm belgeler"
  }
}
```

#### Adım 3: Bilgi içeriği ekleme
```json
{
  "tool": "create_knowledge_from_url",
  "arguments": {
    "kb_id": "dönen bilgi tabanı ID'si",
    "url": "https://docs.company.com/api",
    "enable_multimodel": true
  }
}
```

#### Adım 4: Sohbet oturumu oluşturma
```json
{
  "tool": "create_session",
  "arguments": {
    "kb_id": "bilgi tabanı ID'si",
    "max_rounds": 5,
    "enable_rewrite": true
  }
}
```

#### Adım 5: Sohbeti başlatma
```json
{
  "tool": "chat",
  "arguments": {
    "session_id": "oturum ID'si",
    "query": "Kullanıcı kimlik doğrulama API'si nasıl kullanılır?"
  }
}
```

## Hata işleme örnekleri

### Sık karşılaşılan hatalar ve çözümleri

#### 1. Bağlantı hatası
```json
{
  "error": "Connection refused",
  "solution": "RETHRA_BASE_URL değerinin doğru olduğunu kontrol edin ve servisin çalıştığını doğrulayın"
}
```

#### 2. Kimlik doğrulama hatası
```json
{
  "error": "Unauthorized",
  "solution": "RETHRA_API_KEY değerinin doğru ayarlandığını kontrol edin"
}
```

#### 3. Kaynak bulunamadı
```json
{
  "error": "Knowledge base not found",
  "solution": "Bilgi tabanı ID'sinin doğru olduğunu doğrulayın ya da önce bilgi tabanı oluşturun"
}
```

## Gelişmiş yapılandırma örnekleri

### Özel arama yapılandırması
```json
{
  "tool": "hybrid_search",
  "arguments": {
    "kb_id": "kb_123456",
    "query": "arama sorgusu",
    "vector_threshold": 0.8,
    "keyword_threshold": 0.6,
    "match_count": 15
  }
}
```

### Özel oturum stratejisi
```json
{
  "tool": "create_session",
  "arguments": {
    "kb_id": "kb_123456",
    "max_rounds": 20,
    "enable_rewrite": true,
    "fallback_response": "Mevcut bilgilere göre sorunuzu doğru biçimde yanıtlayamıyorum. Lütfen soruyu farklı şekilde ifade edin ya da teknik destekle iletişime geçin."
  }
}
```

## Performans iyileştirme önerileri

1. **Toplu işlemler**: Bilgi oluşturma ve güncelleme işlemlerini mümkün olduğunca toplu yapın
2. **Önbellek stratejisi**: Doğruluk ile performansı dengelemek için arama eşiğini makul ayarlayın
3. **Oturum yönetimi**: Kaynak tasarrufu için gereksiz oturumları zamanında temizleyin
4. **Günlük izleme**: Performans ölçümlerini izlemek için `--verbose` seçeneğini kullanın

## Entegrasyon örnekleri

### Claude Desktop ile entegrasyon
Claude Desktop yapılandırma dosyasına şunu ekleyin:
```json
{
  "mcpServers": {
    "rethra": {
      "command": "python",
      "args": ["path/to/main.py"],
      "env": {
        "RETHRA_BASE_URL": "http://localhost:8080/api/v1",
        "RETHRA_API_KEY": "your_api_key"
      }
    }
  }
}
```

Proje deposu: https://github.com/acrbaran/rag/tree/main/mcp-server

### Diğer MCP istemcileriyle entegrasyon
Sunucu başlatma komutunu ve ortam değişkenlerini yapılandırmak için ilgili istemcinin belgelerine bakın.

## Sorun giderme

Sorunla karşılaşırsanız:
1. Ortamı kontrol etmek için `python main.py --check-only` çalıştırın
2. Ayrıntılı günlükleri görmek için `python main.py --verbose` kullanın
3. Rethra servisinin düzgün çalıştığını kontrol edin
4. Ağ bağlantısını ve güvenlik duvarı ayarlarını doğrulayın
