# The SecureBrand unified contract (v1.0.0)

**Status:** **v1.0.0**, approved by the owner on 6 November 2026 (Iraq time), decision D-016. It is draft v0.2 unchanged: v0.1 plus every service's review of its own mapping (REQ-012 to REQ-016) and the owner's estimate rule ([`CHANGELOG.md`](CHANGELOG.md)). From now on fields are only added (section "The rules collectors follow"), and CI checks every change against the `v1.0.0` tag. The platform registers the subjects in its own Schema Registry when production is up; collectors never register shared subjects.

## What it is

One agreed format for everything the collectors send into the shared data stream (Kafka): accounts, posts, comments, counts, media, and more. Each kind of thing has a fixed form, with the same fields on every platform. Storage, analysis, alerts and the client app read only these forms, so a dashboard shows Facebook, Instagram, TikTok and YouTube side by side without knowing which collector produced what.

It was built from every collector's field inventory and contract notes (fbs, igs, tks and yts), and from the analysis service's list of the fields each analysis needs. Those answers live in the platform's private repository; the field-by-field comparison is here, in [`docs/field-matrix.md`](docs/field-matrix.md).

**The owner's decisions it follows** (from the platform's decision log):
- **D-003:** personal data is stored as normal data: author IDs, names and handles are ordinary fields.
- **D-004:** every processed record, every version of it, and every analysis result are kept forever. Raw platform responses: failures and a 1% sample forever, the rest for 30 days.
- **D-008:** no discovery caps in production.
- **D-010:** what analysis sends to its AI provider excludes phone numbers, e-mails and author names. That's a rule for analysis, not a field rule here.
- **D-013:** estimates are stored as full values and marked as estimates: "1.2K" is 1,200, and "a week ago" is that day at 00:00 Iraq time (section 4).
- **D-014:** TikTok's own subtitles are collected (after v1.0.0).
- **D-015:** TikTok's age band for commenters is normal data (`tt.predicted_age_group`).
- **R5:** media files are kept only until their analysis is complete, and at least 24 hours (section 8).

## What's here

| Path | What |
|---|---|
| `proto/securebrand/content/v1/` | The entities: Account, AccountSnapshot, Post, Comment, Metrics, Media, Audio, Playlist, plus LiveChatMessage and Transcript (reserved) |
| `proto/securebrand/media/v1/` | The media hand-off to analysis: MediaReady, MediaProcessed, MediaDeleted |
| `proto/securebrand/discovery/v1/` | Candidate: the one hand-off for accounts any discovery source finds |
| `proto/securebrand/coverage/v1/` | JobCoverage: what each job did and didn't get, and why |
| `proto/securebrand/registry/v1/` | AccountRequest: one way to ask any collector to add, change or pause an account |
| `proto/securebrand/analysis/v1/` | What analysis produces: ItemAnalysis, MediaAnalysis, AuthorFlag, TopicUpdate |
| `proto/securebrand/lifecycle/v1/` | Erasure: **dormant**, published only on the owner's written order |
| `registry/` | Platform codes, topics and documented `extras` keys |
| `gen/go/` | Go code generated from `proto/` (`buf generate`) |
| `conformance/` | The conformance checks every collector runs in CI, and their tests |
| `samples/` | Synthetic sample events, one or more per message type, tested |
| `docs/field-matrix.md` | Every unified field, against what each platform gives logged out |

**Releases** are published automatically: when a new `contracts.Version` (in `registry.go`) reaches `main`, the `release` workflow tags it and publishes its `CHANGELOG.md` section.

Check it, from the repository's root:

```bash
buf lint
```

```bash
go test ./...
```

## The design, and why

### 1. One message per kind of thing, one envelope on all of them

Posts and comments are separate (the owner's decision), and counts are separate from content, because counts change far more often. All four collectors and analysis agreed on this. Every content message starts with an `Envelope`:

| Field | Meaning |
|---|---|
| `event_id` | Unique per observation, and the same if one observation is delivered twice; consumers ignore repeats |
| `platform` | A code from `registry/platforms.yaml`: `fb`, `ig`, `tt`, `yt` |
| `kind` | The message kind: `account`, `account_snapshot`, `post`, `comment`, `metrics`, `media`, `audio`, `playlist`… |
| `id` | `<platform>:<entity kind>:<native id>`, the entity the event is about |
| `native_id` | The platform's own ID, as text (TikTok and YouTube IDs don't fit numbers safely) |
| `observed_at` | When we saw it (UTC) |
| `version` | Grows with every change; the highest wins. Collectors use `observed_at` in microseconds (fbs, igs and tks all proposed this). |
| `content_hash` | The same hash means the content didn't change |
| `partial` | Only the set fields are known; unset means "unchanged", never "empty" (deletions) |
| `provenance` | Who produced it and how: collector and version, parser version, source operation and variant, job IDs, how it was acquired, the raw response's location, the exit's country, and any terms deadline |

