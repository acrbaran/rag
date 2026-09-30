# Rethra MCP sunucusunu uv ile çalıştırma

> Python tabanlı MCP servislerini çalıştırmak için `uv` önerilir.
>
> PyPI üzerinden de kurulabilir: `pip install rethra-mcp` veya `uvx --from rethra-mcp rethra-mcp-server` (resmi paket adı `rethra-mcp`, bakımı [acrbaran/rag](https://github.com/acrbaran/rag) tarafından yapılır).

## 1. uv kurulumu

```bash
# macOS/Linux
curl -LsSf https://astral.sh/uv/install.sh | sh

# veya Homebrew ile (macOS)
brew install uv

# Windows
powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"
```

## 2. MCP istemci yapılandırması

### Claude Desktop yapılandırması

Claude Desktop ayarlarına şunu ekleyin:

```json
{
  "mcpServers": {
    "rethra": {
      "args": [
        "--directory",
        "/path/Rethra/mcp-server",
        "run",
        "run_server.py"
      ],
      "command": "uv",
      "env": {
        "RETHRA_API_KEY": "your_api_key_here",
        "RETHRA_BASE_URL": "http://localhost:8080/api/v1"
      }
    }
  }
}
```

### Cursor yapılandırması

Cursor'da MCP yapılandırma dosyasını düzenleyin (genellikle `~/.cursor/mcp-config.json`):

```json
{
  "mcpServers": {
    "rethra": {
      "command": "uv",
      "args": [
        "--directory",
        "/path/Rethra/mcp-server",
        "run",
        "run_server.py"
      ],
      "env": {
        "RETHRA_API_KEY": "your_api_key_here",
        "RETHRA_BASE_URL": "http://localhost:8080/api/v1"
      }
    }
  }
}
```

### KiloCode yapılandırması

KiloCode veya MCP destekleyen diğer editörler için yapılandırma şöyledir:

```json
{
  "mcpServers": {
    "rethra": {
      "command": "uv",
      "args": [
        "--directory",
        "/path/Rethra/mcp-server",
        "run",
        "run_server.py"
      ],
      "env": {
        "RETHRA_API_KEY": "your_api_key_here",
        "RETHRA_BASE_URL": "http://localhost:8080/api/v1"
      }
    }
  }
}
```

### Diğer MCP istemcileri

Genel MCP istemci yapılandırması için:

```json
{
  "mcpServers": {
    "rethra": {
      "command": "uv",
      "args": [
        "--directory",
        "/path/Rethra/mcp-server",
        "run",
        "run_server.py"
      ],
      "env": {
        "RETHRA_API_KEY": "your_api_key_here",
        "RETHRA_BASE_URL": "http://localhost:8080/api/v1"
      }
    }
  }
}
```
