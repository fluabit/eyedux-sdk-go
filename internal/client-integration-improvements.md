# Proposta de melhorias para facilitar integrações

**Status:** proposta  
**Data:** 2026-09-02  
**Escopo:** SDK Go do Eyedux e experiência de implementação nos clients

## Resumo

O SDK já oferece uma API pequena e previsível para chamadas HTTP, mas os
clients acabam criando uma camada de adaptação para resolver configuração,
`projectID`, timeout, conversão de `EventObject`, uso de `EventEyeduxType` e
tratamento de conflitos.

A proposta é manter o `Client` de baixo nível para integrações que precisam de
controle total e adicionar conveniências opcionais para o caso mais comum: um
client configurado uma vez para um projeto e reutilizado para enviar eventos.

O objetivo é que uma integração básica possa começar com poucas linhas, sem
esconder o `context.Context`, sem introduzir estado global e sem transferir
regras de negócio para o SDK.

## Diagnóstico da experiência atual

### 1. Configuração repetida

Cada integração precisa validar manualmente `EYEDUX_API_KEY` e
`EYEDUX_PROJECT_ID`, escolher timeout e formatar os erros de inicialização.

Essa lógica aparece antes de qualquer chamada ao SDK e tende a divergir entre
serviços.

### 2. `projectID` é repetido em todos os eventos

`CreateEventInput` exige `ProjectID`, embora normalmente um client pertença a
um único projeto durante todo o seu ciclo de vida. Isso força o integrador a
guardar o valor em um wrapper e copiá-lo para cada payload.

### 3. Conversão desnecessária de tipos

Um wrapper que expõe sua própria API precisa converter `EventObject` e
`CreateEventInput` para os tipos do SDK. A separação pode ser correta quando o
domínio do client exige nomes ou validações próprias, mas não deveria ser
necessária apenas para configurar o projeto ou o tipo de evento.

### 4. `EventEyeduxType` exige string e ponteiro manual

Hoje `Event.EyeduxType` e `CreateEventInput.EyeduxType` são declarados como
`*string`. Para enviar um tipo predefinido, cada client precisa declarar uma
string local e criar um ponteiro:

```go
eyeduxType := "system-log"

_, err := client.CreateEvent(ctx, eyeduxsdk.CreateEventInput{
    Type:       "api.error",
    EyeduxType: &eyeduxType,
    Properties: properties,
})
```

Além de gerar código repetido, essa API não informa quais valores de
`EyeduxType` são suportados pelo Eyedux. O client precisa conhecer os valores
por documentação externa ou por strings espalhadas no código.

### 5. Política de idempotência fica implícita

A API retorna `409` quando o `external_object` já foi usado. O SDK expõe
`IsConflict`, mas cada client precisa descobrir que esse conflito pode
significar sucesso idempotente e implementar a mesma decisão localmente.

O SDK não deve engolir o conflito por padrão, pois isso é uma decisão do
consumidor. Deve, porém, tornar a identificação do caso específico mais
direta e documentar o padrão recomendado.

### 6. Nome do pacote e documentação precisam permanecer alinhados

O pacote Go atual é `eyeduxsdk`, correção introduzida na versão `v0.2.0`.
Portanto, em um client cujo pacote também se chama `eyedux`, o import normal já
pode ser usado sem alias:

```go
import "github.com/fluabit/eyedux-sdk-go"

client, err := eyeduxsdk.New(apiKey)
```

O alias explícito `eyeduxsdk "github.com/fluabit/eyedux-sdk-go"` continua
válido, mas não é obrigatório. Todos os exemplos e documentos do repositório
devem refletir o nome atual.

## Princípios para a evolução

- Preservar `New(apiKey, opts ...Option)` para não quebrar integrações
  existentes.
- Manter `context.Context` em todos os métodos que fazem I/O.
- Não usar variáveis globais, `init()` ou configuração implícita obrigatória.
- Não fazer retry automático sem opt-in.
- Não transformar regras de negócio, como ignorar duplicidade, em
  comportamento padrão do SDK.
- Permitir que consumidores continuem definindo interfaces pequenas para
  testes.
- Fazer cópias de mapas recebidos quando o SDK mantiver valores como defaults,
  evitando alterações inesperadas por compartilhamento de memória.

## Melhorias propostas

### P0: Configuração de projeto no client

Adicionar uma opção de projeto padrão:

```go
func WithProjectID(projectID string) Option
```

