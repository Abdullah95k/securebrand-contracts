package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	contentv1 "github.com/abdullah95k/securebrand-contracts/gen/go/securebrand/content/v1"
	discoveryv1 "github.com/abdullah95k/securebrand-contracts/gen/go/securebrand/discovery/v1"
	mediav1 "github.com/abdullah95k/securebrand-contracts/gen/go/securebrand/media/v1"
)

// check maps a sample's file-name prefix to its message type and checker.
var check = map[string]struct {
	new   func() proto.Message
	check func(proto.Message, Options) error
}{
	"post":             {func() proto.Message { return &contentv1.Post{} }, func(m proto.Message, o Options) error { return Post(m.(*contentv1.Post), o) }},
	"comment":          {func() proto.Message { return &contentv1.Comment{} }, func(m proto.Message, o Options) error { return Comment(m.(*contentv1.Comment), o) }},
	"account":          {func() proto.Message { return &contentv1.Account{} }, func(m proto.Message, o Options) error { return Account(m.(*contentv1.Account), o) }},
	"account_snapshot": {func() proto.Message { return &contentv1.AccountSnapshot{} }, func(m proto.Message, o Options) error { return AccountSnapshot(m.(*contentv1.AccountSnapshot), o) }},
	"metrics":          {func() proto.Message { return &contentv1.Metrics{} }, func(m proto.Message, o Options) error { return Metrics(m.(*contentv1.Metrics), o) }},
	"media":            {func() proto.Message { return &contentv1.Media{} }, func(m proto.Message, o Options) error { return Media(m.(*contentv1.Media), o) }},
	"media_ready":      {func() proto.Message { return &mediav1.MediaReady{} }, func(m proto.Message, o Options) error { return MediaReady(m.(*mediav1.MediaReady), o) }},
	"candidate":        {func() proto.Message { return &discoveryv1.Candidate{} }, func(m proto.Message, o Options) error { return Candidate(m.(*discoveryv1.Candidate), o) }},
}

func load(t *testing.T, path string) (proto.Message, func(proto.Message, Options) error) {
	t.Helper()
	prefix := strings.SplitN(filepath.Base(path), ".", 2)[0]
	c, ok := check[prefix]
	if !ok {
		t.Fatalf("%s: no checker for prefix %q", path, prefix)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m := c.new()
	// Unknown fields are an error: a sample must use only contract fields.
	if err := (protojson.UnmarshalOptions{}).Unmarshal(b, m); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return m, c.check
}

// Every sample parses into its type and passes the production checks.
func TestSamplesConform(t *testing.T) {
	files, err := filepath.Glob("../samples/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no samples: %v", err)
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			m, fn := load(t, f)
			if err := fn(m, Options{Production: true}); err != nil {
				t.Errorf("%v", err)
			}
		})
	}
}

// Each broken variant of a sample must be rejected.
func TestRulesCatchMistakes(t *testing.T) {
	cases := []struct {
		name, sample, want string
		mutate             func(proto.Message)
	}{
		{"subtype in the ID", "post.tt.json", "not an entity kind", func(m proto.Message) {
			m.(*contentv1.Post).Envelope.Id = "tt:reel:7400000000000000001"
		}},
		{"unregistered platform", "post.tt.json", "not in the registry", func(m proto.Message) {
			p := m.(*contentv1.Post)
			p.Envelope.Platform, p.Envelope.Id, p.AccountId = "zz", "zz:post:1", "zz:account:1"
		}},
		{"reserved platform in production", "post.tt.json", "reserved, not active", func(m proto.Message) {
			p := m.(*contentv1.Post)
			p.Envelope.Platform, p.Envelope.Id, p.AccountId = "x", "x:post:7400000000000000001", "x:account:1"
		}},
		{"time without precision", "post.tt.json", "no precision", func(m proto.Message) {
			m.(*contentv1.Post).PublishedAt.Precision = contentv1.TimePrecision_TIME_PRECISION_UNSPECIFIED
		}},
		{"hidden count with a value", "post.tt.json", "hidden but has a value", func(m proto.Message) {
			m.(*contentv1.Post).Counts.Likes.Hidden = true
		}},
		{"zero version", "post.tt.json", "version must be positive", func(m proto.Message) {
			m.(*contentv1.Post).Envelope.Version = 0
		}},
		{"reply without root", "comment.yt.json", "needs both", func(m proto.Message) {
			m.(*contentv1.Comment).RootCommentId = ""
		}},
		{"top-level comment with a parent", "comment.fb.deleted.json", "has a parent or root", func(m proto.Message) {
			m.(*contentv1.Comment).ParentCommentId = "fb:comment:1"
		}},
		{"metrics kind mismatch", "metrics.fb.json", "differs from target_kind", func(m proto.Message) {
			m.(*contentv1.Metrics).TargetKind = "comment"
		}},
		{"media without a deadline", "media_ready.tt.json", "delete_after missing", func(m proto.Message) {
			m.(*mediav1.MediaReady).DeleteAfter = nil
		}},
		{"discovery cap in production", "candidate.yt-to-fb.json", "not allowed in production", func(m proto.Message) {
			m.(*discoveryv1.Candidate).Outcome = discoveryv1.Outcome_OUTCOME_CAPPED
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, fn := load(t, filepath.Join("..", "samples", tc.sample))
			tc.mutate(m)
			err := fn(m, Options{Production: true})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want an error containing %q, got %v", tc.want, err)
			}
		})
	}
}

// A capped discovery outcome is fine outside production (tests, staging,
// the canary).
func TestCapAllowedOutsideProduction(t *testing.T) {
	m, fn := load(t, "../samples/candidate.yt-to-fb.json")
	m.(*discoveryv1.Candidate).Outcome = discoveryv1.Outcome_OUTCOME_CAPPED
	if err := fn(m, Options{Production: false}); err != nil {
		t.Fatalf("staging should allow CAPPED: %v", err)
	}
}
