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
    client, err := eyeduxsdk.New("sua-api-key", eyeduxsdk.WithProjectID("64f1a2b3c4d5e6f7a8b9c0d1"))
    if err != nil {
        log.Fatal(err)
    }

    event, err := client.CreateEvent(context.Background(), eyeduxsdk.CreateEventInput{
        Type:       "user.signup",
        EyeduxType: eyeduxsdk.EventEyeduxTypeSystemLog,
        Properties: map[string]any{"plan": "pro", "source": "landing_page"},
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(event.ID)
}
```

## Referência da interface pública

### `eyeduxsdk.New`

```go
client, err := eyeduxsdk.New(apiKey string, opts ...eyeduxsdk.Option) (*eyeduxsdk.Client, error)
```

Retorna `ErrEmptyAPIKey` se `apiKey` for vazio. A base URL (`https://api.eyedux.com`) é fixa — não precisa ser configurada.

**Opções disponíveis**

| Opção | Descrição | Padrão |
|-------|-----------|--------|
| `eyeduxsdk.WithTimeout(d)` | Timeout das requisições HTTP | `30s` |
| `eyeduxsdk.WithHTTPClient(c)` | Substitui o `*http.Client` inteiro | `&http.Client{Timeout: 30s}` |
| `eyeduxsdk.WithProjectID(id)` | Define o projeto padrão dos eventos | vazio |
| `eyeduxsdk.WithDefaultMetadata(m)` | Adiciona metadados padrão aos eventos | vazio |

```go
// Timeout customizado
client, err := eyeduxsdk.New("sua-api-key", eyeduxsdk.WithTimeout(10*time.Second))

// HTTP client próprio (útil para proxies, transports customizados, etc.)
client, err := eyeduxsdk.New("sua-api-key", eyeduxsdk.WithHTTPClient(meuHTTPClient))
```

### `eyeduxsdk.NewWithConfig`

Use `NewWithConfig` when the application already has an explicit configuration.
`APIKey` and `ProjectID` are required.

```go
client, err := eyeduxsdk.NewWithConfig(eyeduxsdk.Config{
    APIKey:    "sua-api-key",
    ProjectID: "64f1a2b3c4d5e6f7a8b9c0d1",
    Timeout:   2 * time.Second,
})
```

`NewFromEnv` reads only `EYEDUX_API_KEY`. The project must be supplied with
`WithProjectID` or in each `CreateEventInput`:

```go
client, err := eyeduxsdk.NewFromEnv(
    eyeduxsdk.WithProjectID("64f1a2b3c4d5e6f7a8b9c0d1"),
)
```

---

### `client.CreateEvent`

```go
event, err := client.CreateEvent(ctx context.Context, input eyeduxsdk.CreateEventInput) (*eyeduxsdk.Event, error)
```

Ingere um novo evento. Retorna o evento criado em sucesso (`201`).

```go
extID := "evt_01HX92K"
corrID := "session_abc123"

event, err := client.CreateEvent(ctx, eyeduxsdk.CreateEventInput{
    ProjectID: "64f1a2b3c4d5e6f7a8b9c0d1",
    Type:      "user.signup",
    Properties: map[string]any{"plan": "pro"},
    ExternalObject: &eyeduxsdk.EventObject{
        ID:       extID,
        Property: "orderId",
    },
    CorrelationObject: &eyeduxsdk.EventObject{
        ID:       corrID,
        Property: "sessionId",
    },
    Metadata: map[string]any{"ip": "192.168.1.1"},
})
```

---

### `EventEyeduxType`

Use the predefined constants instead of declaring a string and taking its
address manually:

```go
event, err := client.CreateEvent(ctx, eyeduxsdk.CreateEventInput{
    ProjectID:  "64f1a2b3c4d5e6f7a8b9c0d1",
    Type:       "api.error",
    EyeduxType: eyeduxsdk.EventEyeduxTypeSystemError,
    Properties: map[string]any{"message": "request failed"},
})
```

Available values are `EventEyeduxTypeSystemError`,
`EventEyeduxTypeSystemWarning`, `EventEyeduxTypeSystemLog`,
`EventEyeduxTypeSystemDebug`, `EventEyeduxTypeSystemInfo` and
`EventEyeduxTypeSystemMetric`.

---

### `client.ListEvents`

```go
events, err := client.ListEvents(ctx context.Context, input eyeduxsdk.ListEventsInput) ([]eyeduxsdk.Event, error)
```

Lista todos os eventos da organização. Filtros são opcionais e cumulativos. Retorna `[]Event{}` (nunca `nil`) quando não há resultados.

```go
// Sem filtros
events, err := client.ListEvents(ctx, eyeduxsdk.ListEventsInput{})

// Com filtros
typ := "user.signup"
corrID := "session_abc123"

events, err := client.ListEvents(ctx, eyeduxsdk.ListEventsInput{
    Type:          &typ,
    CorrelationID: &corrID,
})
```

---

### `client.FindEventByExternalID`

```go
event, err := client.FindEventByExternalID(ctx context.Context, externalID string) (*eyeduxsdk.Event, error)
```

Busca um único evento pelo ID definido no `ExternalObject`. Retorna `ErrEmptyExternalID` se `externalID` for vazio.

```go
event, err := client.FindEventByExternalID(ctx, "evt_01HX92K")
if eyeduxsdk.IsNotFound(err) {
    // evento não existe
}
```

---

## Tratamento de erros

Erros da API são do tipo `*eyeduxsdk.APIError` e expõem `StatusCode`, `Code` e `Message`.

```go
event, err := client.CreateEvent(ctx, input)
if err != nil {
    var apiErr *eyeduxsdk.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("status %d — código: %s\n", apiErr.StatusCode, apiErr.Code)
    }
    return err
}
```

**Helpers de inspeção**

```go
eyeduxsdk.IsNotFound(err)    // 404 — external_id não encontrado
eyeduxsdk.IsConflict(err)    // 409 — external_object duplicado
eyeduxsdk.IsExternalObjectConflict(err)
eyeduxsdk.IsRateLimited(err) // 429 — rate limit excedido
eyeduxsdk.IsAuthError(err)   // 422 invalid_api_key
```

**Rate limiting**

Quando `IsRateLimited(err)` for `true`, o campo `RetryAfter` indica quantos segundos aguardar:

```go
if eyeduxsdk.IsRateLimited(err) {
    var apiErr *eyeduxsdk.APIError
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
| `Environment` | `string` | Ambiente da API key |
| `EyeduxType` | `*EventEyeduxType` | `nil` quando não definido |
| `Type` | `string` | Sempre |
| `TypeGroup` | `string` | Vazio quando não definido |
| `Properties` | `map[string]any` | Sempre |
| `Status` | `string` | Sempre (`"active"` ou `"deleted"`) |
| `Timestamp` | `time.Time` | Sempre |
| `CreatedAt` | `time.Time` | Sempre |
| `ExternalObject` | `*EventObject` | `nil` quando não definido |
| `CorrelationObject` | `*EventObject` | `nil` quando não definido |
| `Metadata` | `map[string]any` | `nil` quando não definido |

---

## Documentação pública

| Documento | Audiência | Descrição |
|-----------|-----------|-----------|
| [docs/client-integration.md](docs/client-integration.md) | Clients do SDK | Guia de instalação, configuração e integração com o Eyedux |
| [docs/api-reference.md](docs/api-reference.md) | Devs que integram o SDK | Contrato completo da Public API |

A documentação é publicada como um site estático pelo GitHub Pages. O workflow
em `.github/workflows/pages.yml` envia `docs/` para o Pages. Consulte o [guia de
integração](docs/client-integration.md) para habilitar o source **GitHub
Actions** e executar a publicação.

## Documentação de manutenção

| Documento | Descrição |
|-----------|-----------|
| [internal/sdk-design.md](internal/sdk-design.md) | Arquitetura, convenções e diretrizes de teste do SDK |
| [internal/client-integration-improvements.md](internal/client-integration-improvements.md) | Propostas de evolução da experiência de integração |
