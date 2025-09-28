package usage

import "sync"

type (
	ModelUsage struct {
		tokens int

		mu sync.Mutex
	}
)

func (m *ModelUsage) Update(tokens int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokens = tokens
}

func (m *ModelUsage) Tokens() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.tokens
}
