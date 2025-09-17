package scrollwork

import "sync"

type (
	Usage struct {
		tokens int

		mu sync.Mutex
	}
)

func (u *Usage) Update(tokens int) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.tokens = tokens
}

func (u *Usage) Tokens() int {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.tokens
}
