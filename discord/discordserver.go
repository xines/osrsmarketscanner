package discord

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"osrsmarketscanner/gedb"
	"osrsmarketscanner/osbuddy"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/objectbox/objectbox-go/objectbox"
)

type BotInfo struct {
	Token            string  `json:"Token"`
	ChannelID        string  `json:"ChannelID"`
	ProfitMinimum    int     `json:"ProfitMinimum"`    // Amount in gp
	ProfitPercentage float64 `json:"ProfitPercentage"` // 1.0 = 100%
}

var (
	BotSettings        BotInfo
	isStringAlphabetic = regexp.MustCompile(`^[a-zA-Z0-9'-_ ]*$`).MatchString // For input checking
	Gebox              *gedb.GeDatasBox
)

func init() {
	if err := importSettingsJSON(); err != nil {
		fmt.Println(err.Error())
		return
	}

	flag.StringVar(&BotSettings.Token, "t", BotSettings.Token, "Bot Token")
	flag.Parse()
}

func importSettingsJSON() error {

	jsonFile, err := os.Open("settings.json")
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	json.Unmarshal(byteValue, &BotSettings)

	fmt.Println("Done initializing Settings.")

	return nil
}

//StartDiscordBot starts a new discord session - default args "Console msg", "NO-ID"
func StartDiscordBot(msg string, channelID string) {
	fmt.Println(msg)

	// Create objectbox and store to global
	oBox, err := objectbox.NewBuilder().Model(gedb.ObjectBoxModel()).Build()
	if err != nil {
		log.Fatalf("Could not make new builder for object box.")
		return
	}
	Gebox = gedb.BoxForGeDatas(oBox)

	// Create New Discord Session With Bot Token
	dBot, err := discordgo.New("Bot " + BotSettings.Token)
	if err != nil {
		fmt.Println("Error creating Discord session,", err)
	}

	dBot.AddHandler(messageCreate)

	// Only care about receiving message events.
	dBot.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = dBot.Open()
	if err != nil {
		fmt.Println("Error opening connection,", err)
	}

	if channelID != "NO-ID" {
		dBot.ChannelMessageSend(channelID, "Bot Restarted.")
	}

	// Start analyzing grand exchange on another goroutine
	go geAnalyzer(dBot, BotSettings.ChannelID)

	fmt.Println("Bot is now running waiting for commands.  Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord + objectbox session.
	dBot.Close()
	oBox.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!shutdown" {
		s.ChannelMessageSend(m.ChannelID, "Alright i'll goto sleep.")
		s.Close() // Cleanly close down the Discord session.
	}

	if m.Content == "!restart" {
		s.ChannelMessageSend(m.ChannelID, "BioBop bot restarting...")
		s.Close()                                                 // Cleanly close down the Discord session.
		StartDiscordBot("Starting New Bot Session.", m.ChannelID) // Start new session
	}

	go func(txt string) {
		if strings.Contains(txt, "!ge ") {
			sendGeItemMessageDiscord(txt, s, m.ChannelID)
		}
	}(strings.ToLower(m.Content))
}

