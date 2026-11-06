package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	contracts "github.com/abdullah95k/securebrand-contracts"
	analysisv1 "github.com/abdullah95k/securebrand-contracts/gen/go/securebrand/analysis/v1"
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
	"playlist":         {func() proto.Message { return &contentv1.Playlist{} }, func(m proto.Message, o Options) error { return Playlist(m.(*contentv1.Playlist), o) }},
	"item_analysis":    {func() proto.Message { return &analysisv1.ItemAnalysis{} }, func(m proto.Message, o Options) error { return ItemAnalysis(m.(*analysisv1.ItemAnalysis), o) }},
	"media_processed":  {func() proto.Message { return &mediav1.MediaProcessed{} }, func(m proto.Message, o Options) error { return MediaProcessed(m.(*mediav1.MediaProcessed), o) }},
	"media_deleted":    {func() proto.Message { return &mediav1.MediaDeleted{} }, func(m proto.Message, o Options) error { return MediaDeleted(m.(*mediav1.MediaDeleted), o) }},
	"media_analysis":   {func() proto.Message { return &analysisv1.MediaAnalysis{} }, func(m proto.Message, o Options) error { return MediaAnalysis(m.(*analysisv1.MediaAnalysis), o) }},
	"author_flag":      {func() proto.Message { return &analysisv1.AuthorFlag{} }, func(m proto.Message, o Options) error { return AuthorFlag(m.(*analysisv1.AuthorFlag), o) }},
	"topic_update":     {func() proto.Message { return &analysisv1.TopicUpdate{} }, func(m proto.Message, o Options) error { return TopicUpdate(m.(*analysisv1.TopicUpdate), o) }},
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
		{"day estimate not at Iraq midnight", "comment.yt.json", "isn't 00:00 Iraq time", func(m proto.Message) {
			m.(*contentv1.Comment).PublishedAt.At = timestamppb.New(time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC))
		}},
		{"acknowledgement by an unknown consumer", "media_processed.tt.json", "not in the registry's consumers", func(m proto.Message) {
			m.(*mediav1.MediaProcessed).Consumer = "someone-else"
		}},
		{"skipped without a reason", "media_processed.tt.json", "needs a reason code", func(m proto.Message) {
			x := m.(*mediav1.MediaProcessed)
			x.Status = mediav1.ProcessStatus_PROCESS_STATUS_SKIPPED
			x.Reason = ""
		}},
		{"deletion without a reason", "media_deleted.tt.json", "reason unspecified", func(m proto.Message) {
			m.(*mediav1.MediaDeleted).Reason = mediav1.DeleteReason_DELETE_REASON_UNSPECIFIED
		}},
		{"media analysis of a comment-owned file mislabelled", "media_analysis.tt.json", "differs from owner_id's kind", func(m proto.Message) {
			m.(*analysisv1.MediaAnalysis).OwnerKind = "comment"
		}},
		{"media analysis without provenance", "media_analysis.tt.json", "provenance needs", func(m proto.Message) {
			m.(*analysisv1.MediaAnalysis).Provenance = nil
		}},
		{"transcript without its language", "media_analysis.tt.json", "transcript needs", func(m proto.Message) {
			m.(*analysisv1.MediaAnalysis).TranscriptLang = ""
		}},
		{"unknown bot level", "author_flag.tt.json", "want empty, suspected", func(m proto.Message) {
			m.(*analysisv1.AuthorFlag).Bot = "maybe"
		}},
		{"author flag on a post ID", "author_flag.tt.json", "want account", func(m proto.Message) {
			m.(*analysisv1.AuthorFlag).AuthorId = "tt:post:7400000000000000001"
		}},
		{"sub-topic outside its parent", "topic_update.json", "doesn't start with its parent", func(m proto.Message) {
			m.(*analysisv1.TopicUpdate).ParentId = "services.banking"
		}},
		{"merged topic without a target", "topic_update.json", "needs merged_into", func(m proto.Message) {
			x := m.(*analysisv1.TopicUpdate)
			x.Status = "merged"
			x.MergedInto = ""
		}},
		{"keyword ID not in KeywordID's form", "candidate.yt-to-fb.json", "not contracts.KeywordID's form", func(m proto.Message) {
			c := m.(*discoveryv1.Candidate)
			c.Evidence = append(c.Evidence, &discoveryv1.Evidence{KeywordId: "Baghdad"})
		}},
		{"evidence time estimate not at Iraq midnight", "candidate.yt-to-fb.json", "isn't 00:00 Iraq time", func(m proto.Message) {
			c := m.(*discoveryv1.Candidate)
			c.Evidence = append(c.Evidence, &discoveryv1.Evidence{ContentTime: &contentv1.Time{
				At: timestamppb.New(time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)), Precision: contentv1.TimePrecision_TIME_PRECISION_DAY,
			}})
		}},
		{"hour estimate not at the hour", "comment.yt.json", "start of an hour", func(m proto.Message) {
			c := m.(*contentv1.Comment)
			c.PublishedAt.Precision = contentv1.TimePrecision_TIME_PRECISION_HOUR
			c.PublishedAt.At = timestamppb.New(time.Date(2026, 11, 6, 7, 15, 0, 0, time.UTC))
		}},
		{"count without an estimate mark", "metrics.fb.json", "no exactness", func(m proto.Message) {
			m.(*contentv1.Metrics).Counts.Reactions["love"].Exactness = contentv1.CountExactness_COUNT_EXACTNESS_UNSPECIFIED
		}},
		{"post account differs from account_id", "post.tt.json", "differs from account_id", func(m proto.Message) {
			m.(*contentv1.Post).Account.Id = "tt:account:7000000000000000999"
		}},
		{"media kept under 24 hours", "media_ready.tt.json", "less than 24 hours", func(m proto.Message) {
			r := m.(*mediav1.MediaReady)
			r.DeleteAfter = timestamppb.New(r.StoredAt.AsTime().Add(6 * time.Hour))
		}},
		{"unregistered media consumer", "media_ready.tt.json", "not in the registry's consumers", func(m proto.Message) {
			m.(*mediav1.MediaReady).RequiredConsumers = []string{"analysis"}
		}},
		{"media owner kind mismatch", "media_ready.tt.json", "differs from owner_id's kind", func(m proto.Message) {
			m.(*mediav1.MediaReady).OwnerKind = "playlist"
		}},
		{"playlist item that isn't a post", "playlist.yt.json", "want post", func(m proto.Message) {
			m.(*contentv1.Playlist).Items[0].PostId = "yt:comment:Ugzfake"
		}},
		{"analysis result without the analysed version", "item_analysis.fb.json", "content_version", func(m proto.Message) {
			m.(*analysisv1.ItemAnalysis).ContentVersion = 0
		}},
		{"analysis result kind mismatch", "item_analysis.fb.json", "differs from content_id's kind", func(m proto.Message) {
			m.(*analysisv1.ItemAnalysis).Kind = "post"
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

// A capped discovery outcome is fine outside production (tests and the
// canary).
func TestCapAllowedOutsideProduction(t *testing.T) {
	m, fn := load(t, "../samples/candidate.yt-to-fb.json")
	m.(*discoveryv1.Candidate).Outcome = discoveryv1.Outcome_OUTCOME_CAPPED
	if err := fn(m, Options{Production: false}); err != nil {
		t.Fatalf("tests and the canary should allow CAPPED: %v", err)
	}
}

// A collector's canary runs in production and may cap discovery (D-012).
func TestCapAllowedInACanary(t *testing.T) {
	m, fn := load(t, "../samples/candidate.yt-to-fb.json")
	c := m.(*discoveryv1.Candidate)
	c.Outcome = discoveryv1.Outcome_OUTCOME_CAPPED
	c.Source = discoveryv1.Source_SOURCE_LOCATION_PAGE
	c.Evidence = append(c.Evidence, &discoveryv1.Evidence{
		KeywordId:   contracts.KeywordID("baghdad", "ar", "بَغْدَاد"),
		ContentTime: &contentv1.Time{At: timestamppb.New(time.Date(2026, 11, 2, 21, 0, 0, 0, time.UTC)), Precision: contentv1.TimePrecision_TIME_PRECISION_DAY},
	})
	if err := fn(m, Options{Production: true, Canary: true}); err != nil {
		t.Fatalf("a canary should allow CAPPED: %v", err)
	}
}