Quando essa opção estiver configurada, `CreateEvent` deve usá-la apenas quando
`CreateEventInput.ProjectID` estiver vazio. Se os dois valores forem
informados, o `ProjectID` explícito do input deve prevalecer.

Essa regra preserva compatibilidade e permite migração gradual. Em uma versão
futura, o campo no input pode ser descontinuado para integrações que usam
client escopado.

Também deve ser adicionado um erro próprio para a ausência dos dois valores:

```go
var ErrEmptyProjectID = errors.New("eyedux: project id must not be empty")
```

A validação deve acontecer antes da requisição HTTP. O input não deve ser
alterado; o SDK deve construir internamente o payload com o `projectID`
efetivo.

#### Resultado esperado

```go
client, err := eyeduxsdk.New(
    apiKey,
    eyeduxsdk.WithProjectID(projectID),
    eyeduxsdk.WithTimeout(2*time.Second),
)
if err != nil {
    return err
}

_, err = client.CreateEvent(ctx, eyeduxsdk.CreateEventInput{
    Type:       "api.error",
    TypeGroup:  "backend",
    Properties: properties,
})
```

### P0: Validação e configuração centralizadas

Adicionar uma forma explícita de construir o client a partir de configuração:

```go
type Config struct {
    APIKey     string
    ProjectID  string
    HTTPClient *http.Client
    Timeout    time.Duration
}

func NewWithConfig(config Config) (*Client, error)
```

`NewWithConfig` exige `APIKey` e `ProjectID`, aplica os mesmos defaults de
`New` e mantém uma única lógica interna de construção para evitar diferenças
entre os construtores.

A opção de projeto continua sendo útil para quem prefere a API funcional:

```go
client, err := eyeduxsdk.New(
    apiKey,
    eyeduxsdk.WithProjectID(projectID),
)
```

Não é recomendado remover `ProjectID` de `CreateEventInput` imediatamente. O
campo ainda atende integrações multi-projeto e evita uma mudança incompatível
sem período de transição.

### P0: `EventEyeduxType` tipado e descobrível

O SDK deve expor `EventEyeduxType` como um tipo próprio e disponibilizar constantes
para os valores predefinidos pela plataforma:

```go
type EventEyeduxType string

const (
    EventEyeduxTypeSystemError   EventEyeduxType = "system-error"
    EventEyeduxTypeSystemWarning EventEyeduxType = "system-warning"
    EventEyeduxTypeSystemLog     EventEyeduxType = "system-log"
    EventEyeduxTypeSystemDebug   EventEyeduxType = "system-debug"
    EventEyeduxTypeSystemInfo    EventEyeduxType = "system-info"
    EventEyeduxTypeSystemMetric  EventEyeduxType = "system-metric"
)
```

O tipo permite que o compilador diferencie um tipo de evento de uma string
comum, enquanto as constantes oferecem uma fonte única para os valores
conhecidos. Novos tipos podem ser adicionados sem alterar o padrão da API.

Na entrada de criação, usar o tipo por valor. O valor vazio representa campo
omitido e mantém a semântica opcional atual:

```go
type CreateEventInput struct {
    EyeduxType EventEyeduxType
}
```

O payload interno deve continuar usando `omitempty`:

```go
type createEventBody struct {
    EyeduxType EventEyeduxType `json:"eyedux_type,omitempty"`
}
```

No retorno da API, manter a distinção entre valor ausente e valor preenchido:

```go
type Event struct {
    EyeduxType *EventEyeduxType `json:"eyedux_type"`
}
```

Assim, o uso no client fica direto:

```go
_, err := client.CreateEvent(ctx, eyeduxsdk.CreateEventInput{
    Type:       "api.error",
    EyeduxType: eyeduxsdk.EventEyeduxTypeSystemLog,
    Properties: properties,
})
```

Para um valor recebido dinamicamente, o client ainda poderá fazer uma
conversão explícita:

```go
eventType := eyeduxsdk.EventEyeduxType(configuredType)
```

A conversão deixa claro que o valor não veio da lista de constantes
conhecidas, sem impedir extensões futuras da plataforma.

Alterar `CreateEventInput.EyeduxType` de `*string` para `EventEyeduxType` quebra
integrações que atribuem ponteiros de string diretamente. Como o SDK ainda
está na série `0.x`, a mudança deve ser publicada na próxima versão minor,
com uma nota de migração no changelog:

```go
// Antes
eyeduxType := "system-log"
EyeduxType: &eyeduxType,

// Depois
EyeduxType: eyeduxsdk.EventEyeduxTypeSystemLog,
```

