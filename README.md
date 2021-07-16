# OSRS Market Scanner (Discord Bot)

Discord bot with grand exchange lookup & market scan functionality

## Description
Once up & running it will allow members of your discord group to use keyword "!ge item name" to lookup item stats. Additionally theres a market scanner that displays current margins of items depending on what settings is being used in settings.json.

Program will feed database with new data overtime and provide as accurate infomation as api allows.

## Showcase Images
![](https://i.imgur.com/snxEuyR.png)
![](https://i.imgur.com/FcD69gU.png)


## Getting Started

### Dependencies

* [discordgo](https://github.com/bwmarrin/discordgo)
* [objectbox](https://github.com/objectbox/objectbox-go)

### Installing

* Compile script
* Create discord bot and grab token
* Setup settings.json with accurate information such as channel ids and bot token
* Start bot

### Discord commands

* !ge item keyword or full name
* !shutdown
* !restart


## Version History

* 0.1
    * Initial Release

## License

This project is licensed under the BSD-style License - see the LICENSE.md file for details