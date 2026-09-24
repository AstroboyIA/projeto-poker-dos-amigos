O `TASK.md` anterior foi implementado, porém os testes manuais revelaram dois problemas críticos que precisam ser investigados e corrigidos.

## Problema 1 — Dois jogadores na mesma mesa não compartilham a mesma partida

Cenário reproduzido:

1. Conta A entra na mesa X.
2. Conta B entra na mesma mesa X.
3. Os dois usuários aparecem associados à mesma mesa.
4. Porém, cada navegador aparentemente executa uma partida independente.
5. Os jogadores não conseguem jogar juntos.
6. Na prática, parece existir uma "mesa lógica" compartilhada na listagem, mas duas instâncias independentes do estado/engine da partida.

### Objetivo

Descobrir exatamente onde o estado da partida está sendo duplicado.

Investigue, sem alterar código inicialmente:

* Como `tableId` é obtido pelo frontend.
* Como `tableId` é enviado no `JOIN_TABLE`.
* Como o backend identifica a mesa.
* Como `client.TableID` é preenchido.
* Como `h.tables[tableID]` é populado.
* Se os dois clientes realmente estão no mesmo `h.tables[tableID]`.
* Se existe algum outro mapa/estrutura mantendo o estado da partida.
* Se existe uma instância de Game/Engine/PokerState por cliente.
* Se existe uma instância de Game/Engine/PokerState por mesa.
* Onde o baralho é criado.
* Onde o estado da rodada é criado.
* Onde o turno atual é armazenado.
* Onde o pote é armazenado.
* Onde os jogadores/assentos são armazenados.
* Qual código é executado quando `JOIN_TABLE` é recebido.
* Qual código é executado quando `PLAYER_ACTION` é recebido.
* Qual estado é enviado pelo `BroadcastToTable`.
* Qual estado o `PokerTable.tsx` utiliza para renderizar a partida.
* Se `PokerTable.tsx` ainda executa alguma lógica local de distribuição de cartas, turno, pote ou ações.
* Se `tableRooms.ts` ou `localStorage` ainda interfere no estado da partida.
* Se existe algum `useState`, `useEffect` ou função que recria a partida quando o componente monta.
* Se o frontend recebe um `TABLE_STATE` do backend, mas posteriormente sobrescreve esse estado com estado local.

### Investigação obrigatória

Adicione temporariamente logs de diagnóstico, se necessário, para conseguir responder claramente:

```text
Cliente A:
tableId = ?
client/session/playerId = ?
JOIN_TABLE = ?
backend tableId = ?
backend game state instance = ?

Cliente B:
tableId = ?
client/session/playerId = ?
JOIN_TABLE = ?
backend tableId = ?
backend game state instance = ?
```

Precisamos confirmar se A e B estão realmente apontando para:

```text
MESMA TABLE ID
        +
MESMO GAME STATE
        +
MESMA ENGINE/SALA
```

e não apenas para a mesma representação visual da mesa.

---

# Problema 2 — Players fantasmas / mesa que permanece ocupada

Cenário reproduzido:

1. Usuário entra em uma mesa.
2. O usuário ocupa um assento.
3. Usuário sai da mesa.
4. A mesa continua existindo com o usuário aparentemente ocupando o assento.
5. Ao consultar/abrir novamente a mesa, o jogador antigo continua aparecendo.

### Objetivo

Descobrir onde o jogador está permanecendo registrado.

Investigue:

* `LEAVE_TABLE`.
* fechamento do WebSocket.
* `defer` do `readPump`.
* `defer` do `writePump`.
* função de unregister/disconnect do Hub.
* `client.TableID`.
* `h.tables`.
* estado persistido da mesa.
* endpoint REST que adiciona jogador ao assento.
* endpoint REST que remove jogador do assento.
* qualquer serviço de `join/leave`.
* `tableRooms.ts`.
* `localStorage`.
* qualquer cache frontend.
* qualquer estado persistido no backend.
* o que acontece quando o navegador fecha a aba sem enviar `LEAVE_TABLE`.
* o que acontece quando o WebSocket cai abruptamente.

Determine se existem duas responsabilidades diferentes:

```text
WebSocket membership
        +
ocupação persistida do assento
```

Se existirem, ambas precisam ser tratadas corretamente.

---

# Regra importante sobre LEAVE

Não assuma que:

```text
remover client do h.tables
```

é suficiente.

Também precisamos verificar se o jogador continua ocupando o assento no estado persistente da mesa.

O estado deve ficar consistente:

```text
Jogador saiu
    ↓
WebSocket removido
    ↓
cliente removido da sala
    ↓
assento liberado
    ↓
estado da mesa atualizado
    ↓
BroadcastToTable
    ↓
demais clientes atualizados
```

---

# Fase 1 — Apenas diagnóstico

Antes de modificar qualquer código:

