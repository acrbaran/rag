#!/usr/bin/env python3
"""
Rethra yerel MCP Demo Sunucusu

Rethra "Ayarlar → MCP Hizmetleri" bölümünde istemci bağlantısını test etmek için çalıştırılabilir en küçük harici MCP servisi.
Varsayılan olarak Streamable HTTP ile http://127.0.0.1:8010/mcp üzerinde dinler

Başlat:
  export MCP_SERVER_AUTH_TOKEN=rethra-demo-token
  python server.py

Rethra yapılandırması:
  Aktarım: HTTP Streamable
  URL：http://127.0.0.1:8010/mcp
  Kimlik doğrulama: Bearer, belirteç MCP_SERVER_AUTH_TOKEN ile aynı olmalıdır
"""

from __future__ import annotations

import argparse
import asyncio
import logging
import os
import secrets
import sys
from datetime import datetime, timezone
from typing import Any

from mcp.server import MCPServer

logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")
logger = logging.getLogger("mcp-demo")

mcp = MCPServer("rethra-mcp-demo", version="0.1.0")

# MCP araç çağrılarını test etmek için demo politika derlemi.
DEMO_POLICIES: dict[str, str] = {
    "warranty": "Akıllı Ev Merkezi Pro cihazının tamamı 24 ay, pil türü aksesuarlar ise 12 ay garantilidir; kullanıcı kaynaklı sökme ve su teması garanti kapsamı dışındadır.",
    "offline_voice": "Sesli komutlar bulut üzerinden tanınırsa, internet bağlantısı kesildiğinde yalnızca App ve yerel dokunmatik ekran desteklenir; yerel ses paketi yapılandırıldıktan sonra temel komutlar kullanılmaya devam edilebilir.",
    "device_limit": "Bireysel sürüm hesabına en fazla 3 merkez bağlanabilir; kurumsal sürüm sözleşme yetkilendirmesine göre, varsayılan olarak 50 cihaz içerir.",
    "travel_hotel_tier1": "Birinci kademe şehirlerde (Pekin, Şanghay, Guangzhou, Shenzhen) iş seyahati konaklama gider üst sınırı gecelik 600 yuandır (vergi dahil).",
    "travel_meal": "İş seyahati sırasında yemek giderleri ayrıca karşılanmaz; birinci kademe şehirlerde günlük 150 yuan harcırah verilir.",
    "poc_owner": "Satış sonrası bilgi tabanı POC'sinin teknik sorumlusu Ar-Ge departmanından Zhang Ming, ürün irtibat kişisi Li Wei, test sorumlusu Zhao Lei'dir.",
    "poc_deadline": "Satış sonrası bilgi tabanı POC hedefi, 2024-03-01 tarihinden önce intranet demosunu tamamlamaktır.",
    "matter_cert": "Firmware 3.5'in Mart 2024 sonundan önce kademeli olarak yayınlanması ve Matter 1.2 sertifikasının tamamlanması planlanıyor.",
}

DEMO_CONTACTS: list[dict[str, str]] = [
    {"name": "Chen Hao", "role": "Ürün Direktörü", "department": "Ürün Departmanı"},
    {"name": "Zhang Ming", "role": "Bilgi tabanı ve AI modülü sorumlusu", "department": "Ar-Ge Departmanı"},
    {"name": "Li Wei", "role": "Ürün Operasyonları", "department": "Ürün Departmanı"},
    {"name": "Wang Xue", "role": "Etkileşim tasarımı sorumlusu", "department": "Tasarım departmanı"},
    {"name": "Zhao Lei", "role": "Test müdürü", "department": "Test departmanı"},
]


def network_transport_auth_token() -> str:
    return os.getenv("MCP_SERVER_AUTH_TOKEN", "").strip()


def require_network_transport_auth(transport: str) -> str:
    token = network_transport_auth_token()
    if transport in ("sse", "http") and not token:
        logger.error(
            "MCP_SERVER_AUTH_TOKEN is required for %s transport. "
            "Example: export MCP_SERVER_AUTH_TOKEN=rethra-demo-token",
            transport,
        )
        sys.exit(1)
    return token


class MCPAuthMiddleware:
    """SSE / HTTP aktarımı için Bearer kimlik doğrulama ara yazılımı."""

    def __init__(self, app, token: str):
        self.app = app
        self.token = token

    async def __call__(self, scope, receive, send):
        if scope.get("type") != "http":
            await self.app(scope, receive, send)
            return

        headers = {
            k.decode("latin-1").lower(): v.decode("latin-1")
            for k, v in scope.get("headers", [])
        }
        provided = ""
        auth = headers.get("authorization", "")
        if auth.lower().startswith("bearer "):
            provided = auth[7:].strip()
        elif "x-mcp-auth-token" in headers:
            provided = headers["x-mcp-auth-token"]

        if not provided or not secrets.compare_digest(provided, self.token):
            body = b'{"error":"unauthorized"}'
            await send(
                {
                    "type": "http.response.start",
                    "status": 401,
                    "headers": [[b"content-type", b"application/json"]],
                }
            )
            await send({"type": "http.response.body", "body": body})
            return

        await self.app(scope, receive, send)


@mcp.tool()
def echo(message: str) -> dict[str, Any]:
    """MCP bağlantısını doğrulamak için bir mesajı geri yansıtır."""
    return {"echo": message}


