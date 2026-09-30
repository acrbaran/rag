#!/usr/bin/env python3
"""
Rethra MCP Server başlatma betiği

Not: stdio aktarımında stdout JSON-RPC kanalıdır; tüm tanılama/bilgi iletileri mutlaka
stderr'e yazılmalıdır, aksi takdirde MCP protokol akışı bozulur ve istemci "başlatma başarısız" olarak değerlendirir. Bu betikteki tüm print
stderr üzerinden çıktı verir.
"""

import asyncio
import os
import sys


def check_environment():
    """Ortam yapılandırmasını denetle"""
    base_url = os.getenv("RETHRA_BASE_URL")
    api_key = os.getenv("RETHRA_API_KEY")

    if not base_url:
        print(
            "Uyarı: RETHRA_BASE_URL ortam değişkeni ayarlanmadı, varsayılan değer kullanılıyor: http://localhost:8080/api/v1",
            file=sys.stderr,
        )

    if not api_key:
        print("Uyarı: RETHRA_API_KEY ortam değişkeni ayarlanmadı", file=sys.stderr)

    print(f"Rethra Base URL: {base_url or 'http://localhost:8080/api/v1'}", file=sys.stderr)
    print(f"API Anahtarı: {'ayarlandı' if api_key else 'ayarlanmadı'}", file=sys.stderr)


def main():
    """Ana fonksiyon"""
    print("Rethra MCP Server başlatılıyor...", file=sys.stderr)
    check_environment()

    try:
        from rethra_mcp_server import run

        asyncio.run(run())
    except ImportError as e:
        print(f"İçe aktarma hatası: {e}", file=sys.stderr)
        print("Lütfen tüm bağımlılıkların yüklü olduğundan emin olun: pip install -r requirements.txt", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nSunucu durduruldu", file=sys.stderr)
    except Exception as e:
        print(f"Sunucu çalışma hatası: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
