package tui

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.mau.fi/util/jsontime"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/tui/config"
	"go.mau.fi/gomuks/tui/messages"
)

// WhatsApp's "mention all": the HTML carries each name, members aren't loaded.
func TestMentionsShowNamesNotIDs(t *testing.T) {
	_, room := bubbleView(t)
	content, _ := json.Marshal(map[string]any{
		"msgtype": "m.text",
		"body":    "Keshav Khatri Mansiii",
		"format":  "org.matrix.custom.html",
		"formatted_body": `<a href="https://matrix.to/#/@whyyor:beeper.com">Keshav Khatri</a> ` +
			`<a href="https://matrix.to/#/@whatsapp_lid-71726020960440:beeper.local">Mansiii</a>`,
	})
	evt := &database.Event{ID: "$m:x", RoomID: room.ID, Sender: "@whatsapp_lid-1:beeper.local",
		Type: "m.room.message", Content: content, Timestamp: jsontime.UM(time.Now())}
	prefs := config.UserPreferences{}
	msg := messages.ParseEvent(nil, &prefs, room, evt)
	text := msg.PlainText()
	if strings.Contains(text, "whatsapp_lid") || !strings.Contains(text, "Mansiii") || !strings.Contains(text, "Keshav Khatri") {
		t.Errorf("mentions should read as names: %q", text)
	}
}
