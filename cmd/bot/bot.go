package main

import (
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	redis "gopkg.in/redis.v3"
)

var (
	// discordgo session
	discord *discordgo.Session

	// Redis client connection (used for stats)
	rcli *redis.Client

	// Sound encoding settings
	BITRATE        = 128
	MAX_QUEUE_SIZE = 6

	// Prefix for chat commands
	PREFIX = "!!"

	// Owner
	OWNER string
)

func init() {
	// Seed the random number generator.
	rand.Seed(time.Now().UnixNano()) 
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	log.Info("Recieved READY payload")
	s.UpdateStatus(0, "Overwatch")
}

func processGameplayLoop(ticker *time.Ticker) {
    for {
    	select {
    	case <- ticker.C:        		
    		var processedUsers []string
        	for _, g := range discord.State.Guilds {
	        	for _, p := range g.Presences {
	        		if p.Game != nil && len(p.Game.Name) > 0 &&
	        				!utilStringInSlice(p.User.ID, processedUsers) {

	        			processedUsers = append(processedUsers, p.User.ID)
	            		dbIncGameEntry(p.User.ID, p.Game.Name, 60)
	            	}
	            }
	        }
    	}
    }
}

func onGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable != nil {
		return
	}

	for _, channel := range event.Guild.Channels {
		if channel.ID == event.Guild.ID {
			s.ChannelMessageSend(channel.ID, "**AIRHORN BOT READY FOR HORNING. TYPE `!AIRHORN` WHILE IN A VOICE CHANNEL TO ACTIVATE**")
			return
		}
	}
}

func airhornBomb(cid string, guild *discordgo.Guild, user *discordgo.User, cs string) {
	count, _ := strconv.Atoi(cs)
	discord.ChannelMessageSend(cid, ":ok_hand:"+strings.Repeat(":trumpet:", count))

	// Cap it at something
	if count > 100 {
		return
	}

	play := sndCreatePlay(user, guild, AIRHORN, nil)
	vc, err := discord.ChannelVoiceJoin(play.GuildID, play.ChannelID, true, true)
	if err != nil {
		return
	}

	for i := 0; i < count; i++ {
		AIRHORN.Random().Play(vc)
	}

	vc.Disconnect()
}

