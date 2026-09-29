# TASK — Permissões de mesa, carteira persistente e infraestrutura financeira

## 0. Objetivo

Implementar, de forma incremental e segura, dois requisitos principais no poker online:

1. Somente jogadores autenticados podem ocupar assentos livres de uma mesa.
2. Implementar uma carteira/saldo persistente para cada usuário, integrada ao buy-in e à liquidação da mesa.

A implementação deve preparar a arquitetura para uma futura operação com dinheiro real, **sem implementar dinheiro real nesta task**.

> **Princípio central:** PostgreSQL é a fonte de verdade dos dados persistentes e financeiros; o backend é a autoridade das regras; REST/WebSocket são apenas canais de solicitação; o frontend nunca é autoridade para saldo, stack, buy-in, cash-out ou identidade do criador.

---

# 1. Regras obrigatórias antes de alterar o código

## 1.1 Diagnóstico primeiro

Antes de implementar:

- analisar a arquitetura atual;
- identificar frontend, backend, PostgreSQL, migrations, autenticação, REST, WebSocket e engine do poker;
- identificar os modelos atuais de `User`, `PokerTable`, `Player/Seat`, stack e buy-in;
- identificar como usuários e mesas são atualmente persistidos;
- identificar o fluxo atual de entrada e saída da mesa;
- identificar como o WebSocket autentica e identifica o usuário;
- identificar como o engine mantém o estado da partida;
- identificar as alterações já existentes no worktree.

Primeiro apresentar um diagnóstico resumido contendo:

```text
arquivo/componente
responsabilidade atual
alteração necessária
dependências
risco de regressão
```

Depois iniciar a implementação.

Não reescrever partes funcionais sem necessidade. Reutilizar os padrões existentes.

---

# 2. Preservar alterações existentes

O worktree possui alterações pré-existentes, especialmente:

```text
frontend/src/pages/TableLobby.tsx
frontend/src/services/api.ts
frontend/src/utils/tableRooms.ts
```

Antes de modificar esses arquivos, verificar:

```bash
git diff -- frontend/src/pages/TableLobby.tsx
git diff -- frontend/src/services/api.ts
git diff -- frontend/src/utils/tableRooms.ts
```

Também verificar arquivos novos/não rastreados relacionados à task.

As alterações existentes devem ser preservadas e integradas.

**Não executar para descartar trabalho existente:**

```bash
git reset --hard
git checkout -- .
git restore .
```

---

# 3. Arquitetura financeira obrigatória

## 3.1 Separar dinheiro de fichas

Não utilizar o mesmo campo ou conceito para representar dinheiro e fichas.

Modelo conceitual:

```text
USER
 └── WALLET
      ├── saldo monetário
      ├── saldo disponível
      └── saldo reservado

TABLE
 └── TABLE_PLAYER
      └── stack_chips
```

A carteira representa dinheiro.

A mesa representa fichas.

Durante uma participação:

```text
Wallet
  │
  │ BUY_IN
  ▼
TablePlayer.stack_chips
  │
  │ jogo
  ▼
stack final
  │
  │ TABLE_RETURN
  ▼
Wallet
```

As alterações do stack durante o jogo pertencem ao domínio da partida. Não atualizar a carteira a cada ação de poker.

---

## 3.2 Conversão dinheiro ↔ fichas

Se o projeto não possuir uma regra explícita de conversão, utilizar temporariamente:

```text
R$ 1,00 = 100 fichas
```

Exemplo:

```text
R$ 10,00  = 1.000 fichas
R$ 50,00  = 5.000 fichas
R$ 100,00 = 10.000 fichas
```

Essa é uma convenção de desenvolvimento, não uma regra financeira definitiva.

Centralizar a conversão em um único serviço/regra de domínio:

```text
Money -> Chips
Chips -> Money
```

Não espalhar:

```typescript
amount * 100
amount / 100
```

por handlers, WebSockets ou componentes React.

Nenhum componente frontend deve conhecer diretamente a taxa de conversão.

---

# 4. Fonte de verdade e autoridade

A hierarquia deve ser:

```text
PostgreSQL
    ↓
Backend / domínio
    ↓
REST / WebSocket
    ↓
Frontend
```

O frontend apenas solicita operações.

O backend:

- autentica;
- autoriza;
- valida;
- calcula;
- executa transações;
- persiste;
- publica o resultado.

O frontend nunca é fonte definitiva de:

