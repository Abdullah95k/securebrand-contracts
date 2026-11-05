// Package conformance checks that events follow the unified contract's
// rules. Every collector runs it in CI on the events it produces (standard
// G5), and the platform's writer can run it before storing.
//
// It checks what the type system can't: ID formats, registered platform
// codes, envelope completeness, "absent is not zero", time precision,
// comment threading and the no-caps rule in production.
package conformance

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	contracts "github.com/abdullah95k/securebrand-contracts"
	contentv1 "github.com/abdullah95k/securebrand-contracts/gen/go/securebrand/content/v1"
	discoveryv1 "github.com/abdullah95k/securebrand-contracts/gen/go/securebrand/discovery/v1"
	mediav1 "github.com/abdullah95k/securebrand-contracts/gen/go/securebrand/media/v1"
)

// Registry is the parsed registry/platforms.yaml.
type Registry struct {
	Platforms []struct {
		Code   string `yaml:"code"`
		Status string `yaml:"status"`
	} `yaml:"platforms"`
	EntityKinds  []string            `yaml:"entity_kinds"`
	MessageKinds map[string][]string `yaml:"message_kinds"`
}

var (
	regOnce sync.Once
	reg     Registry
	regErr  error
	idRE    = regexp.MustCompile(`^([a-z]{1,4}):([a-z_]+):(.+)$`)
)

// Load returns the registry embedded in the contract.
func Load() (Registry, error) {
	regOnce.Do(func() { regErr = yaml.Unmarshal(contracts.PlatformsYAML, &reg) })
	return reg, regErr
}

// Problems collects every rule an event breaks.
type Problems []string

func (p *Problems) addf(format string, args ...any) { *p = append(*p, fmt.Sprintf(format, args...)) }

// Err returns nil when there are no problems.
func (p Problems) Err() error {
	if len(p) == 0 {
		return nil
	}
	return fmt.Errorf("contract: %s", strings.Join(p, "; "))
}

// Options change how strict the checks are.
type Options struct {
	// Production forbids test-only values (the CAPPED discovery outcome) and
	// reserved platform codes.
	Production bool
}

func (r Registry) platformStatus(code string) string {
	for _, p := range r.Platforms {
		if p.Code == code {
			return p.Status
		}
	}
	return ""
}

// checkID checks "<platform>:<kind>:<native id>" against the registry and
// returns the entity kind.
func checkID(p *Problems, r Registry, field, id, wantPlatform string, opt Options) string {
	m := idRE.FindStringSubmatch(id)
	if m == nil {
		p.addf("%s %q is not <platform>:<kind>:<native id>", field, id)
		return ""
	}
	platform, kind := m[1], m[2]
	switch st := r.platformStatus(platform); {
	case st == "":
		p.addf("%s: platform %q is not in the registry", field, platform)
	case st != "active" && opt.Production:
		p.addf("%s: platform %q is %s, not active", field, platform, st)
	}
	if wantPlatform != "" && platform != wantPlatform {
		p.addf("%s: platform %q differs from the envelope's %q", field, platform, wantPlatform)
	}
	known := false
	for _, k := range r.EntityKinds {
		if k == kind {
			known = true
		}
	}
	if !known {
		p.addf("%s: kind %q is not an entity kind (a subtype belongs in type or subtype)", field, kind)
	}
	return kind
}

// Envelope checks the envelope of a message of the given kind.
func Envelope(env *contentv1.Envelope, messageKind string, opt Options) Problems {
	var p Problems
	r, err := Load()
	if err != nil {
		p.addf("registry: %v", err)
		return p
	}
	if env == nil {
		p.addf("envelope missing")
		return p
	}
	if env.GetEventId() == "" {
		p.addf("envelope.event_id empty")
	}
	if env.GetKind() != messageKind {
		p.addf("envelope.kind %q, want %q", env.GetKind(), messageKind)
	}
	kind := checkID(&p, r, "envelope.id", env.GetId(), env.GetPlatform(), opt)
	if allowed, ok := r.MessageKinds[messageKind]; ok && kind != "" {
		match := false
		for _, k := range allowed {
			if k == kind {
				match = true
			}
		}
		if !match {
			p.addf("envelope.id kind %q not allowed for a %s (want %v)", kind, messageKind, allowed)
		}
	}
	if m := idRE.FindStringSubmatch(env.GetId()); m != nil && env.GetNativeId() != m[3] {
		p.addf("envelope.native_id %q differs from the ID's %q", env.GetNativeId(), m[3])
	}
	if env.GetObservedAt() == nil {
		p.addf("envelope.observed_at missing")
	}
	if env.GetVersion() <= 0 {
		p.addf("envelope.version must be positive")
	}
	pv := env.GetProvenance()
	if pv.GetCollector() == "" {
		p.addf("provenance.collector empty")
	}
	if pv.GetParserVersion() == "" && pv.GetAcquisition() == contentv1.Acquisition_ACQUISITION_SCRAPED_LOGGED_OUT {
		p.addf("provenance.parser_version empty on scraped data")
	}
	return p
}

