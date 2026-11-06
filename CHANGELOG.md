# Changelog

All notable changes to the contract. Fields are only ever added; a breaking change is a new major version.

## v1.1.0, 6 November 2026

Additions only; every v1.0.0 record is still valid. From the Instagram discovery session's plan (its C1, C2 and K2), decided by the master (D-025, D-027).

- `discovery.v1.Source.SOURCE_LOCATION_PAGE` (18): an account found on a place's page.
- `discovery.v1.Evidence.content_time` (9, a `content.v1.Time`): when the matching post or comment was made. The estimate rule (D-013) applies, and conformance checks it.
- `contracts.KeywordID(group, language, spelling)` and `contracts.FoldKeyword`: one ID per keyword in every service, `<group>:<language>:<folded spelling>`, with no central counter. Conformance checks `Evidence.keyword_id` against it.
- `conformance.Options.Canary`: allows the `CAPPED` outcome in production while a collector's canary runs (owner D-012). Nothing else changes.
- The module now needs `golang.org/x/text` (Unicode normalisation for keyword IDs).
- `created_in_country`: TikTok's web answers don't carry the post's country after all (tks measured 0 of 584 posts on 5 and 6 Nov), so tks leaves it unset. The field stays for platforms that give it. README, field matrix, the field's comment and the TikTok post sample corrected. No schema change.

## v1.0.0, 6 November 2026

Approved by the owner (decision D-016). The content of draft v0.2, unchanged, plus `contracts.Version` (`"1.0.0"`), the value collectors put in the `sb-contract` header. From here on, fields are only added; CI checks every change against this tag.

## v0.2.0 (draft), 6 November 2026

The changes from every session's review of v0.1 (REQ-012 to REQ-016: analysis, fbs, igs, tks, yts) and the owner's decisions D-013 to D-015. Nothing is registered yet; the owner approves the result as v1.0.0.

**Rules**
- Estimates (owner D-013): counts carry the full number with `exactness` as the estimate mark ("1.2K" → 1200, ROUNDED); a relative time is the start of its unit, and for days, weeks, months and years 00:00 Iraq time ("a week ago" read on 6 Nov → 30 Oct, 00:00 Iraq time), with `precision` as the mark. Conformance checks both, and a count with a value must say its exactness.
- `COUNT_EXACTNESS_ROUNDED` means "approximate", not "lower bound" (tks).
- Availability, one rule: DELETED only when the platform says so, or a comment is absent from a complete re-read; PRIVATE when it says private; UNAVAILABLE otherwise, with the platform's code or text (tks, yts).
- `scope` is the registry's decision, not the account's state; new `COLLECTION_SCOPE_NOT_MONITORED` for accounts known only from content (yts, tks).
- `exit_country` is where the platform placed the reader, on every event (tks).
- The playlist entity kind is `playlist` everywhere (tks, yts).
- A time worked out from an ID gets the ID's precision (igs); unknown flags stay unset (igs, tks, fbs).
- Official-API data never reaches `content.*` (yts).

**New fields**
- `Post.account` (the account as seen with the post; analysis) and `Post.author` (when it isn't the account; fbs).
- `Comment.url` (fbs, yts).
- `VideoInfo.live_started_at`, `live_ended_at` (yts).
- `CaptionTrack.source` (uploader, platform speech-to-text, machine translation), replacing `auto_generated` (tks).
- `Media.bitrate_bps` (tks).
- `Audio.status` (tks).
- `MediaReady.attempt` and `MediaProcessed.reason`; `delete_after` at least 24 hours after `stored_at`, 72 hours by default; consumer names in the registry (`sb-analysis`) (analysis).
- `OUTCOME_EXCLUDED` on `Candidate` (tks).
- `JobCoverage.job_count`, for hourly roll-ups (yts).
- Media may belong to a playlist (`owner_kind = playlist`; igs).

**New package:** `securebrand.analysis.v1` (ItemAnalysis, MediaAnalysis, AuthorFlag, TopicUpdate), ported from sb-analysis and revised once before its first registration: contract IDs, platform codes as text, the analysed item's version, routing for the alert engine. Its four topics are in `registry/topics.yaml`.

**Registries:** `media_owner_kinds` and `consumers` in `platforms.yaml`; the analysis topics; extras keys added (`ig.latest_story_at`, `ig.threads_badge`, `yt.family_safe`, `yt.tier`, `yt.added_by`, `yt.added_reason`) and removed where a shared field holds them (`tt.status_code`, `tt.category_code`, `yt.member_badge`, `yt.live_status_raw`).

**Field matrix:** every column corrected by its collector's review. TikTok serves subtitles; YouTube withholds caption tracks logged out.

**Conformance:** new checks for estimates, media lifetime, consumers, media owners, `Post.account`, playlists and analysis results; 21 rule-breaking cases now, each rejected; samples for a playlist and an analysis result.

## v0.1.0 (draft), 6 November 2026

First draft, for every session's review and the owner's approval of v1.0.0.

- `securebrand.content.v1`: Envelope, Provenance, Time, Count, Engagement, Text and TextSpan, AccountRef, Link, Location, Status, Relevance and MediaRef. Entities: Account, AccountSnapshot, Post, Comment, Metrics, Media, Audio and Playlist. LiveChatMessage and Transcript are reserved.
- `securebrand.media.v1`: MediaReady, MediaProcessed, MediaDeleted.
- `securebrand.discovery.v1`: Candidate.
- `securebrand.coverage.v1`: JobCoverage.
- `securebrand.registry.v1`: AccountRequest.
- `securebrand.lifecycle.v1`: Erasure (dormant).
- Registries: platforms (four active, six reserved), topics, documented extras.
- The conformance checks and synthetic samples, with tests.
- Replaces the two incompatible drafts named `securebrand.content.v1` (analysis's `ContentItem`; igs's per-entity messages), and supersedes the `securebrand.common.v1` media and job messages that igs and tks define with different field numbers.