- saldo;
- buy-in;
- stack;
- cash-out;
- resultado financeiro;
- prêmio;
- depósito;
- saque;
- criador da mesa;
- configuração e ocupação de assentos.

---

# 5. Identidade do criador da mesa

Adicionar/persistir:

```text
creator_user_id
```

ou `owner_user_id`, seguindo a convenção do projeto.

Não utilizar nome de usuário como mecanismo de autorização.

Algo como:

```text
CreatedBy = nome
```

pode continuar apenas para apresentação, se necessário.

A autorização deve ser baseada no identificador persistente:

```text
authenticated_user.id == table.creator_user_id
```

---

# 6. Ocupação de assentos

Todos os assentos livres podem ser escolhidos por jogadores autenticados. O backend valida a disponibilidade e impede que duas pessoas ocupem o mesmo assento.

---

# 7. Autenticação do WebSocket

O WebSocket deve identificar o usuário autenticado de forma confiável.

Não utilizar usuário anônimo para operações que dependam da identidade do usuário, incluindo:

- entrar em mesa;
- escolher assento;
- buy-in;
- alterar assento;
- sair da mesa;
- liquidar stack;
- receber/devolver saldo.

Se existir modo anônimo por compatibilidade, separar explicitamente esse modo das operações financeiras e autenticadas.

O servidor deve derivar o `user_id` da autenticação, e não aceitar identidade arbitrária enviada pelo cliente.

---

# 8. Modelo de carteira

Analisar o schema atual e adaptar a solução à arquitetura existente.

A estrutura deve suportar, no mínimo, o conceito:

```text
wallet
------
id
user_id
balance
available_balance
reserved_balance
created_at
updated_at
```

A modelagem pode ser equivalente, desde que preserve as regras abaixo.

## 8.1 Regra de consistência

Se os três campos existirem:

```text
balance = available_balance + reserved_balance
```

Onde:

- `balance`: patrimônio monetário total dentro da carteira;
- `available_balance`: valor disponível para novas operações;
- `reserved_balance`: valor temporariamente bloqueado.

Toda alteração deve ocorrer atomicamente no banco.

O frontend nunca calcula esses valores como fonte de verdade.

---

# 9. Precisão monetária

Nunca utilizar `float` ou `double` para valores financeiros.

Preferir:

```text
INTEGER/BIGINT em centavos
```

ou:

```text
NUMERIC/DECIMAL
```

no PostgreSQL, de acordo com a arquitetura atual.

Exemplo:

```text
R$ 1,00  -> 100 centavos
R$ 25,50 -> 2550 centavos
```

Manter uma única convenção no backend e no banco.

---

# 10. Ledger financeiro

Não depender apenas de um campo de saldo.

Criar, adaptar ou consolidar um ledger/histórico financeiro.

Estrutura conceitual:

```text
wallet_transactions
--------------------
id
wallet_id
user_id
type
amount
balance_before
balance_after
reference_type
reference_id
idempotency_key
created_at
```

Usar os nomes equivalentes à convenção do projeto.

Tipos conceituais:

```text
DEPOSIT
BUY_IN
TABLE_RETURN
WIN
REFUND
WITHDRAWAL
ADJUSTMENT
```

### Regra importante

`TABLE_RETURN` não é automaticamente `WIN`.

Exemplo:

```text
BUY_IN
R$ 100 -> mesa

TABLE_RETURN
R$ 150 -> carteira
```

O jogador terminou com R$ 50 a mais, mas isso não significa que o sistema deva criar automaticamente um evento `WIN`.

`WIN` deve representar uma regra específica de premiação, caso o jogo futuramente possua esse conceito.

O ledger deve ser tratado como registro financeiro imutável.

Não editar uma transação antiga para corrigir erro. Correções devem gerar uma nova transação `ADJUSTMENT` ou equivalente.

---

# 11. Princípio de reconstrução financeira

A partir do banco deve ser possível determinar:

1. quanto o usuário possui;
2. quanto está disponível;
3. quanto está reservado;
4. quais operações alteraram o saldo;
5. quanto foi utilizado para cada buy-in;
6. qual stack foi liquidado ao sair de cada mesa.

Não depender de estado financeiro armazenado apenas em memória.

O engine pode manter estado em memória para performance durante a partida, mas valores financeiros devem possuir persistência e rastreabilidade no banco.

---

# 12. Buy-in

O cliente pode solicitar:

```text
"Quero entrar com R$ 100"
```

Mas o backend decide se o valor é válido.

Fluxo obrigatório:

```text
Cliente
  ↓
solicita buy-in
  ↓
usuário autenticado?
  ↓
mesa existe?
  ↓
mesa aceita entrada?
  ↓
assento disponível?
  ↓
buy-in permitido?
  ↓
saldo disponível suficiente?
  ↓
transação PostgreSQL
  ├── debita/reserva carteira
  ├── cria participação
  ├── cria stack
  └── registra ledger
```

Não aceitar do cliente um `stack` inicial calculado por ele.

Exemplo:

```text
Carteira: R$ 500
Buy-in:   R$ 100

Após operação:
Carteira disponível: R$ 400
Stack da mesa:       valor equivalente em fichas
```

Se o saldo for insuficiente:

```text
Saldo insuficiente para entrar nesta mesa.
```

---

# 13. Atomicidade e idempotência

Buy-in deve ser atômico.

Duas requisições simultâneas não podem gastar o mesmo saldo.

Implementar mecanismo de idempotência para operações financeiras, especialmente:

- depósito;
- buy-in;
- retorno de stack;
- prêmio;
- saque;
- webhook de pagamento.

Conceito:

```text
idempotency_key
      ↓
já processado?
 ├── SIM → retornar resultado existente
 └── NÃO → executar transação
```

Usar constraints/indexes no banco quando apropriado para impedir duplicidade.

---

# 14. Stack da mesa

O stack do jogador pertence ao domínio da partida.

Exemplo:

```text
Carteira disponível: R$ 400
Mesa:
    stack = 10.000 chips
```

Durante o jogo:

```text
stack = 15.000 chips
```

não significa que a carteira passa automaticamente a ter R$ 550.

O resultado financeiro da mesa somente é realizado quando houver uma operação de liquidação válida.

---

# 15. Saída da mesa e liquidação

O cliente deve solicitar somente:

```text
leaveTable
```

Não enviar:

```text
cashOut = 150
```

como valor de liquidação.

O backend deve:

1. autenticar o usuário;
2. localizar a participação real;
3. obter o stack mantido pelo servidor;
4. aplicar a política de saída definida pelo engine;
5. converter stack para dinheiro;
6. executar transação financeira;
7. registrar `TABLE_RETURN`;
8. encerrar a participação;
9. impedir nova liquidação da mesma participação.

Fluxo:

```text
Cliente
  ↓
leaveTable
  ↓
Backend
  ↓
stack real do servidor
  ↓
Money/Chips service
  ↓
transação PostgreSQL
  ├── encerra participação
  ├── registra ledger
  └── libera/credita carteira
```

Prioridades:

```text
não perder stack
não duplicar stack
não duplicar cash-out
```

---

# 16. Saída durante estados intermediários

Antes de definir a política definitiva de liquidação, analisar o comportamento atual do engine quando o jogador:

1. sai durante uma mão;
2. fecha o navegador;
3. perde a conexão WebSocket;
4. atualiza a página;
5. reconecta;
6. entra novamente na mesma mesa.

Não inventar uma regra arbitrária sem entender o engine existente.

Definir e documentar uma política consistente para:

- `leaveTable` explícito;
- disconnect;
- reconnect;
- refresh;
- abandono;
- participação em mão ativa.

A política deve evitar tanto perda de stack quanto liquidação duplicada.

---

# 17. Persistência de mesas e participação

Como o sistema possui PostgreSQL, verificar o que já existe antes de criar novas tabelas.

A arquitetura deve permitir persistir, conforme necessidade:

```text
users
wallets
wallet_transactions
tables
table_players / participations
```

A participação deve possuir identidade própria suficiente para idempotência e liquidação.

Exemplo conceitual:

```text
table_player
------------
id
table_id
user_id
seat
stack_chips
buy_in_reference
status
created_at
updated_at
settled_at
```

Não é obrigatório seguir exatamente esses nomes.

---

# 18. Migrações

Não executar novamente `001_init.sql` como solução para evolução do banco.

Antes de alterar schema:

1. verificar migrations existentes;
2. verificar tabelas existentes;
3. verificar estrutura de `001_init.sql`;
4. verificar se existem dados;
5. verificar campos atualmente usados pelo backend;
6. criar migrations incrementais compatíveis.

Não apagar dados existentes sem estratégia explícita.

As migrations devem ser idempotentes apenas quando a ferramenta/convenção do projeto exigir isso; não duplicar estruturas silenciosamente.

---

# 19. Carteira na interface

Adicionar acesso à carteira à interface principal.

Exemplo:

