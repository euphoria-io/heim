package console

import "euphoria.io/scope"

func init() {
	register("peers", peers{})
}

type peers struct{}

func (peers) run(ctx scope.Context, c *console, args []string) error {
	for i, peer := range c.backend.Peers() {
		c.Printf("%d. %s: version=%s, era=%s\n", i+1, peer.ID, peer.Version, peer.Era)
	}
	return nil
}
