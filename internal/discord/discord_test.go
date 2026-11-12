package discord_test

import (
	"strings"
	"testing"
	"time"

	"github.com/arthurhmarinho/shellcord/internal/discord"
	"github.com/arthurhmarinho/shellcord/internal/discord/discordtest"
)

func TestSetActivityAndClear(t *testing.T) {
	srv := discordtest.New(t, "123")
	c, err := discord.Dial("123", []string{t.TempDir(), srv.Dir})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	err = c.SetActivity(&discord.Activity{Details: "Rodando nvim", Timestamps: &discord.Timestamps{Start: 100}})
	if err != nil {
		t.Fatal(err)
	}
	if got := srv.NextActivity(t, time.Second); !strings.Contains(got, `"details":"Rodando nvim"`) {
		t.Fatalf("atividade recebida: %s", got)
	}
	if err := c.SetActivity(nil); err != nil {
		t.Fatal(err)
	}
	if got := srv.NextActivity(t, time.Second); got != "null" {
		t.Fatalf("limpar deveria mandar activity nula, veio %s", got)
	}
}

func TestInvalidClientID(t *testing.T) {
	srv := discordtest.New(t, "123")
	_, err := discord.Dial("999", []string{srv.Dir})
	if err == nil || !strings.Contains(err.Error(), "Invalid Client ID") {
		t.Fatalf("quero erro de Client ID inválido, veio %v", err)
	}
}

func TestNoDiscord(t *testing.T) {
	if _, err := discord.Dial("123", []string{t.TempDir()}); err == nil {
		t.Fatal("sem socket deveria falhar")
	}
}

func TestPingPongAndClose(t *testing.T) {
	srv := discordtest.New(t, "123")
	c, err := discord.Dial("123", []string{srv.Dir})
	if err != nil {
		t.Fatal(err)
	}
	srv.Ping(t, `{"x":1}`)
	srv.Disconnect()
	select {
	case <-c.Done():
		if c.Err() == nil {
			t.Fatal("Done fechou sem motivo")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("o cliente não percebeu o fechamento")
	}
	if err := c.SetActivity(nil); err == nil {
		t.Fatal("SetActivity depois de fechar deveria falhar")
	}
}
