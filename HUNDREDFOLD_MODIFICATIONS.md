# Hundredfold AI modifications

This repository is a maintained fork of
[`maximhq/bifrost`](https://github.com/maximhq/bifrost). The fork preserves the
upstream Apache License 2.0 and copyright notices.

The `hundredfold/core-v1.11.0` line is based exactly on upstream release
`core/v1.11.0`, commit `b096be9fb6fe83e932fd1718b78e328d678e5555`. It replaces
the `hundredfold/model-gateway-m2` line, which was based on upstream commit
`ec1dd920619955415bd6d61ab9ecff71f170ee22`. Upstream requires Go 1.27 from
this release on.

Hundredfold AI changes on this line:

- fail closed for loopback and all other non-public destination addresses unless
  private-network access is explicitly enabled for development;
- validate every DNS answer before opening a socket, including IPv4-mapped and
  IPv6 transition-address cases; and
- apply an independently configurable hard byte limit to both encoded and
  decoded unary OpenAI-compatible provider responses, including error bodies and
  gzip payloads; and
- allow OpenAI-compatible consumers to disable transport-level stale-connection
  retries for provider attempts that are not safely replayable;
- bound every content-decoding step -- gzip, deflate, brotli and zstd, layered
  codings each in turn -- not only gzip, since upstream decodes all four;
- name the kind of address a refused DNS answer held (private, link-local,
  loopback, unspecified) in the refusal, in upstream's vocabulary;
- count every input token of a Gemini embedding call: the Gemini API's batch
  usageMetadata, and Vertex's per-input statistics summed;
- accept a pre-minted Vertex access token, used as given and never pooled, and
  dial rerank at the configured origin when one is set; and
- let a test binary, and only a test binary, dial loopback test servers
  (`AllowLoopbackDialsForTesting`, inert unless `testing.Testing()`), so
  upstream's own tests run under the fork's dial policy.

Each source or test file changed from the recorded upstream base carries a
`Modified by Hundredfold AI` notice and points to the module notice.

Consumers should pin an exact commit from
`https://github.com/hundredfold-ai/bifrost` and record both that fork commit and
the upstream base above in their third-party notices and software bill of
materials.
