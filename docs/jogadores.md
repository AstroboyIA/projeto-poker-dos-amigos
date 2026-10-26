# Menu de jogadores

O atalho **JOGADORES** do painel do jogador abre a tela de jogadores (`/player/community`). Nela, as seções **Meu perfil** e **Amigos** são alternadas localmente; trocar de seção não navega para outra página.

## Meu perfil

Exibe informações da conta autenticada:

- Nome de exibição, usando o nome completo ou, na ausência dele, o username.
- Username e iniciais para o avatar visual.
- Situação da conta, apresentada como membro ativo quando o status é `ativo`.
- Cidade/estado, data de nascimento e nome, com indicação quando não informados.

O botão **Editar meus dados** leva às configurações da conta, onde esses dados podem ser atualizados.

## Amigos

Mostra o estado vazio da lista e informa que a busca de jogadores e os pedidos de amizade ainda não estão disponíveis. Não há cadastro de amizades, pesquisa, envio ou resposta a solicitações nesta etapa.

## Implementação

- Interface: `frontend/src/pages/PlayerCommunity.tsx`
- Rota: `/player/community`
- Dados do perfil são lidos do usuário autenticado no contexto de autenticação; ainda não há uma API própria para listar jogadores ou administrar amizades.

