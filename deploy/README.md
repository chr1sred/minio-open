# MinIO Open — guia de implantação (CapRover + Typebot)

Guia completo para subir o MinIO Open numa VPS com CapRover e ligá-lo ao
Typebot. Testado numa VPS Debian 13 com 3 vCPU, 4 GB de RAM e 65 GB de disco.
O resumo do projeto está no [README principal](../README.md).

**Tempo total:** ~15 minutos.

| Etapa | O que acontece |
|---|---|
| [1. Imagem](#passo-1--imagem-docker) | Imagem pronta no GHCR (automático) |
| [2. Criar os apps](#passo-2--criar-os-apps-no-caprover-one-click) | Template One-Click cria S3 + Console |
| [3. HTTPS](#passo-3--https) | Certificados nos dois apps |
| [4. Conferir](#passo-4--conferir-logs-e-console) | Logs do bootstrap + console |
| [5. Typebot](#passo-5--configurar-o-typebot) | Variáveis `S3_*` no builder e viewer |
| [6. Testar](#passo-6--testar) | Upload real pelo Typebot |

---

## Passo 1 — Imagem Docker

A cada push no `master`, o workflow **“MinIO Open image”**
([`.github/workflows/open-image.yml`](../.github/workflows/open-image.yml))
compila e publica:

```
ghcr.io/chr1sred/minio-open:latest
```

- Confira em **GitHub → Actions → MinIO Open image** que o último run está verde (~5 min).
- O pacote herda a visibilidade do repositório (público), então o CapRover
  baixa a imagem sem login. Se o repositório for privado, cadastre o GHCR em
  **CapRover → Cluster → Docker Registries** com um token `read:packages`.

> **Por que não compilar na VPS?** Compilar o MinIO consome 2–3 GB de RAM e
> vários minutos de CPU, o que pode derrubar os outros apps de uma VPS de 4 GB.
> Se mesmo assim quiser, veja a [alternativa B](#alternativa-b--compilar-na-própria-vps).

## Passo 2 — Criar os apps no CapRover (One-Click)

> ⚠️ Não use **“Create from scratch”**: ele cria um app só, vazio. O template
> cria os **dois** apps necessários, já configurados.

1. **Apps → Apps/Bancos de Dados One-Click**.
2. Busque por `template` e escolha **`>> TEMPLATE <<`**.
3. Na caixa **Custom One-Click App Template**, cole o **conteúdo** do arquivo
   (não a URL, porque o CapRover não baixa links):
   - abra o [template em texto puro](https://raw.githubusercontent.com/chr1sred/minio-open/master/deploy/caprover/minio-open-one-click.yml);
   - **Ctrl/Cmd + A**, copie e cole na caixa;
   - clique em **Próximo**.
4. Preencha o formulário:

| Campo | Valor recomendado | Observação |
|---|---|---|
| App Name | `minio-open` | Não reutilize o nome de um app que já exista (ex.: um `minio` antigo) |
| Imagem Docker | `ghcr.io/chr1sred/minio-open:latest` | Padrão |
| Usuário admin | `admin` | Login do console |
| Senha admin | gerada automaticamente | **Anote** |
| Bucket do Typebot | `typebot` | Criado com `public/` liberado para leitura |
| Usuário S3 do Typebot | `typebot` | Chave restrita ao bucket |
| Senha S3 do Typebot | gerada automaticamente | **Anote**; vira o `S3_SECRET_KEY` |
| Limite de memória | `1GiB` | Ideal para VPS de 4 GB |

5. Clique em **Deploy** e aguarde 1–3 minutos. São criados:

| App | Função | Endereço |
|---|---|---|
| `minio-open` | API S3 (porta 9000) + volume `/data` | `https://minio-open.seudominio` |
| `minio-open-console` | Proxy nginx (~3 MB RAM) para o console | `https://minio-open-console.seudominio` |

As senhas continuam disponíveis depois em **App Configs → Environment
Variables** do app `minio-open`.

## Passo 3 — HTTPS

Nos **dois** apps: **Configurações HTTP → Enable HTTPS → Force HTTPS → Salvar**.

No `minio-open-console`, confirme que **Websocket Support** está marcado.
As telas de logs e trace usam websocket.

### Domínio próprio (opcional)

Para usar, por exemplo, `s3.empresa.com` e `minio.empresa.com`:

1. Conecte cada domínio no respectivo app (**Configurações HTTP**).
2. No app `minio-open`, atualize as variáveis:
   - `MINIO_SERVER_URL=https://s3.empresa.com`
   - `MINIO_BROWSER_REDIRECT_URL=https://minio.empresa.com`
3. No Typebot, troque `S3_ENDPOINT` para `s3.empresa.com`.

## Passo 4 — Conferir logs e console

### Logs

App `minio-open` → **Logs**. Na subida, o `minio-bootstrap` mostra:

```
[bootstrap] bucket "typebot" created
[bootstrap] anonymous download enabled on typebot/public
[bootstrap] user "typebot" created
[bootstrap] policy "app-typebot-rw" attached to "typebot" (access limited to: typebot)
========================================================================
 MinIO pronto!  /  MinIO is ready!
========================================================================
 Console (admin completo) : https://minio-open-console.seudominio
 API S3                   : https://minio-open.seudominio
 ...
   S3_ACCESS_KEY=typebot
   S3_SECRET_KEY=<o valor de MINIO_INIT_APP_PASSWORD deste app>
   S3_BUCKET=typebot
   S3_ENDPOINT=minio-open.seudominio
   S3_SSL=true
   S3_REGION=us-east-1
```

A cada reinício ele roda de novo, mas nunca apaga nem sobrescreve nada que já
exista. Nos reinícios seguintes as mensagens viram `already exists`.

### Console

Entre em `https://minio-open-console.seudominio` com o usuário e a senha admin.
Não é preciso configurar nada; se quiser, confira:

| Onde | O que deve aparecer |
|---|---|
| **Object Browser** | Bucket `typebot` (0 objetos até o primeiro upload) |
| **Identity → Users** | Usuário `typebot` com a política `app-typebot-rw` |
| **Policies** | 6 políticas: `app-typebot-rw`, `consoleAdmin`, `diagnostics`, `readonly`, `readwrite`, `writeonly` |
| **Buckets → typebot → Anonymous** | Prefixo `public/` com acesso `readonly` |

Política `app-typebot-rw` esperada (acesso somente ao bucket `typebot`):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["s3:GetBucketLocation", "s3:ListBucket", "s3:ListBucketMultipartUploads"],
      "Resource": ["arn:aws:s3:::typebot"]
    },
    {
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject",
                 "s3:AbortMultipartUpload", "s3:ListMultipartUploadParts"],
      "Resource": ["arn:aws:s3:::typebot/*"]
    }
  ]
}
```

> ⚠️ **Não apague** a política `app-typebot-rw`: sem ela os uploads do Typebot
> dão 403. Se acontecer, reinicie o app `minio-open` e o bootstrap a recria.
>
> ⚠️ **Não crie regras de Lifecycle que expirem `public/`**: é lá que ficam as
> imagens e os arquivos dos seus bots.
>
> O menu **Subnet** (License, Health, Performance, Profile, Inspect, Call Home)
> é do serviço comercial da MinIO e pode ser ignorado.

## Passo 5 — Configurar o Typebot

No CapRover, em **typebot-builder** e **typebot-viewer** → **Configurações do
App → Variáveis de Ambiente**, adicione as **mesmas** variáveis nos dois:

| Variável | Valor | Observação |
|---|---|---|
| `S3_ACCESS_KEY` | `typebot` | Nunca use o admin |
| `S3_SECRET_KEY` | senha S3 do Typebot | Igual ao `MINIO_INIT_APP_PASSWORD` |
| `S3_BUCKET` | `typebot` | |
| `S3_ENDPOINT` | `minio-open.seudominio` | Só o host: sem `https://` e sem `/` |
| `S3_SSL` | `true` | Exige HTTPS ativo no app `minio-open` |
| `S3_REGION` | `us-east-1` | |
| `S3_PORT` | (não defina) | Só se usar porta diferente de 443 |
| `S3_PUBLIC_CUSTOM_DOMAIN` | (opcional) | CDN/domínio na frente dos arquivos públicos |

Modelo pronto: [`typebot.env.example`](typebot.env.example). Nomes conferidos
no código do Typebot (`packages/env/src/index.ts`).

**Dicas de revisão:**

- `STORAGE_TYPE` e `UPLOAD_FS_FOLDER_NAME` **não existem** no Typebot. Se
  estiverem lá, podem ser removidas: o Typebot usa S3 só por ter as `S3_*`.
- Mantenha builder e viewer com o mesmo conjunto de variáveis, inclusive
  `DISABLE_SIGNUP`.

Clique em **Salvar & Reiniciar** nos dois apps.

## Passo 6 — Testar

1. No builder, adicione um bloco **Imagem → Upload** e envie uma imagem.
   Ela deve aparecer no editor.
2. Publique um bot com o bloco **Arquivo** e envie um arquivo pelo viewer.
3. No console, abra **Object Browser → typebot → public/**: os arquivos
   aparecem lá.
4. Abra a URL de um arquivo numa aba anônima: deve carregar sem login.

---

## Migrar de um MinIO antigo (opcional)

Se você já tinha outro app MinIO com arquivos do Typebot, entre no container
do `minio-open` (veja [Comandos úteis](#comandos-úteis-mc-já-vem-na-imagem)) e:

```bash
mc alias set antigo https://S3-DO-MINIO-ANTIGO USUARIO_ANTIGO SENHA_ANTIGA
```

```bash
mc alias set novo http://127.0.0.1:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD"
```

```bash
mc mirror --preserve antigo/typebot novo/typebot
```

> **Importante:** o Typebot grava nos bots a **URL completa** dos arquivos
> enviados. Bots antigos continuam apontando para o domínio do MinIO antigo.
> Para não quebrá-los ao desligar o app antigo, depois do `mc mirror`
> remova o domínio antigo do app velho e adicione-o como **domínio extra** no
> app `minio-open` (Configurações HTTP). Assim as URLs antigas passam a ser
> servidas pelo MinIO novo.

Só exclua o app antigo depois de testar os bots existentes.

---

## Alternativa B — compilar na própria VPS

O `captain-definition` na raiz aponta para o `Dockerfile.open`, então você pode
criar um app comum no CapRover e usar **Implantação → GitHub** com este
repositório e o branch `master`. Recomendações:

- crie **swap de 2–4 GB** na VPS antes;
- adicione a variável `WITH_MC=false` para pular o `mc` e encurtar o build;
- `BUILD_JOBS=1` deixa o build mais lento, porém mais leve.

O app do console precisa ser criado à parte; o mais simples é usar o template e
depois trocar a origem do app principal.

## Docker Compose (sem CapRover)

```bash
cp deploy/.env.example deploy/.env
```

Edite as senhas em `deploy/.env` e suba:

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

Acrescente `--build` para compilar localmente em vez de baixar a imagem.

---

## Variáveis de ambiente

| Variável | Padrão na imagem | Descrição |
|---|---|---|
| `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` | — | Credenciais admin (obrigatórias) |
| `MINIO_SERVER_URL` | — | URL pública da API S3 (links e presigned URLs) |
| `MINIO_BROWSER_REDIRECT_URL` | — | URL pública do console |
| `MINIO_CONSOLE_LOCAL_API` | `on` | Console fala com a API via 127.0.0.1, então o login não depende de DNS/SSL (novo neste fork) |
| `MINIO_MEMLIMIT` | `1GiB` | Teto de memória do Go; também ajusta o nº máximo de requisições simultâneas |
| `MINIO_SCANNER_SPEED` | `slow` | Scanner de fundo (`fastest`, `fast`, `default`, `slow`, `slowest`) |
| `MINIO_UPDATE` | `off` | Desliga a checagem de atualização em dl.min.io |
| `MINIO_BROWSER_SESSION_DURATION` | `12h` | Duração da sessão no console |
| `MINIO_VOLUMES` | `/data` (só quando nenhum comando é passado) | Diretório(s) de dados. Se definida, tem prioridade sobre os argumentos de `server` |
| `MINIO_OPTS` | `--console-address :9001` | Argumentos extras do `minio server` |
| `MINIO_INIT_ENABLE` | `on` | `off` desativa o bootstrap |
| `MINIO_INIT_BUCKETS` | `typebot` | Buckets a criar (separados por vírgula) |
| `MINIO_INIT_PUBLIC_PATHS` | `typebot/public` | `bucket/prefixo` com leitura anônima (somente download) |
| `MINIO_INIT_APP_USER` / `MINIO_INIT_APP_PASSWORD` | — | Usuário de aplicação a criar |
| `MINIO_INIT_APP_BUCKETS` | = `MINIO_INIT_BUCKETS` | Buckets que esse usuário acessa |

Variáveis de build (o CapRover também as repassa como build-args):

| Variável | Padrão | Descrição |
|---|---|---|
| `WITH_MC` | `true` | Inclui o `mc` na imagem |
| `BUILD_JOBS` | `2` | Paralelismo do compilador |

Exemplo com vários apps: `MINIO_INIT_BUCKETS=typebot,n8n,backups` e
`MINIO_INIT_PUBLIC_PATHS=typebot/public`.

## Ajuste fino para VPS de 4 GB

| Serviço | RAM típica |
|---|---|
| CapRover + nginx | ~250 MB |
| MinIO Open (ocioso → com uploads) | ~160 MB → até 1 GB |
| Typebot builder + viewer | ~600–900 MB |
| PostgreSQL | ~150–300 MB |
| Folga do sistema | ~1 GB |

Para impor um teto rígido ao container, além do `MINIO_MEMLIMIT`, use no app
`minio-open` → **Configurações do App → Service Update Override**:

```json
{ "TaskTemplate": { "Resources": { "Limits": { "MemoryBytes": 1610612736 } } } }
```

Isso equivale a 1,5 GB. Mantenha esse limite ~50% acima do `MINIO_MEMLIMIT`.

**Backup:** todos os dados ficam no volume `minio-open-data`. Inclua-o na rotina
de backup da VPS.

## Comandos úteis (`mc` já vem na imagem)

No servidor (SSH), ache o container e entre nele:

```bash
docker ps --format '{{.Names}}' | grep 'srv-captain--minio-open\.'
```

```bash
docker exec -it <nome-que-apareceu-acima> sh
```

Dentro do container:

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
| Template: “pull access denied” | Imagem não publicada ou pacote privado | Passo 1 (Actions verde; registry no CapRover se privado) |
| Colei a URL do template e deu erro | O CapRover só aceita o conteúdo | Abra o link raw, copie tudo e cole |
| Nome do app já existe | Já há um app com esse nome | Use outro nome (ex.: `minio-open`) |
| Console abre mas o login falha | `MINIO_CONSOLE_LOCAL_API` desligado e app S3 sem HTTPS | Ligue `MINIO_CONSOLE_LOCAL_API=on` ou ative o HTTPS |
| Console redireciona para endereço errado | `MINIO_BROWSER_REDIRECT_URL` incorreto | Ajuste para a URL pública do app `-console` |
| Upload no Typebot: 403 / `SignatureDoesNotMatch` | `S3_ENDPOINT` com `https://` ou `/`, ou senha errada | Só o host; confira `S3_ACCESS_KEY` / `S3_SECRET_KEY` |
| Upload no Typebot: 403 / `AccessDenied` | Política `app-typebot-rw` removida | Reinicie o app `minio-open` |
| Imagens não aparecem para visitantes (403) | `public/` sem leitura anônima | Buckets → typebot → Anonymous → `public/` = readonly |
| Upload grande falha com 413 | Limite de corpo do nginx do CapRover | Aumente `client_max_body_size` na config nginx do app `minio-open` |
| Telas de Logs/Trace não carregam | Websocket desligado no app do console | Configurações HTTP → Websocket Support |

## Atualizar

- **Nova imagem:** a cada push no `master` o GitHub publica uma nova `latest`.
  No CapRover, app `minio-open` → **Implantação** → **Deploy via ImageName** →
  `ghcr.io/chr1sred/minio-open:latest` → **Deploy**. Os dados do volume `/data`
  são preservados.
- **Trazer correções do MinIO original:** faça merge do `master` de
  `minio/minio` neste fork; o console em `third_party/console` não é afetado.