### P1: Construtor baseado em ambiente

Para clients que seguem o padrão de configuração por ambiente, oferecer uma
conveniência opt-in:

```go
func NewFromEnv(opts ...Option) (*Client, error)
```

`NewFromEnv` lê somente `EYEDUX_API_KEY`. O projeto não é lido do ambiente:
deve ser informado com `WithProjectID` ou no `CreateEventInput`. A função deve
aplicar `strings.TrimSpace` à API key e retornar `ErrEmptyAPIKey` quando ela
estiver ausente. Ela não deve ler configuração no `init()` nem armazenar
valores em estado global.

Exemplo:

```go
client, err := eyeduxsdk.NewFromEnv(
    eyeduxsdk.WithTimeout(2*time.Second),
)
if err != nil {
    return fmt.Errorf("initialize Eyedux: %w", err)
}
```

A leitura de ambiente deve ser uma camada de conveniência. `NewWithConfig`
continua sendo a opção preferida para testes, dependency injection e
aplicações que já possuem uma estrutura de configuração.

### P1: Metadados padrão

Adicionar uma opção para metadados comuns a todos os eventos:

```go
func WithDefaultMetadata(metadata map[string]any) Option
```

Os metadados definidos no evento devem prevalecer em caso de chave repetida.
O SDK deve copiar o mapa na configuração e ao montar cada request.

Isso elimina repetição de informações como `service`, sem impor nomes ou
valores ao domínio do consumidor:

```go
client, err := eyeduxsdk.New(
    apiKey,
    eyeduxsdk.WithProjectID(projectID),
    eyeduxsdk.WithDefaultMetadata(map[string]any{
        "service": "eyedux-ws",
    }),
)
```

### P1: Erro específico para duplicidade

Manter `IsConflict(err)` e adicionar um helper específico para o contrato de
eventos:

```go
func IsExternalObjectConflict(err error) bool
```

O helper deve verificar o código `event_external_object_conflict`, além de
garantir que o erro seja um `*APIError` ou esteja encapsulado por
`fmt.Errorf("...: %w", err)`.

O comportamento recomendado no client continua explícito:

```go
_, err := client.CreateEvent(ctx, input)
if err != nil {
    if eyeduxsdk.IsExternalObjectConflict(err) {
        return nil
    }
    return fmt.Errorf("track event: %w", err)
}
return nil
```

O SDK não deve converter esse conflito em `nil` automaticamente, porque nem
toda integração considera duplicidade um sucesso.

### P1: Exemplos de integração prontos para copiar

Adicionar exemplos compiláveis para os dois cenários mais comuns:

1. Serviço que envia eventos HTTP de sucesso e erro.
2. Serviço que encapsula o SDK e aplica uma política de idempotência.

Os exemplos devem mostrar:

- import sem alias obrigatório;
- `WithProjectID` ou `NewWithConfig`;
- timeout;
- uso do contexto da requisição quando disponível;
- tratamento de `IsExternalObjectConflict`;
- separação entre erro de rastreamento e erro da operação principal;
- uso de `EventEyeduxTypeSystemLog` quando o evento tiver um tipo predefinido.

Isso reduz mais atrito do que adicionar apenas métodos auxiliares, porque
entrega uma integração de referência com decisões já explicitadas.

### P2: Retry opt-in

Adicionar uma política configurável somente depois de estabilizar a
configuração de projeto:

```go
type RetryPolicy struct {
    MaxAttempts int
    Backoff     func(attempt int, err error) time.Duration
}

func WithRetryPolicy(policy RetryPolicy) Option
```

A política deve ser limitada a erros transitórios, como `429`, `502`, `503` e
`504`, respeitar `Retry-After` quando disponível e sempre obedecer ao
cancelamento ou deadline do contexto.

Não retryar `400`, `401`, `403`, `409` ou `422` por padrão. A política deve
evitar duplicar eventos quando a requisição foi aceita pelo servidor, portanto
qualquer suporte a retry deve ser acompanhado de uma estratégia de
idempotência claramente documentada.

### P2: Ingestão assíncrona opcional

Para serviços em que o rastreamento não pode atrasar o fluxo principal, avaliar
um componente separado, como um `Batcher` ou `AsyncClient`, em vez de colocar
fila e goroutines dentro do `Client` padrão.

Esse componente precisaria definir explicitamente:

- limite de memória;
- comportamento durante shutdown;
- política de perda quando a fila estiver cheia;
- propagação de erros;
- garantia, ou ausência de garantia, de entrega.

