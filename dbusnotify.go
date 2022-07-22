package dbusnotify

import (
	"fmt"

	dbus "github.com/godbus/dbus/v5"
)

const (
	objectPath = "/org/freedesktop/Notifications"
	dest       = "org.freedesktop.Notifications"

	methodNotify = "org.freedesktop.Notifications.Notify"
)

type Messenger interface {
	SendInfo(AppName, Subject, Message string) error
	SendError(AppName, Subject, Message string) error
	SendWarning(AppName, Subject, Message string) error

	SendNotification(Notification) error
}

type Action struct {
	Label string
	Value string
}

func (a Action) String() string {
	return fmt.Sprintf("%s,%s", a.Label, a.Value)
}

type Hint struct {
	Name  string
	Value interface{}
}

func (h Hint) ToVariant() dbus.Variant {
	return dbus.MakeVariantWithSignature(h.Value, dbus.SignatureOf(h.Value))
}

type urgency byte

var (
	UrgencyLow    urgency = 0
	UrgencyMedium urgency = 1
	UrgencyHigh   urgency = 2
)

type Notification struct {
	AppName    string
	ReplacesID uint32
	AppIcon    string
	Summary    string
	Body       string
	// Progress overrides info in Hints
	Progress *int
	// Urgency overrides info in Hints
	Urgency urgency
	Actions []Action
	Hints   []Hint
	Sticky  bool
}

type service struct {
	bus *dbus.Conn
}

func NewService() (*service, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("dbusnotify: could not create a connection to the bus: %w", err)
	}
	return &service{bus: conn}, nil
}

func (s *service) SendNotification(n Notification) error {
	actions := []string{}
	for _, a := range n.Actions {
		actions = append(actions, a.Label, a.Value)
	}

	hints := map[string]dbus.Variant{}
	for _, h := range n.Hints {
		hints[h.Name] = h.ToVariant()
	}
	if n.Progress != nil {
		hints["value"] = dbus.MakeVariant(n.Progress)
	}

	hints["urgency"] = dbus.MakeVariant(n.Urgency)

	sticky := -1
	if n.Sticky == true {
		sticky = 0
	}

	obj := s.bus.Object(dest, objectPath)
	flags := dbus.Flags(0)
	call := obj.Call(
		methodNotify,
		flags,
		n.AppName,
		n.ReplacesID,
		n.AppIcon,
		n.Summary,
		n.Body,
		actions,
		hints,
		sticky,
	)

	if call.Err != nil {
		return fmt.Errorf("dbusnotify: could not send message: %w", call.Err)
	}

	return nil
}

func (s *service) SendError(program, subject, body string) error {
	return s.SendNotification(Notification{
		AppName: program,
		Summary: subject,
		Body:    body,
		Urgency: UrgencyHigh,
		Sticky:  true,
	})
}

func (s *service) SendWarning(program, subject, body string) error {
	return s.SendNotification(Notification{
		AppName: program,
		Summary: subject,
		Body:    body,
		Urgency: UrgencyMedium,
		Sticky:  false,
	})
}

func (s *service) SendInfo(program, subject, body string) error {
	return s.SendNotification(Notification{
		AppName: program,
		Summary: subject,
		Body:    body,
		Sticky:  false,
	})
}
