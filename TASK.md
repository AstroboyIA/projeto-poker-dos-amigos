# TASK — Sincronização Multiplayer da Mesa de Poker

## 1. Objetivo

Corrigir a arquitetura multiplayer da mesa de Poker para que **o servidor seja a fonte de verdade do estado da partida**.

Atualmente, cada navegador executa uma partida independente no React, gera seu próprio baralho/cartas e simula os demais jogadores como bots.

Após esta tarefa:

* Todos os jogadores da mesma mesa devem compartilhar o mesmo estado de jogo.
* O backend deve controlar o estado autoritativo da partida.
* O frontend deve receber atualizações da mesa via WebSocket.
* Ações de jogadores devem ser enviadas ao backend via WebSocket.
* O backend deve transmitir as alterações para os demais clientes da mesma mesa.
* Um jogador que entrar ou sair da mesa deve ser refletido nos demais clientes conectados.
* Nenhum jogador humano conectado à mesma mesa deve ser tratado como bot pelo frontend.
* O estado não deve depender exclusivamente do `localStorage` para sincronização entre usuários.

---

# 2. Diagnóstico atual

## Frontend

Arquivo principal:

```text
frontend/src/pages/PokerTable.tsx
```

Problemas identificados:

### 2.1 WebSocket ausente

`PokerTable.tsx` não estabelece conexão WebSocket com o servidor.

Atualmente somente:

```text
TableLobby.tsx
```

utiliza `/ws` para atualizações relacionadas à lista de mesas.

A mesa de Poker precisa possuir sua própria conexão WebSocket.

---

### 2.2 Engine de Poker executada no navegador

`PokerTable.tsx` atualmente executa localmente:

* geração do baralho;
* embaralhamento;
* distribuição das cartas;
* cartas comunitárias;
* controle das rodadas;
* estado dos jogadores;
* ações de jogo.

O embaralhamento utiliza:

```text
shuffleContinuousDeck()
```

aproximadamente em:

```text
PokerTable.tsx#L215
```

Esse comportamento precisa ser revisado.

O frontend não deve criar uma partida independente da partida existente no servidor.

---

### 2.3 Jogadores humanos tratados como bots

Existe lógica semelhante a:

```text
isUser: false
```

para jogadores que não correspondem ao usuário atual.

Isso faz com que o `useEffect` de IA:

```text
PokerTable.tsx#L319-L365
```

assuma o controle desses assentos.

O resultado atual é:

```text
Usuário A
   └── partida local A
       ├── jogador A = humano
       └── jogador B = bot

Usuário B
   └── partida local B
       ├── jogador A = bot
       └── jogador B = humano
```

Esse comportamento deve ser eliminado para jogadores humanos reais.

---

# 3. Diagnóstico do Backend

Arquivo:

```text
backend/internal/ws/hub.go
```

## 3.1 `readPump`

Atualmente o `readPump` trata efetivamente apenas:

```text
MsgPing
```

Mensagens como:

```text
JOIN_TABLE
PLAYER_ACTION
LEAVE_TABLE
```

são recebidas/desempacotadas, mas não possuem o processamento necessário.

---

## 3.2 Registro de clientes nas mesas

O Hub possui:

```text
tables map[uuid.UUID]map[*Client]bool
```

e:

```text
BroadcastToTable
```

Porém:

* clientes não são registrados nas mesas;
* `client.TableID` não é preenchido;
* `BroadcastToTable` não é utilizado adequadamente.

Consequentemente, o servidor não possui atualmente uma associação funcional entre:

```text
cliente WebSocket
        ↓
mesa
        ↓
outros clientes da mesa
```

---

# 4. Diagnóstico de `tableRooms.ts`

A função:

```text
getTableRoomById
```

atualmente depende exclusivamente do:

```text
localStorage
```

do navegador.

Isso impede que alterações feitas por outro usuário sejam refletidas automaticamente nos clientes que já estão na mesa.

