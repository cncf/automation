# Landscape MCP Server

Easily ask questions of the CNCF (Or other!) Landscape. 

## Installation

Add this server to your mcp.json (or equivalent):

```
"cncf-landscape": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "ghcr.io/cncf/landscape-mcp-server:main",
        "--data-url",
        "https://landscape.cncf.io/data/full.json"
      ]
    }
```

## Examples

- How many CNCF projects graduated in 2024?
- When did OpenTelemetry reach incubating?
- What CNCF projects moved levels in 2025?