```text
Saldo: R$ 250,00
```

Ao acessar:

```text
Minha Carteira

Saldo total:       R$ 250,00
Saldo disponível:  R$ 150,00
Em mesas/reservado R$ 100,00
```

Exibir, conforme a modelagem adotada:

- saldo;
- saldo disponível;
- valor reservado/em mesas;
- histórico;
- opção de adicionar saldo.

O frontend deve sempre buscar os valores do backend.

---

# 20. Histórico

Exemplo:

```text
+ R$ 100,00  Depósito
- R$ 50,00   Buy-in
+ R$ 75,00   Retorno da mesa
```

O histórico deve vir do ledger/backend.

Não montar o histórico somente a partir de estado local do React.

---

# 21. Adicionar saldo e preparação para pagamentos

**Não implementar gateway real nesta task.**

Criar uma abstração independente da carteira, por exemplo:

```text
WalletService
    ↓
PaymentService
    ↓
PaymentProvider
    ├── MockPaymentProvider
    └── RealPaymentProvider (futuro)
```

Conceitos futuros:

```text
createDeposit()
getPaymentStatus()
processWebhook()
createWithdrawal()
```

O provider não deve ficar acoplado à lógica da Wallet.

## 21.1 Mock de desenvolvimento

Se necessário, criar:

```text
MockPaymentProvider
```

claramente identificado como `MOCK/DEV`.

Ele deve:

- ficar isolado;
- não representar pagamento real;
- não ser tratado como pagamento confirmado em produção;
- ser facilmente removível/substituível;
- nunca armazenar dados de cartão.

O clique em:

```text
Adicionar saldo
```

não pode ser tratado sozinho como pagamento real.

---

# 22. Fluxo futuro de depósito

A arquitetura deve suportar futuramente:

```text
Usuário
  ↓
escolhe valor
  ↓
Backend cria depósito
  ↓
PaymentProvider
  ↓
usuário realiza pagamento
  ↓
Provider confirma
  ↓
Webhook
  ↓
Backend valida
  ↓
Ledger
  ↓
Wallet
```

O frontend não pode creditar saldo por conta própria.

---

# 23. Preparação para saque futuro

Mesmo sem implementar saque agora, não bloquear a futura arquitetura:

```text
Wallet
  ↓
solicitação de saque
  ↓
validações
  ↓
reserva do valor
  ↓
PaymentProvider
  ↓
processamento
  ↓
confirmação
  ↓
ledger
```

Preparar estados:

```text
PENDING
PROCESSING
COMPLETED
FAILED
CANCELLED
```

Não implementar saque real nesta task.

---

# 24. Preparação para dinheiro real

A arquitetura atual deve ser tecnicamente compatível com futura evolução para dinheiro real, mas esta task **não habilita dinheiro real**.

Não implementar agora:

- gateway real;
- saque real;
- dados de cartão;
- confirmação falsa de pagamento;
- KYC;
- antifraude;
- operação financeira real;
- requisitos regulatórios.

Antes de operar dinheiro real, será necessário tratar separadamente requisitos legais, regulatórios, de pagamentos, identificação, segurança, auditoria, limites e prevenção a fraude aplicáveis à jurisdição e ao modelo de negócio.

---

# 25. Segurança

Toda operação financeira deve exigir, conforme o caso:

- usuário autenticado;
- autorização adequada;
- validação no backend;
- transação de banco;
- idempotência;
- auditoria.

Projetar assumindo que o cliente pode ser manipulado.

Nunca aceitar como verdade:

```text
balance enviado pelo frontend
stack enviado pelo frontend
cashOut enviado pelo frontend
creator enviado pelo frontend
resultado financeiro enviado pelo frontend
```

---

# 26. Estratégia de implementação — não fazer tudo de uma vez

A implementação deve ocorrer em fases verificáveis.

## Fase 0 — Diagnóstico

Entregar primeiro:

- arquitetura atual;
- arquivos afetados;
- fluxo atual de autenticação;
- fluxo atual de WebSocket;
- fluxo atual de buy-in;
- fluxo atual de saída;
- schema/migrations;
- alterações pré-existentes;
- riscos e decisões necessárias.

**Não começar a reescrever tudo antes desse diagnóstico.**

---

## Fase 1 — Banco e domínio financeiro

Implementar:

- conexão PostgreSQL, se ainda não estiver sendo utilizada pelo servidor;
- repositories;
- migrations incrementais;
- persistência de usuários;
- `creator_user_id`;
- wallet;
- ledger;
- participação na mesa;
- unidade monetária;
- conversão Money ↔ Chips centralizada;
- idempotency keys.

