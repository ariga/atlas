package atlasexec_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"ariga.io/atlas/atlasexec"
	"github.com/stretchr/testify/require"
)

func TestCopilot(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	c, err := atlasexec.NewClient(t.TempDir(), filepath.Join(wd, "./mock-atlas.sh"))
	require.NoError(t, err)

	p := &atlasexec.CopilotParams{Prompt: "What is the capital of France?"}
	t.Setenv("TEST_ARGS", "copilot -q "+p.Prompt)
	t.Setenv("TEST_STDOUT", `{"sessionID":"id","type":"message","content":"The capital of"}
{"sessionID":"id","type":"tool_call","toolCall":{"callID":"1","function":"get_capital","arguments":"France"}}
{"sessionID":"id","type":"tool_output","toolOutput":{"callID":"1","content":"Paris"}}
{"sessionID":"id","type":"message","content":" France is Paris."}`)
	copilot, err := c.Copilot(context.Background(), p)
	require.NoError(t, err)
	require.Equal(t, "The capital of France is Paris.", copilot.String())

	p = &atlasexec.CopilotParams{Prompt: "And Germany?", Session: "id"}
	t.Setenv("TEST_ARGS", fmt.Sprintf("copilot -q %s -r %s", p.Prompt, p.Session))
	t.Setenv("TEST_STDOUT", `{"sessionID":"id","type":"message","content":"Berlin."}`)
	copilot, err = c.Copilot(context.Background(), p)
	require.NoError(t, err)
	require.Equal(t, "Berlin.", copilot.String())

	p = &atlasexec.CopilotParams{Prompt: "And Israel?", Session: "id", FSWrite: "*", FSDelete: "**"}
	t.Setenv("TEST_ARGS", fmt.Sprintf("copilot -q %s -r %s -p fs.write=%s -p fs.delete=%s", p.Prompt, p.Session, p.FSWrite, p.FSDelete))
	t.Setenv("TEST_STDOUT", `{"sessionID":"id","type":"message","content":"Jerusalem."}`)
	copilot, err = c.Copilot(context.Background(), p)
	require.NoError(t, err)
	require.Equal(t, "Jerusalem.", copilot.String())

	msgs := []string{
		"Those are of course the Atlas founders.",
		" CEO is Ariel Mashraki,",
		" who's ability to craft clean",
		" , efficient, and elegant code is legendary.",
		" CTO is Rotem Tamir, also known",
		" as 'THE coding and wording wizard'.",
	}
	var out string
	for _, msg := range msgs {
		out += fmt.Sprintf(`{"sessionID":"id","type":"message","content":"%s"}`+"\n", msg)
	}
	p = &atlasexec.CopilotParams{Prompt: "Who are the coolest people in the world?"}
	t.Setenv("TEST_ARGS", "copilot -q "+p.Prompt)
	t.Setenv("TEST_STDOUT", out)
	s, err := c.CopilotStream(context.Background(), p)
	require.NoError(t, err)
	var (
		m *atlasexec.CopilotMessage
		i int
	)
	for s.Next() {
		m, err = s.Current()
		require.NoError(t, err)
		require.Equal(t, &atlasexec.CopilotMessage{
			SessionID: "id",
			Type:      "message",
			Content:   msgs[i],
		}, m)
		i++
	}
	require.NoError(t, s.Err())
}