// sendMessageDiscord sends a discord message for searched item, goroutines ftw
func sendGeItemMessageDiscord(text string, s *discordgo.Session, channelID string) {

	if !isStringAlphabetic(text[4:]) {
		s.ChannelMessageSend(channelID, "Issue trying to find: "+text[4:])
		return
	}

	item, err := osbuddy.GetItemDataByName(text[4:])
	if err != nil {
		fmt.Printf("Error getting item: %s", err)
		s.ChannelMessageSend(channelID, "Error getting item: "+text[4:])
		return
	}

	// Convert bool true/false to string value yes/no
	isMember := "no"
	if item.Members {
		isMember = "Yes"
	}

	// Fetch method
	queryList, err := Gebox.Query(gedb.GeDatas_.ItemID.Equals(int64(item.ID))).Find()
	if err != nil {
		log.Printf("Debug: Gebox.Query - itemID %s %d", item.Name, item.ID)
		return
	}

	// Store item from fetched results
	var itemdb *gedb.GeDatas
	for _, v := range queryList {
		if v.Name == item.Name {
			itemdb = v
		}
	}

	// Get symbol depending on item status and display cached price if Inactive
	buySymbol := "✅"
	sellSymbol := "✅"
	isDatabaseBuy := false
	isDatabaseSell := false

	// If change to database datas
	if itemdb.BuyAverage != int64(item.BuyAverage) && itemdb.BuyAverage != 0 {
		buySymbol = "❓"
		isDatabaseBuy = true
	}
	if itemdb.SellAverage != int64(item.SellAverage) && itemdb.SellAverage != 0 {
		sellSymbol = "❓"
		isDatabaseSell = true
	}

	// Using real time data (if value 0 use db stored datas)
	if !isDatabaseBuy {
		if item.BuyAverage != 0 {
			itemdb.BuyAverage = int64(item.BuyAverage)
		} else {
			buySymbol = "❌"
		}
	}
	if !isDatabaseSell {
		if item.SellAverage != 0 {
			itemdb.SellAverage = int64(item.SellAverage)
		} else {
			sellSymbol = "❌"
		}
	}

	// Convert profits values to displayable text
	profit := (int(itemdb.SellAverage) - int(itemdb.BuyAverage))
	profitpercentage := float64(profit) / float64(int(itemdb.SellAverage)) * 100
	profittext := strconv.Itoa(profit) + " gp" + "\n" + strconv.FormatFloat(profitpercentage, 'f', 2, 64) + "%"

	// Make fields
	fields := make([]*discordgo.MessageEmbedField, 0)
	fields = append(fields, []*discordgo.MessageEmbedField{
		{
			Name:   "Item ID",
			Value:  strconv.Itoa(item.ID),
			Inline: true,
		},
		{
			Name:   "Members Only",
			Value:  isMember,
			Inline: true,
		},
		{
			Name:   "Shop Price",
			Value:  strconv.Itoa(item.Sp) + " gp",
			Inline: true,
		},
		{
			Name:   "\nApprox Avg Profit",
			Value:  profittext,
			Inline: true,
		},
		{
			Name:   "\nBuy Price Avg",
			Value:  strconv.Itoa(int(itemdb.BuyAverage)) + " gp " + buySymbol + "\n" + shortenNumberOSRS(int(itemdb.BuyAverage)),
			Inline: true,
		},
		{
			Name:   "\nSell Price Avg",
			Value:  strconv.Itoa(int(itemdb.SellAverage)) + " gp " + sellSymbol + "\n" + shortenNumberOSRS(int(itemdb.SellAverage)),
			Inline: true,
		},
		{
			Name:   "Buy Quantity",
			Value:  strconv.Itoa(item.BuyQuantity),
			Inline: true,
		},
		{
			Name:   "Sell Quantity",
			Value:  strconv.Itoa(item.SellQuantity),
			Inline: true,
		},
		{
			Name:   "Last Updated",
			Value:  dateUnixFormatDatabase(itemdb.Date),
			Inline: true,
		},
	}...)

	pngURL := "https://secure.runescape.com/m=itemdb_oldschool/obj_sprite.gif?id=" + strconv.Itoa(item.ID)
	pngImgURL := "https://secure.runescape.com/m=itemdb_oldschool/obj_big.gif?id=" + strconv.Itoa(item.ID)

	s.ChannelMessageSendEmbed(channelID, &discordgo.MessageEmbed{
		//URL:         "",
		Type:        discordgo.EmbedTypeImage,
		Title:       item.Name,
		Description: "✅ = Recent data.\n❓ = Last known data.\n❌ = No data, needs more time.",
		Color:       0x00FF00,
		Fields:      fields,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: pngImgURL},
		Image:       &discordgo.MessageEmbedImage{URL: pngURL},
		Footer:      &discordgo.MessageEmbedFooter{Text: "Developed by Xines."},
	})
}

func shortenNumberOSRS(n int) string {
	if n > 1000000000 {
		return strconv.FormatFloat(float64(n)/float64(1000000000), 'f', 2, 64) + "B"
	} else if n > 1000000 {
		return strconv.FormatFloat(float64(n)/float64(1000000), 'f', 2, 64) + "M"
	} else if n > 1000 {
		return strconv.FormatFloat(float64(n)/float64(1000), 'f', 2, 64) + "K"
	} else {
		return strconv.Itoa(n)
	}
}

// dateUnixFormatDatabase get's and formats a stored date in time from unix time in database
// Example output: 2006-01-02 15:04:05
func dateUnixFormatDatabase(d int64) string {
	if d <= 0 {
		return "No datas"
	}
	return time.Unix(int64(d/1000), 0).Format("2006-01-02\n15:04:05")
}