@mcp.tool()
def add(a: float, b: float) -> dict[str, Any]:
    """İki sayının toplamını hesaplar."""
    return {"a": a, "b": b, "sum": a + b}


@mcp.tool()
def server_time() -> dict[str, str]:
    """MCP Demo sunucusunun geçerli UTC saatini döndürür."""
    now = datetime.now(timezone.utc)
    return {
        "iso": now.isoformat(),
        "unix": str(int(now.timestamp())),
    }


@mcp.tool()
def lookup_policy(topic: str) -> dict[str, Any]:
    """Demo politika/proje bilgilerini sorgular. topic için warranty/offline_voice/device_limit/travel_hotel_tier1/travel_meal/poc_owner/poc_deadline/matter_cert veya warranty/reimbursement/POC gibi anahtar sözcükler kullanılabilir."""
    key = topic.strip().lower().replace(" ", "_")
    aliases = {
        "Garanti": "warranty",
        "Kalite garantisi": "warranty",
        "Çevrimdışı": "offline_voice",
        "Ses": "offline_voice",
        "Cihaz sayısı": "device_limit",
        "Konaklama": "travel_hotel_tier1",
        "Masraf iadesi": "travel_hotel_tier1",
        "Yemek": "travel_meal",
        "Ödenek": "travel_meal",
        "Sorumlu": "poc_owner",
        "Zhang Ming": "poc_owner",
        "poc": "poc_owner",
        "Kabul": "poc_deadline",
        "matter": "matter_cert",
        "Sertifikasyon": "matter_cert",
    }
    for alias, mapped in aliases.items():
        if alias in topic:
            key = mapped
            break

    if key in DEMO_POLICIES:
        return {"topic": key, "answer": DEMO_POLICIES[key], "source": "mcp-demo/static"}

    matches = {
        k: v
        for k, v in DEMO_POLICIES.items()
        if key in k or any(ch in k for ch in key if len(key) >= 2)
    }
    if len(matches) == 1:
        only_key = next(iter(matches))
        return {"topic": only_key, "answer": matches[only_key], "source": "mcp-demo/static"}

    return {
        "topic": topic,
        "available_topics": sorted(DEMO_POLICIES.keys()),
        "hint": "topic olarak yukarıdaki anahtar adını veya “garanti”, “masraf iadesi”, “POC” gibi Türkçe anahtar kelimeleri iletin.",
    }


@mcp.tool()
def list_team_contacts(department: str = "") -> dict[str, Any]:
    """Demo proje ekibi üyelerini listeler; bölüm adına göre filtrelenebilir (Ürün Departmanı / Ar-Ge Departmanı / Tasarım Departmanı / Test Departmanı)."""
    rows = DEMO_CONTACTS
    if department.strip():
        needle = department.strip()
        rows = [c for c in rows if needle in c["department"]]
    return {"count": len(rows), "contacts": rows}


@mcp.tool()
def send_demo_alert(channel: str, message: str) -> dict[str, Any]:
    """Harici bir kanala bildirim göndermeyi simüle eder (demo amaçlıdır, gerçekten gönderilmez).

    Rethra içinde MCP aracı manuel onayını test etmek için uygundur: bu aracın onay gerektirecek şekilde işaretlenmesi önerilir.
    """
    return {
        "ok": True,
        "simulated": True,
        "channel": channel,
        "message": message,
        "sent_at": datetime.now(timezone.utc).isoformat(),
    }


async def run_http(host: str, port: int) -> None:
    auth_token = require_network_transport_auth("http")
    try:
        import uvicorn
    except ImportError as e:
        raise ImportError("HTTP transport requires: pip install starlette uvicorn") from e

    starlette_app = MCPAuthMiddleware(
        mcp.streamable_http_app(host=host, stateless_http=True),
        auth_token,
    )
    logger.info("Streamable HTTP MCP demo listening on http://%s:%d/mcp", host, port)
    config = uvicorn.Config(starlette_app, host=host, port=port, log_level="info")
    server = uvicorn.Server(config)
    await server.serve()


async def run_sse(host: str, port: int) -> None:
    auth_token = require_network_transport_auth("sse")
    try:
        import uvicorn
    except ImportError as e:
        raise ImportError("SSE transport requires: pip install starlette uvicorn") from e

    starlette_app = MCPAuthMiddleware(
        mcp.sse_app(host=host, message_path="/sse/messages/"),
        auth_token,
    )
    logger.info("SSE MCP demo listening on http://%s:%d/sse", host, port)
    config = uvicorn.Config(starlette_app, host=host, port=port, log_level="info")
    server = uvicorn.Server(config)
    await server.serve()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Rethra local MCP demo server")
    parser.add_argument(
        "--transport",
        choices=["http", "sse"],
        default=os.getenv("MCP_TRANSPORT", "http"),
        help="Network transport (default: http / Streamable HTTP)",
    )
    parser.add_argument("--host", default=os.getenv("MCP_HOST", "127.0.0.1"))
    parser.add_argument("--port", type=int, default=int(os.getenv("MCP_PORT", "8010")))
    return parser.parse_args()


async def main() -> None:
    args = parse_args()
    if args.transport == "http":
        await run_http(args.host, args.port)
    else:
        await run_sse(args.host, args.port)


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("stopped")
