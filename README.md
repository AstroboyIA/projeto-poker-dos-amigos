# Aplicação de jogos de cartas

Aplicação web para partidas de cartas, com interface interativa e suporte a sessões de jogo.

## Desenvolvimento local

### Requisitos

- Go
- Node.js e npm
- Docker com suporte a Docker Compose

### Executar

Na raiz do repositório, use `make help` para consultar os comandos de desenvolvimento disponíveis. Para iniciar a aplicação localmente:

```bash
make dev
```

Para compilar os componentes:

```bash
make build
```

## Configuração

Use variáveis de ambiente locais para configurar a execução. Não inclua credenciais, tokens, dados pessoais ou configurações privadas em documentação, commits ou logs compartilhados. Valores de exemplo não devem ser usados como segredos em ambientes reais.

## Tecnologias

- Backend em Go
- Interface web em React e TypeScript
- PostgreSQL para persistência local
