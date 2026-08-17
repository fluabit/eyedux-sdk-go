# eyedux-sdk-go

SDK oficial do Eyedux para Go. Framework-agnostic. Zero dependências externas além da stdlib.

## Instalação

```sh
go get github.com/fluabit/eyedux-sdk-go
```

Requer Go 1.21+.

## Uso rápido

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/fluabit/eyedux-sdk-go"
)

func main() {
    client, err := eyedux.New("sua-api-key")
    if err != nil {
        log.Fatal(err)
    }

    event, err := client.CreateEvent(context.Background(), eyedux.CreateEventInput{
        ProjectID:  "64f1a2b3c4d5e6f7a8b9c0d1",
        Type:       "user.signup",
        Properties: map[string]any{"plan": "pro", "source": "landing_page"},
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(event.ID)
}
```

## Referência da interface pública

### `eyedux.New`

```go
client, err := eyedux.New(apiKey string, opts ...eyedux.Option) (*eyedux.Client, error)
```

Retorna `ErrEmptyAPIKey` se `apiKey` for vazio. A base URL (`https://api.eyedux.com`) é fixa — não precisa ser configurada.

**Opções disponíveis**

| Opção | Descrição | Padrão |
|-------|-----------|--------|
| `eyedux.WithTimeout(d)` | Timeout das requisições HTTP | `30s` |
| `eyedux.WithHTTPClient(c)` | Substitui o `*http.Client` inteiro | `&http.Client{Timeout: 30s}` |

```go
// Timeout customizado
client, err := eyedux.New("sua-api-key", eyedux.WithTimeout(10*time.Second))

// HTTP client próprio (útil para proxies, transports customizados, etc.)
client, err := eyedux.New("sua-api-key", eyedux.WithHTTPClient(meuHTTPClient))
```

---

### `client.CreateEvent`

```go
event, err := client.CreateEvent(ctx context.Context, input eyedux.CreateEventInput) (*eyedux.Event, error)
```

Ingere um novo evento. Retorna o evento criado em sucesso (`201`).

```go
extID := "evt_01HX92K"
corrID := "session_abc123"

event, err := client.CreateEvent(ctx, eyedux.CreateEventInput{
    ProjectID:     "64f1a2b3c4d5e6f7a8b9c0d1",
    Type:          "user.signup",
    Properties:    map[string]any{"plan": "pro"},
    ExternalID:    &extID,   // opcional — ID idempotente externo
    CorrelationID: &corrID,  // opcional — agrupa eventos relacionados
    Metadata:      map[string]any{"ip": "192.168.1.1"},
})
```

---

### `client.ListEvents`

```go
events, err := client.ListEvents(ctx context.Context, input eyedux.ListEventsInput) ([]eyedux.Event, error)
```

Lista todos os eventos da organização. Filtros são opcionais e cumulativos. Retorna `[]Event{}` (nunca `nil`) quando não há resultados.

```go
// Sem filtros
events, err := client.ListEvents(ctx, eyedux.ListEventsInput{})

// Com filtros
typ := "user.signup"
corrID := "session_abc123"

events, err := client.ListEvents(ctx, eyedux.ListEventsInput{
    Type:          &typ,
    CorrelationID: &corrID,
})
```

---

### `client.FindEventByExternalID`

```go
event, err := client.FindEventByExternalID(ctx context.Context, externalID string) (*eyedux.Event, error)
```

Busca um único evento pelo `external_id` definido na criação. Retorna `ErrEmptyExternalID` se `externalID` for vazio.

```go
event, err := client.FindEventByExternalID(ctx, "evt_01HX92K")
if eyedux.IsNotFound(err) {
    // evento não existe
}
```

---

## Tratamento de erros

Erros da API são do tipo `*eyedux.APIError` e expõem `StatusCode`, `Code` e `Message`.

```go
event, err := client.CreateEvent(ctx, input)
if err != nil {
    var apiErr *eyedux.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("status %d — código: %s\n", apiErr.StatusCode, apiErr.Code)
    }
    return err
}
```

**Helpers de inspeção**

```go
eyedux.IsNotFound(err)    // 404 — external_id não encontrado
eyedux.IsConflict(err)    // 409 — external_id duplicado
eyedux.IsRateLimited(err) // 429 — rate limit excedido
eyedux.IsAuthError(err)   // 422 invalid_api_key
```

**Rate limiting**

Quando `IsRateLimited(err)` for `true`, o campo `RetryAfter` indica quantos segundos aguardar:

```go
if eyedux.IsRateLimited(err) {
    var apiErr *eyedux.APIError
    errors.As(err, &apiErr)
    if apiErr.RetryAfter != nil {
        time.Sleep(time.Duration(*apiErr.RetryAfter) * time.Second)
    }
}
```

O SDK não implementa retry automático — a política de retry fica a cargo do integrador.

---

## Tipo `Event`

| Campo | Tipo | Presença |
|-------|------|----------|
| `ID` | `string` | Sempre |
| `Type` | `string` | Sempre |
| `Properties` | `map[string]any` | Sempre |
| `Status` | `string` | Sempre (`"active"` ou `"deleted"`) |
| `Timestamp` | `time.Time` | Sempre |
| `CreatedAt` | `time.Time` | Sempre |
| `ExternalID` | `*string` | `nil` quando não definido |
| `CorrelationID` | `*string` | `nil` quando não definido |
| `Metadata` | `map[string]any` | `nil` quando não definido |

---

## Documentação interna

| Documento | Audiência | Descrição |
|-----------|-----------|-----------|
| [docs/api-reference.md](docs/api-reference.md) | Devs + IAs | Contrato completo da Public API |
| [docs/sdk-design.md](docs/sdk-design.md) | Devs + IAs | Arquitetura e convenções do SDK |
