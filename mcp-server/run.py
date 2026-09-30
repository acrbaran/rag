#!/usr/bin/env python3
"""
Rethra MCP Server hızlı başlatma betiği

Bu, en temel işlevleri sağlayan basitleştirilmiş bir başlatma betiğidir.
Daha fazla seçenek için main.py kullanın

Not: stdio aktarımında stdout JSON-RPC kanalıdır; tüm tanılama/bilgi iletileri şuraya yazılmalıdır
stderr, aksi hâlde MCP protokol akışı bozulur ve istemci "başlatma başarısız" olarak değerlendirir. Bu betikteki tüm print
stderr üzerinden çıktı verir.
"""

import os
import sys
from pathlib import Path


def main():
    """Basit başlatma işlevi"""
    # Geçerli dizini Python yoluna ekle
    current_dir = Path(__file__).parent.absolute()
    if str(current_dir) not in sys.path:
        sys.path.insert(0, str(current_dir))

    # Ortam değişkenlerini kontrol et
    base_url = os.getenv("RETHRA_BASE_URL", "http://localhost:8080/api/v1")
    api_key = os.getenv("RETHRA_API_KEY", "")

    print("Rethra MCP Server", file=sys.stderr)
    print(f"Base URL: {base_url}", file=sys.stderr)
    print(f"API Key: {'Ayarlanmış' if api_key else 'Ayarlanmamış'}", file=sys.stderr)
    print("-" * 40, file=sys.stderr)

    try:
        # İçe aktar ve çalıştır
        from main import sync_main

        sync_main()
    except ImportError:
        print("Hata: Gerekli modüller içe aktarılamıyor", file=sys.stderr)
        print("Lütfen şunu çalıştırdığınızdan emin olun: pip install -r requirements.txt", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nSunucu durduruldu", file=sys.stderr)
    except Exception as e:
        print(f"Hata: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
