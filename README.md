# CLEAN – Monitoramento ambiental (MVP)

Sistema de monitoramento de temperatura, umidade e luminosidade com:

- **Backend em Go** (Gin, PostgreSQL, integração com Ollama).
- **Simulador de sensores** (DHT22, MQ-135 e BH1750 simulados) que substitui o hardware nesta versão.
- **Frontend** em React + TypeScript + Vite + shadcn/ui + Recharts.
- **Chat controlado** por intenções: o modelo local interpreta a pergunta e formula a resposta, mas nunca executa SQL nem inventa valores.

## Requisitos

| Ferramenta | Versão | Uso |
|---|---|---|
| Go | 1.27+ | backend |
| Node.js | 22+ | frontend |
| Docker Desktop | recente | PostgreSQL (e, opcionalmente, o backend em container) |
| Ollama | recente | modelo de linguagem local |

## 1. Ollama

Instale o Ollama e baixe o modelo padrão:

```bash
ollama pull qwen3.5:4b
```

O backend local acessa `http://localhost:11434`. Se o backend rodar **dentro do Docker**, o Ollama precisa escutar em todas as interfaces do host. No Windows, defina a variável de ambiente de usuário `OLLAMA_HOST=0.0.0.0:11434` e reinicie o Ollama.

## 2. Configuração

```bash
cp .env.example .env
```

Edite `.env` e troque `POSTGRES_PASSWORD` (e o mesmo valor em `DATABASE_URL`). O arquivo `.env` não deve ser versionado; ele já está no `.gitignore`.

Principais variáveis:

| Variável | Padrão | Descrição |
|---|---|---|
| `POSTGRES_HOST_PORT` | `5433` | porta do host para o PostgreSQL do container (`5432` costuma estar ocupada) |
| `DATABASE_URL` | – (obrigatória) | conexão usada pelo backend local |
| `HTTP_ADDR` | `:18080` | endereço do backend local |
| `APP_TIMEZONE` | `America/Sao_Paulo` | fuso usado para períodos como "hoje" e "ontem" |
| `OLLAMA_URL` | `http://localhost:11434` | endpoint do Ollama |
| `OLLAMA_MODEL` | `qwen3.5:4b` | modelo usado no chat |
| `OLLAMA_TIMEOUT_SECONDS` | `60` | tempo máximo por chamada ao modelo |
| `OLLAMA_THINK` | `false` | raciocínio do modelo (mais lento se ativado) |
| `OLLAMA_KEEP_ALIVE` | `10m` | tempo que o modelo permanece carregado na memória |
| `OLLAMA_NUM_PREDICT` | `220` | limite de tokens gerados por resposta |
| `SIMULATOR_ENABLED` | `true` | liga o simulador de sensores |
| `SIMULATOR_INTERVAL_SECONDS` | `30` | intervalo entre medições simuladas |
| `SIMULATOR_BACKFILL_HOURS` | `24` | histórico gerado na primeira inicialização |
| `API_HOST_PORT` | `18080` | porta do host para o backend no docker-compose |

> **Nota sobre portas:** no Windows, as portas 7991–8090 costumam estar reservadas (Hyper-V/WSL). Por isso o padrão do projeto é `18080`, e não `8080`. Verifique com `netsh interface ipv4 show excludedportrange protocol=tcp`.

## 3. Banco de dados

Com Docker:

```bash
docker-compose up -d postgres
```

(Se o seu Docker tiver o plugin Compose, `docker compose` também funciona.) O banco fica em `localhost:5433` e os dados persistem no volume `pgdata`. As migrações são aplicadas automaticamente pelo backend na inicialização.

## 4. Backend

```bash
cd backend
go run ./cmd/api
```

Na primeira execução, o backend aplica as migrações e gera o histórico simulado das últimas 24 horas. Nas execuções seguintes, ele continua a partir da última medição (não há duplicatas, pois as medições são determinísticas por horário).

## 5. Frontend

```bash
cd frontend
npm install
npm run dev
```

Abra `http://localhost:5173`. O Vite encaminha `/api` para `http://localhost:18080`. Para apontar para outro backend, defina `VITE_API_PROXY_TARGET` no ambiente ou em `frontend/.env.local`.

## 6. Docker Compose (PostgreSQL + backend)

```bash
docker-compose up -d --build
```

Sobe o PostgreSQL e o backend na porta `18080` do host. O backend acessa o Ollama do host por `host.docker.internal` (configurado em `OLLAMA_URL_FROM_DOCKER`). O frontend continua rodando localmente com `npm run dev`.

## Endpoints da API

Base: `http://localhost:18080/api`

