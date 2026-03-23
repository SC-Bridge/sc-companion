package tray

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/SC-Bridge/sc-companion/internal/events"
	"github.com/SC-Bridge/sc-companion/internal/store"
)

// Status holds the current companion app status for display.
type Status struct {
	PlayerHandle string
	CurrentShip  string
	Location     string
	Jurisdiction string
	EventCount   int
	Connected    bool
	LastEvent    time.Time
}

// Controller manages the system tray state and exposes status for UI.
// The actual systray integration is Windows-specific and will use
// platform-specific build tags. This controller is the shared logic.
type Controller struct {
	bus         *events.Bus
	store       *store.Store
	status      Status
	activeShips []string // ordered: most recently boarded last
	onQuit      func()
}

// NewController creates a tray controller.
func NewController(bus *events.Bus, s *store.Store, onQuit func()) *Controller {
	c := &Controller{
		bus:    bus,
		store:  s,
		onQuit: onQuit,
	}

	// Subscribe to events that update status
	bus.Subscribe(func(evt events.Event) {
		c.handleEvent(evt)
	})

	return c
}

// GetStatus returns the current companion status.
func (c *Controller) GetStatus() Status {
	return c.status
}

// StatusLine returns a one-line summary for the tray tooltip.
func (c *Controller) StatusLine() string {
	s := c.status
	if s.PlayerHandle == "" {
		return "SC Bridge Companion — waiting for game"
	}

	line := fmt.Sprintf("%s", s.PlayerHandle)
	if s.CurrentShip != "" {
		line += fmt.Sprintf(" | %s", s.CurrentShip)
	}
	if s.Location != "" {
		line += fmt.Sprintf(" | %s", s.Location)
	}
	line += fmt.Sprintf(" | %d events", s.EventCount)
	return line
}

func (c *Controller) handleEvent(evt events.Event) {
	c.status.EventCount++
	c.status.LastEvent = time.Now()

	switch evt.Type {
	case "player_login":
		c.status.PlayerHandle = evt.Data["handle"]
		slog.Info("player identified", "handle", c.status.PlayerHandle)

	case "ship_boarded":
		ship := evt.Data["ship"]
		// Add to active set if not already present
		found := false
		for _, s := range c.activeShips {
			if s == ship {
				found = true
				break
			}
		}
		if !found {
			c.activeShips = append(c.activeShips, ship)
		}
		c.status.CurrentShip = ship

	case "ship_exited":
		ship := evt.Data["ship"]
		// Remove from active set
		updated := c.activeShips[:0]
		for _, s := range c.activeShips {
			if s != ship {
				updated = append(updated, s)
			}
		}
		c.activeShips = updated
		// Current ship is the most recently boarded one still active
		if len(c.activeShips) > 0 {
			c.status.CurrentShip = c.activeShips[len(c.activeShips)-1]
		} else {
			c.status.CurrentShip = ""
		}

	case "location_change":
		c.status.Location = evt.Data["location"]

	case "jurisdiction_entered":
		c.status.Jurisdiction = evt.Data["jurisdiction"]
	}
}