O `localStorage` pode continuar sendo utilizado para persistência local/cache quando apropriado, mas **não pode ser considerado fonte de verdade para o estado multiplayer da mesa**.

---

# 5. Arquitetura desejada

A arquitetura final deverá seguir este fluxo:

```text
                 BACKEND
                    │
             Estado da mesa
                    │
              WebSocket Hub
                    │
        ┌───────────┼───────────┐
        │           │           │
        ▼           ▼           ▼
     Cliente A   Cliente B   Cliente C
        │           │           │
        └──── ações ┴───────────┘
```

O servidor deve ser a fonte autoritativa.

Exemplo:

```text
Cliente A
   │
   │ PLAYER_ACTION: CHECK
   ▼
Backend
   │
   ├── valida ação
   ├── altera estado da partida
   ├── atualiza turno
   └── calcula novo estado
   │
   ▼
BroadcastToTable()
   │
   ├── Cliente A
   ├── Cliente B
   └── Cliente C
```

Todos os clientes devem receber o mesmo estado resultante.

---

# 6. Escopo da implementação

## Fase 1 — Investigar a arquitetura existente

Antes de modificar código:

1. Analise `frontend/src/pages/PokerTable.tsx`.
2. Analise `frontend/src/pages/TableLobby.tsx`.
3. Analise `tableRooms.ts`.
4. Analise `backend/internal/ws/hub.go`.
5. Localize:

   * tipos das mensagens WebSocket;
   * enumeração/tipos de mensagens;
   * estrutura `Client`;
   * estrutura `Hub`;
   * implementação de `BroadcastToTable`;
   * handlers existentes;
   * modelos de jogador;
   * estado da mesa;
   * engine atual de Poker;
   * endpoints REST relacionados a mesas;
   * mecanismos atuais de JOIN/LEAVE.
6. Procure outras referências a:

   * `JOIN_TABLE`;
   * `PLAYER_ACTION`;
   * `LEAVE_TABLE`;
   * `MsgPing`;
   * `BroadcastToTable`;
   * `TableID`;
   * `shuffleContinuousDeck`;
   * `localStorage`;
   * `isUser`;
   * lógica de bots.

### Regra

**Não implemente uma nova arquitetura antes de entender as estruturas existentes.**

Reutilize tipos, serviços e mecanismos já existentes sempre que possível.

---

# 7. Fase 2 — Corrigir o gerenciamento de clientes no WebSocket

Modificar:

```text
backend/internal/ws/hub.go
```

Implementar o registro correto de um cliente quando receber:

```text
JOIN_TABLE
```

O fluxo esperado:

```text
JOIN_TABLE
    ↓
validar tableID
    ↓
associar client.TableID
    ↓
registrar cliente em h.tables[tableID]
    ↓
notificar clientes da mesa
```

Garantir que:

```text
h.tables[tableID]
```

seja criado quando necessário.

Evitar criação de estruturas duplicadas ou race conditions.

Utilizar os mecanismos de sincronização já existentes no Hub.

---

# 8. Fase 3 — Remoção correta do cliente

Implementar tratamento de:

```text
LEAVE_TABLE
```

e desconexões do WebSocket.

Quando um cliente sair:

```text
h.tables[tableID]
```

deve deixar de conter aquele cliente.

Se a sala ficar vazia, avaliar a remoção da entrada do mapa para evitar vazamento de memória.

O servidor deve notificar os demais clientes quando apropriado.

---

# 9. Fase 4 — Processamento de PLAYER_ACTION

Implementar tratamento de:

```text
PLAYER_ACTION
```

no backend.

O backend deverá:

1. identificar o jogador;
2. identificar a mesa;
3. validar que o jogador pertence à mesa;
4. validar que a ação é permitida;
5. encaminhar a ação para a engine/serviço de Poker existente;
6. atualizar o estado da partida;
7. gerar o novo estado público da mesa;
8. executar:

```text
BroadcastToTable()
```

com o resultado.

### Importante

Não duplicar a engine de Poker dentro do WebSocket Hub.

