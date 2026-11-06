// Package conformance checks that events follow the unified contract's
// rules. Every collector runs it in CI on the events it produces (standard
// G5), and the platform's writer can run it before storing.
//
// It checks what the type system can't: ID formats, registered platform
// codes, envelope completeness, "absent is not zero", estimates marked as
// estimates (owner D-013), comment threading, the media hand-off's deadline
// and consumers, analysis results, and the no-caps rule in production.
package conformance

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	contracts "github.com/abdullah95k/securebrand-contracts"
	analysisv1 "github.com/abdullah95k/securebrand-contracts/gen/go/securebrand/analysis/v1"
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
	EntityKinds     []string            `yaml:"entity_kinds"`
	MessageKinds    map[string][]string `yaml:"message_kinds"`
	MediaOwnerKinds []string            `yaml:"media_owner_kinds"`
	Consumers       []struct {
		Name string `yaml:"name"`
	} `yaml:"consumers"`
}

var (
	regOnce sync.Once
	reg     Registry
	regErr  error
	idRE    = regexp.MustCompile(`^([a-z]{1,4}):([a-z_]+):(.+)$`)
	// Iraq time (Asia/Baghdad), UTC+3 with no daylight saving. A fixed zone,
	// so the check doesn't depend on the host's time-zone data.
	iraq = time.FixedZone("Asia/Baghdad", 3*60*60)
)

// minMediaLifetime is the shortest time a media file is kept (C14).
const minMediaLifetime = 24 * time.Hour

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
	// Canary allows the CAPPED discovery outcome in production while a
	// collector's canary runs (owner D-012). It changes nothing else.
	Canary bool
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