// geAnalyzer(Session ID, Discord Target Channel ID)
func geAnalyzer(s *discordgo.Session, channelID string) {

	for {
		if err := UpdateGeDatabase(); err != nil {
			fmt.Printf("UpdateGeDatabase problem found :( - %s\n", err)
			return
		}

		for _, v := range osbuddy.GeItems {

			// Convert profits values to displayable text
			profit := (v.SellAverage - v.BuyAverage)
			profitpercentage := float64(profit) / float64(v.SellAverage) * 100

			// TODO Idea - make discord bot command for changeable profit & percentage
			if profit >= BotSettings.ProfitMinimum && profitpercentage > BotSettings.ProfitPercentage && profitpercentage != 100.0 {

				itemCached := false
				itemUpdated := false
				for _, val := range osbuddy.CachedItems {
					if v == val {

						itemCached = true

						// Check for new price updates
						if val.BuyAverage != v.BuyAverage || val.SellAverage != v.SellAverage {
							itemUpdated = true
						} else {
							itemUpdated = false
						}

						break
					}
				}

				// Add to cache
				if !itemCached {
					osbuddy.CachedItems = append(osbuddy.CachedItems, v)
					fmt.Printf("CACHEED: %s\n", v.Name)
				}

				// Send text to discord server
				if itemUpdated || !itemCached {
					ptext := "```" + v.Name + " 💲 🡲 " + strconv.FormatFloat(profitpercentage, 'f', 2, 64) + "% 🡰 profit margin 🤑 possible flip reward 🡲 " + shortenNumberOSRS(profit) + " 🔥```"
					s.ChannelMessageSend(channelID, ptext)
					time.Sleep(200 * time.Millisecond)
				}
			}
		}

		// Update every x minute
		time.Sleep(2 * time.Minute)
	}
}

// UpdateGeDatabase Updates all items from json fetch then stores into database
func UpdateGeDatabase() error {

	// Update items to current state
	if err := osbuddy.UpdateGEItems(); err != nil {
		fmt.Printf("UpdateGEItems - Failed to update items.")
		return err
	}

	var (
		id            uint64
		continuecheck bool
		pricedif      bool
	)

	for _, v := range osbuddy.GeItems {

		queryItemList, err := Gebox.Query(gedb.GeDatas_.ItemID.Equals(int64(v.ID))).Find()
		if err != nil {
			log.Printf("Debug: Gebox.Query - itemID %s %d", v.Name, v.ID)
			continue
		}

		continuecheck = false
		for _, val := range queryItemList {

			if val.ItemID == int64(v.ID) {

				// Reset for rechecking
				pricedif = false

				// Check for price changes and store to database if any is found
				// val.BuyAverage != int64(v.BuyAverage) -> only affects item not being updated if price haven't changed ->
				// (pros = less updates)(cons = less accurate price buy/sell date logged)
				if val.BuyAverage >= 0 && v.BuyAverage > 0 && val.BuyAverage != int64(v.BuyAverage) {
					val.BuyAverage = int64(v.BuyAverage)
					pricedif = true
				}

				if val.SellAverage >= 0 && v.SellAverage > 0 && val.SellAverage != int64(v.SellAverage) {
					val.SellAverage = int64(v.SellAverage)
					pricedif = true
				}

				// Update database if price differs
				if pricedif {
					val.Date = int64(time.Now().Unix() * 1000)
					err := Gebox.Update(val)
					if err != nil {
						log.Printf("Gebox.Update: Could not update prices for %s", v.Name)
					}
				}

				continuecheck = true
				break
			}
		}

		if continuecheck {
			continue
		}

		id, err = Gebox.Put(&gedb.GeDatas{
			ItemID:          int64(v.ID),
			Name:            v.Name,
			Members:         v.Members,
			Sp:              int64(v.Sp),
			BuyAverage:      int64(v.BuyAverage),
			BuyQuantity:     int64(v.BuyQuantity),
			SellAverage:     int64(v.SellAverage),
			SellQuantity:    int64(v.SellQuantity),
			OverallAverage:  int64(v.OverallAverage),
			OverallQuantity: int64(v.OverallQuantity),
			Date:            int64(time.Now().Unix() * 1000),
		})

		if err != nil {
			log.Printf("Error adding - itemID %s %d", v.Name, v.ID)
			continue
		}

		log.Printf("Added ID: %d, %s - ItemID: %d\n", id, v.Name, v.ID)
	}

	log.Print("Database updated successfully.\n")
	return nil
}
