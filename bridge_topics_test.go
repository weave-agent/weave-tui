package tui

import tuibridge "github.com/weave-agent/weave-tui/internal/bridge"

const (
	topicPrompt           = tuibridge.TopicPrompt
	topicSteer            = tuibridge.TopicSteer
	topicFollowup         = tuibridge.TopicFollowup
	topicInterrupt        = tuibridge.TopicInterrupt
	topicSessionResume    = tuibridge.TopicSessionResume
	topicModelChange      = tuibridge.TopicModelChange
	topicThinkingChange   = tuibridge.TopicThinkingChange
	topicAuthLoginSuccess = tuibridge.TopicAuthLoginSuccess
)
