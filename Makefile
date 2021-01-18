notesWithTagsTelegramBot:
	go build -o notes-with-tags-telegram-bot main.go

all: dep notesWithTagsTelegramBot

clean:
	rm notes-with-tags-telegram-bot

.PHONY: clean 
