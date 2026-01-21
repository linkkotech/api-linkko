# Guia de Deploy – EasyPanel

Este documento descreve como configurar o Linkko API no EasyPanel para um ambiente de staging ou produção.

## 1. Configuração do Serviço

Ao criar o projeto no EasyPanel, utilize as seguintes definições:

-   **Source**: Repositório GitHub
-   **Build Method**: Dockerfile (o EasyPanel detectará automaticamente o `Dockerfile` na raiz)
-   **Port**: `3001` (ou conforme definido na env `PORT`)

## 2. Variáveis de Ambiente (Environment)

Configure as seguintes variáveis no painel de controle:

| Variável | Valor/Exemplo | Descrição |
| :--- | :--- | :--- |
| `DATABASE_URL` | `postgresql://user:pass@host:5432/db` | Conexão com o Supabase |
| `REDIS_URL` | `redis://:pass@easypanel-redis:6379` | Link do Redis interno do EasyPanel |
| `JWT_SECRET_CRM_V1` | `sua-chave-secreta` | Secret para HS256 |
| `JWT_PUBLIC_KEY_MCP_V1` | `-----BEGIN PUBLIC KEY...` | Chave pública para RS256 |
| `LOG_LEVEL` | `info` | Nível de log (debug, info, error) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://otel-collector:4317` | Opcional: Tracing OTel |

## 3. Fluxo de Migrations (Init Command)

O container principal roda o comando `serve` por padrão. Para garantir que as migrations sejam aplicadas antes do serviço subir:

1.  Vá em **General** -> **Advanced**.
2.  No campo **Initialization Command**, insira:
    ```bash
    /usr/local/bin/entrypoint.sh migrate
    ```
    *Isso garante que o container de "init" execute as migrations e encerre antes do container principal iniciar o `serve`.*

## 4. Health Check

Configure o Health Check para garantir a disponibilidade:

-   **Path**: `/v1/ready` (ou `/health` para liveness simples)
-   **Interval**: `10s`
-   **Timeout**: `5s`

## 5. Troubleshooting

-   **Permissões**: O Dockerfile já define um usuário `appuser` (UID 1000). Certifique-se de que volumes externos (se houver) permitam escrita por este ID.
-   **Logs**: Utilize `docker logs` ou o painel do EasyPanel para monitorar o `entrypoint.sh`.
