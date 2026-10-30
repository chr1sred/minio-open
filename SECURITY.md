# Security Policy

🇧🇷 [Português](#português) · 🇺🇸 [English](#english)

## Português

### Versões suportadas

Só a imagem mais recente recebe correções: `ghcr.io/chr1sred/minio-open:latest`,
publicada a cada push no `master`. Para receber uma correção, basta atualizar
a imagem (veja "Atualizar uma instalação existente" no [README](README.md)).

### Como reportar uma vulnerabilidade

Não abra uma issue pública. Use o reporte privado do GitHub:
**Security → Report a vulnerability** neste repositório.

Descreva o tipo do problema (DoS, bypass de autenticação, vazamento de
informação…), como reproduzir e o que é preciso para explorá-lo (por exemplo,
se exige credenciais).

Se o problema também existir no MinIO original (`minio/minio`), avise
também a MinIO Inc. em security@min.io.

## English

### Supported versions

Only the latest image gets fixes: `ghcr.io/chr1sred/minio-open:latest`,
published on every push to `master`. To get a fix, update the image (see
"Updating an existing install" in the [README](README.en-US.md)).

### Reporting a vulnerability

Please don't open a public issue. Use GitHub's private reporting:
**Security → Report a vulnerability** in this repository.

Describe the kind of issue (DoS, authentication bypass, information
disclosure…), how to reproduce it and what an attacker needs (for example,
whether credentials are required).

If the issue also affects upstream MinIO (`minio/minio`), please notify
MinIO Inc. at security@min.io as well.