### 2. IDs that never change meaning

- `fb:account:100064000000001`, `yt:post:fakeVid0001`, `tt:comment:7410000000000000001`.
- **The middle part is the entity kind, never a subtype.** A reel, a Short or a reply can be re-classified, so a subtype in the ID would change the ID. Subtypes go in `type` and `subtype`. (fbs, igs, yts and analysis all asked for this.)
- One account kind on every platform: `fb:account:…`, not `fb:page:…`.
- Handles and usernames are attributes, never keys. TikTok's second key (`sec_uid`) goes in `alt_ids`.

### 3. New platforms need no redesign

- Platform codes live in `registry/platforms.yaml`. `x`, `tg`, `th`, `li`, `sc` and `web` are reserved now so codes never clash.
- Adding a platform means adding its code and a collector mapping. Messages don't change.
- Reserved for later collection: `LiveChatMessage`, `Transcript`, the `POST_TYPE_CHANNEL_MESSAGE` and `POST_TYPE_ARTICLE` types, playlists and reposts.

### 4. Nothing is lost, and nothing is guessed

- **Absent is not zero.** A count the platform didn't show is left unset, and so is a flag the platform doesn't give (Instagram's business flag, `ai_generated` without a label). "Not known" is never written as 0 or false.
- **Estimates are full values, marked as estimates (owner D-013).**
  - **Counts:** the full number ("1.2K" → 1,200; "1M" → 1,000,000; "10M" → 10,000,000) with `exactness`: `EXACT`, `ROUNDED` (an estimate: the platform rounded or abbreviated it, direction unknown), or `LOWER_BOUND` ("99+"). The text shown is kept in `displayed`. A hidden count has no value and `hidden = true`.
  - **Times:** "5 minutes ago" and "3 hours ago" become the start of that minute or hour. "2 days ago", "1 week ago", "1 month ago", "1 year ago" and a date with no time become **that day at 00:00 Iraq time** (21:00 UTC the evening before). `precision` says how rough it is (61% of YouTube comment times are only "a year ago"), `displayed` keeps the text, and a later, more precise reading replaces it as a new version. A time worked out from an ID gets the ID's precision (Instagram: `MINUTE`).
  - Storage derives a yes/no `is_estimate` column from these marks.
- **Text exactly as received,** with mentions, hashtags and links as positioned spans. The offset unit is stated, because Facebook and TikTok count UTF-16 units.
- **Edits and deletions are new versions,** never overwrites. A deletion is a partial event with its status; storage keeps the earlier text forever (owner D-004).
- **Availability says only what the platform says:** `DELETED` when the platform says deleted, or when a comment is absent from a complete re-read; `PRIVATE` when it says private; `UNAVAILABLE` when it doesn't say why, with its code or text in `reason`. A comment that comes back is a new `VISIBLE` version.
- **Unknown platform fields are kept** in `extras`, under documented keys (`registry/extras.yaml`). A key two platforms use is promoted to a real field.
- **Raw responses** are pointed to by `provenance.raw_ref`: failures and a 1% sample kept forever, the rest for 30 days (owner D-004).

### 5. People and brands link across platforms

- Every author, mention, tag, co-author and share owner is an `AccountRef`, with its account ID. On every platform a commenter is an account, so commenters become accounts too (yts's point), and the entity layer can link them across platforms.
- Accounts carry every link: websites, bio links, About-panel links, and accounts the platform links (Instagram's linked Facebook page, with its ID). Each link says which platform and account it points to.
- A post carries its account as it was when seen (`Post.account`: ID, handle, display name, type, verified), and its author when that isn't the account (`Post.author`). A comment carries its own link (`Comment.url`) when the platform has one.
- Author IDs and names are normal fields (owner D-003). `author_key` is gone: the account ID does its job. What analysis sends to its AI provider is a separate rule (D-010).

### 6. Counts: one structure everywhere

`Engagement` holds views, likes, comments, replies, shares, saves, reposts and quotes; reactions by type with a `complete` flag (Facebook); live viewers; and anything else by name.

- **On Facebook, `likes` is the total of all reactions,** and `reactions` holds each type. So "likes" compares across platforms.
- Account counts: TikTok's "hearts" are `likes`, and YouTube subscribers are `followers`.
- **The volume rule:** a collector publishes `Metrics` when a count changed, and at least once a day for every item it still refreshes. tks and igs showed that publishing every read multiplies rows without new information.

### 7. Why an item was collected, and how completely

- `Relevance`: matched keyword IDs from the shared list, matched rules (`lang-ckb`, `region-IQ`, `phone-964`) and a score. A keyword's ID is `contracts.KeywordID(group, language, spelling)`, for example `baghdad:ar:بغداد`: the spelling is folded with `contracts.FoldKeyword`, the fold fbs and igs already share (Arabic diacritics, tatweel and zero-width joiners dropped; hamza and madda alefs to bare alef; ة and ە to ه; ي and ى to ی; ك to ک; Kurdish letters of their own kept; Latin lower-cased and accents folded), so the same keyword has the same ID in every service, with no central counter. Compute it from the keyword list's stored spelling, never from the text that matched.
- `CollectionScope` on accounts is the registry's decision, not the account's state: full, relevant items only (yts's "mention" channels), profile only (the private second list), or **not monitored** (known only from content: a commenter, a mention, a link target). A private account stays full, and its status says private.
- `ItemCoverage` on posts: comments collected against reported, the state (complete; partial at a cap; truncated by a wall or an error; not read; disabled) and why. `JobCoverage` per account-level job; a collector with many small jobs adds an hourly roll-up per kind (`job_count`).
- `created_in_country` (where a post was made) is a field of its own, for platforms that say. TikTok's web answers don't carry it after all (tks measured 0 of 584 posts on 5 and 6 Nov), so it stays unset there; tks's Iraq check relies on places, phone numbers and `.iq` links in bios and names, captions, and audience.

### 8. Media until analysed, then gone

`MediaReady` names the file, its owner (a post, comment, account, audio or playlist) and its account, its storage object (with SHA-256, and a perceptual hash so the same picture can be matched across platforms after the file is gone), and a hard `delete_after` deadline: **at least 24 hours** after the file is stored, **72 hours by default**. It lists the consumers that must acknowledge it, by their registered names (`sb-analysis`, in `registry/platforms.yaml`). After a retryable failure the producer announces the file again, **at most 3 times**, and `attempt` counts the announcements. `MediaProcessed` is each consumer's acknowledgement (with a short `reason` when skipped or failed), and `MediaDeleted` records the deletion. The Media record itself stays forever. This replaces fbs's `fb.media.*` and the clashing `common.v1` copies in igs and tks.

### 9. Discovery, one format for every platform

`Candidate` carries:
- the platform whose collector should check the account, and the platform it was found on;
- how it was found: keyword search, official API search, mention, tag, co-author, commenter, share, link in content, profile link, followers, following, hashtag, audio or location page, related, graph, client request, import;
- the evidence (the keyword, the content it was seen in, and since v1.1.0 when that content was made) and the Iraq-relevance check;
- the outcome: added, added for relevant items only, already listed, **second list** (private), dormant (with a reason), not relevant, unavailable, **excluded** (removed by the owner).

**No caps in production** (owner D-008): the outcome `CAPPED` exists for tests and the canary only, and the conformance check rejects it in production. A collector's canary runs in production (D-012), so it checks with `Options{Production: true, Canary: true}` until its full run starts. **Official-API data never reaches `content.*`:** official-API search results appear only as candidates, with IDs, never the API's fields (the YouTube terms). The first `content.post` for such a video comes from our own reading of its page. A link found on one platform becomes a candidate for another, which is how new Facebook pages will be found.

### 10. Asking collectors for accounts

`AccountRequest` is what the platform's account registry gateway publishes: add, update, pause, resume or collect now, with a tier (hot, warm, long tail, dormant), a scope and who asked. Client watchlists push their accounts to the hot tier.

### 11. On the wire

- **Plain Protobuf, one message type per topic,** with the headers `sb-type` (the full message name) and `sb-contract` (the contract version, `contracts.Version` in Go: `1.0.0`). That's what fbs, igs and tks already do; only the header name is unified.
- **The platform registers each topic's subject** (`<topic>-value`, `BACKWARD_TRANSITIVE`) from the contract repository's CI. Collectors never register shared subjects.
- **Keys keep the order that matters:** comments and metrics are keyed by their post, so a post's comments and counts stay in order. Every topic and its key is in `registry/topics.yaml`.
- **Retention on Kafka is 7 days.** The permanent copy is ClickHouse plus the event archive in object storage.
- **The platform topics stay** (`fb.*`, `ig.*`, `tt.*`, `yt.*`) alongside `content.*`, as the standard (E2) says.

### 12. What analysis produces

`securebrand.analysis.v1` is part of the contract, because ClickHouse, the alert engine and the client app read it. `ItemAnalysis` (topic `analysis.result.v1`) carries the item's contract IDs (content, account, post, parent, author), the version of the item that was analysed, sentiment, entities, topics, intent, risk labels, the bot level, and why it went to the real-time lane (for the alert engine). `MediaAnalysis`, `AuthorFlag` (keyed by the author's account ID) and `TopicUpdate` complete it. The highest `analysis_version` of an item wins.

## The rules collectors follow

1. Only add fields; never rename, renumber or remove. A breaking change is a new major version (`content.post.v2`), run in parallel during migration.
2. Run `conformance` on what you produce, in CI (standard G5), against the samples and your fake platform. Every message type that can be published has a check (since v1.2.0, the media acknowledgements and deletions and the four analysis messages too).
3. Map by these rules:

   | Platform value | Contract |
   |---|---|
   | Facebook status, photo, album, link, share, event | `POST_TYPE_POST`, with the subtype |
   | Instagram photo; carousel | `POST_TYPE_POST`, subtype `photo`; subtype `carousel` |
   | Facebook video; YouTube video; Instagram non-reel video (feed or IGTV) | `POST_TYPE_VIDEO` (Instagram subtype `video` or `igtv`) |
   | Facebook and Instagram reels, YouTube Shorts, TikTok videos | `POST_TYPE_SHORT_VIDEO` |
   | TikTok photo posts | `POST_TYPE_POST`, subtype `photo` |
   | Live streams | `POST_TYPE_LIVE`, with `video.live_status` (Facebook can't tell live from ended: unset). A stream keeps its ID from upcoming to ended. |
   | Instagram highlight items | `POST_TYPE_STORY`, subtype `highlight`, plus a `Playlist` of type highlight; a 24-hour story, if collected: subtype `story` |
   | YouTube community posts | `POST_TYPE_COMMUNITY`; an attached video is a link, a re-shared post a `Reshare` |
   | A YouTube video's description | `description` (`text` is for community posts and captions) |
   | Replies on YouTube (one level) | depth 1, parent = root = the thread |
   | Replies on TikTok | root = `reply_id`; parent = `reply_to_reply_id`, else the root |
   | Facebook replies (to depth 3) | parent, root and depth as Facebook gives them |
   | Instagram replies | none logged out: `ItemCoverage.replies_available = false` |
   | Account type | Facebook: page, person or unspecified by fbs's own page read; a `User`-typed author or mention is unspecified. Instagram: unspecified. TikTok: organization when it says so, else unspecified. YouTube: channel. |
   | Facebook comment reads | stopped at the per-post cap: `PARTIAL`; stopped by a wall or an error: `TRUNCATED` |
   | Instagram's linked Facebook page | a link with `LINK_SOURCE_LINKED_ACCOUNT`, `target_platform = fb`, the page ID, and `url` built as `https://www.facebook.com/<page id>` |
   | TikTok subtitles | `Post.captions`, `source = CAPTION_SOURCE_PLATFORM_ASR` or `MACHINE_TRANSLATION` |

4. **Fields a platform doesn't have stay unset.** "Not available logged out" is never a zero or a false.
5. **Deletion** is a partial event with `status.availability = DELETED` and `status.since`. It never blanks stored text.

## Adopting it, per service (after v1.0.0)

| Service | How | Estimate (its own, after reviewing v0.1) |
|---|---|---|
| fbs | An emitter writes `content.*` in the same Kafka batch as `fb.*` | about 12–15 days |
| igs | Update its normalizer, and delete its own `securebrand.content.v1` | about 11–14 days |
| tks | Emit `content.*` from its persister, which knows when a row changed. Move the shared types off its `securebrand.common.v1`. | about 5–7 days, plus subtitles |
| yts | The outbox and relay publish `yt.*` and `content.*` in one Kafka transaction | about 4–5 weeks, including new fields |
| analysis | Read `content.post`, `content.comment` and `content.media.ready` into its own internal item; publish `securebrand.analysis.v1` from here. Delete its own `securebrand.content.v1`. | about 6–7 days |

igs and analysis must delete their local `securebrand.content.v1` when they adopt it, because two packages of that name can't be linked into one program. The contract deliberately doesn't use the name `securebrand.common.v1`, which igs and tks define differently, so their platform topics keep working while they move over.

## Open points for review

Each collector session confirms its mapping, and the owner decides the points marked **owner**.

1. **Shared keyword list:** where it lives (a file in this repository, or a small platform service). The seed is fbs's `iraq.yaml`. **Owner**, with the master. Its IDs are settled: `contracts.KeywordID` (v1.1.0).
2. **Instagram highlight items** are published only after one real read-only check shows logged-out Instagram serves them (igs).
3. **Adoption:** each service adopts v1.0.0 in its own request from the owner, and the platform registers the subjects once production is up.

Closed in v0.2: `predicted_age_group` is normal data (D-015); Facebook `likes` is the reaction total (fbs agreed; the client app labels it "reactions" on Facebook); igs's account counts move to snapshots (igs agreed); official-API hits stay out of `content.*` (yts); bot detection keys on the author's account ID (analysis agreed).

## Changelog

See [`CHANGELOG.md`](CHANGELOG.md).
