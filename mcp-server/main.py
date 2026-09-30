#!/usr/bin/env python3
"""
Rethra MCP Server ana giriş noktası

Bu dosya, Rethra MCP sunucusunu başlatmak için birleşik bir giriş noktası sağlar.
Çeşitli şekillerde çalıştırılabilir:
1. python main.py
2. python -m rethra_mcp_server
3. rethra-mcp-server (kurulumdan sonra)

Not: stdio aktarımında stdout JSON-RPC kanalıdır; tüm tanılama/bilgi iletileri mutlaka
stderr'e yazılmalıdır, aksi takdirde MCP protokol akışı bozulur ve istemci "başlatma başarısız" olarak değerlendirir. Bu dosyadaki tüm print
stderr üzerinden çıktı verir.
"""

import argparse
import asyncio
import os
import sys
from pathlib import Path


def setup_environment():
    """Ortamı ve yolları ayarla"""
    # Geçerli dizinin Python yolunda olduğundan emin ol
    current_dir = Path(__file__).parent.absolute()
    if str(current_dir) not in sys.path:
        sys.path.insert(0, str(current_dir))


def check_dependencies():
    """Bağımlılıkların kurulu olup olmadığını kontrol et"""
    try:
        import mcp
        import requests

        return True
    except ImportError as e:
        print(f"Eksik bağımlılık: {e}", file=sys.stderr)
        print("Lütfen çalıştırın: pip install -r requirements.txt", file=sys.stderr)
        return False


def check_environment_variables():
    """Ortam değişkeni yapılandırmasını kontrol et"""
    base_url = os.getenv("RETHRA_BASE_URL")
    api_key = os.getenv("RETHRA_API_KEY")

    print("=== Rethra MCP Server ortam kontrolü ===", file=sys.stderr)
    print(f"Base URL: {base_url or 'http://localhost:8080/api/v1 (varsayılan)'}", file=sys.stderr)
    print(f"API Anahtarı: {'ayarlandı' if api_key else 'ayarlanmadı (uyarı)'}", file=sys.stderr)

    if not base_url:
        print("İpucu: RETHRA_BASE_URL ortam değişkenini ayarlayabilirsiniz", file=sys.stderr)

    if not api_key:
        print("Uyarı: RETHRA_API_KEY ortam değişkenini ayarlamanız önerilir", file=sys.stderr)

    print("=" * 40, file=sys.stderr)
    return True


def parse_arguments():
    """Komut satırı argümanlarını ayrıştır"""
    parser = argparse.ArgumentParser(
        description="Rethra MCP Server - Model Context Protocol server for Rethra API",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Örnek:
  python main.py                    # Varsayılan yapılandırmayla başlat
  python main.py --check-only       # Yalnızca ortamı kontrol et, sunucuyu başlatma
  python main.py --verbose          # Ayrıntılı günlükleri etkinleştir
  
Ortam değişkenleri:
  RETHRA_BASE_URL       Rethra API temel URL'si (varsayılan: http://localhost:8080/api/v1)
  RETHRA_API_KEY        Rethra API anahtarı
  MCP_SERVER_AUTH_TOKEN  SSE/HTTP aktarımı için zorunludur, istemci Authorization: Bearer aracılığıyla iletir
        """,
    )

    parser.add_argument(
        "--check-only", action="store_true", help="Yalnızca ortam yapılandırmasını kontrol et, sunucuyu başlatma"
    )

    parser.add_argument("--verbose", "-v", action="store_true", help="Ayrıntılı günlük çıktısını etkinleştir")

    parser.add_argument(
        "--version", action="version", version="Rethra MCP Server 1.1.1"
    )

    parser.add_argument(
        "--transport",
        choices=["stdio", "sse", "http"],
        default=os.getenv("MCP_TRANSPORT", "stdio"),
        help="Transport type: stdio (default), sse, or http",
    )
    parser.add_argument(
        "--host",
        default=os.getenv("MCP_HOST", "127.0.0.1"),
        help="Bind host for network transports (default: 127.0.0.1)",
    )
    parser.add_argument(
        "--port",
        type=int,
        default=int(os.getenv("MCP_PORT", "8000")),
        help="Bind port for network transports (default: 8000)",
    )

    return parser.parse_args()


async def main():
    """Ana fonksiyon"""
    args = parse_arguments()

    # Ortamı ayarla
    setup_environment()

    # Bağımlılıkları kontrol et
    if not check_dependencies():
        sys.exit(1)

    # Ortam değişkenlerini kontrol et
    check_environment_variables()

    # Yalnızca ortam kontrolü yapılıyorsa çık
    if args.check_only:
        print("Ortam kontrolü tamamlandı.", file=sys.stderr)
        return

    # Günlük kaydı seviyesini ayarla
    if args.verbose:
        import logging

        logging.basicConfig(level=logging.DEBUG)
        print("Ayrıntılı günlük modu etkinleştirildi", file=sys.stderr)

    try:
        print(f"Rethra MCP Server başlatılıyor (transport={args.transport})...", file=sys.stderr)

        from rethra_mcp_server import run_stdio, run_sse, run_http

        # Select transport mode based on CLI argument or MCP_TRANSPORT env var
        # - stdio: Default, used by VS Code Copilot for local integration
        # - sse: Server-Sent Events over HTTP, suitable for cloud/remote deployments
        # - http: Streamable HTTP sessions (MCP 2025-03-26 spec), compatible with REST clients
        if args.transport == "stdio":
            # Stdio mode: communication via stdin/stdout pipes (typical for CLI integrations)
            await run_stdio()
        elif args.transport == "sse":
            # SSE mode: HTTP server with Server-Sent Events for bidirectional streaming
            await run_sse(args.host, args.port)
        elif args.transport == "http":
            # HTTP mode: HTTP REST server with request/response model
            await run_http(args.host, args.port)

    except ImportError as e:
        print(f"İçe aktarma hatası: {e}", file=sys.stderr)
        print("Lütfen tüm dosyaların doğru konumda olduğundan emin olun", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nSunucu durduruldu", file=sys.stderr)
    except Exception as e:
        print(f"Sunucu çalışma hatası: {e}", file=sys.stderr)
        if args.verbose:
            import traceback

            traceback.print_exc()
        sys.exit(1)


def sync_main():
    """entry_points için eşzamanlı sürümün ana fonksiyonu"""
    asyncio.run(main())


if __name__ == "__main__":
    asyncio.run(main())
