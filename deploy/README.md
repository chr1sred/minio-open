# MinIO Open — guia de implantação (CapRover + Typebot)

Este fork devolve ao MinIO tudo o que a comunidade perdeu nas últimas versões
e empacota o servidor para subir em poucos cliques numa VPS pequena com
CapRover, já integrado ao Typebot.

## O que muda em relação ao MinIO oficial

| Recurso | MinIO oficial (master / 2025) | MinIO Open (este fork) |
|---|---|---|
| Console: usuários, grupos, políticas | ❌ removido (só Object Browser) | ✅ |
| Console: OpenID / LDAP | ❌ removido | ✅ |
| Console: Lifecycle (expirar/mover objetos) | ❌ removido desde nov/2024 | ✅ |
| Console: Tiering, Site Replication | ❌ removido desde nov/2024 | ✅ |
| Console: Eventos, notificações, cotas, replicação de bucket | ❌ removido | ✅ |
| Console: Configurações do servidor, monitoramento, logs, trace | ❌ removido | ✅ |
| Imagens Docker / binários oficiais | ❌ não são mais publicados | ✅ imagem própria no GHCR (GitHub Actions) |
| Correções do servidor de 2025 | ✅ | ✅ (servidor do `master` mais recente) |
| Configuração automática p/ Typebot | — | ✅ bucket + pasta pública + usuário dedicado |
| Login do console atrás de proxy (CapRover) | ⚠️ falha se o domínio público não responder de dentro da VPS | ✅ `MINIO_CONSOLE_LOCAL_API=on` |
| Padrões para VPS de 2–4 GB | — | ✅ `MINIO_MEMLIMIT`, scanner lento, imagem Alpine enxuta |

Como foi feito: o servidor continua sendo o `master` mais recente; só o
console (`github.com/minio/console`) foi fixado na **v1.7.3** — a última versão
com a interface administrativa completa — copiada em `third_party/console`
com um patch mínimo de compatibilidade com o `minio-go` atual.

---

## Passo 1 — Publicar a imagem (automático)

A cada `push` neste branch, o workflow **“MinIO Open image”**
(`.github/workflows/open-image.yml`) compila a imagem e publica em:

```
ghcr.io/chr1sred/minio-open:latest
```

1. No GitHub: **Actions** → habilite os workflows do fork, se pedido →
   abra **MinIO Open image** → aguarde ficar verde (~10–15 min na 1ª vez).
   Também dá para rodar manualmente em **Run workflow**.
2. Em **github.com/chr1sred → Packages → minio-open → Package settings**,
   deixe a visibilidade como **Public** (assim o CapRover baixa sem login).
   Prefere privado? Cadastre o GHCR em **CapRover → Cluster → Docker Registries**
   com um token (`read:packages`).

