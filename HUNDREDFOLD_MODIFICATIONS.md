# Hundredfold AI modifications

This repository is a maintained fork of
[`maximhq/bifrost`](https://github.com/maximhq/bifrost). The fork preserves the
upstream Apache License 2.0 and copyright notices.

The `hundredfold/model-gateway-m2` line is based exactly on upstream commit
`ec1dd920619955415bd6d61ab9ecff71f170ee22`.

Hundredfold AI changes on this line:

- fail closed for loopback and all other non-public destination addresses unless
  private-network access is explicitly enabled for development;
- validate every DNS answer before opening a socket, including IPv4-mapped and
  IPv6 transition-address cases; and
- apply an independently configurable hard byte limit to both encoded and
  decoded unary OpenAI-compatible provider responses, including error bodies and
  gzip payloads.

Consumers should pin an exact commit from
`https://github.com/hundredfold-ai/bifrost` and record both that fork commit and
the upstream base above in their third-party notices and software bill of
materials.
