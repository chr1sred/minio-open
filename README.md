# MinIO Open

**MinIO com o console administrativo completo de volta**, empacotado para subir
em poucos cliques numa VPS pequena com **CapRover**, e já integrado ao **Typebot**.

Fork de [minio/minio](https://github.com/minio/minio), que entrou em modo de
manutenção em 2025: a comunidade perdeu o console administrativo e deixou de
receber imagens Docker e binários oficiais. Este fork devolve as duas coisas.

[![license](https://img.shields.io/badge/license-AGPL%20V3-blue)](LICENSE)
[![image](https://img.shields.io/badge/imagem-ghcr.io%2Fchr1sred%2Fminio--open-2ea44f)](https://github.com/chr1sred/minio-open/pkgs/container/minio-open)

---

## Por que este fork

| Recurso | MinIO oficial (2025) | MinIO Open |
|---|---|---|
| Console: usuários, grupos, políticas, access keys | ❌ removido (sobrou só o Object Browser) | ✅ |
| Console: OpenID / LDAP | ❌ removido | ✅ |
| Console: Lifecycle (expirar/mover objetos) | ❌ removido desde nov/2024 | ✅ |
| Console: Tiering, Site Replication | ❌ removido desde nov/2024 | ✅ |
| Console: eventos, notificações, cotas, replicação de bucket | ❌ removido | ✅ |
| Console: configurações do servidor, monitoramento, logs, trace | ❌ removido | ✅ |
| Imagem Docker pronta | ❌ não é mais publicada | ✅ `ghcr.io/chr1sred/minio-open` |
| Correções do servidor de 2025 | ✅ | ✅ (servidor do `master` mais recente) |
| Configuração automática para o Typebot | — | ✅ bucket, pasta pública e usuário dedicado |
| Login do console atrás de proxy (CapRover, Traefik) | ⚠️ falha se o domínio público não responde de dentro da VPS | ✅ `MINIO_CONSOLE_LOCAL_API=on` |
| Padrões para VPS de 2–4 GB | — | ✅ ~160 MB de RAM ocioso, teto configurável |

**Como foi feito:** o servidor é o `master` mais recente do MinIO. Só o console
(`github.com/minio/console`) foi fixado na **v1.7.3**, a última versão com a
interface administrativa completa. Ele fica em
[`third_party/console`](third_party/console), com um patch mínimo de
compatibilidade com o `minio-go` atual.

---

## Começo rápido

### CapRover (recomendado)

1. **CapRover → Apps → One-Click Apps/Databases → `>> TEMPLATE <<`**.
2. Cole o conteúdo de [`deploy/caprover/minio-open-one-click.yml`](deploy/caprover/minio-open-one-click.yml)
   (versão em texto puro: [raw](https://raw.githubusercontent.com/chr1sred/minio-open/master/deploy/caprover/minio-open-one-click.yml)).
3. Nomeie o app (ex.: `minio-open`), anote as senhas geradas e clique em **Deploy**.
4. Ative **HTTPS** nos dois apps criados (`minio-open` e `minio-open-console`).
5. Copie as variáveis `S3_*` dos **App Logs** do `minio-open` para o builder e o viewer do Typebot.

Passo a passo completo, com prints do que esperar, variáveis e solução de
problemas: **[deploy/README.md](deploy/README.md)**.

### Docker

```bash
docker run -d --name minio -p 9000:9000 -p 9001:9001 -v minio-data:/data -e MINIO_ROOT_USER=admin -e MINIO_ROOT_PASSWORD=troque-esta-senha ghcr.io/chr1sred/minio-open:latest
```

API S3 em `http://localhost:9000` e console em `http://localhost:9001`.

### Docker Compose

```bash
cp deploy/.env.example deploy/.env
```

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

---

## O que vem na imagem

| Componente | Descrição |
|---|---|
| `minio` | Servidor S3 + console admin completo (porta 9000 API, 9001 console) |
| `minio-bootstrap` | Roda a cada início, sem nunca sobrescrever nada: cria buckets, libera leitura pública de prefixos, cria um usuário de aplicação com política restrita e imprime nos logs o guia de configuração do Typebot |
| `mc` | MinIO Client, para administração pela linha de comando |

Base Alpine, ~160 MB, healthcheck embutido em `/minio/health/live`.

### Variáveis principais

| Variável | Padrão | Descrição |
|---|---|---|
| `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` | — | Credenciais admin (obrigatórias) |
| `MINIO_SERVER_URL` | — | URL pública da API S3 |
| `MINIO_BROWSER_REDIRECT_URL` | — | URL pública do console |
| `MINIO_CONSOLE_LOCAL_API` | `on` | Console fala com a API via 127.0.0.1 (novo neste fork) |
| `MINIO_MEMLIMIT` | `1GiB` | Teto de memória; também ajusta a concorrência máxima |
| `MINIO_SCANNER_SPEED` | `slow` | Scanner de fundo mais leve |
| `MINIO_INIT_BUCKETS` | — | Buckets criados automaticamente (o template do CapRover usa `typebot`) |
| `MINIO_INIT_PUBLIC_PATHS` | — | Prefixos com leitura anônima, somente download (o template usa `typebot/public`) |
| `MINIO_INIT_APP_USER` / `MINIO_INIT_APP_PASSWORD` | — | Usuário de aplicação a criar |

Lista completa em [deploy/README.md](deploy/README.md#variáveis-de-ambiente).

---

## Integração com o Typebot

O bootstrap deixa o MinIO no formato que o Typebot espera: bucket `typebot`,
prefixo `public/` com leitura anônima e um usuário com acesso só a esse bucket.
No builder **e** no viewer do Typebot:

| Variável | Valor |
|---|---|
| `S3_ACCESS_KEY` | usuário do Typebot (ex.: `typebot`) |
| `S3_SECRET_KEY` | senha do Typebot |
| `S3_BUCKET` | `typebot` |
| `S3_ENDPOINT` | `minio-open.seudominio.com` (só o host, sem `https://`) |
| `S3_SSL` | `true` |
| `S3_REGION` | `us-east-1` |

Modelo pronto em [`deploy/typebot.env.example`](deploy/typebot.env.example).
Validado em produção com a versão atual do Typebot: upload pelo builder e
exibição pública das imagens.

---

## Compilar a partir do código-fonte

Requer Go 1.24+.

```bash
CGO_ENABLED=0 go build -trimpath -o minio .
```

```bash
CGO_ENABLED=0 go build -trimpath -o minio-bootstrap ./deploy/bootstrap
```

Imagem Docker:

```bash
docker build -f Dockerfile -t minio-open .
```

A cada push no `master`, o workflow [`open-image.yml`](.github/workflows/open-image.yml)
publica `ghcr.io/chr1sred/minio-open:latest`.

---

## Estrutura do fork

| Caminho | Função |
|---|---|
| `third_party/console/` | Console v1.7.3 completo (+ patch de compatibilidade) |
| `cmd/common-main.go` | Opção `MINIO_CONSOLE_LOCAL_API` |
| `deploy/bootstrap/` | Código do `minio-bootstrap` |
| `deploy/caprover/` | Template One-Click do CapRover |
| `deploy/docker-compose.yml`, `deploy/.env.example` | Uso fora do CapRover |
| `deploy/typebot.env.example` | Variáveis prontas para o Typebot |
| `deploy/README.md` | Guia completo de implantação |
| `Dockerfile`, `dockerscripts/open-entrypoint.sh` | Imagem Docker |
| `captain-definition` | Deploy direto do GitHub no CapRover (compila na VPS) |
| `.github/workflows/open-image.yml` | Publicação da imagem no GHCR |

O restante do repositório é o MinIO original, sem alterações.

---

## Limitações conhecidas

- O console é de **nov/2024 (v1.7.3)**. As versões 1.7.4–1.7.6 trouxeram
  basicamente as remoções de recursos. Use sempre HTTPS e senhas fortes.
- O menu **Subnet** do console (License, Health, Call Home…) pertence ao
  serviço comercial da MinIO e não tem função aqui; pode ser ignorado.
- Pensado para instalação em um único servidor/disco (`/data`). Para
  clusters distribuídos, consulte a documentação original do MinIO.

---

## Licença e créditos

- Código-fonte: [GNU AGPLv3](LICENSE), igual ao MinIO. Ao oferecer este
  serviço modificado a terceiros, o código das modificações deve continuar
  disponível; este repositório público já cumpre isso.
- Documentação original do MinIO: [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
- [Conformidade de licença](COMPLIANCE.md) · [Créditos](CREDITS)

MinIO é marca registrada da MinIO, Inc. Este é um fork comunitário e
independente, sem afiliação com a MinIO, Inc.
