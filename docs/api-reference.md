# Eyedux Public API — Referência Completa

> Este documento é a fonte de verdade para o desenvolvimento do SDK. Consolida todos os contratos da Public API: autenticação, endpoints, rate limiting e códigos de erro.

## Índice

1. [Base URL e ambiente](#base-url-e-ambiente)
2. [Autenticação](#autenticação)
3. [Envelope de resposta](#envelope-de-resposta)
4. [Rate limiting](#rate-limiting)
5. [Endpoints](#endpoints)
   - [POST /public/logs — Criar evento](#post-publiclogs--criar-evento)
   - [GET /public/logs — Listar eventos](#get-publiclogs--listar-eventos)
   - [GET /public/logs/external/:external_id — Buscar por ID externo](#get-publiclogsexternalexternal_id--buscar-por-id-externo)
6. [Modelos de dados](#modelos-de-dados)
7. [Códigos de erro](#códigos-de-erro)

---

## Base URL e ambiente

Todos os clientes utilizam a mesma base URL:

```
https://api.eyedux.com/
```

A URL é fixa no SDK — o integrador não precisa (e não pode) configurá-la.

---

## Autenticação

Todas as rotas da Public API exigem o header:

```
Authorization: Bearer <api_key>
```

A `api_key` é estática — gerada no painel do Eyedux e vinculada a uma organização. O uso de `Bearer` para API keys estáticas é intencional e alinhado com o padrão da indústria (Stripe, OpenAI, Anthropic, GitHub). O backend infere a organização automaticamente a partir da chave — o integrador **não envia** `organization_id` no body.

O SDK deve injetar o header `Authorization: Bearer <apiKey>` em todas as requisições automaticamente, a partir da chave configurada no cliente.

### Comportamento em caso de falha de autenticação

| Cenário | Status | Código de erro |
|--------|--------|----------------|
| Header `Authorization` ausente ou sem `Bearer` | `400 Bad Request` | — |
| API key inválida ou revogada | `422 Unprocessable Entity` | `invalid_api_key` |

---

## Envelope de resposta

### Sucesso

```json
{
  "data": { ... }
}
```

Ou, para listas:

```json
{
  "data": [ ... ]
}
```

### Erro

```json
{
  "error": {
    "code": "<código_legível_por_máquina>",
    "message": "<mensagem_textual>"
  }
}
```

O SDK deve sempre inspecionar o campo `error.code` para categorizar erros. O campo `message` é apenas informativo.

---

## Rate limiting

O rate limiting é aplicado por API key em todas as rotas `/public/...`.

### Limites padrão do servidor

| Parâmetro | Valor padrão | Descrição |
|-----------|-------------|-----------|
| RPS | `10` | Requisições por segundo por API key |
| Burst | `20` | Pico momentâneo máximo acima do RPS |

Esses valores são configurados no servidor — o SDK não os controla, apenas reage ao `429`.

### Resposta quando o limite é excedido

```
HTTP 429 Too Many Requests
Retry-After: 1
```

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "too many requests"
  }
}
```

### Comportamento esperado do SDK

- Identificar `429` como erro de rate limit.
- Expor o valor do header `Retry-After` na estrutura de erro quando presente.
- **Não implementar retry automático por padrão** — deixar a decisão de retry para o código do integrador.
- Expor uma interface opcional de retry (ex: `WithRetryPolicy`) para quem quiser ativar comportamento automático.

---

## Endpoints

### POST /public/logs — Criar evento

Ingere um novo evento na organização autenticada.

#### Request

```
POST /public/logs
Authorization: Bearer <api_key>
Content-Type: application/json
```

**Body**

| Campo | Tipo | Obrigatório | Regra |
|-------|------|-------------|-------|
| `project_id` | `string` | Sim | ObjectID hexadecimal (24 chars) |
| `type` | `string` | Sim | Categoria do evento; não pode ser vazio |
| `type_group` | `string` | Não | Grupo da categoria do evento |
| `eyedux_type` | `string` | Não | Tipo predefinido do Eyedux |
| `properties` | `object` | Sim | Payload livre; não pode ser vazio (`{}` é rejeitado) |
| `external_object` | `object` | Não | Referência externa (`id`, `property` e `source`) |
| `correlation_object` | `object` | Não | Referência correlacionada (`id`, `property` e `source`) |
| `metadata` | `object` | Não | Dados de contexto livres (IP, user agent, etc.) |

**Exemplo de body**

```json
{
  "project_id": "64f1a2b3c4d5e6f7a8b9c0d1",
  "type": "user.signup",
  "type_group": "identity",
  "eyedux_type": "system-log",
  "properties": {
    "plan": "pro",
    "source": "landing_page"
  },
  "external_object": {
    "id": "evt_01HX92K",
    "property": "orderId"
  },
  "correlation_object": {
    "id": "session_abc123",
    "property": "sessionId"
  },
  "metadata": {
    "ip": "192.168.1.1",
    "user_agent": "Mozilla/5.0"
  }
}
```

#### Resposta de sucesso

```
HTTP 201 Created
```

```json
{
  "data": {
    "id": "64f1a2b3c4d5e6f7a8b9c0d1",
    "environment": "production",
    "eyedux_type": "system-log",
    "type": "user.signup",
    "type_group": "identity",
    "properties": { "plan": "pro", "source": "landing_page" },
    "status": "active",
    "timestamp": "2026-08-13T10:00:00Z",
    "created_at": "2026-08-13T10:00:01Z",
    "external_object": { "id": "evt_01HX92K", "property": "orderId" },
    "correlation_object": { "id": "session_abc123", "property": "sessionId" },
    "metadata": { "ip": "192.168.1.1", "user_agent": "Mozilla/5.0" }
  }
}
```

| Campo | Tipo | Presença |
|-------|------|----------|
| `id` | `string` | Sempre |
| `environment` | `string` | Sempre; ambiente da API key |
| `eyedux_type` | `string` ou `null` | Sempre |
| `type` | `string` | Sempre |
| `type_group` | `string` | Sempre |
| `properties` | `object` | Sempre |
| `status` | `string` | Sempre (`active` na criação) |
| `timestamp` | `string` (RFC3339) | Sempre |
| `created_at` | `string` (RFC3339) | Sempre |
| `external_object` | `object` ou `null` | Sempre |
| `correlation_object` | `object` ou `null` | Sempre |
| `metadata` | `object` ou `null` | Sempre |

#### Respostas de erro

| Status | Cenário | Código |
|--------|---------|--------|
| `400` | JSON inválido, `project_id` ausente ou formato inválido, `type` ou `properties` ausentes | — |
| `409` | `external_object` já utilizado por outro evento | `event_external_object_conflict` |
| `422` | API key inválida/revogada; `type` enviado mas vazio; `properties` enviado mas vazio | `invalid_api_key` / `event_type_required` / `event_properties_empty` |
| `429` | Rate limit excedido | `RATE_LIMIT_EXCEEDED` |
| `500` | Falha interna do servidor | `INTERNAL_SERVER_ERROR` |

---

### GET /public/logs — Listar eventos

Lista todos os eventos da organização autenticada, com filtros opcionais.

#### Request

```
GET /public/logs
Authorization: Bearer <api_key>
```

**Query params**

| Parâmetro | Tipo | Obrigatório | Regra |
|-----------|------|-------------|-------|
| `type` | `string` | Não | Filtra por tipo de evento |
| `correlation_id` | `string` | Não | Filtra por correlation ID |

Filtros são cumulativos quando ambos são enviados.

#### Resposta de sucesso

```
HTTP 200 OK
```

```json
{
  "data": [
    {
      "id": "64f1a2b3c4d5e6f7a8b9c0d1",
      "type": "user.signup",
      "properties": { "plan": "pro" },
      "status": "active",
      "timestamp": "2026-08-13T10:00:00Z",
      "created_at": "2026-08-13T10:00:01Z",
      "external_object": null,
      "correlation_object": null,
      "metadata": null
    }
  ]
}
```

Campos por item idênticos ao retorno de criação (campos opcionais retornam `null`, nunca omitidos). Ordenação: `created_at` decrescente.

Não há paginação neste endpoint. O backend retorna todos os eventos que correspondem aos filtros em um único array. `data` pode ser `[]` quando não há resultados — isso não é um erro.

#### Respostas de erro

| Status | Cenário | Código |
|--------|---------|--------|
| `400` | Header `Authorization` ausente ou malformado | — |
| `422` | API key inválida ou revogada | `invalid_api_key` |
| `429` | Rate limit excedido | `RATE_LIMIT_EXCEEDED` |
| `500` | Falha interna do servidor | `INTERNAL_SERVER_ERROR` |

---

### GET /public/logs/external/:external_id — Buscar por ID externo

Busca um único evento pelo seu `external_id`.

#### Request

```
GET /public/logs/external/<external_id>
Authorization: Bearer <api_key>
```

**Path param**

| Parâmetro | Tipo | Obrigatório | Regra |
|-----------|------|-------------|-------|
| `external_id` | `string` | Sim | ID dentro de `external_object`; não pode ser vazio |

#### Resposta de sucesso

```
HTTP 200 OK
```

```json
{
  "data": {
    "id": "64f1a2b3c4d5e6f7a8b9c0d1",
    "type": "user.signup",
    "properties": { "plan": "pro" },
    "status": "active",
    "timestamp": "2026-08-13T10:00:00Z",
    "created_at": "2026-08-13T10:00:01Z",
    "external_id": "evt_01HX92K",
    "correlation_id": null,
    "metadata": null
  }
}
```

A busca é **global** — não filtra por organização ou projeto. O ID externo deve ser suficientemente único na integração do cliente.

#### Respostas de erro

| Status | Cenário | Código |
|--------|---------|--------|
| `400` | Header `Authorization` ausente ou malformado | — |
| `404` | Nenhum evento com o `external_id` informado | `event_external_id_not_found` |
| `422` | API key inválida/revogada; `external_id` vazio no path | `invalid_api_key` / `event_external_id_required` |
| `429` | Rate limit excedido | `RATE_LIMIT_EXCEEDED` |
| `500` | Falha interna do servidor | `INTERNAL_SERVER_ERROR` |

---

## Modelos de dados

### Event

Representa um evento retornado pela API. Campos opcionais são sempre presentes no JSON, com valor `null` quando não preenchidos.

| Campo | Tipo Go sugerido | Observação |
|-------|-----------------|------------|
| `id` | `string` | ObjectID hexadecimal |
| `type` | `string` | |
| `properties` | `map[string]any` | |
| `status` | `string` | `"active"` ou `"deleted"` |
| `timestamp` | `time.Time` | RFC3339 |
| `created_at` | `time.Time` | RFC3339 |
| `environment` | `string` | Ambiente da API key |
| `eyedux_type` | `*EventEyeduxType` | `null` quando não definido |
| `type_group` | `string` | Vazio quando não definido |
| `external_object` | `*EventObject` | `null` quando não definido |
| `correlation_object` | `*EventObject` | `null` quando não definido |
| `metadata` | `map[string]any` | `null` quando não definido |

### CreateEventInput

Entrada para criação de evento.

| Campo | Tipo Go sugerido | Obrigatório |
|-------|-----------------|-------------|
| `ProjectID` | `string` | Sim |
| `Type` | `string` | Sim |
| `Properties` | `map[string]any` | Sim |
| `TypeGroup` | `string` | Não |
| `EyeduxType` | `EventEyeduxType` | Não |
| `ExternalObject` | `*EventObject` | Não |
| `CorrelationObject` | `*EventObject` | Não |
| `Metadata` | `map[string]any` | Não |

### ListEventsInput

Filtros para listagem de eventos.

| Campo | Tipo Go sugerido | Obrigatório |
|-------|-----------------|-------------|
| `Type` | `*string` | Não |
| `CorrelationID` | `*string` | Não |

### EventEyeduxType

O SDK expõe os tipos predefinidos de evento como constantes descobríveis pelo
autocomplete:

| Constante | Valor JSON |
|-----------|------------|
| `EventEyeduxTypeSystemError` | `system-error` |
| `EventEyeduxTypeSystemWarning` | `system-warning` |
| `EventEyeduxTypeSystemLog` | `system-log` |
| `EventEyeduxTypeSystemDebug` | `system-debug` |
| `EventEyeduxTypeSystemInfo` | `system-info` |
| `EventEyeduxTypeSystemMetric` | `system-metric` |

`CreateEventInput.EyeduxType` usa o valor vazio quando o campo não deve ser
enviado. `Event.EyeduxType` é `nil` quando a API retorna `null`.

---

## Códigos de erro

Tabela consolidada de todos os códigos de erro conhecidos da Public API.

| Código | Status HTTP | Descrição |
|--------|-------------|-----------|
| `invalid_api_key` | `422` | API key inválida ou revogada |
| `event_type_required` | `422` | Campo `type` enviado mas vazio |
| `event_properties_empty` | `422` | Campo `properties` enviado mas vazio |
| `event_external_object_conflict` | `409` | `external_object` duplicado |
| `event_external_id_not_found` | `404` | Nenhum evento com o `external_id` informado |
| `event_external_id_required` | `422` | `external_id` vazio no path param |
| `RATE_LIMIT_EXCEEDED` | `429` | Rate limit excedido |
| `INTERNAL_SERVER_ERROR` | `500` | Erro interno não mapeado |
