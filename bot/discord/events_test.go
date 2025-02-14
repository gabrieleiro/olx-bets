package discord

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestParseGuess(t *testing.T) {
	type ParseGuessTest struct {
		Input       string
		Expected    int
		ExpectedErr bool
		AssertMsg   string
	}

	tests := []ParseGuessTest{
		{"oi", 0, true, "NaN as argument"},
		{"20", 20, false, "Parsing plain numbers"},
		{"20k", 20_000, false, "Expanding k-ending numbers"},
		{"20k500", 20_500, false, "Expanding k-in-the-middle"},
		{"20k50", 20_050, false, "Expanding k-in-the-middle"},
		{"20k5", 20_005, false, "Expanding k-in-the-middle"},
		{"10kk", 0, true, "Double k"},
		{"10k8k", 0, true, "Double k"},
		{"  300 ", 300, false, "Spaces"},
		{"R$1400", 1400, false, "R$"},
		{"r$1400", 1400, false, "R$"},
		{"$1400", 1400, false, "$"},
		{" $1400", 1400, false, "$"},
		{"100 reais", 100, false, "reais"},
		{"100 Reais", 100, false, "Reais"},
		{"100 reais  ", 100, false, "reais"},
		{"100 re  ", 0, true, "reais"},
		{"100 re", 0, true, "reais"},
		{"-100", 0, true, "negative guess"},
		{"9999999999999999", 0, true, "long input"},
	}

	for _, current := range tests {
		msg := discordgo.MessageCreate{
			Message: &discordgo.Message{Content: current.Input},
		}

		res, err := ParseGuess(&msg)

		if res != current.Expected {
			t.Fatalf("%s\nWant: %d\nGot: %d\nInput: %s", current.AssertMsg, current.Expected, res, current.Input)
		}

		if err == nil && current.ExpectedErr {
			t.Fatalf("%s\nInput: %s", current.AssertMsg, current.Input)
		}
	}
}
