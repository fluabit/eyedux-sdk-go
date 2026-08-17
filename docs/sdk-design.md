# SDK Design — Diretrizes de Arquitetura

> Este documento define as decisões de design, convenções de interface e regras de implementação do `eyedux-sdk-go`. Deve ser lido antes de qualquer contribuição de código.

## Índice

1. [Princípios gerais](#princípios-gerais)
2. [Estrutura de pacotes](#estrutura-de-pacotes)
3. [Cliente](#cliente)
4. [Configuração](#configuração)
5. [Métodos do cliente](#métodos-do-cliente)
6. [Tipos públicos](#tipos-públicos)
7. [Tratamento de erros](#tratamento-de-erros)
8. [HTTP e transport](#http-e-transport)
9. [Testes](#testes)
10. [Convenções de código](#convenções-de-código)

---

## Princípios gerais

- **Framework-agnostic**: zero dependências além da stdlib do Go. Nenhum framework HTTP, nenhuma lib de serialização externa.
- **Sem magia**: a interface pública deve ser previsível. O que está escrito na assinatura do método é o que acontece.
- **Sem retry automático por padrão**: o integrador decide como lidar com falhas transitórias. O SDK expõe os erros de forma rica o suficiente para que o integrador implemente sua própria política.
- **Configuração explícita**: todas as opções relevantes são configuráveis; nenhum comportamento depende de variáveis de ambiente.
- **Context-first**: todos os métodos que fazem I/O aceitam `context.Context` como primeiro argumento.

---

## Estrutura de pacotes

```
eyedux-sdk-go/
├── eyedux.go          # Client, New, opções de configuração
├── event.go           # Tipos Event, CreateEventInput, ListEventsInput
├── errors.go          # Tipos de erro, APIError, códigos de erro como constantes
├── http.go            # Lógica de transporte HTTP (interno — não exportado)
├── eyedux_test.go     # Testes do cliente
└── docs/
    ├── api-reference.md
    └── sdk-design.md
```

Pacote único: `eyedux`. Não criar subpacotes — o SDK é pequeno o suficiente para viver em um único pacote.

---

## Cliente

### Struct

```go
type Client struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
}
```

Todos os campos são privados. O estado interno não deve ser mutável após a criação.

### Construtor

```go
func New(apiKey string, opts ...Option) (*Client, error)
```

- `apiKey` obrigatório e não vazio — retornar erro se vazio.
- `baseURL` fixo em `https://api.eyedux.com/` — não exposto como opção.
- `opts` aplicados em sequência sobre uma config interna com defaults.
- Retornar `(*Client, error)` para permitir validação no momento da criação.

---

## Configuração

Usar o padrão **functional options**:

```go
type Option func(*config)

func WithHTTPClient(client *http.Client) Option
func WithTimeout(d time.Duration) Option
```

### Defaults

| Opção | Valor padrão |
|-------|-------------|
| `BaseURL` | `https://api.eyedux.com/` (fixo, não configurável) |
| `Timeout` | `30s` |
| `HTTPClient` | `&http.Client{Timeout: 30s}` |

---

## Métodos do cliente

Todos os métodos seguem a assinatura:

```go
func (c *Client) <Método>(ctx context.Context, input <Input>) (<Output>, error)
```

### CreateEvent

```go
func (c *Client) CreateEvent(ctx context.Context, input CreateEventInput) (*Event, error)
```

- Serializa `input` para JSON e faz `POST /public/logs`.
- Em sucesso (`201`), deserializa `data` para `*Event` e retorna.
- Em erro, retorna `nil, <APIError>`.

### ListEvents

```go
func (c *Client) ListEvents(ctx context.Context, input ListEventsInput) ([]Event, error)
```

- Monta query params a partir de `input` e faz `GET /public/logs`.
- Em sucesso (`200`), deserializa `data` (array) para `[]Event`.
- Lista vazia é resultado válido — retornar `[]Event{}`, não `nil`.

### FindEventByExternalID

```go
func (c *Client) FindEventByExternalID(ctx context.Context, externalID string) (*Event, error)
```

- Faz `GET /public/logs/external/<externalID>`.
- `externalID` vazio deve ser validado antes da chamada HTTP — retornar `ErrExternalIDRequired`.
- Em `404`, retornar `nil, <APIError com código event_external_id_not_found>`.

---

## Tipos públicos

### Event

```go
type Event struct {
    ID            string         `json:"id"`
    Type          string         `json:"type"`
    Properties    map[string]any `json:"properties"`
    Status        string         `json:"status"`
    Timestamp     time.Time      `json:"timestamp"`
    CreatedAt     time.Time      `json:"created_at"`
    ExternalID    *string        `json:"external_id,omitempty"`
    CorrelationID *string        `json:"correlation_id,omitempty"`
    Metadata      map[string]any `json:"metadata,omitempty"`
}
```

### CreateEventInput

```go
type CreateEventInput struct {
    ProjectID     string
    Type          string
    Properties    map[string]any
    ExternalID    *string
    CorrelationID *string
    Metadata      map[string]any
}
```

### ListEventsInput

```go
type ListEventsInput struct {
    Type          *string
    CorrelationID *string
}
```

---

## Tratamento de erros

### APIError

Erro retornado pelo servidor. Deve expor status HTTP, código e mensagem:

```go
type APIError struct {
    StatusCode int
    Code       string
    Message    string
    RetryAfter *int // preenchido em respostas 429 com header Retry-After
}

func (e *APIError) Error() string {
    return fmt.Sprintf("eyedux: %s (status %d)", e.Code, e.StatusCode)
}
```

### Constantes de código de erro

```go
const (
    ErrCodeInvalidAPIKey             = "invalid_api_key"
    ErrCodeEventTypeRequired         = "event_type_required"
    ErrCodeEventPropertiesEmpty      = "event_properties_empty"
    ErrCodeEventExternalIDConflict   = "event_external_id_conflict"
    ErrCodeEventExternalIDNotFound   = "event_external_id_not_found"
    ErrCodeEventExternalIDRequired   = "event_external_id_required"
    ErrCodeRateLimitExceeded         = "RATE_LIMIT_EXCEEDED"
    ErrCodeInternalServerError       = "INTERNAL_SERVER_ERROR"
)
```

### Helpers de inspeção

```go
func IsNotFound(err error) bool
func IsConflict(err error) bool
func IsRateLimited(err error) bool
func IsAuthError(err error) bool
```

Cada helper verifica se `err` é um `*APIError` e compara o `StatusCode` ou `Code` correspondente.

### Erros do SDK (não da API)

Usar `errors.New` simples para erros de validação interna do SDK:

```go
var (
    ErrEmptyAPIKey      = errors.New("eyedux: api key must not be empty")
    ErrEmptyBaseURL     = errors.New("eyedux: base URL must not be empty")
    ErrEmptyExternalID  = errors.New("eyedux: external_id must not be empty")
)
```

---

## HTTP e transport

### Construção das requisições

- Usar `net/http` da stdlib.
- Montar a URL concatenando `baseURL + path`. Garantir que não haja double slash.
- Serializar body com `encoding/json`.
- Sempre setar `Content-Type: application/json` em requisições com body.
- Sempre setar `Authorization: Bearer <apiKey>`.
- Sempre setar `Accept: application/json`.

### Leitura das respostas

- Ler o body inteiro com `io.ReadAll` antes de fechar — chamar `defer resp.Body.Close()` logo após a chamada.
- Deserializar com `encoding/json`.
- Qualquer status >= 400 deve ser tratado como erro: deserializar o envelope `{"error": {...}}` e retornar `*APIError`.
- Para `429`, tentar ler o header `Retry-After` e preencher `APIError.RetryAfter`.

### Envelope interno

Para deserialização de sucesso, usar tipos internos (não exportados):

```go
type successEnvelope[T any] struct {
    Data T `json:"data"`
}

type errorEnvelope struct {
    Error struct {
        Code    string `json:"code"`
        Message string `json:"message"`
    } `json:"error"`
}
```

---

## Testes

- Testar cada método do cliente com um `httptest.Server` que simula as respostas da API.
- Cobrir: sucesso, cada categoria de erro (400, 404, 409, 422, 429, 500).
- Cobrir: validações internas do SDK (apiKey vazio, externalID vazio).
- Não mockar a interface do cliente — testar o `Client` concreto contra um servidor fake.
- Arquivo de teste: `eyedux_test.go` no mesmo pacote (`package eyedux`).

### Padrão de helper de servidor fake

```go
func newTestServer(t *testing.T, handler http.HandlerFunc) *Client {
    t.Helper()
    srv := httptest.NewServer(handler)
    t.Cleanup(srv.Close)
    c, _ := New("test-api-key", srv.URL)
    return c
}
```

---

## Convenções de código

- `gofmt` e `go vet` obrigatórios antes de qualquer commit.
- Nomes exportados seguem Go idiomático: `CreateEvent`, não `CreateEventRequest`.
- Nenhum `init()`.
- Nenhuma variável global mutável.
- Comentário de pacote em `eyedux.go`: uma linha descrevendo o pacote.
- Cada tipo e função exportada deve ter comentário godoc.
