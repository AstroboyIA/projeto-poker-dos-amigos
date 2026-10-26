# Menu de configurações

O menu está disponível para jogadores e gerentes pelo atalho **CONFIGURAÇÕES** dos respectivos painéis. Ambos abrem a mesma tela (`/account/settings`), sem navegar entre páginas ao selecionar um tópico: a navegação lateral atualiza o conteúdo no painel da direita.

## Tópicos

### Conta

- Exibe username e e-mail como informações somente leitura.
- Permite editar nome completo, telefone e data de nascimento.
- A localização é selecionada em caixas encadeadas: primeiro estado, depois cidade correspondente, carregadas pela API pública do IBGE. A informação é salva no formato `Cidade/UF`. É necessária conexão à internet para carregar as opções.
- Salva os dados no backend e atualiza o usuário da sessão.
- Permite trocar a senha informando a senha atual, uma nova senha e sua confirmação.
- A nova senha deve ter entre 8 e 72 bytes; o backend verifica a senha atual e armazena a nova com bcrypt.

### Sons, Temas, Mesa de jogo, Notificações e Privacidade

Esses tópicos já aparecem na navegação e mostram informações introdutórias. Os controles e a persistência de preferências ainda não foram implementados; selecionar as opções exibidas não altera configurações.

## Implementação

- Interface: `frontend/src/pages/PlayerSettings.tsx`
- Rotas: `/account/settings` e `/player/settings`
- Atualização de dados da conta: `PUT /api/auth/profile`
- Troca de senha: `PUT /api/auth/password`
- As duas rotas da API exigem autenticação.
