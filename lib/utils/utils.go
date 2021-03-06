package utils

import (
	"encoding/json"
	"errors"
	"math/rand"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
)

var (
	soundRegex     *regexp.Regexp
	emoteRegex     *regexp.Regexp
	urlRegex       *regexp.Regexp
	soundBlacklist = []string{
		"!airhorn",
	}
)

func init() {
	soundRegex = regexp.MustCompile(`!(?P<group>[^\s]+)\s*(?P<effect>[^\s]*)`)
	emoteRegex = regexp.MustCompile(`<([^>]+)>`)
	urlRegex = regexp.MustCompile(`http.*`)
}

func ParseText(text string) string {
	soundNames := soundRegex.SubexpNames()
	soundMatches := soundRegex.FindAllStringSubmatch(text, -1)
	if len(soundMatches) > 0 {
		return ""
		soundMatch := soundMatches[0]
		soundNameMap := map[string]string{}

		for i, n := range soundMatch {
			soundNameMap[soundNames[i]] = n
			log.Printf("%s", n)
		}

		for _, b := range soundBlacklist {
			if b == soundNameMap["group"] {
				log.Printf("Excluding: %s", b)
				return ""
			}
		}
	}

	if Scontains(text, "http") {
		log.Printf("Excluding: %s", text)
		return ""
	}

	text = emoteRegex.ReplaceAllString(text, "")
	text = urlRegex.ReplaceAllString(text, "")

	text = strings.TrimSpace(text)

	text = strings.Replace(text, "&lt;", "<", -1)
	text = strings.Replace(text, "&gt;", ">", -1)
	text = strings.Replace(text, "&amp;", "&", -1)

	return text
}

func LowerFirst(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[n:]
}

func IntInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func BooltoInt(b bool) int {
	if b {
		return 1
	}
	return 0

}

func Scontains(key string, options ...string) bool {
	for _, item := range options {
		if item == key {
			return true
		}
	}
	return false
}

// Attempts to find the current users voice channel inside a given guild
func GetCurrentVoiceChannel(s *discordgo.Session, user *discordgo.User, guild *discordgo.Guild) *discordgo.Channel {
	for _, vs := range guild.VoiceStates {
		if vs.UserID == user.ID {
			channel, _ := s.State.Channel(vs.ChannelID)
			return channel
		}
	}
	return nil
}

// Returns a random integer between min and max
func RandomRange(min, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Intn(max-min) + min
}

func GetMentioned(s *discordgo.Session, m *discordgo.MessageCreate) *discordgo.User {
	for _, mention := range m.Mentions {
		if mention.ID != s.State.Ready.User.ID {
			return mention
		}
	}
	return nil
}
func GetPreferredName(guild *discordgo.Guild, UserID string) string {
	for _, member := range guild.Members {
		if member.User.ID == UserID {
			if member.Nick != "" {
				return member.Nick
			}
			return member.User.Username
		}
	}
	return "Failed to get Preferred Name"
}

func GetMember(guild *discordgo.Guild, UserID string) *discordgo.Member {
	for _, member := range guild.Members {
		if member.User.ID == UserID {
			return member
		}
	}
	return nil
}

func InTimeSpan(start, end, check time.Time) bool {
	return check.After(start) && check.Before(end)
}

func GetDaysTillWeekday(startDay int, weekday int) int {
	return (weekday + 7 - startDay) % 7
}

func LogJSON(obj interface{}) {
	b, _ := json.MarshalIndent(obj, "", "    ")
	log.Infof("%v", string(b))
}

func EnableToBool(s string) (bool, error) {
	if s == "enable" {
		return true, nil
	} else if s == "disable" {
		return false, nil
	}
	return false, errors.New("Invalid argument")
}

func ToWeekday(s string) time.Weekday {
	m := map[string]time.Weekday{
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
	}
	return m[s]
}

func FindUser(guild *discordgo.Guild, name string) (*discordgo.User, error) {
	for _, member := range guild.Members {
		if name == strings.ToLower(member.Nick) ||
			name == strings.ToLower(member.User.Username) {
			return member.User, nil
		}
	}

	return nil, errors.New("User not found")
}
