#!/usr/bin/env python3
"""
Manuel MCP içe aktarma denetim betiği (unittest testi değildir).

Kullanım: python check_imports.py
"""

try:
    import mcp.server.stdio  # noqa: F401

    print("✓ mcp.server.stdio içe aktarıldı")
except ImportError as e:
    print(f"✗ mcp.server.stdio içe aktarılamadı: {e}")

try:
    # mcp 2.x high-level server API (MCPServer, formerly FastMCP).
    from mcp.server import MCPServer

    print("✓ MCPServer mcp.server içinden başarıyla içe aktarıldı")
except ImportError as e:
    print(f"✗ MCPServer içe aktarılamadı: {e}")

try:
    from mcp.server.mcpserver.exceptions import ToolError  # noqa: F401

    print("✓ ToolError mcp.server.mcpserver.exceptions içinden başarıyla içe aktarıldı")
except ImportError as e:
    print(f"✗ ToolError içe aktarılamadı: {e}")

try:
    import mcp_types  # noqa: F401  # standalone protocol types in mcp 2.x

    print("✓ mcp_types içe aktarıldı")
except ImportError as e:
    print(f"✗ mcp_types içe aktarılamadı: {e}")

# MCP paket yapısını denetle
import mcp

print(f"\nMCP paket sürümü: {getattr(mcp, '__version__', 'bilinmiyor')}")
print(f"MCP paket yolu: {mcp.__file__}")