// checkTime checks a Time's precision, and that an estimate sits at the
// start of its unit (owner D-013): minutes and hours at the start of the
// minute or hour; days, weeks, months and years at 00:00 Iraq time.
func checkTime(p *Problems, field string, t *contentv1.Time) {
	if t == nil || t.GetAt() == nil {
		return
	}
	prec := t.GetPrecision()
	if prec == contentv1.TimePrecision_TIME_PRECISION_UNSPECIFIED {
		p.addf("%s has a time but no precision", field)
		return
	}
	at := t.GetAt().AsTime()
	switch prec {
	case contentv1.TimePrecision_TIME_PRECISION_MINUTE:
		if at.Second() != 0 || at.Nanosecond() != 0 {
			p.addf("%s has minute precision but isn't at the start of a minute (D-013)", field)
		}
	case contentv1.TimePrecision_TIME_PRECISION_HOUR:
		if at.Minute() != 0 || at.Second() != 0 || at.Nanosecond() != 0 {
			p.addf("%s has hour precision but isn't at the start of an hour (D-013)", field)
		}
	case contentv1.TimePrecision_TIME_PRECISION_DAY, contentv1.TimePrecision_TIME_PRECISION_WEEK,
		contentv1.TimePrecision_TIME_PRECISION_MONTH, contentv1.TimePrecision_TIME_PRECISION_YEAR:
		local := at.In(iraq)
		if local.Hour() != 0 || local.Minute() != 0 || local.Second() != 0 || local.Nanosecond() != 0 {
			p.addf("%s has %s precision but isn't 00:00 Iraq time, 21:00 UTC (D-013)", field,
				strings.TrimPrefix(prec.String(), "TIME_PRECISION_"))
		}
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
	if c.Value != nil && c.GetExactness() == contentv1.CountExactness_COUNT_EXACTNESS_UNSPECIFIED {
		p.addf("%s has a value but no exactness: say exact, rounded or lower bound (D-013)", field)
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
	for k, c := range e.Other {
		checkCount(p, field+".other["+k+"]", c)
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
	checkTime(&p, "handle_changed_at", m.GetHandleChangedAt())
	checkTime(&p, "name_changed_at", m.GetNameChangedAt())
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
	checkTime(&p, "video.scheduled_start", m.GetVideo().GetScheduledStart())
	checkTime(&p, "video.live_started_at", m.GetVideo().GetLiveStartedAt())
	checkTime(&p, "video.live_ended_at", m.GetVideo().GetLiveEndedAt())
	checkTime(&p, "flags.edited_at", m.GetFlags().GetEditedAt())
	checkEngagement(&p, "counts", m.GetCounts())
	for i, a := range m.GetMentions() {
		checkRef(&p, r, fmt.Sprintf("mentions[%d]", i), a, opt)
	}
	if a := m.GetAccount(); a != nil {
		checkRef(&p, r, "account", a, opt)
		if a.GetId() != "" && a.GetId() != m.GetAccountId() {
			p.addf("account.id %q differs from account_id %q", a.GetId(), m.GetAccountId())
		}
	}
	checkRef(&p, r, "author", m.GetAuthor(), opt)
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
	checkTime(&p, "flags.edited_at", m.GetFlags().GetEditedAt())
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
	checkOwner(&p, r, m.GetOwnerId(), m.GetOwnerKind(), env.GetPlatform(), opt)
	if m.GetType() == contentv1.MediaType_MEDIA_TYPE_UNSPECIFIED {
		p.addf("media.type unspecified")
	}
	return p.Err()
}

// checkOwner checks a media file's owner: a contract ID whose entity kind is
// one media may belong to, and equal to owner_kind.
func checkOwner(p *Problems, r Registry, id, kind, platform string, opt Options) {
	k := checkID(p, r, "owner_id", id, platform, opt)
	if k == "" {
		return
	}
	ok := false
	for _, allowed := range r.MediaOwnerKinds {
		if k == allowed {
			ok = true
		}
	}
	if !ok {
		p.addf("owner_id kind %q can't own media (want one of %v)", k, r.MediaOwnerKinds)
	}
	if kind != "" && kind != k {
		p.addf("owner_kind %q differs from owner_id's kind %q", kind, k)
	}
}

// Playlist checks a content.playlist.v1 event.
func Playlist(m *contentv1.Playlist, opt Options) error {
	r, _ := Load()
	env := m.GetEnvelope()
	p := Envelope(env, "playlist", opt)
	if k := checkID(&p, r, "account_id", m.GetAccountId(), env.GetPlatform(), opt); k != "" && k != "account" {
		p.addf("account_id kind %q, want account", k)
	}
	for i, it := range m.GetItems() {
		if k := checkID(&p, r, fmt.Sprintf("items[%d].post_id", i), it.GetPostId(), env.GetPlatform(), opt); k != "" && k != "post" {
			p.addf("items[%d].post_id kind %q, want post", i, k)
		}
		checkTime(&p, fmt.Sprintf("items[%d].added_at", i), it.GetAddedAt())
	}
	checkCount(&p, "item_count", m.GetItemCount())
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
	checkOwner(&p, r, m.GetOwnerId(), m.GetOwnerKind(), m.GetPlatform(), opt)
	if m.GetDeleteAfter() == nil {
		p.addf("delete_after missing: every file has a deadline")
	} else if m.GetStoredAt() != nil && m.GetDeleteAfter().AsTime().Sub(m.GetStoredAt().AsTime()) < minMediaLifetime {
		p.addf("delete_after is less than 24 hours after stored_at")
	}
	if len(m.GetRequiredConsumers()) == 0 {
		p.addf("required_consumers empty")
	}
	for _, c := range m.GetRequiredConsumers() {
		if !r.consumer(c) {
			p.addf("required consumer %q is not in the registry's consumers", c)
		}
	}
	if a := m.GetAttempt(); a < 0 || a > 4 {
		p.addf("attempt %d: the first announcement is 1, and a file is announced again at most 3 times", a)
	}
	return p.Err()
}

func (r Registry) consumer(name string) bool {
	for _, c := range r.Consumers {
		if c.Name == name {
			return true
		}
	}
	return false
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
	if opt.Production && !opt.Canary && m.GetOutcome() == discoveryv1.Outcome_OUTCOME_CAPPED {
		p.addf("outcome CAPPED is not allowed in production outside a canary (owner D-008, D-012)")
	}
	for i, e := range m.GetEvidence() {
		if e.GetContentTime() != nil {
			checkTime(&p, fmt.Sprintf("evidence[%d].content_time", i), e.GetContentTime())
		}
		if id := e.GetKeywordId(); id != "" && !contracts.ValidKeywordID(id) {
			p.addf("evidence[%d].keyword_id %q is not contracts.KeywordID's form", i, id)
		}
	}
	return p.Err()
}

// ItemAnalysis checks an analysis.result.v1 event.
func ItemAnalysis(m *analysisv1.ItemAnalysis, opt Options) error {
	var p Problems
	r, _ := Load()
	if m.GetEventId() == "" {
		p.addf("event_id empty")
	}
	kind := checkID(&p, r, "content_id", m.GetContentId(), m.GetPlatform(), opt)
	if kind != "" && kind != "post" && kind != "comment" {
		p.addf("content_id kind %q, want post or comment", kind)
	}
	if m.GetKind() != kind {
		p.addf("kind %q differs from content_id's kind %q", m.GetKind(), kind)
	}
	if k := checkID(&p, r, "account_id", m.GetAccountId(), m.GetPlatform(), opt); k != "" && k != "account" {
		p.addf("account_id kind %q, want account", k)
	}
	if kind == "comment" {
		if k := checkID(&p, r, "post_id", m.GetPostId(), m.GetPlatform(), opt); k != "" && k != "post" {
			p.addf("post_id kind %q, want post", k)
		}
		if m.GetDepth() > 0 && m.GetParentCommentId() == "" {
			p.addf("a reply's result needs parent_comment_id")
		}
	}
	if a := m.GetAuthorId(); a != "" {
		if k := checkID(&p, r, "author_id", a, "", opt); k != "" && k != "account" {
			p.addf("author_id kind %q, want account", k)
		}
	}
	if m.GetContentVersion() <= 0 {
		p.addf("content_version must be the analysed item's envelope.version")
	}
	checkProvenance(&p, m.GetProvenance())
	return p.Err()
}

// MediaProcessed checks a content.media.processed.v1 event: a consumer's
// acknowledgement of one file. Since v1.2.0.
func MediaProcessed(m *mediav1.MediaProcessed, opt Options) error {
	var p Problems
	r, _ := Load()
	if m.GetEventId() == "" {
		p.addf("event_id empty")
	}
	if k := checkID(&p, r, "media_id", m.GetMediaId(), m.GetPlatform(), opt); k != "" && k != "media" {
		p.addf("media_id kind %q, want media", k)
	}
	if !r.consumer(m.GetConsumer()) {
		p.addf("consumer %q is not in the registry's consumers", m.GetConsumer())
	}
	switch m.GetStatus() {
	case mediav1.ProcessStatus_PROCESS_STATUS_UNSPECIFIED:
		p.addf("status unspecified")
	case mediav1.ProcessStatus_PROCESS_STATUS_PROCESSED:
	default:
		if m.GetReason() == "" {
			p.addf("a %s acknowledgement needs a reason code", m.GetStatus())
		}
	}
	if m.GetProcessedAt() == nil {
		p.addf("processed_at missing")
	}
	return p.Err()
}

// MediaDeleted checks a content.media.deleted.v1 event. Since v1.2.0.
func MediaDeleted(m *mediav1.MediaDeleted, opt Options) error {
	var p Problems
	r, _ := Load()
	if m.GetEventId() == "" {
		p.addf("event_id empty")
	}
	if k := checkID(&p, r, "media_id", m.GetMediaId(), m.GetPlatform(), opt); k != "" && k != "media" {
		p.addf("media_id kind %q, want media", k)
	}
	if m.GetReason() == mediav1.DeleteReason_DELETE_REASON_UNSPECIFIED {
		p.addf("reason unspecified")
	}
	if m.GetDeletedAt() == nil {
		p.addf("deleted_at missing")
	}
	return p.Err()
}

// checkProvenance: who made an analysis result, which version of it, when.
func checkProvenance(p *Problems, pv *analysisv1.Provenance) {
	if pv.GetProducer() == "" || pv.GetAnalysisVersion() == 0 || pv.GetAnalyzedAt() == nil {
		p.addf("provenance needs producer, analysis_version and analyzed_at")
	}
}

// MediaAnalysis checks an analysis.media.v1 event. Since v1.2.0.
func MediaAnalysis(m *analysisv1.MediaAnalysis, opt Options) error {
	var p Problems
	r, _ := Load()
	if m.GetEventId() == "" {
		p.addf("event_id empty")
	}
	if k := checkID(&p, r, "media_id", m.GetMediaId(), m.GetPlatform(), opt); k != "" && k != "media" {
		p.addf("media_id kind %q, want media", k)
	}
	checkOwner(&p, r, m.GetOwnerId(), m.GetOwnerKind(), m.GetPlatform(), opt)
	if a := m.GetAccountId(); a != "" {
		if k := checkID(&p, r, "account_id", a, m.GetPlatform(), opt); k != "" && k != "account" {
			p.addf("account_id kind %q, want account", k)
		}
	}
	if m.GetType() == contentv1.MediaType_MEDIA_TYPE_UNSPECIFIED {
		p.addf("type unspecified")
	}
	if m.GetTranscript() != "" && (m.GetTranscriptLang() == "" || m.GetTranscriptSource() == "") {
		p.addf("a transcript needs transcript_lang and transcript_source")
	}
	checkProvenance(&p, m.GetProvenance())
	return p.Err()
}

var botLevels = map[string]bool{"": true, "suspected": true, "likely": true, "confirmed": true}

// AuthorFlag checks an analysis.author.v1 event. Since v1.2.0.
func AuthorFlag(m *analysisv1.AuthorFlag, opt Options) error {
	var p Problems
	r, _ := Load()
	if m.GetEventId() == "" {
		p.addf("event_id empty")
	}
	if k := checkID(&p, r, "author_id", m.GetAuthorId(), m.GetPlatform(), opt); k != "" && k != "account" {
		p.addf("author_id kind %q, want account", k)
	}
	if !botLevels[m.GetBot()] {
		p.addf("bot %q: want empty, suspected, likely or confirmed", m.GetBot())
	}
	if m.GetScore() < 0 {
		p.addf("score %d is negative", m.GetScore())
	}
	if m.GetScoredAt() == nil {
		p.addf("scored_at missing")
	}
	return p.Err()
}

// TopicUpdate checks an analysis.topic.v1 event. Since v1.2.0.
func TopicUpdate(m *analysisv1.TopicUpdate, opt Options) error {
	var p Problems
	if m.GetEventId() == "" {
		p.addf("event_id empty")
	}
	id := m.GetTopicId()
	if id == "" {
		p.addf("topic_id empty")
	}
	if par := m.GetParentId(); par != "" && !strings.HasPrefix(id, par+".") {
		p.addf("topic_id %q doesn't start with its parent %q and a dot", id, par)
	}
	switch m.GetStatus() {
	case "active", "rejected":
		if m.GetMergedInto() != "" {
			p.addf("merged_into is set, but status is %q", m.GetStatus())
		}
	case "merged":
		if m.GetMergedInto() == "" || m.GetMergedInto() == id {
			p.addf("a merged topic needs merged_into, another topic's ID")
		}
	default:
		p.addf("status %q: want active, merged or rejected", m.GetStatus())
	}
	if m.GetTaxonomyVersion() == "" {
		p.addf("taxonomy_version empty")
	}
	if m.GetUpdatedAt() == nil {
		p.addf("updated_at missing")
	}
	return p.Err()
}
