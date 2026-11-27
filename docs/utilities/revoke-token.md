# Revoke Token (Go SDK)

Revoke the API token the client is currently authenticated with.

## Making Requests

Revoke the current token with the `RevokeToken` method on the `client.Utilities` service. It takes no parameters and no options, but requires a valid API token — the one it revokes. It comes in two call styles:

| Method                 | Return                          | Description                                                             |
|-------------------------|---------------------------------|--------------------------------------------------------------------------|
| **`GetRevokeToken()`** | `error`                         | Convenience; uses a background context and discards response metadata. |
| **`RevokeToken(ctx)`** | `(*response.Response, error)`   | Context-aware; also returns the raw response plus rate-limit metadata. |

## RevokeToken

```go
func (s *Service) RevokeToken(ctx context.Context) (*response.Response, error)
func (s *Service) GetRevokeToken() error
```

Revoke the client's current API token from the `DELETE /v1/user/token/` endpoint. Unlike [Status](./api-status.md), [Headers](./headers.md), and [User](./user.md), which are served unversioned, this endpoint is versioned.

### Parameters

- `ctx` (`context.Context`) — request context for cancellation and deadlines (context method only).

This endpoint takes no functional options, but it requires a valid API token.

### Returns

- `*response.Response` — raw response plus rate-limit metadata (context method only).
- `error` — non-nil on request or decoding failure.

### Notes

- **This revokes the caller's own token, with no way to name a different one.** Once revoked, every further request made with this client — including any retry already in flight — fails with an `AuthenticationError`.
- There is no typed result: a successful call carries nothing beyond the confirmation that the token is revoked.

### Get (simple)

```go
package main

import (
	"log"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

func main() {
	client, err := marketdata.NewClient(marketdata.WithToken("YOUR_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	if err := client.Utilities.GetRevokeToken(); err != nil {
		log.Fatal(err)
	}
}
```

### Context + Response

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
)

func main() {
	client, err := marketdata.NewClient(marketdata.WithToken("YOUR_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	resp, err := client.Utilities.RevokeToken(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Revoked (status %d)\n", resp.StatusCode)
}
```
