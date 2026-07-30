# Hundredfold AI modifications to Bifrost Core

This module is distributed from the maintained fork at
`https://github.com/hundredfold-ai/bifrost` under the upstream Apache License
2.0. It is based on upstream commit
`ec1dd920619955415bd6d61ab9ecff71f170ee22`.

Hundredfold AI modified the provider network dial policy and OpenAI-compatible
unary response handling for the Model Gateway:

- non-public destinations, including loopback and IPv6 transition forms, fail
  closed unless a constrained development private-network opt-in applies; and
- both encoded and decoded response bodies are rejected at a configured hard
  byte limit; and
- OpenAI-compatible deployments can disable transport-level stale-connection
  retries when a provider attempt is not safely replayable.

Each source or test file changed from the recorded upstream base carries a
`Modified by Hundredfold AI` notice that points here.

The root `HUNDREDFOLD_MODIFICATIONS.md` contains the complete fork notice.
