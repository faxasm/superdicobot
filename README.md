# Bot  pour untimeout en masse sur twitch
## (et surement plein de truc plus tard)

### 1) annule tous les timeouts  de moins de "MAX_TIMEOUT_DURATION"   

#

## configuration
    placer un fichier de conf config.yaml  dans le dossier ./config
    avec ces parametres:
#

```
loggerLevel: "info" #niveau de log global
loggerFile: "./log/main.log"
bots:
 -
  user: "botName"
  oauth: "oauth:xxx"
  loggerLevel: "info" # niveau de log pour le bot
  loggerFile: "./log/botName.log" 
  administrator: "userAdmin" #admin du bot
  channels:
    -
     channel: "oneChannel" # channel twitch
     unTimeoutCmd: "!unto" # commande de untimeout en masse
     pingCmd: "!ping superdicobot" # commande de ping sur la chaine
     maxTimeoutDuration: 600 # temps max de timeout qui sera untimeout
     loggerLevel: "info" # niveau de log pour le channel
     loggerFile: "./log/botName/oneChannel.log" # fichier de log pour ce chan
    -
     channel: "secondChannel"
     unTimeoutCmd: "!unto"
     pingCmd: "!ping superdicobot"
     maxTimeoutDuration: 600
     loggerLevel: "info"
     loggerFile: "./log/botName/secondChannel.log"
```yaml



##  en local

    make run

##  build l'application

    make build 

##  lancer l'application

    ./target/superdicobot
