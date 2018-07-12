package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/gomodule/redigo/redis"
)

// Guild configuration information per guild
type Guild struct {
	GuildName           string
	GuildPrefix         string
	GuildID             string
	Users               []User
	BlacklistedChannels []string
	BlacklistedMembers  []string
	WelcomeMessage      string
	WelcomeChannel      string
	GoodbyeMessage      string
	GoodbyeChannel      string
	Events              []string
	DisabledCommands    []string
	BirthdayRole        string
	MutedRole           string
	AutoRole            []string
}

// User information
type User struct {
	Username          string
	Discriminator     string
	Nickname          string
	ID                string
	Age               string
	JoinedAt          string
	PreviousUsernames []string
	PreviousNicknames []string
	Roles             []string
	Warnings          []Warnings
	Kicks             []Kicks
	Bans              []Bans
	Mutes             []Mutes
}

// Warnings information for a user
type Warnings struct {
	Channel string
	Reason  string
	Time    time.Time
}

// Kicks information for a user
type Kicks struct {
	Channel string
	Reason  string
	Time    time.Time
}

// Bans information for a user
type Bans struct {
	Channel string
	Reason  string
	Time    time.Time
}

// Mutes information for a user
type Mutes struct {
	Channel string
	Reason  string
	Time    time.Time
	Length  time.Duration
}

// DialNewPool connectes to a local Redis database by port pass-in
func DialNewPool(net string, port string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:   80,
		MaxActive: 12000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(net, port)
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
}

// DialNewPoolURL connectes to a Redis database by URL pass-in
func DialNewPoolURL(url string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:   80,
		MaxActive: 12000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(os.Getenv(url))
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
}

// DeleteGuild removes a guild from the database
func DeleteGuild(guild *discordgo.Guild) (interface{}, error) {
	n, err := p.Do("DEL", guild.ID)
	if err != nil {
		log.Println(err)
		return n, err
	}
	return n, nil
}

// GuildExists returns true if guild key exists, false if not
func GuildExists(guild *discordgo.Guild) bool {
	n, err := p.Do("EXISTS", guild.ID)
	if err != nil {
		log.Println(err)
	}

	// FIX THIS FLUSH

	if n == 1 {
		return true
	}

	return false
}

// RegisterNewGuild creates a key with a guild ID
func RegisterNewGuild(guild *discordgo.Guild) (interface{}, error) {
	users := InitializeUsers(guild)

	g := &Guild{
		GuildName:   guild.Name,
		GuildPrefix: "+",
		GuildID:     guild.ID,
		Users:       users,
	}

	serialized, err := json.Marshal(g)

	if err != nil {
		log.Println(err)
	}

	n, err := p.Do("SETNX", guild.ID, serialized)
	if err != nil {
		log.Println(err)
	}

	return n, nil
}

// InitializeUsers fetches user defaults upon guild initialization
func InitializeUsers(guild *discordgo.Guild) []User {
	// mem, err := ctx.Session.State.Member(ctx.Guild.ID, ctx.Event.Author.ID)
	user := []User{}
	for _, usr := range guild.Members {
		age, err := CreationTime(usr.User.ID)
		iso, err := time.Parse(time.RFC3339Nano, usr.JoinedAt)

		if err != nil {
			log.Println(err)
		}

		user = append(user, User{
			Username:          usr.User.Username,
			Discriminator:     usr.User.Discriminator,
			Nickname:          usr.Nick,
			ID:                usr.User.ID,
			Age:               age.Format("01/02/06 03:04:05 PM MST"),
			JoinedAt:          iso.Format("01/02/06 03:04:05 PM MST"),
			PreviousUsernames: []string{},
			PreviousNicknames: []string{},
			Roles:             []string{},
			Warnings:          []Warnings{},
			Kicks:             []Kicks{},
			Bans:              []Bans{},
			Mutes:             []Mutes{},
		})
	}

	return user
}

// LogWarning logs a warning to a user's record in the database
func LogWarning(ctx Context, userID string, reason string) {
	data, err := redis.Bytes(p.Do("GET", ctx.Guild.ID))
	if err != nil {
		panic(err.Error())
	}

	var g Guild
	err = json.Unmarshal(data, &g)

	if err != nil {
		log.Println(err)
	}

	for k := range g.Users {
		if g.Users[k].ID == userID {
			g.Users[k].Warnings = append(g.Users[k].Warnings, Warnings{
				Channel: fmt.Sprintf("%s / %s", ctx.Channel.Name, ctx.Channel.ID),
				Reason:  reason,
				Time:    time.Now(),
			})
		}
	}

	serialized, err := json.Marshal(g)

	if err != nil {
		log.Println(err)
	}

	_, err = p.Do("SET", ctx.Guild.ID, serialized)
	if err != nil {
		log.Println(err)
	}
}

// LogKick logs a kick to a user's record in the database
func LogKick(ctx Context, userID string, reason string) {
	data, err := redis.Bytes(p.Do("GET", ctx.Guild.ID))
	if err != nil {
		panic(err.Error())
	}

	var g Guild
	err = json.Unmarshal(data, &g)

	if err != nil {
		log.Println(err)
	}

	for k := range g.Users {
		if g.Users[k].ID == userID {
			g.Users[k].Kicks = append(g.Users[k].Kicks, Kicks{
				Channel: fmt.Sprintf("%s / %s", ctx.Channel.Name, ctx.Channel.ID),
				Reason:  reason,
				Time:    time.Now(),
			})
		}
	}

	serialized, err := json.Marshal(g)

	if err != nil {
		log.Println(err)
	}

	_, err = p.Do("SET", ctx.Guild.ID, serialized)
	if err != nil {
		log.Println(err)
	}
}

// LogBan logs a ban to a user's record in the database
func LogBan(ctx Context, userID string, reason string) {
	data, err := redis.Bytes(p.Do("GET", ctx.Guild.ID))
	if err != nil {
		panic(err.Error())
	}

	var g Guild
	err = json.Unmarshal(data, &g)

	if err != nil {
		log.Println(err)
	}

	for k := range g.Users {
		if g.Users[k].ID == userID {
			g.Users[k].Bans = append(g.Users[k].Bans, Bans{
				Channel: fmt.Sprintf("%s / %s", ctx.Channel.Name, ctx.Channel.ID),
				Reason:  reason,
				Time:    time.Now(),
			})
		}
	}

	serialized, err := json.Marshal(g)

	if err != nil {
		log.Println(err)
	}

	_, err = p.Do("SET", ctx.Guild.ID, serialized)
	if err != nil {
		log.Println(err)
	}
}
