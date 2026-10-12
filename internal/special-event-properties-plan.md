# Plano e implementação: propriedades especiais de eventos

## Objetivo

Documentar e implementar o envio livre de campos personalizados no objeto
`properties` do evento. `code` e `workflow` podem seguir uma convenção
reconhecida pela interface, mas o SDK não valida nem transforma esses valores.

## Estado atual confirmado no SDK

`EmitError` e os helpers de diagnóstico trabalham com estas chaves em
`Properties`:

| Chave | Tipo | Origem atual |
|-------|------|--------------|
| `error` | string | `EmitError`/`ErrorProperties`, quando um erro é informado |
| `operation` | string | `EmitError`/`ErrorProperties`, quando a operação é informada |
| `source_file` | string | Arquivo do caller; apenas o nome base é incluído |
| `source_line` | integer | Linha do caller |
| `source_function` | string | Função do caller |

As chaves são serializadas em `snake_case`. Os três campos de origem são
incluídos quando o SDK consegue identificar o caller. Os helpers copiam o mapa
de propriedades recebido, em vez de modificá-lo.

## Formato implementado

O SDK envia as propriedades dentro do objeto `properties`. O formato abaixo é
uma convenção para exibição especial, não uma estrutura validada pelo SDK:

```json
{
  "properties": {
    "code": {
      "source_file": "orders.go",
      "source_line": 42,
      "source_function": "example.com/app.CreateOrder"
    },
    "workflow": {
      "source_workflow": "orders",
      "source_node": "save-order",
      "source_execution_id": "execution-123"
    },
    "order_id": "order-456"
  }
}
```

`CreateEventInput.Properties` continua sendo o ponto de entrada Go e suas
chaves e valores são preservados ao serializar dentro de `properties`. `code` e
`workflow` podem conter qualquer valor JSON. Chaves individuais, como
`source_file`, não são agrupadas automaticamente. `Event.Properties` contém
todos os campos personalizados da resposta, inclusive `code` e `workflow`.

## Trabalho realizado

- Documentar o formato em `docs/api-reference.md`,
  `docs/client-integration.md` e no README.
- Serializar as propriedades em um objeto `properties`, sem validar `code`,
  `workflow` ou nomes de propriedades.
- Manter propriedades com nomes iguais aos campos do evento aninhadas, sem
  sobrescrever o envelope.

## Limites

- O SDK não coleta metadados de workflow automaticamente nem depende de n8n.

## Fora de escopo nesta etapa

- Inferir ou preencher automaticamente os valores de workflow.

## Validação

- Cobrir serialização, desserialização e propriedades com nomes iguais aos
  campos do envelope com testes Go.
- Executar `go test ./...` e o build estrito do site de documentação.
