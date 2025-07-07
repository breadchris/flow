You will only create tsx components that have no backend. They will only use react and no other external dependencies.
Components will be created in ./data and available at http://64.23.152.104:8082/code/render/data/<path>

There is no need to perform research of the existing codebase, you will only create new components based on the provided requirements.
You will first elaborate on the component requirements, then provide the code for the component. Think about the user experience, functionality, and design. The component should be reusable and maintainable.
The component will have an emphasis on simplicity and clarity, ensuring that it is easy to understand and integrate into existing projects.
The component should be styled using Tailwind CSS for a modern and responsive design suitable for mobile devices and larger screens. It should also be accessible, ensuring that it can be used by all users, including those with disabilities.

## Data Persistence with Supabase Key-Value Store

Prototype components can now persist data using a built-in key-value store powered by Supabase. This allows your prototypes to maintain state across sessions, store user preferences, cache data, and implement complex application logic with persistent storage.

### Key Features

- **Worklet-scoped data isolation**: Each prototype can only access its own data
- **Namespace support**: Organize data logically within your prototype
- **Full CRUD operations**: Create, Read, Update, Delete with batch support
- **Type-safe TypeScript interface**: Complete IntelliSense and type checking
- **Automatic collision prevention**: Entropy-rich keys prevent data conflicts
- **Performance optimized**: Batch operations and efficient querying

### Import and Setup

```tsx
import { createWorkletKVStore, generateEntropyKey } from './data/supabase-kv';

// Initialize the KV store (workletId is automatically provided)
const kvStore = createWorkletKVStore(workletId);
```

### Basic Usage Examples

#### Simple Key-Value Operations
```tsx
// Set a value
await kvStore.set('user_prefs', 'theme', 'dark');

// Get a value
const theme = await kvStore.get<string>('user_prefs', 'theme');

// Check if key exists
const hasTheme = await kvStore.has('user_prefs', 'theme');

// Delete a key
const deleted = await kvStore.delete('user_prefs', 'theme');
```

#### Working with Complex Data
```tsx
// Store user preferences
const userPrefs = {
  theme: 'dark',
  language: 'en',
  notifications: true,
  layout: 'grid'
};
await kvStore.set('user_prefs', 'settings', userPrefs);

// Retrieve and use
const settings = await kvStore.get<typeof userPrefs>('user_prefs', 'settings');
if (settings?.theme === 'dark') {
  // Apply dark theme
}
```

#### Batch Operations
```tsx
// Set multiple values at once
await kvStore.mset('app_state', {
  current_page: 'dashboard',
  sidebar_open: true,
  last_saved: new Date().toISOString(),
  user_count: 42
});

// Get multiple values
const appState = await kvStore.mget('app_state', ['current_page', 'sidebar_open']);
```

#### Namespaces for Organization
```tsx
// Use different namespaces for logical separation
await kvStore.set('user_prefs', 'theme', 'dark');
await kvStore.set('app_state', 'current_tab', 'profile');
await kvStore.set('cache', 'api_response', { data: [...], timestamp: Date.now() });

// List all namespaces
const namespaces = await kvStore.listNamespaces(); // ['user_prefs', 'app_state', 'cache']
```

#### Advanced Operations
```tsx
// Increment a counter
const newCount = await kvStore.increment('analytics', 'page_views', 1);

// Append to a string
await kvStore.append('logs', 'debug_output', '\nNew log entry');

// Generate entropy-rich keys for unique identifiers
const sessionId = generateEntropyKey('session');
await kvStore.set('sessions', sessionId, { userId: 123, startTime: Date.now() });
```

### React Component Example with Persistence

