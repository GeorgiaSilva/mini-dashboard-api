# API niveladora de front-end

Backend REST pequeno, em Go, para praticar login JWT, perfis de acesso, consulta com filtros, paginação, tabelas e gráficos. A persistência é feita nos arquivos JSON de `data/`; portanto, alterações feitas pela API continuam após reiniciá-la.

## Requisitos e execução

- Go 1.22 ou superior

```bash
go mod tidy
cp .env.example .env # opcional: use a variável no seu terminal
export JWT_SECRET='uma-chave-segura-para-o-seu-ambiente'
go run ./cmd/api
```

Sem `JWT_SECRET`, a API usa uma chave padrão exclusivamente para desenvolvimento local. Ela fica disponível em `http://localhost:8080`. Para outra porta, defina `PORT`; para outro diretório de dados, defina `DATA_DIR`.

Para rodar os testes:

```bash
go test ./...
```

Os testes usam cópias temporárias dos JSONs e não modificam `data/`.

## Insomnia

Importe o arquivo [insomnia.json](./insomnia.json) no Insomnia (`Import > File`). Execute **Login — Desenvolvedor**, copie o valor de `accessToken` retornado e cole-o na variável `token` do ambiente **Local**. As demais requisições autenticadas passarão a usar o token automaticamente.

## Usuários de teste

Todos usam a senha `123456` (armazenada como hash bcrypt no arquivo).

| Nome | E-mail | Perfil |
|---|---|---|
| Desenvolvedor | dev@empresa.com | DEVELOPER |
| Maria Desenvolvedora | maria.dev@empresa.com | DIRECTOR |
| Carlos Diretor | diretor@empresa.com | DIRECTOR |
| Ana Diretora | ana.diretora@empresa.com | DIRECTOR |

| Recurso | DEVELOPER | DIRECTOR |
|---|---|---|
| Dashboard | sim | sim |
| Vendas | sim | sim |
| Usuários | sim | não |

## Autenticação

Rotas públicas: `GET /health` e `POST /auth/login`. Todas as outras exigem:

```http
Authorization: Bearer SEU_TOKEN
```

```bash
curl -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"dev@empresa.com","password":"123456"}'
```

A resposta traz `accessToken` e o usuário sem senha. O JWT contém `sub`, `name`, `email`, `role`, `iat` e `exp`, dura uma hora e nunca inclui a senha. `GET /auth/me` consulta o usuário atual no arquivo, assim refletindo mudanças recentes de perfil ou status.

## Rotas

| Método | Rota | Acesso | Descrição |
|---|---|---|---|
| GET | /health | público | `{"status":"ok"}` |
| POST | /auth/login | público | gera o token |
| GET | /auth/me | autenticado | usuário atualizado |
| GET | /dashboard | DEVELOPER, DIRECTOR | cards e dados de gráficos |
| GET | /sales | DEVELOPER, DIRECTOR | vendas agrupadas por dia |
| GET | /sales/filters | DEVELOPER, DIRECTOR | bandeiras, status e entes |
| GET | /sales | DEVELOPER, DIRECTOR | lista paginada e agrupada |
| GET | /sales/{id} | DEVELOPER, DIRECTOR | consulta uma venda |
| GET | /users | DEVELOPER | lista paginada |
| GET | /users/{id} | DEVELOPER | consulta um usuário |
| POST | /users | DEVELOPER | cria usuário |
| PATCH | /users/{id} | DEVELOPER | altera usuário |
| DELETE | /users/{id} | DEVELOPER | desativa usuário |
| PATCH | /users/{id}/password | DEVELOPER | altera senha |

O `DELETE` de usuário é lógico (`active: false`). Não há cadastro público. E-mail é único, e um desenvolvedor não pode desativar a própria conta. Vendas vêm de outro produto de pagamentos e são somente leitura nesta API.

## Filtros e exemplos

```text
GET /sales?startDate=2026-09-01&endDate=2026-09-30&brand=VISA&publicEntityId=2&status=APPROVED&page=1&limit=20
GET /users?search=maria&role=DEVELOPER&active=true&page=1&limit=10
GET /dashboard?startDate=2026-09-01&endDate=2026-09-30
```

Datas usam `YYYY-MM-DD`. Os limites vão de 1 a 100. Vendas são retornadas da mais recente para a mais antiga e os cálculos de resumo levam em conta somente os registros filtrados.

Exemplo de criação de usuário:

```json
{"name":"Novo usuário","email":"novo@empresa.com","password":"123456","role":"DIRECTOR"}
```

## Formato de resposta

`GET /sales` retorna `summary`, `groups` e `pagination`. Cada grupo possui `date`, `totalAmount`, `totalSales` e `sales`. O dashboard calcula receita bruta a partir de vendas aprovadas, receita líquida como 90% dela e evita divisão por zero.

Falhas têm sempre a mesma forma:

```json
{"error":{"code":"VALIDATION_ERROR","message":"Mensagem para o cliente."}}
```

Status usados: `200` sucesso, `201` criado, `204` sem conteúdo, `400` entrada inválida, `401` autenticação inválida, `403` sem permissão, `404` não encontrado, `409` conflito de e-mail e `500` erro interno.

## Estrutura

```text
cmd/api/               inicialização e testes HTTP
internal/handlers/     regras de cada recurso
internal/middleware/   JWT, perfis e erros
internal/models/       estruturas de dados
internal/repository/   JSON, trava RWMutex e escrita atômica
data/                  users.json e sales.json
```

O CORS permite o front-end local em `http://localhost:5173`, incluindo o cabeçalho `Authorization`.
