# ♠️ Club Poker dos Amigos - Texas Hold'em

Projeto completo de Poker Online desenvolvido com arquitetura moderna, de alta performance e segurança:

- **Backend**: Go (Golang) com REST API, WebSockets em tempo real e motor de embaralhamento contínuo com memória física.
- **Banco de Dados**: PostgreSQL com migrações estruturadas.
- **Frontend Web**: React 19 + TypeScript + Vite + Tailwind CSS (Tema Dark VIP Cassino com gradientes dourados).
- **Mobile (Fase 2)**: Flutter (Dart) integrado ao mesmo backend.

---

## 📂 Estrutura do Projeto

```text
projeto-pokr-dos-amigos/
├── backend/                  # Servidor e Engine em Go
│   ├── cmd/server/main.go    # Entrypoint HTTP/WebSocket
│   ├── internal/
│   │   ├── auth/             # JWT, Bcrypt e Middlewares RBAC
│   │   ├── config/           # Carregamento de variáveis de ambiente
│   │   ├── database/         # Conexão e pooling do PostgreSQL
│   │   ├── engine/           # Shuffler com CSPRNG e cortes + Regras Texas Hold'em
│   │   ├── handlers/         # Controllers dos 8 módulos e autenticação
│   │   ├── models/           # Structs de Usuário, Mesa, Torneios, Financeiro
│   │   └── ws/               # Hub e Conexões WebSocket
│   ├── migrations/           # Schemas SQL para o PostgreSQL
│   ├── pkg/response/         # Helpers de resposta JSON
│   ├── go.mod
│   └── .env.example
│
├── frontend/                 # Aplicação Web em React + TypeScript
│   ├── src/
│   │   ├── components/       # Componentes visuais (HeaderLogo, Botões, etc.)
│   │   ├── context/          # AuthContext e Sessão
│   │   ├── pages/            # Login, Cadastro, Menu Gerente, Menu Jogador, Mesa
│   │   ├── services/         # Clientes REST (api.ts) e WebSocket
│   │   ├── types/            # Interfaces TypeScript compartilhadas
│   │   ├── index.css         # Estilos customizados Dark/Gold
│   │   ├── App.tsx           # Roteamento SPA
│   │   └── main.tsx
│   ├── package.json
│   └── vite.config.ts
│
└── Jogo de poker oline Poker dos Amigos.pdf # Documento original de requisitos
```

---

## 🚀 Como Executar

### Atalhos Rápidos

Os principais comandos podem ser executados pela raiz do projeto:

```bash
make help
```

```bash
make db-up
# Sobe o PostgreSQL em Docker na porta local 5433
```

```bash
make backend
# Sobe a API Go em http://localhost:8080
```

```bash
make frontend
# Sobe a aplicação React/Vite em http://localhost:5173
```

```bash
make dev
# Sobe banco, backend e frontend juntos
```

Se a porta `5433` também estiver ocupada, escolha outra porta local:

```bash
POSTGRES_PORT=15432 make dev
```

Para parar o banco:

```bash
make db-down
```

Para apagar os dados locais do PostgreSQL e recriar o banco do zero:

```bash
make db-reset
```

### Banco de Dados (PostgreSQL)

O banco local roda com Docker Compose:

```bash
docker compose up -d postgres
```

Configuração padrão:

- Host: `localhost`
- Porta local: `5433`
- Porta interna do container: `5432`
- Banco: `poker_dos_amigos`
- Usuário: `postgres`
- Senha: `postgres`
- URL: `postgres://postgres:postgres@localhost:5433/poker_dos_amigos?sslmode=disable`

As migrações em `backend/migrations` são carregadas automaticamente na primeira criação do volume do PostgreSQL.

### Backend (Go)
```bash
cd backend
go run cmd/server/main.go
# O servidor iniciará em http://localhost:8080 (REST e WebSocket)
```

Você pode copiar `backend/.env.example` para `backend/.env` se quiser customizar portas, JWT ou URL do banco.

### Frontend (React)
```bash
cd frontend
npm run dev
# A aplicação web estará acessível em http://localhost:5173
```

---

## 🔑 Credenciais Pré-configuradas para Testes

- **Sócios / Gerentes (Direitos Iguais)**:
  - `jonatas@pokerdosamigos.com` | Senha: `poker123`
  - `felipe@pokerdosamigos.com` | Senha: `poker123`
- **Jogador Membro VIP**:
  - `jogador@pokerdosamigos.com` | Senha: `jogador123`
# projeto-poker-dos-amigos