```tsx
import React, { useState, useEffect } from 'react';
import { createWorkletKVStore } from './data/supabase-kv';

interface PersistentCounterProps {
  workletId: string;
}

const PersistentCounter: React.FC<PersistentCounterProps> = ({ workletId }) => {
  const [count, setCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const kvStore = createWorkletKVStore(workletId);

  // Load saved count on component mount
  useEffect(() => {
    const loadCount = async () => {
      try {
        const savedCount = await kvStore.get<number>('app_state', 'counter');
        if (savedCount !== null) {
          setCount(savedCount);
        }
      } catch (error) {
        console.error('Failed to load count:', error);
      } finally {
        setLoading(false);
      }
    };
    loadCount();
  }, []);

  // Save count whenever it changes
  const updateCount = async (newCount: number) => {
    setCount(newCount);
    try {
      await kvStore.set('app_state', 'counter', newCount);
    } catch (error) {
      console.error('Failed to save count:', error);
    }
  };

  if (loading) {
    return <div className="p-4">Loading...</div>;
  }

  return (
    <div className="p-6 bg-white rounded-lg shadow-md max-w-md mx-auto mt-8">
      <h2 className="text-2xl font-bold text-gray-800 mb-4">Persistent Counter</h2>
      <p className="text-gray-600 mb-4">Count: {count}</p>
      <div className="space-x-2">
        <button
          onClick={() => updateCount(count + 1)}
          className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded"
        >
          Increment
        </button>
        <button
          onClick={() => updateCount(count - 1)}
          className="bg-red-500 hover:bg-red-700 text-white font-bold py-2 px-4 rounded"
        >
          Decrement
        </button>
        <button
          onClick={() => updateCount(0)}
          className="bg-gray-500 hover:bg-gray-700 text-white font-bold py-2 px-4 rounded"
        >
          Reset
        </button>
      </div>
    </div>
  );
};

export default PersistentCounter;
```

### Common Patterns

#### Configuration Management
```tsx
// Save app configuration
const config = {
  apiEndpoint: 'https://api.example.com',
  timeout: 5000,
  retries: 3
};
await kvStore.set('config', 'api_settings', config);

// Load configuration
const apiConfig = await kvStore.get('config', 'api_settings');
```

#### User Session Management
```tsx
// Store user session
const session = {
  userId: 'user123',
  loginTime: Date.now(),
  permissions: ['read', 'write'],
  preferences: { theme: 'dark' }
};
await kvStore.set('session', 'current_user', session);

// Check authentication
const currentUser = await kvStore.get('session', 'current_user');
const isAuthenticated = currentUser !== null;
```

#### Caching API Responses
```tsx
// Cache API response with timestamp
const cacheData = {
  data: apiResponse,
  timestamp: Date.now(),
  ttl: 5 * 60 * 1000 // 5 minutes
};
await kvStore.set('cache', 'user_list', cacheData);

// Check cache validity
const cached = await kvStore.get('cache', 'user_list');
const isValid = cached && (Date.now() - cached.timestamp) < cached.ttl;
```

#### Analytics and Metrics
```tsx
// Track user interactions
await kvStore.increment('analytics', 'button_clicks');
await kvStore.increment('analytics', 'page_views');

// Store custom events
const eventKey = generateEntropyKey('event');
await kvStore.set('events', eventKey, {
  type: 'user_action',
  action: 'button_click',
  timestamp: Date.now(),
  metadata: { buttonId: 'submit-form' }
});
```

### Best Practices

1. **Use meaningful namespaces** to organize your data logically
2. **Handle errors gracefully** with try-catch blocks
3. **Use TypeScript types** for better development experience
4. **Batch operations** when setting multiple values
5. **Validate data** before storing to prevent issues
6. **Use entropy keys** for unique identifiers
7. **Consider data size limits** (1MB per value max)
8. **Implement loading states** for better UX

### API Reference

See `/data/supabase-kv.ts` for the complete TypeScript interface including:
- `WorkletKVStore` class with all methods
- Error types and handling
- Configuration options
- Utility functions

### Error Handling

```tsx
import { KVError, KVConnectionError, KVValidationError } from './data/supabase-kv';

try {
  await kvStore.set('namespace', 'key', value);
} catch (error) {
  if (error instanceof KVValidationError) {
    console.error('Invalid input:', error.message);
  } else if (error instanceof KVConnectionError) {
    console.error('Connection failed:', error.message);
  } else if (error instanceof KVError) {
    console.error('KV operation failed:', error.message);
  }
}
```

This key-value store enables sophisticated prototype applications with persistent data, user sessions, caching, and much more while maintaining data isolation between different worklets.