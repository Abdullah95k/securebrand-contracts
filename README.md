# The SecureBrand unified contract (draft v0.1)

**Status:** draft for review, 6 November 2026 (Iraq time). Every collector session reviews its own mapping, and the owner approves **v1.0.0**. Until v1.0.0 is tagged, **nothing is registered in any shared Schema Registry**.

## What it is

One agreed format for everything the collectors send into the shared data stream (Kafka): accounts, posts, comments, counts, media, and more. Each kind of thing has a fixed form, with the same fields on every platform. Storage, analysis, alerts and the client app read only these forms, so a dashboard shows Facebook, Instagram, TikTok and YouTube side by side without knowing which collector produced what.

It was built from every collector's field inventory and contract notes (fbs, igs, tks and yts), and from the analysis service's list of the fields each analysis needs. Those answers live in the platform's private repository; the field-by-field comparison is here, in [`docs/field-matrix.md`](docs/field-matrix.md).

**The owner's decisions it follows** (from the platform's decision log):
- **D-003:** personal data is stored as normal data: author IDs, names and handles are ordinary fields.
- **D-004:** every processed record, every version of it, and every analysis result are kept forever. Raw platform responses: failures and a 1% sample forever, the rest for 30 days.
- **D-008:** no discovery caps in production.
- **D-010:** what analysis sends to its AI provider excludes phone numbers, e-mails and author names. That's a rule for analysis, not a field rule here.
- **R5:** media files are kept only until their analysis is complete.

## What's here

| Path | What |
|---|---|
| `proto/securebrand/content/v1/` | The entities: Account, AccountSnapshot, Post, Comment, Metrics, Media, Audio, Playlist, plus LiveChatMessage and Transcript (reserved) |
| `proto/securebrand/media/v1/` | The media hand-off to analysis: MediaReady, MediaProcessed, MediaDeleted |
| `proto/securebrand/discovery/v1/` | Candidate: the one hand-off for accounts any discovery source finds |
| `proto/securebrand/coverage/v1/` | JobCoverage: what each job did and didn't get, and why |
| `proto/securebrand/registry/v1/` | AccountRequest: one way to ask any collector to add, change or pause an account |
| `proto/securebrand/lifecycle/v1/` | Erasure: **dormant**, published only on the owner's written order |
| `registry/` | Platform codes, topics and documented `extras` keys |
| `gen/go/` | Go code generated from `proto/` (`buf generate`) |
| `conformance/` | The conformance checks every collector runs in CI, and their tests |
| `samples/` | Synthetic sample events, one or more per message type, tested |
| `docs/field-matrix.md` | Every unified field, against what each platform gives logged out |

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

- **Absent is not zero.** A count the platform didn't show is left unset. Every count is a `Count` that says whether it is exact or rounded ("1.2K", with the text kept) and whether the owner hid it (Instagram likes, YouTube subscribers).
- **Every time carries its precision.** 61% of YouTube comment times are only "a year ago". `Time` keeps the estimate, its precision and the text shown, and a later, more precise reading replaces it.
- **Text exactly as received,** with mentions, hashtags and links as positioned spans. The offset unit is stated, because Facebook and TikTok count UTF-16 units.
- **Edits and deletions are new versions,** never overwrites. A deletion is a partial event with the status `DELETED`; storage keeps the earlier text forever (owner D-004).
- **Unknown platform fields are kept** in `extras`, under documented keys (`registry/extras.yaml`). A key two platforms use is promoted to a real field.
- **Raw responses** are pointed to by `provenance.raw_ref`: failures and a 1% sample kept forever, the rest for 30 days (owner D-004).

### 5. People and brands link across platforms