1. Reproduza mentalmente ou através dos testes existentes os dois cenários.
2. Trace o fluxo completo de `JOIN_TABLE`.
3. Trace o fluxo completo de `PLAYER_ACTION`.
4. Trace o fluxo completo de `LEAVE_TABLE`.
5. Trace o fluxo de desconexão inesperada.
6. Identifique a fonte de verdade do estado da partida.
7. Identifique todas as estruturas que representam uma mesa.
8. Identifique todas as estruturas que representam um jogador/assento.
9. Identifique onde o estado é criado e onde é destruído.
10. Identifique onde o frontend pode estar sobrescrevendo o estado recebido do backend.

Não faça uma nova implementação antes de entender isso.

Ao terminar essa etapa, apresente:

### A. Fluxo atual

```text
JOIN
 ↓
...
 ↓
GAME STATE
```

### B. Fluxo esperado

```text
JOIN
 ↓
TABLE
 ↓
SHARED GAME STATE
 ↓
CLIENTS
```

### C. Causa raiz do problema 1

Explique exatamente por que dois clientes estão executando partidas independentes.

### D. Causa raiz do problema 2

Explique exatamente por que o jogador continua ocupando a mesa.

### E. Arquivos que precisam ser alterados

Liste cada arquivo e explique o motivo.

Não altere código antes de apresentar esse diagnóstico.

---

# Fase 2 — Correção

Depois do diagnóstico, implemente a menor alteração arquitetural necessária.

## Requisitos obrigatórios

### Estado compartilhado

Deve existir exatamente um estado autoritativo da partida por `tableId`.

Conceitualmente:

```text
tableId
   ↓
GameState
   ├── players
   ├── seats
   ├── deck
   ├── communityCards
   ├── pot
   ├── currentPlayer
   └── phase
```

Os clientes WebSocket daquela mesa devem apontar para esse mesmo estado.

Não criar:

```text
Client A → GameState A
Client B → GameState B
```

quando ambos estão na mesma mesa.

---

## Ações

O fluxo deve ser:

```text
Client A
   ↓
PLAYER_ACTION
   ↓
Backend
   ↓
GameState da tableId
   ↓
validação
   ↓
atualização
   ↓
BroadcastToTable(tableId)
   ↓
Client A + Client B
```

O frontend não deve executar uma segunda versão da engine.

---

## Entrada

Quando B entrar:

```text
JOIN_TABLE(tableId)
       ↓
backend localiza GameState da tableId
       ↓
registra B
       ↓
atualiza estado
       ↓
BroadcastToTable
       ↓
A recebe B
B recebe estado atual
```

---

## Saída

Quando A sair:

```text
LEAVE_TABLE
       ↓
remover A do Hub
       ↓
liberar assento de A
       ↓
atualizar GameState
       ↓
BroadcastToTable
       ↓
B recebe estado atualizado
```

Também trate:

```text
WebSocket disconnect
```

como uma possível saída da mesa.

Não dependa exclusivamente de `LEAVE_TABLE`, pois o navegador pode fechar, perder conexão ou cair.

---

# Fase 3 — Testes obrigatórios

Depois das correções, valide pelo menos:

### Teste A — Dois clientes

```text
Cliente A → Mesa X
Cliente B → Mesa X
```

Confirmar:

* mesmo `tableId`;
* mesmo estado de partida;
* ambos aparecem na mesma mesa;
* somente uma instância da partida existe;
* ambos recebem as mesmas cartas comunitárias;
* ações de A aparecem para B;
* ações de B aparecem para A.

### Teste B — Ação

A executa:

```text
CHECK
```

B deve receber a alteração.

Depois B executa:

```text
BET
```

A deve receber a alteração.

### Teste C — Saída

A sai.

Confirmar em B:

* A desaparece;
* assento é liberado;
* estado é atualizado;
* não existe player fantasma.

### Teste D — Fechamento abrupto

Fechar a aba de A sem executar explicitamente `LEAVE_TABLE`.

Confirmar que o backend detecta a desconexão e libera corretamente o jogador.

### Teste E — Reentrada

A entra novamente.

Confirmar:

* não cria jogador duplicado;
* não ocupa dois assentos;
* recebe o estado atual;
* não cria uma nova instância da partida.

### Teste F — Mesa vazia

Todos os jogadores saem.

Confirmar:

* nenhum player permanece no estado da mesa;
* `h.tables[tableId]` não mantém clientes inexistentes;
* a mesa não apresenta jogadores fantasmas;
* se a arquitetura exigir remoção da sala vazia, ela é removida corretamente.

---

# Critério de conclusão

Não considere a tarefa concluída apenas porque:

* o frontend compila;
* o backend compila;
* o WebSocket conecta;
* dois usuários aparecem na mesma mesa.

A tarefa somente estará concluída quando houver evidência de que:

```text
2 clientes
   ↓
1 tableId
   ↓
1 GameState
   ↓
1 partida
   ↓
Broadcast
   ↓
2 interfaces sincronizadas
```

e quando:

```text
jogador sai
   ↓
assento liberado
   ↓
estado atualizado
   ↓
nenhum player fantasma
```

Ao final, informe:

* causa raiz encontrada;
* arquivos modificados;
* alterações realizadas;
* testes executados;
* resultado de cada teste;
* eventuais limitações restantes.
