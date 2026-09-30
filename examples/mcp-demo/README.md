# Rethra yerel MCP Demo

Rethra'nın **MCP istemcisi olarak** üçüncü taraf araçlara bağlanmasını test etmek için kullanılan en küçük harici MCP servisi.

6 demo aracı sunar:

| Araç | İşlevi |
| --- | --- |
| `echo` | Bağlantı öz denetimi |
| `add` | İki sayıyı toplar |
| `server_time` | Sunucunun UTC saatini döndürür |
| `lookup_policy` | Demo politikalarını sorgular (garanti, masraf iadesi, POC vb.) |
| `list_team_contacts` | Demo proje ekibinin üyelerini listeler |
| `send_demo_alert` | Dışarıya bildirim göndermeyi taklit eder (araçların manuel onayını test etmek için uygundur) |

## 1. Başlatma

```bash
cd examples/mcp-demo
chmod +x start.sh
./start.sh
```

`start.sh` otomatik olarak `.venv` oluşturur ve bağımlılıkları kurar. Varsayılan olarak `http://127.0.0.1:8010/mcp` adresini dinler; kimlik doğrulama belirteci `rethra-demo-token` olur.

Özelleştirme:

```bash
export MCP_SERVER_AUTH_TOKEN=my-secret
export MCP_PORT=9000
./start.sh
```

## 2. Öz denetim

Yeni bir terminal açın:

```bash
cd examples/mcp-demo
source .venv/bin/activate
python test_tools.py
```

6 araç listelenmelidir.

## 3. Rethra'ya bağlama

1. **Ayarlar → MCP servisleri → Yeni** menüsünü açın
2. Şunları doldurun:

| Alan | Değer |
| --- | --- |
| Ad | `Yerel MCP Demo` |
| Aktarım | **HTTP Streamable** |
| URL | `http://127.0.0.1:8010/mcp` |
| Kimlik doğrulama | **Bearer** |
| Belirteç | `rethra-demo-token` (`MCP_SERVER_AUTH_TOKEN` ile aynı) |

3. Kaydettikten sonra **Bağlantıyı test et** düğmesine basın; 6 araç bulunmalıdır.
4. **Ajan** yapılandırmasında bu MCP servisini seçin (veya tüm araçları seçin).
5. (İsteğe bağlı) `send_demo_alert` için **manuel onayı** açın; sohbette Agent aracı çağırmadan önce bir onay penceresi açılır.

## 4. Denenebilecek sorular

Agent sohbetinde şunları sorun:

- "MCP aracını çağırarak akıllı ev kontrol merkezinin garanti süresini bul" → `lookup_policy` tetiklenmeli
- "Ar-Ge departmanının POC sorumlusu kim?" → `lookup_policy` veya `list_team_contacts`
- "MCP Demo sunucusunda saat kaç?" → `server_time`

## 5. Dikkat edilecekler

- Rethra arayüzü **stdio** aktarımını desteklemez; **HTTP Streamable** veya **SSE** kullanılmalıdır.
- Demo yalnızca `127.0.0.1` adresine bağlanır; herkese açık ağa açmayın.
- `send_demo_alert` gerçekten mesaj göndermez, yalnızca sahte bir sonuç döndürür.

## 6. SSE modu (isteğe bağlı)

```bash
MCP_TRANSPORT=sse MCP_PORT=8011 ./start.sh
```

Rethra'da aktarım olarak **SSE** seçin, URL alanına `http://127.0.0.1:8011/sse` yazın.
