# Field matrix (6 November 2026)

Every unified field of the contract ([`../README.md`](../README.md)), against what each collector gets today. Built from:
- the field inventories of fbs, igs, tks and yts (in the platform's private repository);
- analysis's input needs (its `docs/master/analysis-contract-notes.md`);
- the master's reading of each collector's code.

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

Measured shares are given where the inventories have them.

## Account (`content.account.v1`)

| Unified field | fb (fbs) | ig (igs) | tt (tks) | yt (yts) |
|---|---|---|---|---|
| id / native_id | `page_id` **A** | `user_pk` **A** | `user_id` **A** | channel ID **A** |
| alt_ids | — | — | `sec_uid` **A** | — |
| handle | `vanity` **O** | `username` **A** | `username` **A** | `@handle` **A** |
| display_name | `name` **A** | `full_name` **O** | `nickname` **A** | `title` **A** |
| type | page **A** | profile (business flag **N**) | `organization` **A** | channel **A** |
| bio | `intro` **S** | `biography` **O** | `bio` **O** | `description` **O** (92%) |
| links | `contact.website` **S** (first only: **X** for the rest) | `bio_links` **S**; linked Facebook page **R** (ID dropped: **X**) | `bio_link` **S** (24%) | About links **O** (89%); description links **X** |
| contact (e-mails, phones, addresses) | **S** | **N** | — | — |
| categories | **O** | **N** (empty logged out) | — | — |
| verified | **O** | **A** | **A** | **X** (badge not parsed) |
| is_private | — (pages are public) | **A** | **A** | — |
| stated_country | from the address **S** | — | — | `country` **O** (95%) |
| platform_language | — | — | `language` **O** | — |
| created_at (with precision) | — | — | `createTime` **A**, exact | "Joined" **O** (97%), day precision; not parsed yet: **X** |
| handle / name changed at | — | — | **O** | — |
| picture | **A** (expires) | **A** (expires) | **A** (expires in hours) | **A** |
| cover or banner | **S** | — | — | banner **O** (95%) |
| keywords | — | — | — | channel keywords (share unknown) |
| status | active, unavailable, restricted **A** | active, private, unavailable **A** | active, private, unavailable, banned **A** | removed after 2 missed checks **A** |

## Account snapshot (`content.account_snapshot.v1`)

| Unified field | fb | ig | tt | yt |
|---|---|---|---|---|
| followers | **O**, rounded ("125K") | **A**, exact | **A**: exact daily, rounded on lists | subscribers **A**, rounded; can be hidden |
| following | — | **A** | **A** | — |
| posts | — | **A** | **A** | videos (share unknown) |
| likes | page likes **O**, rounded | — | hearts **A** | — |
| views | — | — | — | total views **O** (97%) |
| likes_given, friends | — | — | **A**, **O** | — |
| How often | daily | daily | every check, plus daily exact | per tier (6 h to weekly) |

## Post (`content.post.v1`)

| Unified field | fb | ig | tt | yt |
|---|---|---|---|---|
| id / native_id | `post_id` **A** | `media_pk` **A** | `post_id` **A** | video ID **A**; community post ID **A** |
| account_id | `page_id` **A** | `owner_pk` **A** | `user_id` **A** | `channel_id` **A** |
| type / subtype | `type` **A** | `kind`, `product_type` **A** | video or photo **A** | video, short or live **A**; community **A** |
| url | `permalink` **A** | `permalink` **A** | built **A** | built **A** |
| published_at | exact **A** | exact **A** | exact **A** | exact **O** (85%), else relative; community posts relative only |
| title | — | — | — | **A** |
| text | **O** | caption **O** | caption **O** | description **O** (87%); community text **O** |
| text spans (positions) | `text_ranges` **S** (UTF-16) | **N** (no positions served) | `text_ranges` **S** (UTF-16) | not extracted: **X** |
| platform_language | — | — | `textLanguage` **O** | not parsed: **X** |
| hashtags | **S** | **S** (from the caption) | **S** (24%) | — |
| uploader_tags | — | — | — | `tags` **S** (63%) |
| mentions | **S** (with IDs) | **S** (usernames only) | **S** (with IDs) | — |
| tagged | — | `usertags` **S** (10%) | — | — |
| coauthors | — | **R** (6%) | — | collaborations not collected |
| links | **S** | — (captions have none) | — | in the description: **X** |
| location | — | **S** (10%) | not served (unknown) | — |
| created_in_country | — | — | `region` **O** | — |
| reshare | `shared_post` **S** | — | duets and stitches: **X** | — |
| audio | — | `music` **O** (50%) | `music_id` **A** | — |
| video.duration | via media **S** | **A** (videos) | **A** | **A** (94%) |
| video.live_status, scheduled_start | type `live` | — | — | live status **A**; scheduled start **X** |
| video.chapters, caption_languages | — | — | — | **R** (1%), **R** (0%) |
| media (in order) | `media_ids` **S** | `media_ids` **A** | cover, photos **A** | thumbnail **A** |
| labels.ai_generated | — | **R** (7%) | **A** | — |
| labels.paid_partnership / ad | **S** / — | **N** / — | — / `isAd` **A** | — |
| labels.age_restricted, members_only, family_safe | — | — | warnings **R** | family-safe **O**; age and members-only **X** |
| flags.pinned | **S** | **R** (inferred) | **R** | — |
| flags.comments_disabled | — | **A** | **A** | **A** |
| category | — | — | code **A** | name **O** (85%) |
| platform_topics | — | topics **O** (89%) | — | — |
| playlists | — | highlight items | `playlist_id` **S** | playlist items **A** |
| poll | — | — | — | yes/no only: **X** |
| captions | — | — | — | **N** (need tokens) |
| status | visible, unavailable **A** | visible, unavailable **A** | visible, unavailable, private, with a code **A** | removed **A** |
| coverage | completeness, state **A** | per post not written: **X** | `comments_state`, completeness **A** | completeness, partial **A** |

## Counts (`Engagement`, on `content.metrics.v1`)

| Unified field | fb | ig | tt | yt |
|---|---|---|---|---|
| views | videos **S** | reels **O** (99.4%); photos and carousels **N** | plays **A** (rounded from 10,000) | **A**, exact |
| likes | reaction total **O** | **O** (97%); hidden 3%: `hidden` | **A** | **A**, exact |
| comments | **O** | **A** | **A** | **O** (70%) |
| shares | **O** | **N** | **A** | — |
| saves, reposts | — | **N** | **A**, **O** | — |
| reactions by type | **O**, with `complete` | — | — | — |
| concurrent_viewers | — | — | — | live **S** |
| Comment likes, replies | latest only | series **A** | latest only | latest only (series: **X**) |

## Comment (`content.comment.v1`)

| Unified field | fb | ig | tt | yt |
|---|---|---|---|---|
| id / native_id | **A** | **A** | **A** | **A** |
| post_id | **A** | **A** | **A** | **A** (video or community post) |
| parent, root, depth | depth 0–3 **A** | replies **N** (depth 0 only) | depth 0–1 **A**; reply-to **S** | depth 0–1 **A** |
| author.account.id | **O** (Facebook hides some) | **A** | **A** | **O** |
| author handle, name | name **O** | handle **A** (no display name: **N**) | handle and name **A** | handle **A** |
| author.account.type | person or page **O** | **N** | **N** | channel **A** |
| author verified, badges | **S** | **A** | **S** | **S**; member badge **R** |
| text | **O** (empty for stickers) | **A** | **A** | **A** |
| text spans | **S** | — | **S** | — |
| platform_language | — | — | **O** | — |
| published_at | exact **A** | exact **A** | exact **A** | relative only **A** (61% year precision in backfill) |
| attachments | **S** | GIFs dropped: **X** | images **R** | — |
| flags.by_post_owner | `is_op` **S** | **A** (derived) | derivable, not stored: **X** | **A** |
| flags.pinned, liked_by_creator | — | — | **A**, **A** | **R**, **R** |
| flags.edited | **S**, plus hash | detected by text change | detected by text change | **S** |
| counts.likes, replies | **O**, **O** | **A**, **A** | **A**, **A** | **A** (0 when no number shown), **A** |
| status: deleted | **A** (complete re-reads) | never detected: **X** | **A** (when the count still rises) | **A** (two sweeps) |

## Media, audio and playlists

| Unified field | fb | ig | tt | yt |
|---|---|---|---|---|
| Media kinds | photo, video, reel, thumbnail, comment media, GIF, sticker, cover, profile picture | photo, video, reel, carousel child, thumbnail, highlight cover, profile picture | cover (3 variants), video, photo, avatar, audio cover, comment image | thumbnail, avatar, banner, commenter avatar |
| Source URL expiry | days | days | hours | none in the URL |
| Downloaded today | off | off | off | off (images only, when storage is set) |
| Audio | — | `music` (title, artist, ID, original) **O** | `Music` **A** (title, author, original, duration, album, cover) | — |
| Playlist | — | highlights **A** (items unverified) | `playlist_id` only: **X** | playlists and items **A** |

## Discovery, coverage and provenance

| Unified field | fb | ig | tt | yt |
|---|---|---|---|---|
| Discovery sources today | mention checker (off); keyword search has no source | tags, co-authors, mentions, commenters (on in the chart) | entry check built; no source runs yet | website search, Data API search, commenters, graph |
| Private second list | `private` | `private` | dormant with a reason | — (channels are public) |
| Job coverage | `fb.job.v1` **A** | `ig.job.v1` **A** | `tt.job.v1` **A** | no event: **X** |
| provenance.parser_version | **A** | **A** | **A** | to add with the outbox |
| provenance.raw_ref | failures only | **A** (all kept) | failures only | none: **X** |
| provenance.exit_country | — | — | `status_region` **S** | — |

## Gaps that matter to analysis

| Analysis needs | Status | Where it's fixed |
|---|---|---|
| `by_post_owner` on every comment | tks derives but doesn't store it | tks's content mapper (contract adoption) |
| Mention spans with positions | Instagram serves none; YouTube not extracted | yts description links (REQ-010, fix 20); Instagram: not available |
| Link titles on posts | fbs **S**; others have none in posts | — |
| Captions or subtitles | YouTube **N** (needs tokens); TikTok none | — |
| A thread's post text for short comments | Available via `post_id` on every platform | — |
| Author type (person or page) | fbs and yts only | Instagram and TikTok don't serve it |
