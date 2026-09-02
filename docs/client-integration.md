# Guia de integração para clients

Este guia explica como uma aplicação cliente pode integrar o Eyedux usando o
SDK oficial para Go. O SDK envia eventos para a Public API e pode ser usado
com qualquer framework HTTP, worker ou serviço em background.

## Antes de começar

Você precisa de:

- Go 1.21 ou superior;
- uma API key criada no painel do Eyedux;
- o `project_id` do projeto que receberá os eventos.

A API key autentica a organização e deve permanecer no servidor. Não a
exponha em JavaScript do navegador, aplicações mobile ou repositórios. O SDK
usa a base fixa `https://api.eyedux.com` e envia automaticamente o header
`Authorization: Bearer <api_key>`.

## Instalação

Adicione o módulo ao projeto:

```sh
go get github.com/fluabit/eyedux-sdk-go
```

Importe o pacote como `eyeduxsdk`:

```go
import "github.com/fluabit/eyedux-sdk-go"
```

## Configuração recomendada

Mantenha os segredos fora do código e injete-os por variáveis de ambiente:

```sh
export EYEDUX_API_KEY="sua-api-key"
export EYEDUX_PROJECT_ID="64f1a2b3c4d5e6f7a8b9c0d1"
```

O construtor mais explícito é `NewWithConfig`:

```go
package eyeduxclient

import (
	"os"

	eyeduxsdk "github.com/fluabit/eyedux-sdk-go"
)

func New() (*eyeduxsdk.Client, error) {
	return eyeduxsdk.NewWithConfig(eyeduxsdk.Config{
		APIKey:    os.Getenv("EYEDUX_API_KEY"),
		ProjectID: os.Getenv("EYEDUX_PROJECT_ID"),
	})
}
```

`NewWithConfig` valida `APIKey` e `ProjectID` antes de criar o client. Para
usar a conveniência baseada em ambiente, `NewFromEnv` lê somente
`EYEDUX_API_KEY`; o projeto ainda precisa ser passado com `WithProjectID` ou
em cada evento:

```go
client, err := eyeduxsdk.NewFromEnv(
	eyeduxsdk.WithProjectID(os.Getenv("EYEDUX_PROJECT_ID")),
)
```

Crie uma instância por configuração e reutilize-a durante a vida da
aplicação. O `Client` é seguro para uso concorrente, mas não deve ser copiado
depois do primeiro uso.

## Enviando o primeiro evento

Com um projeto padrão configurado, `ProjectID` pode ser omitido do input:

```go
package eyeduxclient

import (
	"context"
	"fmt"

	eyeduxsdk "github.com/fluabit/eyedux-sdk-go"
)

func TrackSignup(ctx context.Context, client *eyeduxsdk.Client, userID string) error {
	event, err := client.CreateEvent(ctx, eyeduxsdk.CreateEventInput{
		Type:       "user.signup",
		TypeGroup:  "identity",
		EyeduxType: eyeduxsdk.EventEyeduxTypeSystemLog,
		Properties: map[string]any{
			"user_id": userID,
			"plan":    "pro",
			"source":  "landing_page",
		},
	})
	if err != nil {
		return err
	}

	fmt.Println("evento criado:", event.ID)
	return nil
}
```

`Properties` é o payload principal do evento e deve conter pelo menos um
valor. Os valores precisam ser serializáveis como JSON. `Type` é obrigatório;
`TypeGroup`, `EyeduxType`, `Metadata` e os objetos de referência são
opcionais.

## Contexto e timeout

Passe o contexto recebido pela aplicação para cada chamada. Em handlers HTTP,
isso permite cancelar a requisição quando o caller desconecta; em workers,
prefira um timeout explícito:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

_, err := client.CreateEvent(ctx, eyeduxsdk.CreateEventInput{
	Type:       "payment.completed",
	Properties: map[string]any{"payment_id": "pay_123"},
})
```

O timeout do contexto e o configurado no HTTP client atuam juntos. O padrão
do SDK é `30s`; ajuste-o ao construir o client quando necessário:

```go
client, err := eyeduxsdk.NewWithConfig(eyeduxsdk.Config{
	APIKey:    os.Getenv("EYEDUX_API_KEY"),
	ProjectID: os.Getenv("EYEDUX_PROJECT_ID"),
	Timeout:   10 * time.Second,
})
```

## Referências e correlação

Use `ExternalObject` para associar o evento a um identificador do sistema de
origem. Use `CorrelationObject` para agrupar eventos relacionados, como os de
uma mesma sessão ou requisição:

```go
externalID := "order_01HX92K"
sessionID := "session_abc123"

event, err := client.CreateEvent(ctx, eyeduxsdk.CreateEventInput{
	Type: "order.paid",
	Properties: map[string]any{
		"amount":   149.90,
		"currency": "BRL",
	},
	ExternalObject: &eyeduxsdk.EventObject{
		ID:       externalID,
		Property: "order_id",
	},
	CorrelationObject: &eyeduxsdk.EventObject{
		ID:       sessionID,
		Property: "session_id",
	},
	})
