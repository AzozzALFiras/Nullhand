# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| latest  | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in Nullhand, please report it responsibly:

1. **Do NOT open a public issue.**
2. Email the details to the maintainer or use [GitHub's private vulnerability reporting](https://github.com/AzozzALFiras/nullhand/security/advisories/new).
3. Include steps to reproduce the vulnerability if possible.
4. Allow reasonable time for a fix before public disclosure.

You can expect an initial response within **72 hours**.

## Security Architecture

Nullhand is a **personal-use, self-hosted** agent that runs locally on your Mac. Key security properties:

- **Telegram whitelist** — Only your Telegram user ID can send commands. All other messages are silently ignored.
- **No public IP** — Your Mac never exposes any ports. Nullhand polls Telegram; the connection is always outbound.
- **Config file protection** — `~/.nullhand/` is `700` and `config.json` is `600` (owner-only access).
- **Shell command whitelist** — Only whitelisted commands can be executed. `sudo`, `bash`, and `sh` are blocked.
- **Local AI option** — Using Ollama, all AI processing stays on your Mac with zero external data sharing.

For full details, see [docs/privacy.md](docs/privacy.md).