| Método | Caminho | Descrição |
|---|---|---|
| GET | `/health` | estado do banco, do Ollama, do modelo e da fonte de dados |
| GET | `/sensors/latest` | última medição |
| GET | `/sensors/history?from=…&to=…&bucket=15m` | histórico agregado (`bucket`: `1m`, `5m`, `15m`, `1h`) ou bruto com `limit` |
| GET | `/dashboard/summary?period=last_24_hours` | resumo (`last_hour`, `last_2_hours`, `last_24_hours`, `today`) |
| POST | `/chat` | pergunta em linguagem natural; corpo `{"message": "..."}` |

Exemplos:

```bash
curl -s http://localhost:18080/api/health
curl -s http://localhost:18080/api/sensors/latest
curl -s "http://localhost:18080/api/sensors/history?from=2026-10-27T08:00:00-03:00&to=2026-10-27T12:00:00-03:00&bucket=15m"
curl -s "http://localhost:18080/api/dashboard/summary?period=last_24_hours"
curl -s -X POST http://localhost:18080/api/chat -H "Content-Type: application/json" \
  -d '{"message":"Qual foi a temperatura média nas últimas 24 horas?"}'
```

Perguntas suportadas pelo chat: leituras atuais; médias, máximas e mínimas de temperatura e umidade; tendência; comparação de períodos; quando um limite foi ultrapassado. Períodos: última hora, últimas 2 horas, últimas 24 horas, hoje, ontem, últimos 7 dias, esta semana (desde segunda-feira), semana passada ou intervalo de datas.

Se o Ollama estiver indisponível, o chat responde com erro (HTTP 503) e uma mensagem amigável; nenhuma resposta da IA é inventada.

## Testes

```bash
cd backend
go test ./...
go vet ./...

cd ../frontend
npm run build
```

Os testes do backend usam fakes e `httptest`; não exigem PostgreSQL nem Ollama.

## Estrutura

```
backend/
  cmd/api/              ponto de entrada (configuração, migrações, simulador, HTTP)
  internal/api/         rotas e handlers Gin
  internal/chat/        intents, validação, prompts, cliente Ollama, fatos em pt-BR
  internal/config/      carregamento de variáveis de ambiente
  internal/ingest/      consumo das medições de uma fonte (Source) e gravação
  internal/sensors/     modelo Measurement e interface Source (futuro MQTT)
  internal/simulator/   simulador determinístico e backfill
  internal/storage/     PostgreSQL (pgx) e migrações
  migrations/           SQL versionado
frontend/               aplicação React
docker-compose.yml      PostgreSQL e backend
.env.example            modelo de configuração (sem credenciais reais)
```

## Decisões principais

- **Chat sem SQL gerado pelo modelo.** O modelo devolve apenas um JSON com intent, métrica, período e limite (validado contra um schema e uma lista branca). A consulta é feita por código com parâmetros, e os fatos são formatados no servidor. O modelo só redige a frase final a partir desses fatos.
- **Ollama não é exposto ao frontend.** Todas as chamadas passam pelo backend.
- **Simulador determinístico.** O valor de cada medição depende apenas do horário, o que torna o backfill e o restart idempotentes.
- **Fonte de dados abstraída.** A interface `sensors.Source` permite plugar MQTT (ESP32) depois, sem mudar a API.

## Limitações

- Não há hardware real: os dados vêm do simulador. MQTT ainda não foi implementado (apenas a interface).
- Sem autenticação; a API e o banco são pensados para uso local.
- O chat depende de um modelo local pequeno. A interpretação pode falhar em perguntas ambíguas; nesses casos o sistema pede esclarecimento em vez de chutar.
- O primeiro carregamento do `qwen3.5:4b` pode demorar dezenas de segundos; ajuste `OLLAMA_TIMEOUT_SECONDS` se necessário.
- Períodos e datas são calculados no fuso `APP_TIMEZONE`; o banco armazena `TIMESTAMPTZ` em UTC.
- O histórico consultável tem limite de 31 dias por requisição.

## Problemas comuns

- **`DATABASE_URL é obrigatória`**: crie o `.env` a partir do `.env.example`.
- **Porta 5433 ou 18080 ocupada**: altere `POSTGRES_HOST_PORT`, `DATABASE_URL` ou `HTTP_ADDR`.
- **`bind: ... access permissions` na porta 8080 (Windows)**: a porta está reservada pelo sistema; use `18080`.
- **Chat responde "A IA está indisponível"**: confirme que o Ollama está rodando (`ollama list`) e que o modelo foi baixado. Em Docker, confirme `OLLAMA_HOST=0.0.0.0:11434`.
- **Erro do runner do Ollama (CUDA / `llama-server process has terminated`)**: problema do driver da GPU ou do modelo; reinicie o Ollama, atualize o driver ou use `OLLAMA_MODEL=qwen3.5:2b` (modelo menor, também instalado com `ollama pull qwen3.5:2b`).
