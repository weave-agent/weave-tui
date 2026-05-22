package tui

import (
	"fmt"

	"github.com/weave-agent/weave/sdk"
	sdkmodel "github.com/weave-agent/weave/sdk/model"

	tuiauth "github.com/weave-agent/weave-tui/internal/auth"
	tuibridge "github.com/weave-agent/weave-tui/internal/bridge"
	"github.com/weave-agent/weave-tui/internal/components/messages"
	tuievents "github.com/weave-agent/weave-tui/internal/events"
	tuiproviders "github.com/weave-agent/weave-tui/internal/providers"

	tea "charm.land/bubbletea/v2"
)

func (m Model) onLoginFlowResult(msg tuievents.LoginFlowResultMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		am := messages.NewAssistantMessage()
		am.Finalize(fmt.Sprintf("OAuth login failed for %s: %v", tuiauth.DisplayNameForProvider(msg.Provider), msg.Error))
		m.chat = m.chat.AddItem(am)

		return m, nil
	}

	if msg.Credential.AccessToken == "" {
		am := messages.NewAssistantMessage()
		am.Finalize(fmt.Sprintf("OAuth login failed for %s: received empty access token", tuiauth.DisplayNameForProvider(msg.Provider)))
		m.chat = m.chat.AddItem(am)

		return m, nil
	}

	if err := sdk.SetOAuthCredential(msg.Provider, msg.Credential); err != nil {
		am := messages.NewAssistantMessage()
		am.Finalize(fmt.Sprintf("Failed to save OAuth credentials for %s: %v", tuiauth.DisplayNameForProvider(msg.Provider), err))
		m.chat = m.chat.AddItem(am)

		return m, nil
	}

	// Update in-memory auth status so the provider is immediately usable.
	sdkmodel.SetProviderAuth(msg.Provider, true)

	am := messages.NewAssistantMessage()
	am.Finalize(fmt.Sprintf("Successfully logged in to %s.", tuiauth.DisplayNameForProvider(msg.Provider)))
	m.chat = m.chat.AddItem(am)

	// If we were in noConfigured state, re-evaluate now that auth exists.
	if m.noConfigured {
		models := tuiproviders.ListModels()
		if len(models) > 0 {
			m.noConfigured = false
			cur := tuiproviders.CurrentModel(models, m.ps)
			m.currentModel = cur
			m.footer = m.footer.SetModel(cur.Model, cur.Provider)
			m.footer = m.footer.SetReasoning(tuiproviders.ModelReasoning(cur))
		}
	}

	var cmds []tea.Cmd

	if m.bus != nil {
		cmds = append(cmds, tuibridge.PublishAuthLoginSuccess(m.bus, msg.Provider))

		// If we transitioned out of noConfigured, publish model.change so the
		// agent loop switches to the newly available provider.
		if !m.noConfigured {
			cmds = append(cmds, tuibridge.PublishModelChange(m.bus, m.currentModel))
		}
	}

	return m, tea.Batch(cmds...)
}