- Every author, mention, tag, co-author and share owner is an `AccountRef`, with its account ID. On every platform a commenter is an account, so commenters become accounts too (yts's point), and the entity layer can link them across platforms.
- Accounts carry every link: websites, bio links, About-panel links, and accounts the platform links (Instagram's linked Facebook page, with its ID). Each link says which platform and account it points to.
- Author IDs and names are normal fields (owner D-003). `author_key` is gone: the account ID does its job. What analysis sends to its AI provider is a separate rule (D-010).

### 6. Counts: one structure everywhere

`Engagement` holds views, likes, comments, replies, shares, saves, reposts and quotes; reactions by type with a `complete` flag (Facebook); live viewers; and anything else by name.

- **On Facebook, `likes` is the total of all reactions,** and `reactions` holds each type. So "likes" compares across platforms.
- Account counts: TikTok's "hearts" are `likes`, and YouTube subscribers are `followers`.
- **The volume rule:** a collector publishes `Metrics` when a count changed, and at least once a day for every item it still refreshes. tks and igs showed that publishing every read multiplies rows without new information.

### 7. Why an item was collected, and how completely

- `Relevance`: matched keyword IDs from the shared list, matched rules (`lang-ckb`, `region-IQ`, `phone-964`) and a score.
- `CollectionScope` on accounts: full, relevant items only (yts's "mention" channels), or profile only (the private second list).
- `ItemCoverage` on posts: comments collected against reported, the state (complete, partial, truncated, not read) and why. `JobCoverage` per job.
- TikTok's `created_in_country` (where a post was made) is a field of its own: tks found it the strongest Iraq signal.

### 8. Media until analysed, then gone

`MediaReady` names the file, its owner and its account, its storage object (with SHA-256, and a perceptual hash so the same picture can be matched across platforms after the file is gone), and a hard `delete_after` deadline. It also lists the consumers that must acknowledge it. `MediaProcessed` is each consumer's acknowledgement, and `MediaDeleted` records the deletion. The Media record itself stays forever. This replaces fbs's `fb.media.*` and the clashing `common.v1` copies in igs and tks.

### 9. Discovery, one format for every platform

`Candidate` carries:
- the platform whose collector should check the account, and the platform it was found on;
- how it was found: keyword search, official API search, mention, tag, co-author, commenter, share, link in content, profile link, followers, following, hashtag or audio page, related, graph, client request, import;
- the evidence and the Iraq-relevance check;
- the outcome: added, added for relevant items only, already listed, **second list** (private), dormant (with a reason), not relevant, unavailable.

**No caps in production** (owner D-008): the outcome `CAPPED` exists for tests and staging only, and the conformance check rejects it in production. Official-API search results carry IDs only, never the API's fields (the YouTube terms). A link found on one platform becomes a candidate for another, which is how new Facebook pages will be found.

### 10. Asking collectors for accounts

`AccountRequest` is what the platform's account registry gateway publishes: add, update, pause, resume or collect now, with a tier (hot, warm, long tail, dormant), a scope and who asked. Client watchlists push their accounts to the hot tier.

### 11. On the wire

- **Plain Protobuf, one message type per topic,** with the headers `sb-type` (the full message name) and `sb-contract` (the contract version). That's what fbs, igs and tks already do; only the header name is unified.
- **The platform registers each topic's subject** (`<topic>-value`, `BACKWARD_TRANSITIVE`) from the contract repository's CI. Collectors never register shared subjects.
- **Keys keep the order that matters:** comments and metrics are keyed by their post, so a post's comments and counts stay in order. Every topic and its key is in `registry/topics.yaml`.
- **Retention on Kafka is 7 days.** The permanent copy is ClickHouse plus the event archive in object storage.
- **The platform topics stay** (`fb.*`, `ig.*`, `tt.*`, `yt.*`) alongside `content.*`, as the standard (E2) says.

## The rules collectors follow

1. Only add fields; never rename, renumber or remove. A breaking change is a new major version (`content.post.v2`), run in parallel during migration.
2. Run `conformance` on what you produce, in CI (standard G5), against the samples and your fake platform.
3. Map by these rules:

   | Platform value | Contract |
   |---|---|
   | Facebook status, photo, album, link, share, event | `POST_TYPE_POST`, with the subtype |
   | Facebook video; YouTube video | `POST_TYPE_VIDEO` |
   | Facebook and Instagram reels, YouTube Shorts, TikTok videos | `POST_TYPE_SHORT_VIDEO` |
   | TikTok photo posts | `POST_TYPE_POST`, subtype `photo` |
   | Live streams | `POST_TYPE_LIVE`, with `video.live_status` |
   | Instagram highlight items | `POST_TYPE_STORY`, subtype `highlight`, plus a `Playlist` of type highlight |
   | YouTube community posts | `POST_TYPE_COMMUNITY` |
   | Replies on YouTube (one level) | depth 1, parent = root = the thread |
   | Replies on TikTok | root = `reply_id`; parent = `reply_to_reply_id`, else the root |
   | Facebook replies (to depth 3) | parent, root and depth as Facebook gives them |
   | Instagram replies | none logged out: `ItemCoverage.replies_available = false` |

4. **Fields a platform doesn't have stay unset.** "Not available logged out" is never a zero or a false.
5. **Deletion** is a partial event with `status.availability = DELETED` and `status.since`. It never blanks stored text.

## Adopting it, per service (after v1.0.0)

| Service | How | Estimate (its own) |
|---|---|---|
| fbs | An emitter writes `content.*` in the same Kafka batch as `fb.*` | 1–2 weeks |
| igs | Update its normalizer, and delete its own `securebrand.content.v1` | 7–8 days |
| tks | A small mapper from `tt.*` to `content.*`. Move the shared types off its `securebrand.common.v1`. | One milestone |
| yts | The relay publishes `yt.*` and `content.*` in one Kafka transaction | 3–4 days, after the outbox |
| analysis | Read `content.post`, `content.comment` and `content.media.ready` into its own internal item. Delete its own `securebrand.content.v1`. | 1–2 days |

igs and analysis must delete their local `securebrand.content.v1` when they adopt it, because two packages of that name can't be linked into one program. The contract deliberately doesn't use the name `securebrand.common.v1`, which igs and tks define differently, so their platform topics keep working while they move over.

## Open points for review

Each collector session confirms its mapping, and the owner decides the points marked **owner**.

1. **Shared keyword list:** where it lives (a file in this repository, or a small platform service). The seed is fbs's `iraq.yaml`. **Owner**, with the master.
2. **tks:** `predicted_age_group` rides as the documented extras key `tt.predicted_age_group`, as normal data (D-003). **Owner** to confirm.
3. **fbs:** `likes` as the reaction total on Facebook (section 6). fbs to confirm.
4. **igs:** account counts move from `content.metrics` with `target = account` to `content.account_snapshot`. igs agreed in its notes.
5. **yts:** official-API search hits never appear as `content.*`, only as `discovery.candidate` with IDs (section 9). yts proposed this.
6. **analysis:** whether `author_key` can be replaced by the author's account ID in its bot detection. analysis to confirm.

## Changelog

See [`CHANGELOG.md`](CHANGELOG.md).
