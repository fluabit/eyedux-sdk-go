# eyedux-sdk-go

SDK oficial do Eyedux para Go. Framework-agnostic. Zero dependências externas além da stdlib.

## O que é

O Eyedux é uma plataforma de ingestão e consulta de eventos (logs) para aplicações. Este SDK encapsula a [Public API](docs/api-reference.md) do Eyedux, expondo uma interface idiomática em Go para:

- enviar eventos
- listar eventos da organização
- buscar um evento por ID externo

## Autenticação

Toda integração é feita via **API key da organização**, obtida no painel do Eyedux.

## Documentação

| Documento | Descrição |
|-----------|-----------|
| [docs/api-reference.md](docs/api-reference.md) | Contrato completo da Public API (fonte de verdade para o SDK) |
| [docs/sdk-design.md](docs/sdk-design.md) | Diretrizes de arquitetura, interface e convenções do SDK |

## Status

Em desenvolvimento.