> Por que não compilar na VPS? Compilar o MinIO usa 2–3 GB de RAM e ~10 min
> de CPU. Na sua VPS (3 vCPU / 4 GB) isso pode derrubar os outros apps.
> Mesmo assim, há a [alternativa B](#alternativa-b--compilar-na-própria-vps).

## Passo 2 — Criar os apps no CapRover (One-Click)

1. CapRover → **Apps** → **One-Click Apps/Databases**.
2. Role até o fim e escolha **`>> TEMPLATE <<`**.
3. Cole **todo** o conteúdo de
   [`deploy/caprover/minio-open-one-click.yml`](caprover/minio-open-one-click.yml).
4. Preencha o formulário (os padrões já servem; senhas são geradas
   automaticamente — **anote-as**):

| Campo | Padrão | Para que serve |
|---|---|---|
| Nome do app | ex.: `minio` | Gera `minio.seudominio` (S3) e `minio-console.seudominio` |
| Imagem | `ghcr.io/chr1sred/minio-open:latest` | Imagem do passo 1 |
| Usuário / senha admin | `admin` / aleatória | Login do console |
| Bucket do Typebot | `typebot` | Criado sozinho, com `public/` liberado para leitura |
| Usuário/senha S3 do Typebot | `typebot` / aleatória | Chave restrita ao bucket (o Typebot nunca usa o admin) |
| Limite de memória | `1GiB` | Teto de RAM do MinIO |

5. Clique **Deploy**. São criados dois apps:
   - **`minio`** → API S3 (porta 9000), com volume persistente `/data`;
   - **`minio-console`** → proxy nginx (~3 MB de RAM) para o console.

## Passo 3 — HTTPS

Nos **dois** apps: **HTTP Settings → Enable HTTPS → Force HTTPS**.
No `minio-console`, confirme que **Websocket Support** está marcado
(o template já pede isso; algumas telas — logs, trace — usam websocket).

Usando domínio próprio (ex.: `s3.empresa.com` e `minio.empresa.com`)?
Conecte-o em **HTTP Settings** e atualize, no app `minio`, as variáveis
`MINIO_SERVER_URL=https://s3.empresa.com` e
`MINIO_BROWSER_REDIRECT_URL=https://minio.empresa.com`.

## Passo 4 — Conferir o guia nos logs

Abra o app `minio` → **App Logs**. Ao subir, o `minio-bootstrap` imprime:

```
[bootstrap] bucket "typebot" created
[bootstrap] anonymous download enabled on typebot/public
[bootstrap] user "typebot" created
[bootstrap] policy "app-typebot-rw" attached to "typebot" (access limited to: typebot)
========================================================================
 MinIO pronto!  /  MinIO is ready!
========================================================================
 Console (admin completo) : https://minio-console.seudominio.com
 API S3                   : https://minio.seudominio.com
 ...
 Cole no Typebot (builder E viewer) -> App Configs > Environment Variables:

   S3_ACCESS_KEY=typebot
   S3_SECRET_KEY=<o valor de MINIO_INIT_APP_PASSWORD deste app>
   S3_BUCKET=typebot
   S3_ENDPOINT=minio.seudominio.com
   S3_SSL=true
   S3_REGION=us-east-1
```

Ele é **idempotente**: roda a cada reinício, mas nunca apaga nem sobrescreve
nada que já exista (senhas, políticas de bucket editadas por você etc.).

## Passo 5 — Configurar o Typebot

No CapRover, nos apps do **builder** e do **viewer** do Typebot, adicione as
variáveis acima (modelo em [`deploy/typebot.env.example`](typebot.env.example))
e clique **Save & Update**.

| Variável Typebot | Valor | Observação |
|---|---|---|
| `S3_ACCESS_KEY` | usuário do Typebot | não use o admin |
| `S3_SECRET_KEY` | senha do Typebot | |
| `S3_BUCKET` | `typebot` | |
| `S3_ENDPOINT` | `minio.seudominio.com` | só o host: sem `https://`, sem `/` |
| `S3_SSL` | `true` | |
| `S3_REGION` | `us-east-1` | |
| `S3_PORT` | (vazio) | só se usar porta diferente de 443 |
| `S3_PUBLIC_CUSTOM_DOMAIN` | (opcional) | CDN/domínio na frente dos arquivos públicos |

Teste: num fluxo do Typebot, envie uma imagem num bloco de upload; ela deve
aparecer no console em **Object Browser → typebot → public/…**.

---

## Alternativa B — compilar na própria VPS

O `captain-definition` na raiz aponta para o `Dockerfile.open`, então dá
para criar um app comum no CapRover e usar **Deploy → Method 3 (GitHub)**
com este repositório/branch. Recomendações:

- crie **swap de 2–4 GB** na VPS antes (`fallocate -l 4G /swapfile && chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile`);
- adicione a variável `WITH_MC=false` para pular o `mc` e encurtar o build;
- `BUILD_JOBS=1` deixa o build mais lento, porém mais leve.

Nesse caso crie o app do console à mão (ou use o template e só troque a
imagem do app principal depois).

## Docker Compose (sem CapRover)

```bash
cp deploy/.env.example deploy/.env      # edite as senhas
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d --build
```

---

## Variáveis de ambiente

| Variável | Padrão na imagem | Descrição |
|---|---|---|
| `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` | — | Credenciais admin (obrigatórias) |
| `MINIO_SERVER_URL` | — | URL pública da API S3 (links e presigned URLs) |
| `MINIO_BROWSER_REDIRECT_URL` | — | URL pública do console |
| `MINIO_CONSOLE_LOCAL_API` | `on` | Console fala com a API via 127.0.0.1 (login não depende de DNS/SSL) — **novo neste fork** |
| `MINIO_MEMLIMIT` | `1GiB` | Teto de memória do Go; também ajusta o nº máximo de requisições simultâneas |
| `MINIO_SCANNER_SPEED` | `slow` | Scanner de fundo mais leve (`fastest`…`slowest`) |
| `MINIO_UPDATE` | `off` | Desliga a checagem de update em dl.min.io |
| `MINIO_BROWSER_SESSION_DURATION` | `12h` | Duração da sessão no console |
| `MINIO_VOLUMES` | `/data` | Diretório(s) de dados |
| `MINIO_OPTS` | `--console-address :9001` | Argumentos extras do `minio server` |
| `MINIO_INIT_ENABLE` | `on` | `off` desativa o bootstrap |
| `MINIO_INIT_BUCKETS` | `typebot` | Buckets a criar (separe por vírgula) |
| `MINIO_INIT_PUBLIC_PATHS` | `typebot/public` | `bucket/prefixo` com leitura anônima (somente download) |
| `MINIO_INIT_APP_USER` / `MINIO_INIT_APP_PASSWORD` | — | Usuário de aplicação a criar |
| `MINIO_INIT_APP_BUCKETS` | = `MINIO_INIT_BUCKETS` | Buckets que esse usuário acessa |

Exemplo com vários apps: `MINIO_INIT_BUCKETS=typebot,n8n,backups` e
`MINIO_INIT_PUBLIC_PATHS=typebot/public`.

## Ajuste fino para a VPS (3 vCPU / 4 GB / 65 GB)

Orçamento de RAM sugerido:

| Serviço | RAM típica |
|---|---|
| CapRover + nginx | ~250 MB |
| MinIO Open (ocioso → com uploads) | ~300 MB → até 1 GB |
| Typebot builder + viewer | ~600–900 MB |
| PostgreSQL | ~150–300 MB |
| Folga do sistema | ~1 GB |

Para impor um teto rígido ao container (além do `MINIO_MEMLIMIT`), no app
`minio` → **App Configs → Service Update Override**:

```json
{ "TaskTemplate": { "Resources": { "Limits": { "MemoryBytes": 1610612736 } } } }
```

(= 1,5 GB). Mantenha o limite ~50% acima de `MINIO_MEMLIMIT`.

Dicas de disco: crie uma regra de **Lifecycle** (console → Buckets →
typebot → Lifecycle) para expirar arquivos temporários, e acompanhe o uso
em **Monitoring → Metrics**.

## Comandos úteis (`mc` já vem na imagem)

No servidor (SSH), ache o container do app `minio` e entre nele:

```bash
docker ps --format '{{.Names}}' | grep 'srv-captain--minio\.'
```

```bash
docker exec -it <nome-que-apareceu-acima> sh
```

Dentro dele:

```bash
mc alias set local http://127.0.0.1:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD"
```

```bash
mc admin info local
```

```bash
mc ls local/typebot/public
```

```bash
mc anonymous get local/typebot
```

## Solução de problemas

| Sintoma | Causa provável | Correção |
|---|---|---|
| Template falha no deploy: “pull access denied” | Pacote do GHCR privado ou ainda não publicado | Passo 1 (Actions verde + pacote público) |
| Console abre mas login dá erro | `MINIO_CONSOLE_LOCAL_API` desligado e domínio S3 sem HTTPS | Ligue `MINIO_CONSOLE_LOCAL_API=on` ou ative o HTTPS do app S3 |
| Console redireciona para endereço errado | `MINIO_BROWSER_REDIRECT_URL` incorreto | Ajuste para a URL pública do app `-console` |
| Upload no Typebot falha (403/SignatureDoesNotMatch) | `S3_ENDPOINT` com `https://` ou `/`, ou chave errada | Use só o host; confira `S3_ACCESS_KEY`/`S3_SECRET_KEY` |
| Imagens do Typebot não aparecem (403) | Pasta `public/` sem leitura anônima | Console → Buckets → typebot → Anonymous → prefixo `public/` = readonly |
| Upload grande falha com 413 | Limite de corpo do nginx do CapRover | Aumente `client_max_body_size` na config nginx do app `minio` |
| Telas de Logs/Trace não carregam | Websocket desligado no app do console | HTTP Settings → Websocket Support |

## Arquivos deste fork

| Arquivo | Função |
|---|---|
| `third_party/console/` | Console v1.7.3 completo (+ patch de compatibilidade) |
| `cmd/common-main.go` | Opção `MINIO_CONSOLE_LOCAL_API` |
| `deploy/bootstrap/` | `minio-bootstrap`: buckets, acesso público, usuário do app, guia nos logs |
| `Dockerfile.open` | Imagem Alpine compilada do código-fonte (minio + bootstrap + mc) |
| `dockerscripts/open-entrypoint.sh` | Entrypoint: sobe o servidor e o bootstrap |
| `captain-definition` | Deploy direto do GitHub no CapRover (compila na VPS) |
| `deploy/caprover/minio-open-one-click.yml` | Template One-Click (S3 + Console) |
| `deploy/docker-compose.yml`, `deploy/.env.example` | Uso fora do CapRover |
| `deploy/typebot.env.example` | Variáveis prontas para o Typebot |
| `.github/workflows/open-image.yml` | Publica a imagem no GHCR |

Licença: GNU AGPLv3 (igual ao MinIO). Ao oferecer este serviço modificado a
terceiros, o código-fonte das modificações deve continuar disponível — este
repositório público já cumpre isso.