O Hub deve ser responsável pela comunicação.

A lógica de negócio deve permanecer em uma camada apropriada da aplicação.

Preferência:

```text
WebSocket
   ↓
Handler
   ↓
Poker/Table Service
   ↓
Game State
   ↓
Hub.BroadcastToTable()
```

e não:

```text
WebSocket Hub
   ↓
toda a engine de Poker
```

---

# 10. Fase 5 — Evento de novo jogador

Quando um jogador entrar em uma mesa:

```text
JOIN_TABLE
```

os demais clientes conectados àquela mesa devem receber uma atualização.

A mensagem deve permitir que o frontend descubra:

* jogador entrou;
* assento ocupado;
* identificação pública do jogador;
* estado atualizado dos assentos;
* demais informações necessárias para renderização.

Evitar enviar informações privadas de outros jogadores.

### Segurança

Nunca enviar as cartas fechadas de um jogador para os demais clientes.

O payload público deve respeitar as regras do Poker.

---

# 11. Fase 6 — Conexão WebSocket em PokerTable.tsx

Modificar:

```text
frontend/src/pages/PokerTable.tsx
```

A tela deve abrir uma conexão WebSocket ao entrar na mesa.

Fluxo:

```text
PokerTable mount
      ↓
abrir WebSocket
      ↓
conexão estabelecida
      ↓
JOIN_TABLE(tableId)
      ↓
receber estado inicial
      ↓
renderizar mesa
```

A conexão deve ser encerrada corretamente quando o componente for desmontado.

Evitar múltiplas conexões para a mesma mesa devido a re-renderizações do React.

---

# 12. Fase 7 — Sincronização do estado da mesa

Ao receber mensagens WebSocket, `PokerTable.tsx` deverá atualizar seu estado local.

O estado local deve ser uma representação do estado recebido do servidor.

Exemplo:

```text
WebSocket message
       ↓
parse
       ↓
validar tipo
       ↓
atualizar estado React
       ↓
renderizar
```

Não utilizar o React como autoridade da partida.

---

# 13. Fase 8 — Ações do jogador

Ações como:

```text
CHECK
CALL
BET
RAISE
FOLD
ALL_IN
```

quando suportadas pela aplicação, devem seguir:

```text
click
  ↓
PLAYER_ACTION
  ↓
WebSocket
  ↓
Backend
  ↓
validação
  ↓
engine
  ↓
novo estado
  ↓
BroadcastToTable
  ↓
todos os clientes
```

O frontend não deve simplesmente alterar o pote/turno/cartas e assumir que a ação foi aceita.

A UI pode apresentar estado transitório de "processando", se necessário.

---

# 14. Fase 9 — Eliminar bots locais de jogadores humanos

Revisar:

```text
PokerTable.tsx#L319-L365
```

A IA local não pode assumir automaticamente o controle de qualquer assento que não seja o usuário atual.

A distinção deve ser algo semelhante a:

```text
PLAYER_HUMAN
PLAYER_BOT
```

e não:

```text
é meu jogador → humano
não é meu jogador → bot
```

Se a aplicação possuir bots reais suportados pelo backend, o backend deve ser responsável por eles.

Se os bots atuais existirem somente para simular jogadores enquanto o multiplayer não estava implementado, remover/desabilitar essa simulação para mesas multiplayer.

**Não remover código de bot definitivamente sem antes verificar se ele é utilizado em outro fluxo da aplicação.**

---

# 15. Fase 10 — Baralho e cartas

O baralho deve deixar de ser criado independentemente em cada navegador para uma partida multiplayer.

Atualmente existe:

```text
shuffleContinuousDeck()
```

em `PokerTable.tsx`.

Investigar a engine existente no backend.

Preferencialmente:

```text
Backend
   ↓
cria/embaralha baralho
   ↓
distribui cartas
   ↓
mantém estado
   ↓
envia apenas informações públicas/permitidas
```

O frontend deve apenas renderizar as cartas recebidas.

### Cartas privadas

