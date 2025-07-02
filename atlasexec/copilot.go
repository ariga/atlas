// Copyright 2021-present The Atlas Authors. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package atlasexec

import (
	"context"
	"encoding/json"
	"strings"
)

// CopilotTypeMessage, CopilotTypeToolCall, and CopilotTypeToolOutput are the
// types of messages that can be emitted by the Copilot execution.
const (
	CopilotTypeMessage    = "message"
	CopilotTypeToolCall   = "tool_call"
	CopilotTypeToolOutput = "tool_output"
)

type (
	// CopilotParams are the parameters for the Copilot execution.
	CopilotParams struct {
		Prompt, Session string
		// FSWrite and FSDelete glob patterns to specify file permissions.
		FSWrite, FSDelete string
	}
	// Copilot is the result of a Copilot execution.
	Copilot []*CopilotMessage
	// CopilotMessage is the JSON message emitted by the Copilot OneShot execution.
	CopilotMessage struct {
		// Session ID for the Copilot session.
		SessionID string `json:"sessionID,omitempty"`

		// Type of the message. Can be "message", "tool_call", or "tool_output".
		Type string `json:"type"`

		// Content, ToolCall and ToolOutput are mutually exclusive.
		Content    string             `json:"content,omitempty"`
		ToolCall   *ToolCallMessage   `json:"toolCall,omitempty"`
		ToolOutput *ToolOutputMessage `json:"toolOutput,omitempty"`
	}
	// ToolCallMessage is the input to a tool call.
	ToolCallMessage struct {
		CallID    string `json:"callID"`
		Function  string `json:"function"`
		Arguments string `json:"arguments"`
	}
	// ToolOutputMessage is the output of a tool call.
	ToolOutputMessage struct {
		CallID  string `json:"callID"`
		Content string `json:"content"`
	}
)

// Copilot executes a one-shot Copilot session with the provided options.
func (c *Client) Copilot(ctx context.Context, params *CopilotParams) (Copilot, error) {
	args := []string{"copilot", "-q", params.Prompt}
	if params.Session != "" {
		args = append(args, "-r", params.Session)
	}
	if params.FSWrite != "" {
		args = append(args, "-p", "fs.write="+params.FSWrite)
	}
	if params.FSDelete != "" {
		args = append(args, "-p", "fs.delete="+params.FSDelete)
	}
	return jsonDecode[CopilotMessage](c.runCommand(ctx, args))
}

type copilotStream struct {
	s   Stream[string]
	cur *CopilotMessage
	err error
}

// Next advances the stream to the next CopilotMessage.
func (s *copilotStream) Next() bool {
	s.cur = nil
	s.err = nil
	return s.s.Next()
}

// Current returns the current CopilotMessage from the stream.
func (s *copilotStream) Current() (*CopilotMessage, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.cur == nil {
		cur, err := s.s.Current()
		if err != nil {
			s.err = err
			return nil, err
		}
		var m CopilotMessage
		if s.err = json.Unmarshal([]byte(cur), &m); s.err != nil {
			return nil, s.err
		}
		s.cur = &m
	}
	return s.cur, nil
}

// Err returns the error encountered during the stream processing.
func (s *copilotStream) Err() error {
	if s.err != nil {
		return s.err
	}
	return s.s.Err()
}

var _ Stream[*CopilotMessage] = (*copilotStream)(nil)

// CopilotStream executes a one-shot Copilot session, streaming the result.
func (c *Client) CopilotStream(ctx context.Context, params *CopilotParams) (Stream[*CopilotMessage], error) {
	args := []string{"copilot", "-q", params.Prompt}
	if params.Session != "" {
		args = append(args, "-r", params.Session)
	}
	if params.FSWrite != "" {
		args = append(args, "-p", "fs.write="+params.FSWrite)
	}
	if params.FSDelete != "" {
		args = append(args, "-p", "fs.delete="+params.FSDelete)
	}
	s, err := c.runCommandStream(ctx, args)
	if err != nil {
		return nil, err
	}
	return &copilotStream{s: s}, nil
}

func (c Copilot) String() string {
	var buf strings.Builder
	for _, msg := range c {
		if msg.Type == CopilotTypeMessage {
			buf.WriteString(msg.Content)
		}
	}
	return buf.String()
}
