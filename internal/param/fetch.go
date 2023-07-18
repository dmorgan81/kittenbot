package param

import "context"

type Fetcher interface {
	Fetch(context.Context, string) (string, error)
	FetchAll(context.Context, string) ([]string, error)
}
