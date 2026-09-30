#!/usr/bin/env python3
"""
Rethra MCP Server Package

A Model Context Protocol server that provides access to the Rethra knowledge management API.
"""

__version__ = "1.1.1"
__author__ = "Rethra Team"
__description__ = "Rethra MCP Server - Model Context Protocol server for Rethra API"

from rethra_mcp_server import RethraClient, run

__all__ = ["RethraClient", "run"]
