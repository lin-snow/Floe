package memory

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Memory represents a thread-safe storage for workflow variables.
type Memory struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewMemory creates a new Memory instance.
func NewMemory() *Memory {
	return &Memory{
		data: make(map[string]interface{}),
	}
}

// Set stores a value at the given path.
// Path format: "key" or "key.subkey" (simple support for now, mostly top-level or one level deep if we implement nested map logic, 
// but for this MVP we'll stick to flat keys or simple map assignment if the value is a map).
// For the requirement "global.xxx", we can treat "global" as a key or just use the full string as key if we want flat.
// However, the requirements mention "global.a.b". Let's implement a simple nested map support or just flat keys if acceptable.
// The requirements say: "Memory 路径约定：global.xxx.yyy（用点分层）".
// And "Get/Set 通过 strings.Split(path, ".") 逐级遍历 map".
func (m *Memory) Set(path string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	keys := strings.Split(path, ".")
	if len(keys) == 0 {
		return fmt.Errorf("empty path")
	}

	// If it's just one key, set it directly
	if len(keys) == 1 {
		m.data[keys[0]] = value
		return nil
	}

	// Traverse/Create nested maps
	current := m.data
	for i := 0; i < len(keys)-1; i++ {
		k := keys[i]
		val, exists := current[k]
		if !exists {
			newMap := make(map[string]interface{})
			current[k] = newMap
			current = newMap
		} else {
			nextMap, ok := val.(map[string]interface{})
			if !ok {
				return fmt.Errorf("path segment '%s' is not a map", k)
			}
			current = nextMap
		}
	}

	current[keys[len(keys)-1]] = value
	return nil
}

// Get retrieves a value from the given path.
func (m *Memory) Get(path string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := strings.Split(path, ".")
	if len(keys) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	var current interface{} = m.data
	for _, k := range keys {
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot traverse path '%s', segment is not a map", path)
		}
		val, exists := currentMap[k]
		if !exists {
			return nil, fmt.Errorf("path '%s' not found", path)
		}
		current = val
	}

	return current, nil
}

// ResolveInterpolation replaces ${path} with values from memory.
func (m *Memory) ResolveInterpolation(str string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(str, func(match string) string {
		// match is like "${global.foo}", submatch is "global.foo"
		// We need to extract the content inside ${}
		path := match[2 : len(match)-1] // remove ${ and }
		
		val, err := m.Get(path)
		if err != nil {
			// For MVP, log warning or return empty? 
			// Requirements: "执行失败并记录 trace 错误" or "当作空字符串".
			// Let's return empty string or keep original if we want to be safe, 
			// but "replace with memory.Get(path) string form" implies replacement.
			return "" 
		}
		return fmt.Sprintf("%v", val)
	})
}

// Snapshot returns a copy of the current data.
func (m *Memory) Snapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Deep copy is better, but for MVP shallow copy of top level or JSON marshal/unmarshal
	// Let's do a simple recursive copy if needed, or just return m.data for read-only snapshot (unsafe if modified).
	// For trace, we usually want a snapshot.
	// Let's do a simple JSON roundtrip or manual copy. Manual copy is safer.
	return deepCopyMap(m.data)
}

func deepCopyMap(src map[string]interface{}) map[string]interface{} {
	dest := make(map[string]interface{})
	for k, v := range src {
		if vm, ok := v.(map[string]interface{}); ok {
			dest[k] = deepCopyMap(vm)
		} else {
			dest[k] = v
		}
	}
	return dest
}