Validar schema e migrations.

**Ao concluir a Fase 1, parar e apresentar:**

```text
migrations
models
repositories
constraints
transações
decisões de arquitetura
```

Só avançar após verificar que a base financeira está consistente.

---

## Fase 2 — Autenticação e autorização

Implementar:

- autenticação obrigatória nas operações WS relevantes;
- identificação real do usuário;
- proteção REST;
- proteção WebSocket;

Testar com duas contas.

---

## Fase 3 — Buy-in

Implementar:

- validação de saldo;
- validação de mesa;
- validação de assento;
- débito/reserva atômica;
- criação da participação;
- stack inicial;
- ledger;
- idempotência;
- tratamento de concorrência.

Testar duas entradas simultâneas.

---

## Fase 4 — Liquidação

Implementar:

- `leaveTable`;
- stack real do servidor;
- política de disconnect/reconnect;
- conversão Chips → Money;
- `TABLE_RETURN`;
- crédito/liberação de saldo;
- encerramento da participação;
- idempotência.

Testar:

- saída normal;
- refresh;
- disconnect;
- reconnect;
- saída duplicada;
- duas requisições simultâneas.

---

## Fase 5 — Frontend

Implementar:

- carteira;
- saldo no header/navbar;
- histórico;
- adicionar saldo;
- buy-in integrado ao saldo;
- mensagens de erro/saldo insuficiente;
- atualização após buy-in e liquidação.

Não duplicar regra financeira no React.

---

## Fase 6 — Mock de pagamento

Implementar somente se necessário:

- `PaymentProvider`;
- `MockPaymentProvider`;
- fluxo DEV claramente identificado;
- estados de depósito;
- idempotência.

Não simular um gateway real de forma enganosa.

---

## Fase 7 — Validação

Executar:

```text
backend tests
frontend tests
lint
build
```

Depois realizar testes manuais com:

```text
Conta A = criador
Conta B = jogador
Conta C = jogador adicional
```

e pelo menos:

```text
Mesa 1
Mesa 2
```

---

# 27. Cenários obrigatórios de teste

## 27.1 Assentos

- dois jogadores não podem ocupar o mesmo assento;
- jogadores podem escolher somente assentos livres.

---

## 27.2 Persistência da carteira

Verificar:

- saldo inicial conforme regra definida;
- logout/login preserva saldo;
- refresh preserva saldo;
- outro dispositivo/sessão obtém o mesmo saldo do backend;
- histórico persiste.

---

## 27.3 Buy-in

Cenário:

```text
Saldo: R$ 500
Buy-in: R$ 100
```

Esperado:

```text
Saldo disponível: R$ 400
Mesa: equivalente a R$ 100 em fichas
```

Cenário:

```text
Saldo: R$ 50
Buy-in: R$ 100
```

Esperado:

```text
entrada recusada
saldo inalterado
```

---

## 27.4 Concorrência

Testar simultaneamente:

- dois buy-ins com o mesmo saldo;
- dois pedidos de saída;
- refresh durante buy-in;
- reconnect durante uma participação;
- repetição da mesma requisição com a mesma idempotency key.

Garantir:

- sem saldo negativo indevido;
- sem débito duplicado;
- sem crédito duplicado;
- sem stack duplicado;
- sem participação duplicada;
- sem liquidação duplicada.

---

## 27.5 Liquidação

Exemplo:

```text
Saldo disponível: R$ 400
Stack: equivalente a R$ 150
```

Após saída válida:

```text
Saldo disponível: R$ 550
Stack: 0 / participação encerrada
```

conforme a política de liquidação definida pelo engine.

---

# 28. Critérios de aceite

A implementação somente será considerada concluída quando:

1. WebSocket não depender de usuário anônimo para operações autenticadas;
2. cada usuário possuir carteira persistente;
3. valores monetários utilizarem representação exata;
4. dinheiro e fichas forem conceitos separados;
5. conversão Money ↔ Chips estiver centralizada;
6. buy-in validar saldo no backend;
7. buy-in for atômico;
8. buy-in possuir idempotência;
9. stack da mesa não for controlado pelo frontend;
10. saída usar o stack real do servidor;
11. liquidação não aceitar `cashOut` calculado pelo cliente;
12. saída não puder creditar duas vezes;
13. saldo disponível e reservado obedecerem à regra definida;
14. ledger registrar movimentações relevantes;
15. ledger permitir reconstrução financeira;
16. carteira sobreviver a logout/login/refresh;
17. existir interface de carteira;
18. existir histórico;
19. existir estrutura de depósito preparada para provider futuro;
20. mock, se existir, estiver explicitamente isolado como DEV;
21. gateway real não for integrado;
22. testes automatizados passarem;
23. lint e build passarem;
24. testes manuais com múltiplas contas e mesas passarem;
25. alterações existentes no worktree forem preservadas.