// Handles bot operator messages, should be refactored (lmao)
func handleBotControlMessages(s *discordgo.Session, m *discordgo.MessageCreate, parts []string, g *discordgo.Guild) {
	if utilScontains(parts[1], "status") {
		rdDisplayBotStats(m.ChannelID)

	} else if utilScontains(parts[1], "stats") {
		if len(m.Mentions) >= 2 {
			rdDisplayUserStats(m.ChannelID, utilGetMentioned(s, m).ID)
		} else if len(parts) >= 3 {
			rdDisplayUserStats(m.ChannelID, parts[2])
		} else {
			rdDisplayServerStats(m.ChannelID, g.ID)
		}

	} else if utilScontains(parts[1], "bomb") && len(parts) >= 4 {
		airhornBomb(m.ChannelID, g, utilGetMentioned(s, m), parts[3])

	} else if utilScontains(parts[1], "aps") {
		s.ChannelMessageSend(m.ChannelID, ":ok_hand: give me a sec m8")
		go rdCalculateAirhornsPerSecond(m.ChannelID)

	} else if utilScontains(parts[1], "save_messages") && len(parts) >= 4 {
		s.ChannelMessageSend(m.ChannelID, ":ok_hand: give me a sec m8")
		go mkFetchAndSaveMessages(m.ChannelID, g, m.Author, parts[2], parts[3])

	} else if utilScontains(parts[1], "generate_chain") && len(parts) >= 4 {
		s.ChannelMessageSend(m.ChannelID, ":ok_hand: give me a sec m8")
		go mkGenerateChain(m.ChannelID, g, m.Author, parts[2], parts[3])

	} else if utilScontains(parts[1], "load_chain") && len(parts) >= 3 {
		s.ChannelMessageSend(m.ChannelID, ":ok_hand: give me a sec m8")
		go mkLoadChain(m.ChannelID, g, m.Author, parts[2])

	} else if utilScontains(parts[1], "get_message") && len(parts) >= 3 {
		s.ChannelMessageSend(m.ChannelID, ":ok_hand: give me a sec m8")
		go mkGetMessage(m.ChannelID, g, m.Author, parts[2])
	}
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(m.Content) <= 0 {
		return
	}

	channel, _ := discord.State.Channel(m.ChannelID)
	if channel == nil {
		log.WithFields(log.Fields{
			"channel": m.ChannelID,
			"message": m.ID,
		}).Warning("Failed to grab channel")
		return
	}

	guild, _ := discord.State.Guild(channel.GuildID)
	if guild == nil {
		log.WithFields(log.Fields{
			"guild":   channel.GuildID,
			"channel": channel,
			"message": m.ID,
		}).Warning("Failed to grab guild")
		return
	}

	// If we have a message not starting with "!", then handle markov stuff
	if (!strings.HasPrefix(m.Content, "!") && len(m.Mentions) < 1) {
		mkWriteMessage(guild, m.Content)
		rando := rand.Intn(100)
		if rando < 10 {
			log.Printf("Sending markov message")
			go mkGetMessage(m.ChannelID, guild, m.Author, "1")
		}
		return
	}

	msg := strings.Replace(m.ContentWithMentionsReplaced(), s.State.Ready.User.Username, "username", 1)
	parts := strings.Split(strings.ToLower(msg), " ")

	// If this is a mention, it should come from the owner (otherwise we don't care)
	if len(m.Mentions) > 0 && m.Author.ID == OWNER && len(parts) > 0 {
		mentioned := false
		for _, mention := range m.Mentions {
			mentioned = (mention.ID == s.State.Ready.User.ID)
			if mentioned {
				break
			}
		}

		if mentioned {
			handleBotControlMessages(s, m, parts, guild)
		}
		return
	}

	// Filter out commands for airhornbot
	if (!strings.HasPrefix(m.Content, PREFIX)) {
		log.Printf("Filtering out airhornbot command")
		return
	}

	baseCommand := strings.Replace(parts[0], PREFIX, "", 1)

	if utilScontains(baseCommand, "text") {
		if len(parts) > 1 {
			go mkGetMessage(m.ChannelID, guild, m.Author, parts[1])
		} else {
			go mkGetMessage(m.ChannelID, guild, m.Author, "1")
		}
		return
	} else if utilScontains(baseCommand, "stats") {
		go gpPrintStats(m.ChannelID, m.Author)
		return
	}

	// Find the collection for the command we got
	for _, coll := range COLLECTIONS {
		if utilScontains(baseCommand, coll.Commands...) {

			// If they passed a specific sound effect, find and select that (otherwise play nothing)
			var sound *Sound
			if len(parts) > 1 {
				for _, s := range coll.Sounds {
					if parts[1] == s.Name {
						sound = s
					}
				}

				if sound == nil {
					return
				}
			}

			go sndEnqueuePlay(m.Author, guild, coll, sound)
			return
		}
	}
}

// Handle updating of presences in the current session, because the API doesnt...
func onPresenceUpdate(s *discordgo.Session, u *discordgo.PresenceUpdate) {
	if s == nil {
		return
	}

	guild, err := s.Guild(u.GuildID)
	if err != nil {
		return
	}

	s.Lock()
	defer s.Unlock()

	for i, p := range guild.Presences {
		if p.User.ID == u.User.ID {
			guild.Presences[i].Status = u.Status
			guild.Presences[i].Game = u.Game
			return
		}
	}

	return
}

func main() {
	var (
		Token      = flag.String("t", "", "Discord Authentication Token")
		Redis      = flag.String("r", "", "Redis Connection String")
		Owner      = flag.String("o", "", "Owner ID")
		err        error
	)
	flag.Parse()

	if *Owner != "" {
		OWNER = *Owner
	}

	// Preload all the sounds
	log.Info("Preloading sounds...")
	for _, coll := range COLLECTIONS {
		coll.Load()
	}

	// If we got passed a redis server, try to connect
	if *Redis != "" {
		log.Info("Connecting to redis...")
		rcli = redis.NewClient(&redis.Options{Addr: *Redis, DB: 0})
		_, err = rcli.Ping().Result()

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Failed to connect to redis")
			return
		}
	}

	dbOpen("./Drumpf.db")

	// Create a discord session
	log.Info("Starting discord session...")
	discord, err = discordgo.New(*Token)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create discord session")
		return
	}

	discord.AddHandler(onReady)
	discord.AddHandler(onGuildCreate)
	discord.AddHandler(onMessageCreate)
	discord.AddHandler(onPresenceUpdate)

	err = discord.Open()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create discord websocket connection")
		return
	}

	// We're running!
	log.Info("AIRHORNBOT is ready to horn it up.")

	log.Info("Setting up Game watch tick")
	ticker := time.NewTicker(time.Second * 60)
    go processGameplayLoop(ticker)

	// Wait for a signal to quit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
}
