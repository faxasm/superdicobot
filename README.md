# Bot  pour untimeout en masse sur twitch
## (et surement plein de truc plus tard)

### 1) annule tous les timeouts  de moins de "MAX_TIMEOUT_DURATION"   

#

## configuration
    placer un fichier de conf app.env  dans le dossier ./config
    avec ces parametres:
#
        TWITCH_USER=userbot
        TWITCH_OAUTH=oauth:xxx
        TWITCH_CHANNEL=botChannel
        UNTIMEOUT_CMD=!unto
        MAX_TIMEOUT_DURATION=600   

##  en local

    make run

##  build l'application

    make build 

##  lancer l'application

    ./target/app
