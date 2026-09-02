# Docs — eyedux-sdk-go

Documentação pública do SDK e guia de integração para clients.

| Documento | Audiência | Descrição |
|-----------|-----------|-----------|
| [client-integration.md](client-integration.md) | Clients do SDK | Instalação, configuração, envio de eventos, erros, idempotência e publicação com GitHub Pages |
| [api-reference.md](api-reference.md) | Devs que integram o SDK | Contrato completo da Public API do Eyedux: autenticação, todos os endpoints, rate limiting, todos os códigos de erro e modelos de dados |

## Fluxo recomendado de leitura

1. `client-integration.md` — integrar uma aplicação ao Eyedux
2. `api-reference.md` — entender **o que** o SDK encapsula

As decisões de arquitetura e propostas de evolução ficam em `internal/` e não
fazem parte do site público.
