# Rate Limiter

Um middleware HTTP em Go que limita requisições por
endereço IP ou por token de acesso, com a persistência plugável através do padrão Strategy.

Quando o limite é excedido, o cliente recebe **429** e fica bloqueado por um tempo configurável.

## Como rodar

Tudo funciona só com Docker:

```bash
make up      # sobe redis + app, com a app em :8080
make logs    # acompanha os logs
make down    # derruba tudo e remove os volumes
```

O `docker-compose.yaml` já define as variáveis necessárias, então **não é preciso criar um `.env`**
para subir o projeto.

Para rodar sem Docker nenhum, a strategy em memória dispensa infra externa:

```bash
STORAGE_STRATEGY=memory go run ./cmd/ratelimiter
```

### Testes

```bash
make test         # local
make test-docker   
```

## Vendo o limiter funcionar

Com `make up`, os limites são 10 req/s por IP e 100 req/s por token.

**Limite por IP**  
As dez primeiras respondem `ok`, da 11ª em diante vem a mensagem de bloqueio:

```bash
for i in $(seq 1 12); do curl -s localhost:8080/; echo; done
```
```
ok
ok
ok
ok
ok
ok
ok
ok
ok
ok
you have reached the maximum number of requests or actions allowed within a certain time frame
you have reached the maximum number of requests or actions allowed within a certain time frame
```

**Com token**  
O mesmo IP que acabou de ser bloqueado, agora enviando o header, passa a
valer 100 req/s e nenhuma requisição é barrada:

```bash
for i in $(seq 1 12); do curl -s localhost:8080/ -H "API_KEY: test123"; echo; done
```
```
ok
ok
ok
ok
ok
ok
ok
ok
ok
ok
ok
ok
```

**O estado no Redis**:

```bash
docker compose exec redis redis-cli --scan --pattern 'ratelimit:*'
# ratelimit:count:ip:172.19.0.1:1785244718
# ratelimit:block:ip:172.19.0.1
```

Para repetir os testes sem esperar o bloqueio expirar:
`docker compose exec redis redis-cli FLUSHALL`.

## Configuração

Variáveis de ambiente. Com default, a aplicação sobe mesmo sem nenhuma configurada.
Um `.env` na raiz é carregado quando existe : [.env.example](.env.example).

| Variável | Default | O que faz |
| --- | --- | --- |
| `HTTP_PORT` | `8080` | porta do servidor |
| `HTTP_GRACEFUL_TIMEOUT` | `10s` | quanto esperar as requisições em voo no shutdown |
| `STORAGE_STRATEGY` | `memory` | `memory` ou `redis` |
| `RATE_LIMIT_IP_MAX_REQUESTS` | `10` | requisições por segundo por IP |
| `RATE_LIMIT_IP_BLOCK_DURATION` | `1m` | quanto tempo o IP fica bloqueado após estourar |
| `RATE_LIMIT_TOKEN_MAX_REQUESTS` | `100` | requisições por segundo por token |
| `RATE_LIMIT_TOKEN_BLOCK_DURATION` | `1m` | quanto tempo o token fica bloqueado após estourar |
| `REDIS_ADDR` | `localhost:6379` | endereço do Redis |
| `REDIS_PASSWORD` | vazio | senha do Redis |
| `REDIS_DB` | `0` | banco do Redis |
| `MEMORY_CLEANUP_INTERVAL` | `1m` | varredura das chaves vencidas na strategy em memória |

A ordem de prioridade é: 

variável de ambiente > `.env` > default declarado na tag. 

O ambiente sempre vence,
comportamento necessário para o docker compose, onde não existe `.env` dentro do container.

Configuração inválida derruba a aplicação **no boot**, não a cada requisição:

```bash
REDIS_DB=abc STORAGE_STRATEGY=redis go run ./cmd/ratelimiter
# {"level":"ERROR","msg":"application terminated",
#  "error":"env: parse error on field \"DB\" of type \"int\": strconv.ParseInt: parsing \"abc\": invalid syntax"}
```

