package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

// types/types.go

// UserID is a unique identifier for a Telegram user or bot.
type UserID int

// ReplierRepository returns a replier for a given user or bot.
type ReplierRepository interface {
	ProvideReplier(UserID) Replier
	SaveReplier(UserID, Replier)
	DeleteReplier(UserID)
}

// Replier replies to a given update on the Reply call.
// It returns the reply message and the next Replier if communication is pending.
type Replier interface {
	Reply(Update) (string, Replier)
}

// Update is a message from a user or bot.
type Update struct {
	IsCommand bool
	Cmd       string
	Args      []string
	Text      string
}

// DBProvider provides a DB for a given user.
type DBProvider interface {
	ProvideDB(UserID) DB
}

// DB stores all the data of a given user.
type DB interface {
	CreateNote(txt string, tags []string)
	ListNotes(tags []string) string
}

// prototype/replier_repository.go

type replierRepository struct {
	sync.RWMutex
	repo map[UserID]Replier
	db   DBProvider
}

// replierRepository implements the ReplierRepository interface.
var _ ReplierRepository = (*replierRepository)(nil)

// NewReplierRepository creates a replier repository.
func NewReplierRepository(db DBProvider) ReplierRepository {
	return &replierRepository{
		repo: map[UserID]Replier{},
		db:   db,
	}
}

// ProvideReplier returns the relevant replier for the given user.
func (rp *replierRepository) ProvideReplier(uid UserID) Replier {
	rp.RLock()
	defer rp.RUnlock()

	if result := rp.repo[uid]; result != nil {
		return result
	}

	return NewCmdExecer(rp.db.ProvideDB(uid))
}

// SaveReplier saves the replier for coninuing the conversation.
func (rp *replierRepository) SaveReplier(uid UserID, r Replier) {
	rp.Lock()
	defer rp.Unlock()

	rp.repo[uid] = r
}

// DeleteReplier drops the conversation when it's over.
func (rp *replierRepository) DeleteReplier(uid UserID) {
	rp.Lock()
	defer rp.Unlock()

	delete(rp.repo, uid)
}

// prototype/repliers.go

// TODO: consider moving repliers to some core or something (for they have no logic, for all the logic is inside the cmd package)

// cmdExecer executes a Telegram command.
type cmdExecer struct {
	db DB
}

// cmdExecer implements the Replier interface.
var _ Replier = (*cmdExecer)(nil)

// NewCmdExecer creates a Telegram command executor.
func NewCmdExecer(db DB) Replier {
	return &cmdExecer{
		db: db,
	}
}

// Reply executes a Telegram command.
func (ce cmdExecer) Reply(u Update) (string, Replier) {
	if !u.IsCommand {
		return GetUsage(), nil
	}

	// TODO: register commands in a nice way in the cmd/ package and use them over here

	if u.Cmd == "listnotes" {
		result := ce.db.ListNotes(toTags(u.Args))
		if result == "" {
			result = "No notes satisfy the search criteria! :("
		}

		return result, nil
	}

	if u.Cmd == "createnote" {
		var next bodyExpector = func(txt string) {
			ce.db.CreateNote(txt, toTags(u.Args))
		}

		return "Please, enter the body of the new note!", &next
	}

	return GetUsage(), nil
}

// bodyExpector expects a new note body.
type bodyExpector func(string)

// bodyExecutor implements the Replier interface.
var _ Replier = (*bodyExpector)(nil)

// Reply add the new message to the registry and outputs a happy reply.
func (be bodyExpector) Reply(u Update) (string, Replier) {
	be(u.Text)

	return "Successfully added a new note! Hooray!", nil
}

// TODO: use pflags or something

func toTags(args []string) []string {
	// TODO: add normal validation and erroring

	if len(args) != 2 {
		return nil
	}

	if args[0] != "--tag" {
		return nil
	}

	return strings.Split(args[1], ",")
}

// cmd/cmd.go
// cmd/createnote.go
// cmd/listnotest.go

// TODO: clear all DB data at some point

