# Huefy Go SDK Lab

Verifies the core email contract through the real Go email client against a local stub server.

## Run

```bash
go run .
```

from `sdks/go/sdk-lab/`.

## Scenarios

1. Initialization
2. Single-send contract shaping
3. Bulk-send contract shaping
4. Validation rejection for invalid single input
5. Validation rejection for invalid bulk input
6. SDK health path behavior
7. Cleanup

## Notes

- The lab runs through the real email client methods.
- It checks transport shaping, parsed responses, and that invalid input never reaches the transport layer.
