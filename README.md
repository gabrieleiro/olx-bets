https://github.com/user-attachments/assets/a210f573-1703-4196-8722-49c7adf575f9

# Como rodar sua própria instância
Caso não tenha experiência criando bots de discord, comece pelo [guia oficial](https://discord.com/developers/docs/intro).

1. Após registrar o seu próprio bot, habilite as instalações em servidores (guild installs) e adicione os scopes "applications.commands" e "bot, as permissões "Add Reactions", "Create Public Threads", "Read Message History", "Send Messages", "Send Messages in Threads" e "View Channels" e habilite os privileged intents "Server members" e "Message content".
2. Crie um arquivo .ENV na pasta "bot" com as variáveis DB_URL (url para um DB hospedado na [Turso](https://turso.tech/) ou caminho para um arquivo sqlite), ENV (deve ser "development" para desenvolvimento local e "production" quando estiver deployado em prod), BOT_TOKEN e DEV_GUILD(o servidor que você usará para testar o bot localmente)
3. Na pasta bot, rode o comando `go build && ./bot` ou `go run main.go`
