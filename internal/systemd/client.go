package systemd

import (
	"context"
	"strings"

	sysdbus "github.com/coreos/go-systemd/v22/dbus"

	"laxy-services/internal/models"
)

type Client struct {
	conn *sysdbus.Conn
}

func NewClient() (*Client, error) {
	conn, err := sysdbus.NewSystemConnectionContext(context.Background())
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

func (c *Client) Close() { c.conn.Close() }

func (c *Client) ListServices() ([]models.Service, error) {
	units, err := c.conn.ListUnitsContext(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.Service, 0, len(units))
	for _, u := range units {
		if strings.HasSuffix(u.Name, ".service") {
			out = append(out, models.Service{
				Name: u.Name, Description: u.Description,
				LoadState: u.LoadState, ActiveState: u.ActiveState, SubState: u.SubState,
			})
		}
	}
	return out, nil
}

type connActionFn func(context.Context, string, string, chan<- string) (int, error)

func (c *Client) execUnit(fn connActionFn, name string) error {
	ch := make(chan string, 1)
	_, err := fn(context.Background(), name, "replace", ch)
	return err
}

func (c *Client) StartUnit(name string) error   { return c.execUnit(c.conn.StartUnitContext, name) }
func (c *Client) StopUnit(name string) error    { return c.execUnit(c.conn.StopUnitContext, name) }
func (c *Client) RestartUnit(name string) error { return c.execUnit(c.conn.RestartUnitContext, name) }
