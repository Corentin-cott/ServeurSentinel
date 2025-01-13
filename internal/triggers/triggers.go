package triggers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/Corentin-cott/ServeurSentinel/config"
	"github.com/Corentin-cott/ServeurSentinel/internal/console"
)

// ----------------- Start of global variables -----------------
var (
	playerJoinedRegex       = regexp.MustCompile(`\[(\d{2}:\d{2}:\d{2})\] \[Server thread/INFO\]: (.+) joined the game`)
	playerDisconnectedRegex = regexp.MustCompile(`\[(\d{2}:\d{2}:\d{2})\] \[Server thread/INFO\]: (.+) lost connection: Disconnected`)
)

// ----------------- End of global variables -----------------

// ----------------- Start of Trigger structs -----------------
func GetTriggers() []console.Trigger {
	return []console.Trigger{
		{
			Condition: func(line string) bool {
				return strings.Contains(line, "joined the game")
			},
			Action: func(line string) {
				// On utilise une expression régulière pour récupérer le nom du joueur
				matches := playerJoinedRegex.FindStringSubmatch(line)
				if len(matches) < 3 {
					fmt.Println("Erreur lors de la récupération du nom du joueur")
					return
				}
				fmt.Println("Player joined: ", matches[2])
				SendToDiscord("Player joined: " + matches[2])
				WriteToLogFile("/var/log/serversentinel/playerjoined.log", matches[2])
			},
		},
		{
			Condition: func(line string) bool {
				return strings.Contains(line, "lost connection: Disconnected")
			},
			Action: func(line string) {
				// On utilise une expression régulière pour récupérer le nom du joueur
				matches := playerDisconnectedRegex.FindStringSubmatch(line)
				if len(matches) < 3 {
					fmt.Println("Erreur lors de la récupération du nom du joueur")
					return
				}
				fmt.Println("Player disconnected: ", matches[2])
				SendToDiscord("Player disconnected: " + matches[2])
				WriteToLogFile("/var/log/serversentinel/playerdisconnected.log", matches[2])
			},
		},
		{
			Condition: func(line string) bool {
				return strings.HasPrefix(line, "[ERROR]")
			},
			Action: func(line string) {
				fmt.Println("Error detected: ", line)
				fmt.Println("Sending error to the error log...")
				WriteToLogFile("/var/log/serversentinel/errors.log", line)
			},
		},
	}
}

// ----------------- End of triggers structs -----------------

// ----------------- Start of utils functions -----------------
// Écrit une ligne dans un fichier log
func WriteToLogFile(logPath, line string) error {
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("erreur d'ouverture du fichier log : %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(line + "\n")
	if err != nil {
		return fmt.Errorf("erreur d'écriture dans le fichier log : %v", err)
	}
	return nil
}

// Envoi un message à un serveur Discord
func SendToDiscord(message string) {
	config.LoadConfig("/opt/serversentinel/config.json")
	botToken := config.AppConfig.BotToken
	channelID := config.AppConfig.DiscordChannelID

	switch {
	case botToken == "" && channelID == "":
		fmt.Println("Bot token and channel ID not set. Skipping Discord message.")
		return
	case botToken == "":
		fmt.Println("Bot token not set. Skipping Discord message.")
		return
	case channelID == "":
		fmt.Println("Channel ID not set. Skipping Discord message.")
		return
	}

	apiURL := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", channelID)

	type DiscordBotMessage struct {
		Content string `json:"content"`
	}

	payload := DiscordBotMessage{Content: message}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Erreur lors de la sérialisation du message Discord : %v\n", err)
		return
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("Erreur lors de la création de la requête : %v\n", err)
		return
	}

	req.Header.Set("Authorization", "Bot "+botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Erreur lors de l'envoi à Discord : %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		fmt.Printf("Erreur lors de l'envoi à Discord : Status %d\n", resp.StatusCode)
	} else {
		fmt.Println("Message envoyé à Discord avec succès.")
	}
}
