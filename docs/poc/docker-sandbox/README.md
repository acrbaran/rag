# Docker sandbox backend fizibilite PoC'si

Güncel uygulama ve dağıtım gereksinimleri için bkz. [Sandbox dağıtımı ve sorun giderme](../../../website-docs/06-development/04-sandbox-deployment.md). Bu PoC yalnızca geçmiş fizibilite deneyini korur, güncel adaptör davranışını temsil etmez.

Bu program doğrudan Docker Engine API'sini çağırır ve «Docker, Rethra'nın
`RemoteSandboxClient` sözleşmesini + E2B'nin Snapshot iş akışını taşıyabilir mi» sorusunu madde madde doğrular. Ürün kodu değildir, ana modül derlemesine de katılmaz
(kendi `go.mod` dosyası vardır); yalnızca araştırma sonuçlarının yeniden üretilebilir kanıtıdır.

Her madde `PASS/FAIL` ve gerçek gözlenen değeri yazdırır. `(GAP)` işaretli adımlar «Docker'ın neyi yapamadığını» doğrular;
bunlar da PASS ile biter — PASS, farkın yeniden üretildiği anlamına gelir, yeteneğin var olduğu anlamına gelmez.

## Çalıştırma

Erişilebilir bir Docker daemon'ı (`DOCKER_HOST` dikkate alınır) ve `python:3.11-slim` çekebilme gerekir:

```bash
cd docs/poc/docker-sandbox
go run .            # daemon yerel unix socket üzerindeyse sudo -E gerekebilir
```

Program, uid 1000 `user` hesabına sahip bir şablon imajı kendisi oluşturur (E2B şablon kuralına uygun) ve bitince oluşturduğu konteynerleri siler.
Kalan `rethra-poc/*` imajlarını `docker image rm` ile temizleyin.

## Kapsanan içerik

- Yaşam döngüsü: Create / Connect (istemci değiştirip yeniden bağlanma) / List (label filtresi) / Delete / Pause / Stop+Start
- Çalıştırma: user, workdir, env, stdin, stdout+stderr ayrımı, çıkış kodu, zaman aşımı
- Dosya yüzeyi: archive API ile okuma/yazma ve stat, exec ile mkdir/ls/rm
- Oturum semantiği: `pip install` ve dosya yazma exec'ler arasında kalıcıdır
- Snapshot: imaja commit, anlık görüntüden yeni sandbox başlatma, v1→v2 artımlı, label ile listeleme, katman sayısı ve boyut
- Farkların yeniden üretimi: istemci iptali süreci öldürmez, CapDrop ALL sonrası root izin bitlerini aşamaz, anlık görüntü bellek durumunu içermez,
  daemon'da boşta TTL yoktur, `docker run` öldürülünce hâlâ çalışan konteyner kalır
