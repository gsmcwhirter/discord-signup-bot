package stats

import (
	"context"
	"fmt"
	"sort"
	"strings"

	bstats "github.com/gsmcwhirter/discord-bot-lib/v19/stats"
	"github.com/gsmcwhirter/go-util/v8/errors"
	"golang.org/x/sync/errgroup"
)

type Hub struct {
	stats map[string]*bstats.ActivityRecorder
}

var ErrDuplicate = errors.New("duplicate stat name")

func NewHub() *Hub {
	return &Hub{
		stats: map[string]*bstats.ActivityRecorder{},
	}
}

func (h *Hub) Get(name string) (*bstats.ActivityRecorder, bool) {
	ar, ok := h.stats[name]
	return ar, ok
}

func (h *Hub) Add(name string, ar *bstats.ActivityRecorder) error {
	if _, ok := h.stats[name]; ok {
		return ErrDuplicate
	}

	h.stats[name] = ar

	return nil
}

func (h *Hub) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, ar := range h.stats {
		ar := ar
		g.Go(func() error { return ar.Run(ctx) })
	}

	return g.Wait()
}

type rep struct {
	name           string
	avg, timescale float64
}

func (r rep) Format() string {
	return fmt.Sprintf("%s: %.2f / %.0fs", r.name, r.avg, r.timescale)
}

func (h *Hub) Report(indent string) string {
	reps := make([]rep, 0, len(h.stats))
	for name, ar := range h.stats {
		reps = append(reps, rep{
			name:      name,
			avg:       ar.Avg(),
			timescale: ar.Timescale(),
		})
	}

	fmts := make([]string, 0, len(reps))
	for _, r := range reps {
		fmts = append(fmts, r.Format())
	}

	sort.Strings(fmts)

	return indent + strings.Join(fmts, fmt.Sprintf("\n%s", indent))
}