// TODO: use cobra or something for commands

// CmdID is an ID of a Telegram command.
type CmdID string

// TODO: drop it in favor of registering per file

var Cmds []Cmd = []Cmd{
	{
		ID:    "createnote",
		Usage: "/createnote [--tag work,concentration]",
	},
	{
		ID:    "listnotes",
		Usage: "/listnotes [--tag work]",
	},
}

// Cmd describes a Telegram command.
type Cmd struct {
	ID    string
	Usage string
}

// GetUsage returns usage of all the Telegram commands.
func GetUsage() string {
	result := []string{}
	for _, cmd := range Cmds {
		result = append(result, cmd.Usage)
	}

	return fmt.Sprintf(`Run one of

%s

to let the magic happen!
`, strings.Join(result, "\n"))
}

// prototype/db_provider.go

// TODO: consider moving it to core or something (with the injected DB creator)

func NewDBProvider() DBProvider {
	return &dbProvider{
		repo: map[UserID]DB{},
	}
}

type dbProvider struct {
	sync.RWMutex
	repo map[UserID]DB
}

// dbProvider implements the DBProvider interface.
var _ DBProvider = (*dbProvider)(nil)

// ProvideDB returns a prototype DB for a given user.
func (dbp *dbProvider) ProvideDB(uid UserID) DB {
	if db := dbp.getDB(uid); db != nil {
		return db
	}

	dbp.Lock()
	defer dbp.Unlock()

	db := NewDB()

	dbp.repo[uid] = db

	return db
}

// getDB safely returns a DB from the provider.
func (dbp *dbProvider) getDB(uid UserID) DB {
	dbp.RLock()
	defer dbp.RUnlock()

	return dbp.repo[uid]
}

// prototype/db.go

// TODO: use some normal DB

// NewDB creates a new prototype DB.
func NewDB() DB {
	return &db{}
}

// db is a prototype db.
type db struct {
	repo []Entry
}

// Entry represents a registered note.
type Entry struct {
	Text string
	Tags []string
}

// db implements the DB interface.
var _ DB = (*db)(nil)

// CreateNote adds a note to a prototype DB.
func (db *db) CreateNote(txt string, tags []string) {
	db.repo = append(db.repo, Entry{
		Text: txt,
		Tags: tags,
	})
}

// ListNotes returns seleted notes for a prototype DB.
func (db *db) ListNotes(tags []string) string {
	result := []string{}
	for _, e := range db.repo {
		skip := false
		for _, tag := range tags {
			found := false
			for _, t := range e.Tags {
				if t == tag {
					found = true
					break
				}
			}

			if !found {
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		result = append(result, e.Text)
	}

	return strings.Join(result, "\n\n")
}

// main.go

func main() {
	// Creating a bot.
	bot, err := tgbotapi.NewBotAPI("TOKEN")
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Configuring the bot.
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Getting the update channel.
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	// Preparing the db and the replier provider.
	db := NewDBProvider()
	replierProvider := NewReplierRepository(db)

	// Accepting updates.
	for updateGlobal := range updates {
		// Enabling the parallel execution.
		go func() {
			// Capturing the update.
			update := updateGlobal
			
			// Skipping irrelevant input.
			if update.Message == nil {
				return
			}

			// Loggging debug info.
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			// Preparing the reply.
			uid := UserID(update.Message.From.ID)
			reply := replierProvider.ProvideReplier(uid)
			msg := update.Message
			u := Update{
				IsCommand: msg.IsCommand(),
				Cmd:       msg.Command(),
				Args:      strings.Split(msg.CommandArguments(), " "),
				Text:      msg.Text,
			}

			// Replying.
			txt, next := reply.Reply(u)
			if next == nil {
				replierProvider.DeleteReplier(uid)
			} else {
				replierProvider.SaveReplier(uid, next)
			}

			// Sending the reply.
			r := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			r.Text = txt
			bot.Send(r)
		}()
	}

	// TODO: exit gracefully
	// TODO: backup
	// TODO: restore
}