---

# 29. Compatibilidade e não regressão

Antes de finalizar:

- revisar todos os fluxos existentes de autenticação;
- revisar criação/entrada/saída de mesas;
- revisar WebSocket;
- revisar engine;
- revisar lobby;
- revisar API;
- revisar migrations;
- revisar testes existentes.

Não criar uma segunda fonte de verdade para usuários, autenticação ou mesas.

Não substituir silenciosamente um mecanismo existente sem justificar a mudança.

Se uma decisão estrutural for necessária para compatibilidade, documentá-la no relatório final.

---

# 30. Relatório final obrigatório

Ao terminar cada fase, informar:

```text
FASE:
STATUS:

Arquivos alterados:
- ...

Migrations:
- ...

Models:
- ...

Repositories/Services:
- ...

Endpoints:
- ...

Eventos WebSocket:
- ...

Regras de negócio:
- ...

Testes executados:
- ...

Resultado:
- PASS / FAIL

Problemas encontrados:
- ...

Decisões de arquitetura:
- ...

Pendências:
- ...
```

Ao final da task, apresentar também:

- resumo da arquitetura final;
- arquivos alterados;
- migrations criadas;
- endpoints criados/alterados;
- eventos WebSocket criados/alterados;
- regras financeiras implementadas;
- estratégia de idempotência;
- estratégia de reconexão;
- testes automatizados;
- testes manuais;
- limitações conhecidas;
- pontos que dependem de integração externa.

---

# 31. Resultado arquitetural esperado

```text
                         ┌──────────────────┐
                         │       USER       │
                         └────────┬─────────┘
                                  │
                                  ▼
                         ┌──────────────────┐
                         │      WALLET      │
                         │                  │
                         │ balance          │
                         │ available        │
                         │ reserved         │
                         └────────┬─────────┘
                                  │
                              BUY_IN
                                  │
                                  ▼
                         ┌──────────────────┐
                         │      TABLE       │
                         │                  │
                         │ TablePlayer      │
                         │ stack_chips      │
                         └────────┬─────────┘
                                  │
                               GAME
                                  │
                                  ▼
                         ┌──────────────────┐
                         │   FINAL STACK    │
                         └────────┬─────────┘
                                  │
                           TABLE_RETURN
                                  │
                                  ▼
                         ┌──────────────────┐
                         │      WALLET      │
                         │ saldo atualizado │
                         └──────────────────┘

                 ┌────────────────────────────┐
                 │      WALLET LEDGER         │
                 │                            │
                 │ DEPOSIT                    │
                 │ BUY_IN                     │
                 │ TABLE_RETURN               │
                 │ WIN / REFUND / ADJUSTMENT  │
                 │ WITHDRAWAL (futuro)        │
                 └────────────────────────────┘

                 ┌────────────────────────────┐
                 │      PAYMENT LAYER         │
                 │                            │
                 │ PaymentProvider             │
                 │ ├─ MockPaymentProvider      │
                 │ └─ RealProvider (futuro)    │
                 └────────────────────────────┘
```

A carteira e a mesa são domínios distintos.

A comunicação financeira entre eles ocorre por operações de domínio controladas pelo backend e persistidas de forma transacional.

---

# 32. Regra final de execução

**Não fazer uma reescrita monolítica.**

A ordem obrigatória é:

```text
1. Diagnóstico
2. Banco/domínio/persistência
3. Autenticação/autorização
4. Buy-in
5. Liquidação
6. Frontend
7. Mock de pagamento, se necessário
8. Testes
9. Relatório final
```

Depois de cada fase relevante, executar os testes correspondentes e verificar o estado do projeto antes de avançar.

Se uma alteração exigir mudança de arquitetura, explicar a decisão antes de aplicar uma substituição ampla.

O objetivo é corrigir os requisitos atuais preservando o sistema existente e, simultaneamente, estabelecer uma base financeira consistente para futura evolução — sem implementar dinheiro real nesta etapa.