As cartas fechadas do próprio jogador podem ser enviadas apenas ao cliente correspondente.

Nunca transmitir as cartas privadas de todos os jogadores para todos os clientes.

---

# 16. Fase 11 — `tableRooms.ts`

Revisar:

```text
getTableRoomById
```

O `localStorage` não deve ser a fonte principal para o estado atual da mesa.

Determinar a melhor integração com a API/estado WebSocket existente.

Objetivo:

```text
REST
  ↓
dados persistentes da mesa

WebSocket
  ↓
estado em tempo real da partida
```

Evitar fazer polling contínuo se o WebSocket já fornecer os eventos necessários.

---

# 17. Fase 12 — Modelo de mensagens

Antes de implementar mensagens novas, localizar o protocolo existente.

Se já existir um padrão de mensagem, reutilizá-lo.

Caso seja necessário criar/ajustar tipos, manter um formato consistente.

Exemplo conceitual:

```json
{
  "type": "JOIN_TABLE",
  "tableId": "..."
}
```

```json
{
  "type": "PLAYER_ACTION",
  "tableId": "...",
  "action": "CHECK"
}
```

Resposta:

```json
{
  "type": "TABLE_STATE",
  "tableId": "...",
  "state": {}
}
```

Os nomes exatos devem seguir os tipos já existentes no projeto.

**Não criar uma segunda implementação de protocolo se já existir uma.**

---

# 18. Tratamento de erros

Implementar tratamento para:

* WebSocket indisponível;
* desconexão;
* reconexão;
* JOIN_TABLE inválido;
* mesa inexistente;
* jogador não pertencente à mesa;
* ação inválida;
* ação fora do turno;
* payload inválido;
* mensagem desconhecida.

O frontend deve apresentar um estado coerente quando perder conexão.

Não deixar a interface aparentar que uma ação foi executada quando o servidor não confirmou a ação.

---

# 19. Concorrência e consistência

O estado da mesa deve ser protegido contra duas ações simultâneas.

Exemplo:

```text
Cliente A → CHECK
Cliente B → BET
```

O backend deve determinar a ordem e validar cada ação contra o estado atual.

O frontend não deve decidir quem possui a vez.

A regra de turno deve ser determinada pelo backend.

---

# 20. Testes obrigatórios

## Cenário 1 — Dois jogadores

Abrir a mesma mesa em:

```text
Browser A
Browser B
```

Com jogadores diferentes.

Validar:

* ambos aparecem na mesma mesa;
* ambos recebem atualização quando um jogador entra;
* nenhum é transformado em bot;
* ambos recebem o mesmo estado público.

---

## Cenário 2 — Ação de jogador

No Browser A:

```text
CHECK
```

Validar no Browser B:

* ação refletida;
* turno atualizado;
* estado da mesa atualizado.

---

## Cenário 3 — Aposta

No Browser A:

```text
BET
```

Validar:

* pote atualizado;
* stack atualizado;
* ação refletida no Browser B;
* turno atualizado.

---

## Cenário 4 — Fold

No Browser A:

```text
FOLD
```

Validar:

* jogador marcado corretamente;
* turno/rodada atualizados;
* Browser B recebe a atualização.

---

## Cenário 5 — Entrada de jogador

Com Browser A conectado à mesa:

1. abrir Browser B;
2. entrar na mesma mesa;
3. ocupar outro assento.

Validar que Browser A recebe automaticamente a atualização.

---

## Cenário 6 — Saída

Desconectar Browser B.

Validar que Browser A recebe a alteração correspondente.

---

## Cenário 7 — Reconexão

Desconectar e reconectar um cliente.

Validar que:

* a conexão é restabelecida;
* o cliente volta à mesa;
* recebe o estado atual;
* não cria uma segunda inscrição duplicada.

---

## Cenário 8 — Cartas

Validar que:

* todos os clientes veem as mesmas cartas comunitárias;
* as cartas privadas de cada jogador permanecem privadas;
* não existem baralhos diferentes por navegador.

