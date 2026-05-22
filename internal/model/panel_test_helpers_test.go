package model

import (
	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

type mockPanelDrawer struct {
	updateCount int
	drawCount   int
	lastArea    uv.Rectangle
	lastMsg     tea.Msg
}

func (m *mockPanelDrawer) Draw(_ uv.Screen, area uv.Rectangle) {
	m.drawCount++
	m.lastArea = area
}

func (m *mockPanelDrawer) Update(msg tea.Msg) (PanelDrawer, tea.Cmd) {
	m.updateCount++
	m.lastMsg = msg

	return m, nil
}

func (m *mockPanelDrawer) Handles(_ tea.Msg) bool { return true }

type selectivePanelDrawer struct {
	mockPanelDrawer
}

func (m *selectivePanelDrawer) Handles(msg tea.Msg) bool {
	key, ok := msg.(tea.KeyPressMsg)
	return ok && key.Code == 'h'
}
