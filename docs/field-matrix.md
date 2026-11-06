# Field matrix (6 November 2026, v1.0.0)

Every unified field of the contract ([`../README.md`](../README.md)), against what each collector gets today. Built from:
- the field inventories of fbs, igs, tks and yts (in the platform's private repository);
- analysis's input needs (its `docs/master/analysis-contract-notes.md`);
- the master's reading of each collector's code.

The column corrections come from each collector's review of draft v0.1 (REQ-012 to REQ-016).

**Key:**

| Mark | Meaning |
|---|---|
| **A** | Always |
| **O** | Often (over 50%) |
| **S** | Sometimes |
| **R** | Rare |
| **N** | Not available logged out |
| **X** | Available, but not collected yet (a gap with a fix in that service's request) |
| **—** | Doesn't exist on that platform |
| **unknown** | Not checked yet whether the platform serves it logged out |

Measured shares are given where the inventories have them. An unknown flag stays unset, never `false` (C25).

## Account (`content.account.v1`)

| Unified field | fb (fbs) | ig (igs) | tt (tks) | yt (yts) |
|---|---|---|---|---|
| id / native_id | `page_id` **A** | `user_pk` **A** | `user_id` **A** | channel ID **A** |
| alt_ids | — | — | `sec_uid` **A** | — |
| handle | `vanity` **O** | `username` **A** | `username` **A** | `@handle` **A** |
| display_name | `name` **A** | `full_name` **O** | `nickname` **A** | `title` **A** |
| type | page, profile or unknown **A** (`pages.kind`) | unset: person or business can't be told (**N**) | `type` **N**: ORGANIZATION only when `isOrganization` is true | channel **A** |
| is_business | — | **N** (unset, never `false`) | `isOrganization` **A** | — |
| bio | `intro` **S** | `biography` **O** | `bio` **O** | `description` **O** (92%) |
| links | websites **S** (first only today: **X** for the rest); social links **X** | `bio_links` **S**; linked Facebook page **R** (ID dropped: **X**) | `bio_link` **S** (24%) | About links **O** (89%); description links **X** |
| contact (e-mails, phones, addresses) | **S** | **N** | — | **N** (the business e-mail needs a sign-in: never collected) |
| categories | **R** (none of the pilot's 150 pages) | **N** (empty logged out) | — | — |
| verified | **O** | **A** | **A** | **X** (badge not parsed) |
| is_private | — (pages are public) | **A** | **A** | — |
| stated_country | — (pages state none; address and phone are relevance signals) | **N** (not tried; probably login-only) | — | `country` **O** (95%) |
| platform_language | — | — | `language` **O** | — |
| created_at (with precision) | **N** (Page Transparency, not read logged out) | **N** (not tried) | `createTime` **A**, exact | "Joined" **O** (97%), day precision; not parsed yet: **X** |
| handle / name changed at | — | — | **O** | — |
| picture | **A** (expires; as a Media record: **X**) | **A** (expires) | **A** (expires in hours) | **A** |
| cover or banner | **S** (as a Media record: **X**) | — | — | banner **O** (95%) |
| keywords | — | — | — | channel keywords (share unknown) |
| status | active, unavailable, restricted **A** | active, private, unavailable ("no user") **A** | active, private, unavailable, banned **A** | removed after 2 missed checks **A** (`UNAVAILABLE`, C4) |

## Account snapshot (`content.account_snapshot.v1`)

| Unified field | fb | ig | tt | yt |
|---|---|---|---|---|
| followers | **O**, rounded or exact as shown | **A**, exact | **A**: exact on the profile, rounded on lists | subscribers **A**, rounded; can be hidden (`hidden` not set yet: **X**) |
| following | **X** (some pages show it; not parsed) | **A** | **A** | — |
| posts | — | **O** (a private account's answer may lack it) | **A** | videos (share unknown) |
| likes | page likes **O**, rounded or exact as shown | — | hearts **A** | — |
| views | — | — | — | total views **O** (97%) |
| likes_given, friends | — | — | **A**, **O** | — |
| How often | daily | daily, plus the first read on registration | every check, plus daily exact | per tier (6 h to weekly) |

## Post (`content.post.v1`)

| Unified field | fb | ig | tt | yt |
|---|---|---|---|---|
| id / native_id | `post_id` **A** | `media_pk` **A** | `post_id` **A** | video ID **A**; community post ID **A** |
| account_id | `page_id` **A** | `owner_pk` **A** | `user_id` **A** | `channel_id` **A** |
| account (as seen with the post, C7) | **A** | **A** | **A** | **A** |
| author (when not the account, C29) | `author_id` **S** | — | — | — |
| type / subtype | `type` **A** (status, photo, album, link, share, event, video, reel, live) | `kind`, `product_type` **A** (photo, carousel, reel, video or IGTV, story) | video or photo **A** | video, short or live **A**; community **A** |
| url | `permalink` **A** | `permalink` **A** | built **A** | built **A** |
| published_at | exact **A** | exact **A**; pk-derived times (highlight items): minute precision (C24) | exact **A** | exact **O** (85%), else relative (an estimate, D-013); community posts relative only |
| title | — | — | photo posts' `imagePost.title` **R**, **X** | **A** |
| text | **O** | caption **O** | caption **O** | community post text **O** (videos use `description`, C8) |
| description | — | — | — | video description **O** (87%) |
| text spans (positions) | `text_ranges` **S** (UTF-16) | **N** (no positions served) | `text_ranges` **S** (UTF-16) | description links, hashtags, mentions: **X** |
| platform_language | — | — | `textLanguage` **O** | unknown |
| hashtags | **S** | **S** (from the caption) | **S** (24%) | in titles and descriptions: **X** |
| uploader_tags | — | — | — | `tags` **S** (63%) |
| mentions | **S** (with IDs) | **S** (usernames only; an ID when also tagged or a co-author) | **S** (with IDs) | @mentions in descriptions: **X** |
| tagged | "with" tags **X** (unconfirmed) | `usertags` **S** (10%) | — | — |
| coauthors | — | **R** (6%) | — | collaborations: unknown |
| links | attachments **S** (text links come from the spans) | — (captions have none) | — | in the description: **X** |
| location | check-ins **X** (unconfirmed) | **S** (10%) | unknown (no `poiInfo` seen) | unknown |
| created_in_country | — | — | `region` **O** | — |
| reshare | `shared_post` **S** | — | duets and stitches: unknown (none seen in 253 posts) | community re-shares **X** |
| audio | — | `music` **O** (50%) | `music_id` **A** | Shorts sound: unknown |
| video.duration | via media **S** | **A** (videos) | **A** | **A** (94%) |
| video.width, height | via media **S** | **A** (videos) | **A** | not read |
| video.live_status, scheduled_start | type `live` **A**; status **N** | — | — (lives not collected) | live status **A**; scheduled start **X** |
| video.live_started_at, live_ended_at (C9) | **N** | — | — | **X** (from the player) |
| video.chapters, caption_languages | — | — | — | **R** (1%); caption tracks **N** in practice |
| media (in order) | `media_ids` **S** | `media_ids` **A** | cover, photos **A** | thumbnail **A** |
| labels.ai_generated | — | **R** (7%); unset when there's no label | **A** | "altered or synthetic" label **X** |
| labels.ai_label_text | — | `ai_label` **R** | `AIGCDescription` **X** | — |
| labels.paid_partnership / ad | **S** / — | **N** / — | — / `isAd` **A** | "includes paid promotion" **X** / — |
| labels.age_restricted, members_only, family_safe | — | — | — | family-safe **O**; age and members-only **X** |
| labels.warnings | — | — | **R** | — |
| flags.pinned | **S** | **R** (inferred) | **R** | — |
| flags.comments_disabled | **X** (unconfirmed: comments off reads as zero comments) | **A** | **A** | **A** |
| flags.edited | not read | `caption_is_edited` **S** (8%), no time | caption change detected **A** | — |
| category | — | — | code **A** | name **O** (85%) |
| platform_topics | — | topics **O** (89%) | — | — |
| playlists | — | highlight items (unverified: one real check first) | `playlist_id` **S** (30%: 75 of 253) | playlist items **A** |
| poll | — | — | — | options and total votes **X**; per-option votes **N** |
| captions | — | — | `video.subtitleInfos` **S**–**O** (14 of 30 videos; mostly Arabic speech-to-text), **X** until collected after v1.0.0 (D-014) | **N** (tracks withheld logged out) |
| status | visible, unavailable **A** | visible, unavailable **A**; walled posts (2.6%) **X** | visible, unavailable, private, with a code **A** | removed **A** (private, deleted or unavailable, C4) |
| coverage | completeness, state **A** (cap stops recorded as complete: fix) | per post not written: **X** | `comments_state`, completeness **A** | completeness, partial **A** |

## Counts (`Engagement`, on `content.metrics.v1`)

| Unified field | fb | ig | tt | yt |
|---|---|---|---|---|
| views | videos **S** | reels **O** (99.4%, reels tab only); photos and carousels **N**; non-reel videos unknown | plays **A** (rounded from 10,000; no text) | **A**, exact |
| likes | reaction total **O** (rounded or exact as shown) | **O** (97%); hidden 3%: `hidden` | **A** (rounded from 10,000) | **A**, exact |
| comments | **O** | **A** | **A** | **O** (70% on 4 Nov; to re-measure) |
| shares | **O** | **N** | **A** | — |
| saves, reposts | — | **N** | **A**, **O** | — |
| reactions by type | **O**, with `complete` | — | — | — |
| concurrent_viewers | — | — | — | live **S** |
| Post counts as a series | **A** (every read today; the volume rule: **X**) | reels **A** (reels tab); photos and carousels **X** | **A** | **A** |
| Comment likes, replies | latest only | series **A** | latest only | latest only (series: **X**) |

## Comment (`content.comment.v1`)

| Unified field | fb | ig | tt | yt |
|---|---|---|---|---|
| id / native_id | **A** | **A** | **A** | **A** |
| post_id | **A** | **A** | **A** | **A** (video or community post) |
| parent, root, depth | depth 0–3 **A** | replies **N** (depth 0 only) | depth 0–1 **A**; reply-to **S** | depth 0–1 **A** (parent = root; the direct parent **N**) |
| url (C28) | **A** | unknown | unknown | built from the IDs **A** |
| author.account.id | **O** (Facebook hides some) | **A** | **A** | **O** (expected **A**; not measured) |
| author handle, name | name **O**; handle **S** (from the profile URL) | handle **A** (no display name: **N**) | handle and name **A** | handle **A** |
| author.account.type | Page-typed: page **O**; User-typed: unspecified | **N** | **N** | channel **A** |
| author verified, badges | **S** | **A** | **S** | **S**; member badge **R** |
| text | **O** (empty for stickers) | **A** | **A** | **A** |
| text spans | **S** | — | **S** | links, mentions, timestamps, hashtags: **X** |
| platform_language | — | — | **O** | — |
| published_at | exact **A** | exact **A** | exact **A** | relative only **A** (an estimate, D-013; 61% year precision in backfill) |
| attachments | **S** | GIFs dropped: **X** | images **R** | — |
| flags.by_post_owner | `is_op` **S** | **A** (derived) | derivable by the mapper: **A** once adopted (**X** today) | **A** |
| flags.pinned, liked_by_creator | **N** | unknown (not parsed) | **A**, **A** | **R**, **R** |
| flags.edited | **S**, plus hash | kept in igs's Postgres; unset on the event | detected by text change | **S** (no time) |
| counts.likes, replies | **O**, **O** | **A**, **A** | **A**, **A** | **A** (0 when no number shown), **A** |
| status: deleted | **A** (complete re-reads; a restored comment comes back `VISIBLE`) | never detected: **X** | **A** (absent from a complete re-read) | **A** (two sweeps) |

## Media, audio and playlists

| Unified field | fb | ig | tt | yt |
|---|---|---|---|---|
| Media kinds | photo, video, reel, thumbnail, comment media, GIF, sticker, cover, profile picture | photo, video, reel, carousel child, thumbnail, highlight cover, highlight-item media, profile picture | cover (3 variants), video, photo, avatar, audio cover, comment image | thumbnail, avatar, banner, commenter avatar |
| Source URL expiry | days | days | hours | none in the URL |
| Downloaded today | off | off | off | off (images only, when storage is set) |
| bitrate (C11) | not read | not read | `bitrate` **A** | — (no video files) |
| Audio | — | `music` (title, artist, ID, original; author when shown) **O** | `Music` **A** (title, author name, original, duration, album, cover); copyrighted, commercial **O**; `uses` **X**; author as an account **N** | Shorts sound: unknown |
| Playlist | — | highlights **A** (items unverified: one real check first) | `playlist_id` **S** (30%); the playlists themselves: **X** | playlists and items **A** |

## Discovery, coverage and provenance

| Unified field | fb | ig | tt | yt |
|---|---|---|---|---|
| Discovery sources today | mention checker (off); keyword search has no source | tags, co-authors, mentions, commenters (on in the chart); profile links to other platforms **X** | entry check built; no source runs yet | website search, Data API search, commenters, graph; About-panel links to other platforms **X** |
| Private second list | `private` (reasons `no_public_posts`, `timeline_hidden`); `restricted` → DORMANT | `private` | dormant with a reason | — (channels are public) |
| Job coverage | `fb.job.v1` **A** | `ig.job.v1` **A** | `tt.job.v1` **A** | no event: **X** (hourly roll-ups, C18) |
| provenance.parser_version | **A** | **A** (per query, `"<query>:<n>"`) | **A** | to add with the outbox |
| provenance.raw_ref | failures only; 1% sample **X** | **A** (all kept) | failures only; 1% sample **X** | none: **X** |
| provenance.exit_country | **A** (the proxy tier's configured country) | **X** (needs a country per proxy tier) | `status_region` **S**; **A** once carried on every event (**X** today) | **X** (needs a country per proxy) |

## Gaps that matter to analysis

| Analysis needs | Status | Where it's fixed |
|---|---|---|
| The account's name on posts | On every platform from v0.2 (`Post.account`, C7) | Each collector's contract adoption |
| `by_post_owner` on every comment | tks derives it in its mapper (**A** once adopted) | tks's content mapper (contract adoption) |
| Mention spans with positions | Instagram serves none (a mention also tagged or a co-author gets an ID); YouTube not extracted yet | yts description and comment spans (REQ-010, fix 20); Instagram: not available |
| Link titles on posts | fbs **S**; others have none in posts | — |
| Captions or subtitles | TikTok serves its own subtitles (**S**–**O**, mostly Arabic speech-to-text), not collected yet; YouTube **N** (tracks withheld logged out) | tks, after v1.0.0 (D-014); YouTube: not available |
| A thread's post text for short comments | Available via `post_id` on every platform | — |
| Author type (person or page) | fbs (Page-typed authors only) and yts only | Instagram and TikTok don't serve it |
