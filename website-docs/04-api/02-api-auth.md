# API Başvurusu: Kimlik Doğrulama ve Kullanıcılar

Kayıt, giriş, belirteç yenileme, profil ve davet işleme arayüzleri sağlar. Kimlik doğrulama gereksinimleri arayüze göre değişir; herkese açık arayüzler her öğede belirtilmiştir.

Aksi özellikle belirtilmedikçe, bu gruptaki arayüzler kimlik doğrulama ara yazılımından sonra yalnızca “oturum açmış olmayı” gerektirir; asgari rol koşulu yoktur. Kimlik doğrulaması gerektirmeyen arayüzler her öğede belirtilmiştir.

## Kimlik Doğrulama (/api/v1/auth)

### POST /api/v1/auth/register

Amaç: Yeni kullanıcı kaydı (kendi kendine kayıt modu). Kimlik doğrulama gerekmez. İşleyici: `internal/handler/auth.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `username` | string | Evet | Kullanıcı adı, 2–50 karakter |
| `email` | string | Evet | E-posta |
| `password` | string | Evet | Parola (8–32 karakter, harf+rakam; karmaşık modda ayrıca büyük/küçük harf ve özel karakter gerekir) |

Kişisel alanın otomatik oluşturulup oluşturulmayacağı sunucudaki `auth.default_tenant_mode` tarafından belirlenir; istek gövdesinde belirtilemez. `invite_only` modunda 403 döner.

Yanıt: 201 `{"success":true,"message":"...","user":{User}}`

```bash
curl -X POST $BASE/api/v1/auth/register -H 'Content-Type: application/json' \
  -d '{"username":"alice","email":"a@ex.com","password":"secret123"}'
```

### POST /api/v1/auth/register-by-invite

Amaç: Davet/paylaşım bağlantısı token ile kayıt olmak ve alana katılmak. Kimlik doğrulama gerekmez, IP hız sınırı dakika başına 30 istektir. İşleyici: `internal/handler/auth_register_by_invite.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `token` | string | Evet (`binding:"required"`) | Davet token |
| `email` | string | Evet (`binding:"required,email"`) | E-posta |
| `username` | string | Evet (`binding:"required"`) | Kullanıcı adı |
| `password` | string | Evet (`binding:"required,min=6"`) | Parola (8–32 karakter, harf+rakam; karmaşık mod ayrıca büyük/küçük harf ve özel karakter gerektirir) |

Yanıt: 201, Login ile aynı (`user/active_tenant/memberships/token/refresh_token`).

```bash
curl -X POST $BASE/api/v1/auth/register-by-invite -H 'Content-Type: application/json' \
  -d '{"token":"<invite_token>","email":"a@ex.com","username":"alice","password":"secret123"}'
```

### POST /api/v1/auth/invitations/lookup

Amaç: Davet tokenine karşılık gelen alan bilgilerini anonim olarak sorgulamak (kayıt öncesi önizleme). Kimlik doğrulama gerekmez, IP hız sınırı uygulanır. Handler: `internal/handler/auth_register_by_invite.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `token` | string | Evet (`binding:"required"`) | Davet tokeni |

Yanıt: 200 `{"success":true,"data":{"tenant_id","tenant_name","role","expires_at"}}`

```bash
curl -X POST $BASE/api/v1/auth/invitations/lookup -H 'Content-Type: application/json' -d '{"token":"<invite_token>"}'
```

### POST /api/v1/auth/login

Amaç: E-posta ve parola ile giriş. Kimlik doğrulama gerekmez. Handler: `internal/handler/auth.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `email` | string | Evet (`binding:"required"`) | E-posta |
| `password` | string | Evet (`binding:"required"`) | Parola |

Yanıt: 200 `{"success":true,"user":{...},"active_tenant":{...},"memberships":[...],"token":"...","refresh_token":"..."}`

```bash
curl -X POST $BASE/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"a@ex.com","password":"secret123"}'
```

### GET /api/v1/auth/config

Amaç: Kayıt modu gibi kimlik doğrulama yapılandırmalarını sorgulamak. Kimlik doğrulama gerekmez. Handler: `internal/handler/auth.go`

Yanıt: 200 `{"success":true,"registration_mode":"self_serve|invite_only","complex_password_enabled":false}`

```bash
curl $BASE/api/v1/auth/config
```

### POST /api/v1/auth/switch-tenant

