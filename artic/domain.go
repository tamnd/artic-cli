package artic

import (
	"context"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

// Domain is the artic driver. It carries no state; the per-run client is
// built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "artic",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "artic",
			Short:  "A command line for the Art Institute of Chicago.",
			Long: `A command line for the Art Institute of Chicago public API.

artic reads artwork data from api.artic.edu over HTTPS, shapes it into
clean records, and prints output that pipes into the rest of your tools.
No API key required. The collection spans 132,000+ artworks.`,
			Site: Host,
			Repo: "https://github.com/tamnd/artic-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "search", Group: "read", List: true,
		Summary: "Search the Art Institute of Chicago collection",
		Args:    []kit.Arg{{Name: "query", Help: "search query"}}}, searchArtworks)

	kit.Handle(app, kit.OpMeta{Name: "recent", Group: "read", List: true,
		Summary: "List recently published artworks"}, listArtworks)

	kit.Handle(app, kit.OpMeta{Name: "artwork", Group: "read", Single: true,
		Resolver: true, URIType: "artwork",
		Summary: "Fetch an artwork by ID",
		Args:    []kit.Arg{{Name: "id", Help: "artwork ID"}}}, getArtwork)
}

// newClient builds the Client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClientWithConfig(c), nil
}

// --- input structs ---

type searchInput struct {
	Query  string  `kit:"arg"          help:"search query"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Page   int     `kit:"flag"         help:"page number (1-based)"`
	Client *Client `kit:"inject"`
}

type listInput struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Page   int     `kit:"flag"         help:"page number (1-based)"`
	Client *Client `kit:"inject"`
}

type artworkRef struct {
	ID     string  `kit:"arg"    help:"artwork ID"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func searchArtworks(ctx context.Context, in searchInput, emit func(*Artwork) error) error {
	artworks, _, err := in.Client.SearchArtworks(ctx, in.Query, in.Limit, in.Page)
	if err != nil {
		return err
	}
	for _, a := range artworks {
		if err := emit(a); err != nil {
			return err
		}
	}
	return nil
}

func listArtworks(ctx context.Context, in listInput, emit func(*Artwork) error) error {
	artworks, _, err := in.Client.ListArtworks(ctx, in.Limit, in.Page)
	if err != nil {
		return err
	}
	for _, a := range artworks {
		if err := emit(a); err != nil {
			return err
		}
	}
	return nil
}

func getArtwork(ctx context.Context, in artworkRef, emit func(*Artwork) error) error {
	a, err := in.Client.GetArtwork(ctx, in.ID)
	if err != nil {
		return err
	}
	return emit(a)
}

// Classify turns any accepted input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	id = strings.TrimSpace(input)
	if id == "" {
		return "", "", errs.Usage("artic: empty reference")
	}
	return "artwork", id, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	if uriType != "artwork" {
		return "", errs.Usage("artic has no resource type %q", uriType)
	}
	return "https://www.artic.edu/artworks/" + id, nil
}