func checkTime(p *Problems, field string, t *contentv1.Time) {
	if t == nil {
		return
	}
	if t.GetAt() != nil && t.GetPrecision() == contentv1.TimePrecision_TIME_PRECISION_UNSPECIFIED {
		p.addf("%s has a time but no precision", field)
	}
}

func checkCount(p *Problems, field string, c *contentv1.Count) {
	if c == nil {
		return
	}
	if c.Hidden && c.Value != nil {
		p.addf("%s is hidden but has a value", field)
	}
	if c.Value != nil && *c.Value < 0 {
		p.addf("%s is negative", field)
	}
	if c.GetExactness() == contentv1.CountExactness_COUNT_EXACTNESS_ROUNDED && c.Value == nil {
		p.addf("%s is rounded but has no value", field)
	}
}

func checkEngagement(p *Problems, field string, e *contentv1.Engagement) {
	if e == nil {
		return
	}
	for name, c := range map[string]*contentv1.Count{
		"views": e.Views, "likes": e.Likes, "comments": e.Comments, "replies": e.Replies,
		"shares": e.Shares, "saves": e.Saves, "reposts": e.Reposts, "quotes": e.Quotes,
		"concurrent_viewers": e.ConcurrentViewers,
	} {
		checkCount(p, field+"."+name, c)
	}
	for k, c := range e.Reactions {
		checkCount(p, field+".reactions["+k+"]", c)
	}
}

func checkRef(p *Problems, r Registry, field string, a *contentv1.AccountRef, opt Options) {
	if a == nil {
		return
	}
	if a.GetId() == "" && a.GetHandle() == "" && a.GetUrl() == "" && a.GetNativeId() == "" {
		p.addf("%s has no id, native id, handle or url", field)
	}
	if a.GetId() != "" {
		if k := checkID(p, r, field+".id", a.GetId(), "", opt); k != "" && k != "account" {
			p.addf("%s.id kind %q, want account", field, k)
		}
	}
}

// Account checks a content.account.v1 event.
func Account(m *contentv1.Account, opt Options) error {
	p := Envelope(m.GetEnvelope(), "account", opt)
	if !m.GetEnvelope().GetPartial() && m.GetDisplayName() == "" && m.GetHandle() == "" {
		p.addf("account has neither display_name nor handle")
	}
	checkTime(&p, "created_at", m.GetCreatedAt())
	return p.Err()
}

// Post checks a content.post.v1 event.
func Post(m *contentv1.Post, opt Options) error {
	r, _ := Load()
	env := m.GetEnvelope()
	p := Envelope(env, "post", opt)
	if k := checkID(&p, r, "account_id", m.GetAccountId(), env.GetPlatform(), opt); k != "" && k != "account" {
		p.addf("account_id kind %q, want account", k)
	}
	if !env.GetPartial() {
		if m.GetType() == contentv1.PostType_POST_TYPE_UNSPECIFIED {
			p.addf("post.type unspecified")
		}
		if m.GetPublishedAt().GetAt() == nil {
			p.addf("post.published_at missing")
		}
	}
	checkTime(&p, "published_at", m.GetPublishedAt())
	checkEngagement(&p, "counts", m.GetCounts())
	for i, a := range m.GetMentions() {
		checkRef(&p, r, fmt.Sprintf("mentions[%d]", i), a, opt)
	}
	return p.Err()
}