Amaç: Geçerli etkin alanı değiştirmek ve tokeni yenilemek. Oturum açılması gerekir (alan olmadan da çağrılabilir). Handler: `internal/handler/auth.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `tenant_id` | uint64 | Evet (`binding:"required"`) | Hedef alan ID'si |
| `refresh_token` | string | Hayır | Yeni token yenilemek için kullanılır |

Yanıt: 200, Login ile aynı.

Başarılı token yenileme, hedef alanı hesap düzeyindeki "son etkin kiracı" tercihine yazar; sonraki girişte (parola/OIDC/cihaz değişimi) ve refresh işleminde bu alana dönülür. Refresh JWT, `tenant_id` içermez; bu nedenle tercih yazımı başarısız olursa token yenileme işleminin tamamı başarısız olur ve yeni token verilmez. API istemcilerinin artık ek olarak `PUT /auth/me/preferences` göndermesine gerek yoktur. Web UI, alan değiştirirken bu arayüzü kullanmaz. Bir token yenileme, bu kullanıcının tüm cihazlarında sonraki varış noktasını değiştirir.

```bash
curl -X POST $BASE/api/v1/auth/switch-tenant -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"tenant_id":2}'
```

### GET /api/v1/auth/oidc/config

Amaç: OIDC'nin etkin olup olmadığını ve görünen adını sorgulamak. Kimlik doğrulama gerekmez. Handler: `internal/handler/auth.go`

Yanıt: 200 `{"success":true,"enabled":bool,"provider_display_name":"..."}`

```bash
curl $BASE/api/v1/auth/oidc/config
```

### GET /api/v1/auth/oidc/start

Oturum açma gerekmez; doğrudan 302 ve IdP'ye yönlendiren Location döndürülür, kurumsal portal bağlantılarında kullanılabilir. Önce JSON yetkilendirme adresi istemeye gerek yoktur; geri çağırma, istek origininden `/api/v1/auth/oidc/callback` olarak oluşturulur. Geri çağırma sonrası giriş sonucu, özgün OIDC akışıyla aynıdır.

```bash
curl -i "$BASE/api/v1/auth/oidc/start"
```

### GET /api/v1/auth/oidc/url

Amaç: OIDC yetkilendirme yönlendirme URL'sini almak. Kimlik doğrulama gerekmez. Handler: `internal/handler/auth.go`

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `redirect_uri` | string | Evet | Geri çağrı adresi |

Yanıt: 200 `{"success":true,"authorization_url":"...","nonce":"..."}`

```bash
curl "$BASE/api/v1/auth/oidc/url?redirect_uri=https://app.example.com/callback"
```

### GET /api/v1/auth/oidc/callback

Amaç: OIDC yetkilendirme geri çağrısı (tarayıcı yönlendirmesiyle girilir). Kimlik doğrulama gerekmez. Handler: `internal/handler/auth.go`

Sorgu parametreleri: `code`, `state`, `error`, `error_description` (tamamı OIDC sağlayıcısı tarafından geri gönderilir).

Yanıt: 302 ön uca yönlendirilir; başarıda `#oidc_result=<base64url>`, başarısızlıkta `#oidc_error=...` taşınır.

```bash
curl -i "$BASE/api/v1/auth/oidc/callback?code=xxx&state=yyy"
```

### POST /api/v1/auth/refresh

Amaç: refresh token kullanarak yeni token almak. Kimlik doğrulama gerekmez. Handler: `internal/handler/auth.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `refreshToken` | string | Evet (`binding:"required"`) | refresh token |

Yanıt: 200 `{"success":true,"access_token":"...","refresh_token":"..."}`

```bash
curl -X POST $BASE/api/v1/auth/refresh -H 'Content-Type: application/json' -d '{"refreshToken":"<rt>"}'
```

### GET /api/v1/auth/validate

Amaç: Mevcut tokenın geçerli olup olmadığını doğrulamak. Oturum açılması gerekir (çalışma alanı olmadan çağrılabilir). Handler: `internal/handler/auth.go`

Yanıt: 200 `{"success":true,"message":"Token is valid","user":{UserInfo}}`

```bash
curl $BASE/api/v1/auth/validate -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/auth/logout

Amaç: Oturumu kapatmak (mevcut tokenı geçersiz kılar). Oturum açılması gerekir. İstek gövdesi yoktur. Handler: `internal/handler/auth.go`

Yanıt: 200 `{"success":true,"message":"Logout successful"}`

```bash
curl -X POST $BASE/api/v1/auth/logout -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/auth/me

Amaç: Mevcut çağıranın kimliğini sorgulamak (kullanıcı/çalışma alanı/üyelik ilişkisi/yetenekler). Oturum açılması gerekir; API anahtarı da kullanılabilir (politika `apiKeyAny()`, herhangi bir geçerli anahtar). Handler: `internal/handler/auth.go`

Yanıt: 200 `{"success":true,"data":{"user":{UserInfo},"tenant":{TenantResponse},"memberships":[...],"tenant_required":bool,"capabilities":{"can_create_tenant":bool,"auto_accept_invitation":bool}}}`

```bash
curl $BASE/api/v1/auth/me -H "X-API-Key: $API_KEY"
```

### PUT /api/v1/auth/me/preferences

Amaç: Kişisel tercihleri güncellemek (en son etkin çalışma alanı). Oturum açılması gerekir. Handler: `internal/handler/auth.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `last_active_tenant_id` | *uint64 | Hayır | Pozitif tam sayı ayarlar/değiştirir; `0` temizler (sonraki girişte ana sayfaya döner); atlanırsa değiştirilmez. `POST /auth/switch-tenant` başarılı olduğunda sunucu aynı alanı yazar. |