```

`Metadata` é adequado para contexto comum a vários eventos. Metadados padrão
podem ser definidos uma vez e sobrescritos por evento:

```go
client, err := eyeduxsdk.New(
	os.Getenv("EYEDUX_API_KEY"),
	eyeduxsdk.WithProjectID(os.Getenv("EYEDUX_PROJECT_ID")),
	eyeduxsdk.WithDefaultMetadata(map[string]any{
		"service":     "checkout-api",
		"environment": "production",
	}),
)
```

Quando o mesmo client envia eventos para mais de um projeto, informe
`CreateEventInput.ProjectID`. Esse valor explícito tem precedência sobre o
projeto configurado com `WithProjectID`.

## Idempotência com `ExternalObject`

O Eyedux retorna `409` quando o `external_object` já está associado a um
evento. O SDK não converte esse conflito em sucesso automaticamente, porque a
decisão depende da regra de negócio do client.

Para tratar a criação como idempotente, recupere o evento existente pelo ID
externo:

```go
event, err := client.CreateEvent(ctx, eyeduxsdk.CreateEventInput{
	Type: "order.paid",
	Properties: map[string]any{"order_id": externalID},
	ExternalObject: &eyeduxsdk.EventObject{
		ID:       externalID,
		Property: "order_id",
	},
})
if err != nil && eyeduxsdk.IsExternalObjectConflict(err) {
	event, err = client.FindEventByExternalID(ctx, externalID)
}
if err != nil {
	return err
}

fmt.Println("evento confirmado:", event.ID)
```

Esse padrão só deve ser usado quando a repetição do mesmo ID externo
representar a mesma operação no domínio da aplicação.

## Consultando eventos

Liste eventos sem filtros ou combine os filtros disponíveis:

```go
eventType := "order.paid"
sessionID := "session_abc123"

events, err := client.ListEvents(ctx, eyeduxsdk.ListEventsInput{
	Type:          &eventType,
	CorrelationID: &sessionID,
})
if err != nil {
	return err
}
```

Para uma busca pontual por identificador externo:

```go
event, err := client.FindEventByExternalID(ctx, "order_01HX92K")
if eyeduxsdk.IsNotFound(err) {
	// Nenhum evento possui esse identificador.
}
```

## Tratamento de erros

Há dois grupos de erros:

- erros de configuração do SDK, como `ErrEmptyAPIKey`, `ErrEmptyProjectID` e
  `ErrEmptyExternalID`;
- erros da API, representados por `*eyeduxsdk.APIError`, com `StatusCode`,
  `Code` e `Message`.

Inspecione o tipo do erro sem depender apenas da mensagem textual:

```go
event, err := client.CreateEvent(ctx, input)
if err != nil {
	var apiErr *eyeduxsdk.APIError
	if errors.As(err, &apiErr) {
		log.Printf("Eyedux retornou %d (%s): %s",
			apiErr.StatusCode, apiErr.Code, apiErr.Message)
	}
	return err
}
```

Os helpers mais comuns são:

| Helper | Uso |
|--------|-----|
| `IsAuthError(err)` | API key inválida ou revogada (`422`) |
| `IsNotFound(err)` | Evento externo não encontrado (`404`) |
| `IsConflict(err)` | Qualquer conflito (`409`) |
| `IsExternalObjectConflict(err)` | `external_object` duplicado |
| `IsRateLimited(err)` | Limite de requisições excedido (`429`) |

### Rate limit

O SDK não faz retry automático. Quando `IsRateLimited(err)` for verdadeiro,
consulte `RetryAfter` e aplique a política apropriada para o worker ou serviço:

```go
if eyeduxsdk.IsRateLimited(err) {
	var apiErr *eyeduxsdk.APIError
	if errors.As(err, &apiErr) && apiErr.RetryAfter != nil {
		time.Sleep(time.Duration(*apiErr.RetryAfter) * time.Second)
	}
}
```

Em aplicações de produção, prefira uma fila ou backoff limitado para não
bloquear handlers HTTP. Preserve o mesmo `ExternalObject` ao repetir um
evento para manter a deduplicação do lado da API.

## Proxy e transporte HTTP

Use `WithHTTPClient` quando a aplicação já possui um `*http.Client` com proxy,
transport ou observabilidade próprios:

```go
httpClient := &http.Client{
	Timeout: 15 * time.Second,
}

client, err := eyeduxsdk.New(
	os.Getenv("EYEDUX_API_KEY"),
	eyeduxsdk.WithProjectID(os.Getenv("EYEDUX_PROJECT_ID")),
	eyeduxsdk.WithHTTPClient(httpClient),
)
```

Quando `WithHTTPClient` é usado, `WithTimeout` não altera o client fornecido.

## Checklist de produção

- API key armazenada em secret manager ou variável de ambiente;
- `project_id` configurado explicitamente;
- uma instância do SDK reutilizada pela aplicação;
- contexto com timeout em cada operação;
- `Properties` sem dados sensíveis desnecessários;
- `ExternalObject` definido quando o evento precisa ser idempotente;
- tratamento separado para autenticação, conflitos, not-found e rate limit;
- retry implementado no worker ou serviço, com backoff e limite;
- logs sem imprimir a API key ou outros segredos.

## Publicação com GitHub Pages

Sim. Esta documentação é Markdown e pode ser publicada diretamente pelo
GitHub Pages sem um servidor Go. O GitHub Pages hospeda os arquivos estáticos;
ele não executa o SDK, não guarda a API key com segurança e não deve receber
eventos diretamente do navegador.

Este repositório já contém o workflow
`.github/workflows/pages.yml`, que publica a pasta `docs/`:

1. Faça push do repositório para o GitHub.
2. Abra `Settings > Pages` no repositório.
3. Em **Build and deployment > Source**, escolha **GitHub Actions**.
4. Salve e execute o workflow `Deploy documentation to GitHub Pages`, ou faça
	push na branch `main`.
5. Aguarde o job de deploy terminar e abra a URL informada pelo ambiente
	`github-pages`.

O arquivo `docs/index.md` funciona como a página inicial e aponta para este
guia e a referência da API. Não é necessário adicionar um backend para
publicar a documentação. Se o projeto precisar de busca, navegação avançada ou
versionamento, um gerador estático como MkDocs ou Docusaurus pode ser adicionado
depois ao job de build.

## Próximo documento

- [Referência completa da API](api-reference.md)