// Comment checks a content.comment.v1 event, including threading.
func Comment(m *contentv1.Comment, opt Options) error {
	r, _ := Load()
	env := m.GetEnvelope()
	p := Envelope(env, "comment", opt)
	if k := checkID(&p, r, "post_id", m.GetPostId(), env.GetPlatform(), opt); k != "" && k != "post" {
		p.addf("post_id kind %q, want post", k)
	}
	switch d := m.GetDepth(); {
	case d < 0:
		p.addf("depth is negative")
	case d == 0 && (m.GetParentCommentId() != "" || m.GetRootCommentId() != ""):
		p.addf("a top-level comment (depth 0) has a parent or root")
	case d > 0 && (m.GetParentCommentId() == "" || m.GetRootCommentId() == ""):
		p.addf("a reply (depth %d) needs both parent_comment_id and root_comment_id", d)
	case d == 1 && m.GetParentCommentId() != m.GetRootCommentId():
		p.addf("a depth-1 reply's parent must be its root")
	}
	if !env.GetPartial() && m.GetPublishedAt().GetAt() == nil {
		p.addf("comment.published_at missing")
	}
	checkTime(&p, "published_at", m.GetPublishedAt())
	if a := m.GetAuthor(); a != nil && !a.GetHidden() {
		checkRef(&p, r, "author.account", a.GetAccount(), opt)
	}
	checkEngagement(&p, "counts", m.GetCounts())
	return p.Err()
}

// Metrics checks a content.metrics.v1 event.
func Metrics(m *contentv1.Metrics, opt Options) error {
	p := Envelope(m.GetEnvelope(), "metrics", opt)
	if t := m.GetTargetKind(); t != "post" && t != "comment" {
		p.addf("target_kind %q, want post or comment", t)
	} else if mm := idRE.FindStringSubmatch(m.GetEnvelope().GetId()); mm != nil && mm[2] != t {
		p.addf("envelope.id kind %q differs from target_kind %q", mm[2], t)
	}
	if m.GetCounts() == nil {
		p.addf("metrics without counts")
	}
	checkEngagement(&p, "counts", m.GetCounts())
	return p.Err()
}

// AccountSnapshot checks a content.account_snapshot.v1 event.
func AccountSnapshot(m *contentv1.AccountSnapshot, opt Options) error {
	p := Envelope(m.GetEnvelope(), "account_snapshot", opt)
	if c := m.GetCounts(); c != nil {
		for name, v := range map[string]*contentv1.Count{"followers": c.Followers, "following": c.Following, "posts": c.Posts, "likes": c.Likes, "views": c.Views} {
			checkCount(&p, "counts."+name, v)
		}
	}
	return p.Err()
}

// Media checks a content.media.v1 event.
func Media(m *contentv1.Media, opt Options) error {
	r, _ := Load()
	env := m.GetEnvelope()
	p := Envelope(env, "media", opt)
	checkID(&p, r, "owner_id", m.GetOwnerId(), env.GetPlatform(), opt)
	if m.GetType() == contentv1.MediaType_MEDIA_TYPE_UNSPECIFIED {
		p.addf("media.type unspecified")
	}
	return p.Err()
}

// MediaReady checks a content.media.ready.v1 event.
func MediaReady(m *mediav1.MediaReady, opt Options) error {
	var p Problems
	r, _ := Load()
	if m.GetEventId() == "" {
		p.addf("event_id empty")
	}
	if k := checkID(&p, r, "media_id", m.GetMediaId(), m.GetPlatform(), opt); k != "" && k != "media" {
		p.addf("media_id kind %q, want media", k)
	}
	o := m.GetObject()
	if o.GetBucket() == "" || o.GetKey() == "" || o.GetSha256() == "" {
		p.addf("object needs bucket, key and sha256")
	}
	if m.GetDeleteAfter() == nil {
		p.addf("delete_after missing: every file has a deadline")
	}
	if len(m.GetRequiredConsumers()) == 0 {
		p.addf("required_consumers empty")
	}
	return p.Err()
}

// Candidate checks a discovery.candidate.v1 event.
func Candidate(m *discoveryv1.Candidate, opt Options) error {
	var p Problems
	r, _ := Load()
	if m.GetEventId() == "" {
		p.addf("event_id empty")
	}
	switch st := r.platformStatus(m.GetTargetPlatform()); {
	case st == "":
		p.addf("target_platform %q not in the registry", m.GetTargetPlatform())
	case st != "active" && opt.Production:
		p.addf("target_platform %q is %s", m.GetTargetPlatform(), st)
	}
	checkRef(&p, r, "account", m.GetAccount(), opt)
	if m.GetAccount() == nil {
		p.addf("account missing")
	}
	if m.GetFoundAt() == nil {
		p.addf("found_at missing")
	}
	if m.GetSource() == discoveryv1.Source_SOURCE_UNSPECIFIED {
		p.addf("source unspecified")
	}
	if opt.Production && m.GetOutcome() == discoveryv1.Outcome_OUTCOME_CAPPED {
		p.addf("outcome CAPPED is not allowed in production (owner D-008: no discovery caps)")
	}
	return p.Err()
}