>Nota:
 o docker-compose fixa esses valores em `environment:`, então variáveis do shell não o afetam para testar, use `docker compose run -e VAR=valor app` ou rode localmente como acima.

## Regras de limitação

- **Por IP**: o IP de origem da conexão define o grupo.
- **Por token**: quando o header `API_KEY` vem preenchido, ele define o grupo.
- **Precedência**: o limite do token **substitui** o do IP; Não soma, e não escolhe o menor. Com IP
  a 10 req/s e token a 100 req/s, uma requisição com token vale 100 req/s, mesmo que aquele IP já
  esteja bloqueado.

A janela é fixa de 1 segundo. Ao estourar o limite, a chave de bloqueio é gravada com o TTL
configurado e **todas** as requisições seguintes são rejeitadas até ele expirar, inclusive as de
uma janela nova.

Com `RATE_LIMIT_IP_MAX_REQUESTS=10`, exatamente 10 requisições passam: a décima ainda é atendida, a
décima-primeira dispara o bloqueio.

> **Sobre o token:** o desafio não define emissão nem validação de tokens, e o limite configurado é
> global. Portanto qualquer valor não-vazio em `API_KEY` é aceito e recebe o limite de token. A
> consequência prática é que um cliente bloqueado por IP consegue contornar o limite inventando um
> token sem autenticação.

## Trocando a strategy de persistência

Só uma variável de ambiente:

```bash
STORAGE_STRATEGY=memory make up   # sem Redis
STORAGE_STRATEGY=redis make up    # com Redis
```

### Adicionando uma nova strategy

A interface do domínio, em [internal/entity/interface.go](internal/entity/interface.go), tem três
métodos:

```go
type LimiterStorage interface {
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)
	IsBlocked(ctx context.Context, key string) (bool, error)
	Block(ctx context.Context, key string, duration time.Duration) error
}
```

Para plugar um backend novo:

1. Implemente a interface em `internal/infra/storage/<nome>.go`.
2. Declare `<Nome>StorageOptions` com as tags `env` em
   [internal/infra/storage/options.go](internal/infra/storage/options.go).
3. Acrescente o `case` em [internal/infra/storage/factory.go](internal/infra/storage/factory.go).

## Arquitetura

Clean Architecture, espelhando o layout do `20-CleanArch` do curso.

```
cmd/ratelimiter/       wiring: config → storage → usecase → middleware → webserver
configs/               carregamento do .env e os limites da aplicação
internal/
  entity/              regra pura: precedência, resolução de limite e formato das chaves
  usecase/             orquestração do fluxo de verificação
  infra/
    storage/           as strategies (memory, redis) e a factory
    web/middleware/    o middleware HTTP
    web/webserver/     encapsula o echo
test/integration/      concorrência HTTP real contra memória e Redis
```

O fluxo de uma requisição:

1. O middleware extrai o IP e o header `API_KEY` e delega.
2. O usecase resolve qual limite vale, consulta o bloqueio, incrementa o contador e decide.
3. A strategy só sabe contar, expirar e bloquear chaves.

### Decisões

**Chave com o segundo embutido.** O contador é gravado em
`ratelimit:count:<tipo>:<id>:<unix_second>`. Como a chave muda a cada segundo, a janela expira
sozinha em qualquer backend, sem código de expiração específico. O bloqueio fica numa chave
separada, `ratelimit:block:<tipo>:<id>`.

**A resposta 429 é texto puro.** O corpo é exatamente a mensagem exigida pelo enunciado, sem
envelope JSON e sem newline final.

## Stack

* Go 1.26, [echo v5](https://github.com/labstack/echo) como framework HTTP
* [go-redis v9](https://github.com/redis/go-redis)
* [caarlos0/env](https://github.com/caarlos0/env) para configuração e
* [testify](https://github.com/stretchr/testify) nos testes.

## Limitações conhecidas

- Não há autenticação: qualquer `API_KEY` não-vazia é aceita.
- O limite de token é único e global, não configurável por token(mas daria para ser).
- Atrás de um proxy reverso, o IP precisaria de configuração adicional para ser identificado.
