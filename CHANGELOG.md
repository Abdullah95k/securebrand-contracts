# Changelog

All notable changes to the contract. Fields are only ever added; a breaking change is a new major version.

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