---

# 21. Critérios de aceite

A tarefa será considerada concluída somente quando:

* [ ] `PokerTable.tsx` possuir conexão WebSocket funcional.
* [ ] `JOIN_TABLE` registrar corretamente o cliente no Hub.
* [ ] `client.TableID` for preenchido corretamente.
* [ ] `h.tables[tableID]` possuir os clientes conectados.
* [ ] `BroadcastToTable` for utilizado para eventos da mesa.
* [ ] `PLAYER_ACTION` for processado pelo backend.
* [ ] `LEAVE_TABLE` for processado.
* [ ] desconexões removerem corretamente os clientes.
* [ ] jogadores humanos não forem tratados como bots.
* [ ] o estado da partida for autoritativo no backend.
* [ ] ações forem sincronizadas entre os clientes.
* [ ] pote e turno forem sincronizados.
* [ ] entrada/saída de jogadores for sincronizada.
* [ ] cartas comunitárias forem iguais para todos os clientes.
* [ ] cartas privadas não sejam expostas a outros jogadores.
* [ ] `localStorage` não seja utilizado como fonte de verdade do estado multiplayer.
* [ ] não existam múltiplas conexões WebSocket indevidas.
* [ ] reconexão não gere inscrições duplicadas.
* [ ] testes dos cenários acima sejam executados.

---

# 22. Regras para o agente

1. **Não reescrever o projeto inteiro.**
2. **Não criar uma nova engine de Poker se já existir uma no backend.**
3. **Não duplicar tipos ou protocolos WebSocket existentes.**
4. **Não remover funcionalidades sem verificar seus usos.**
5. **Não alterar a arquitetura geral sem necessidade.**
6. Antes de implementar, apresentar um resumo das estruturas existentes e do plano de alteração.
7. Implementar em etapas pequenas.
8. Após cada etapa relevante, executar os testes/lint/build disponíveis.
9. Corrigir erros introduzidos antes de avançar.
10. Manter as alterações compatíveis com a arquitetura atual.
11. Não utilizar `Math.random()` no frontend para determinar ações ou estado da partida multiplayer.
12. O backend deve ser a autoridade para:

    * turno;
    * ações;
    * pote;
    * cartas;
    * estado da rodada;
    * jogadores.
13. O frontend é responsável principalmente por:

    * interface;
    * conexão WebSocket;
    * envio de ações;
    * renderização do estado recebido.
14. Não enviar informações privadas de um jogador para outros clientes.
15. Antes de considerar a tarefa concluída, testar com pelo menos dois clientes simultâneos.

---

# 23. Resultado esperado

A arquitetura final deve se comportar aproximadamente assim:

```text
                    BACKEND
                       │
              Poker Game State
                       │
                 WebSocket Hub
                       │
          ┌────────────┼────────────┐
          │            │            │
          ▼            ▼            ▼
      Browser A     Browser B    Browser C
          │            │            │
          │            │            │
       ação A        ação B       ação C
          │            │            │
          └────────────┼────────────┘
                       │
                       ▼
                  BACKEND
                       │
                valida/processa
                       │
                       ▼
                novo Game State
                       │
                       ▼
               BroadcastToTable
                       │
          ┌────────────┼────────────┐
          ▼            ▼            ▼
      Browser A     Browser B    Browser C
```

O objetivo final é que **todos os clientes conectados à mesma mesa estejam visualizando uma única partida real**, e não cópias independentes executadas localmente.

---

# 24. Primeira ação do agente

Antes de alterar qualquer arquivo:

1. mapear o protocolo WebSocket atual;
2. identificar a engine/serviço de Poker existente no backend;
3. identificar a estrutura atual de estado da mesa;
4. identificar como jogadores e assentos são persistidos;
5. identificar todos os usos de `tableRooms.ts`;
6. identificar todos os usos de `shuffleContinuousDeck`;
7. identificar todos os usos da lógica de bots;
8. apresentar o plano de implementação;
9. somente depois iniciar as alterações.