É uma melhoria de maior risco operacional e não deve fazer parte da primeira
versão da API simplificada.

## API alvo recomendada

Com as melhorias P0 e P1, uma integração típica poderia ficar assim:

```go
package eyedux

import (
    "context"
    "fmt"
    "time"

    "github.com/fluabit/eyedux-sdk-go"
)

type Client struct {
    events *eyeduxsdk.Client
}

func NewClient(apiKey, projectID string) (*Client, error) {
    events, err := eyeduxsdk.New(
        apiKey,
        eyeduxsdk.WithProjectID(projectID),
        eyeduxsdk.WithTimeout(2*time.Second),
        eyeduxsdk.WithDefaultMetadata(map[string]any{
            "service": "eyedux-ws",
        }),
    )
    if err != nil {
        return nil, fmt.Errorf("initialize Eyedux: %w", err)
    }
    return &Client{events: events}, nil
}

func (c *Client) TrackError(ctx context.Context, properties map[string]any) error {
    _, err := c.events.CreateEvent(ctx, eyeduxsdk.CreateEventInput{
        Type:       "api.error",
        TypeGroup:  "backend",
        EyeduxType: eyeduxsdk.EventEyeduxTypeSystemLog,
        Properties: properties,
    })
    if err != nil {
        if eyeduxsdk.IsExternalObjectConflict(err) {
            return nil
        }
        return fmt.Errorf("track error event: %w", err)
    }
    return nil
}
```

A camada do client ainda pode manter tipos próprios, como `RequestEvent` e
`ErrorEvent`, quando isso melhora o contrato do domínio. A melhoria proposta
apenas remove a necessidade de repetir configuração, mapeamento e ponteiros
por motivos puramente técnicos.

## Plano de implementação

1. Atualizar `internal/sdk-design.md` e exemplos para `package eyeduxsdk` e
   assinaturas reais do SDK.
2. Adicionar `ErrEmptyProjectID`, `WithProjectID` e a resolução do projeto no
   payload.
3. Cobrir precedência entre `CreateEventInput.ProjectID` e o projeto padrão.
4. Adicionar `Config` e `NewWithConfig` usando a mesma construção interna de
   `New`.
5. Adicionar `NewFromEnv` com testes isolados para variáveis de ambiente.
6. Adicionar `WithDefaultMetadata` com testes de cópia e precedência de
   chaves.
7. Adicionar `EventEyeduxType`, suas constantes e testes de serialização.
8. Adicionar `IsExternalObjectConflict` e atualizar a referência de erros.
9. Publicar exemplos compiláveis de integração.
10. Avaliar retry e ingestão assíncrona em uma proposta separada, após validar
    a API básica.

## Critérios de aceite

- Uma integração de projeto único não precisa informar `ProjectID` em cada
  `CreateEvent`.
- A ausência de API key ou de projeto efetivo falha antes da chamada HTTP.
- A API atual `New(apiKey, opts ...Option)` continua compilando.
- Um `ProjectID` informado explicitamente no input continua tendo precedência.
- Metadados padrão não são mutados pelo SDK nem pelo consumidor após a
  configuração.
- O client pode usar `eyeduxsdk.EventEyeduxTypeSystemLog` sem declarar uma string
  local ou criar ponteiro manualmente.
- `eyedux_type` continua sendo omitido quando nenhum tipo é informado.
- Respostas com `eyedux_type: null` continuam distinguíveis de respostas com
  um tipo preenchido.
- O JSON enviado permanece compatível com o contrato atual da API.
- Valores predefinidos ficam descobríveis pelo autocomplete do Go.
- Duplicidade de `external_object` pode ser identificada sem inspecionar
  strings de erro.
- Os exemplos de documentação compilam com `go test`.
- Nenhum retry ou envio assíncrono é ativado sem configuração explícita.

## Decisões tomadas

- `NewFromEnv` lê somente `EYEDUX_API_KEY`; o projeto fica em `WithProjectID`
    ou no `CreateEventInput`.
- `NewWithConfig` exige `APIKey` e `ProjectID`.
- O construtor público preferido é `NewWithConfig`; `NewFromConfig` não faz
    parte da API.
- `CreateEventInput.ProjectID` permanece por compatibilidade e suporte
    temporário a múltiplos projetos. Sua remoção exige uma major version.
- Os valores predefinidos de `EventEyeduxType` são `system-error`,
    `system-warning`, `system-log`, `system-debug`, `system-info` e
    `system-metric`.