Yanıt: 200 `{"success":true,"data":{UserPreferences}}`

```bash
curl -X PUT $BASE/api/v1/auth/me/preferences -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"last_active_tenant_id":2}'
```

Parola politikası `GET /auth/config` tarafından sağlanır. Karmaşık modda izin verilen özel karakter kümesi `!@#$%^&*()_+-=[]{}|;:,.<>?` şeklindedir; tam doğrulama kuralları yalnızca yapı binding uzunluk etiketlerinden çıkarılamaz.

### POST /api/v1/auth/change-password

Amaç: Parolayı değiştirmek. Oturum açılması gerekir. Handler: `internal/handler/auth.go`

| Alan | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `old_password` | string | Evet (`binding:"required"`) | Eski parola |
| `new_password` | string | Evet (`binding:"required"`) | Yeni parola (8–32 karakter, harf+rakam; karmaşık modda ayrıca büyük/küçük harf ve özel karakter gerekir) |

Yanıt: 200 `{"success":true,"message":"Password changed successfully"}`

```bash
curl -X POST $BASE/api/v1/auth/change-password -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"old_password":"old","new_password":"NewPass123!"}'
```

Parola değiştirmek için doğru eski parola verilmelidir; yeni parola eski parolayla aynı olamaz. Başarılı olduktan sonra tüm oturumlar iptal edilir ve yeniden giriş yapılması gerekir. Varsayılan politika 8–32 karakter, harfler ve rakamlardır; karmaşık mod büyük/küçük harf, rakam ve özel karakter gerektirir. Hata ayrıntıları `invalid_old_password`, `password_policy`, `same_password` olabilir; tümü 400 döndürür.

### POST /api/v1/me/invitations/accept-by-token

Giriş gereklidir, yalnızca mevcut kullanıcı üzerinde işlem yapılır; alanı olmayan yeni kullanıcılar da çağırabilir. İstek `{"token":"<invite-token>"}`; geçerli bir paylaşımlı davet, mevcut kullanıcıyı ilgili alana ekler. Geçersiz, süresi dolmuş veya iptal edilmiş bağlantılar 410 döndürür; boş token 400 döndürür. Başarılı yanıt `{success:true,data:{membership:{tenant_id,role,status,joined_at},tenant_name}}` şeklindedir; ön yüz buna göre alan değiştirir.

```bash
curl -X POST "$BASE/api/v1/me/invitations/accept-by-token" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"token":"<invite-token>"}'
```

## Davetlerim (/api/v1/me/invitations)

Servis katmanı “yalnızca davet edilen kişi kabul edebilir/reddedebilir” kuralını garanti eder; rol alt sınırı yoktur (alanı olmayan yeni kullanıcılar da kullanabilir). Handler: `internal/handler/tenant_invitation.go`

### GET /api/v1/me/invitations

Amaç: Bana gönderilen davetleri listelemek.

| Sorgu parametresi | Tür | Zorunlu | Açıklama |
| --- | --- | --- | --- |
| `include_terminal` | bool | Hayır | `true` olduğunda tamamlanmış davetleri içerir |

Yanıt: 200 `{"success":true,"data":{"invitations":[TenantInvitationResponse],"total":N}}`

```bash
curl $BASE/api/v1/me/invitations -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/me/invitations/pending-count

Amaç: Bekleyen davet sayısı (hafif yoklama).

Yanıt: 200 `{"success":true,"data":{"pending_count":N}}`

```bash
curl $BASE/api/v1/me/invitations/pending-count -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/me/invitations/:inv_id/accept

Amaç: Daveti kabul eder ve üyelik ilişkisini yazar. Yol parametresi: `inv_id` davet ID'si. İstek gövdesi yoktur.

Yanıt: 200 `{"success":true,"data":{"membership":{"tenant_id","role","status","joined_at"}}}`

```bash
curl -X POST $BASE/api/v1/me/invitations/12/accept -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/me/invitations/:inv_id/decline

Amaç: Daveti reddetmek. Yol parametresi: `inv_id`. İstek gövdesi yoktur.

Yanıt: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/me/invitations/12/decline -H "Authorization: Bearer $TOKEN"
```

## Uygulama referansı

Rota kaydı: `internal/router/routes_auth_tenant.go` içindeki `RegisterAuthRoutes` ve `RegisterMyInvitationRoutes`. Handler: `internal/handler/auth.go`, `internal/handler/auth_register_by_invite.go`, `internal/handler/tenant_invitation.go`.
