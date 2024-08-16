# rbac

Provides a role based policy management package using Casbin for access control and OpenSearch for storing policies.

## Overview

The system includes the following components:
- **`auth`**: Middleware for enforcing access control using Casbin.
- **`opensearch`**: Adapter for managing Casbin policies with OpenSearch.
- **`config`**: Configuration loader for setting up the system.

## Components

### `auth` Package

The `auth` package provides middleware for enforcing access control based on Casbin policies.

#### `EnforcerMiddleware`

- **Fields:**
  - `Enforcer`: A Casbin enforcer instance.

- **Methods:**
  - `NewEnforcerMiddleware(enforcer *casbin.Enforcer) *EnforcerMiddleware`: Initializes the middleware with a Casbin enforcer.
  - `Middleware(c *fiber.Ctx) error`: Middleware function that checks user access permissions.

#### Functions

- `determineAction(method string) string`: Maps HTTP methods to actions.
- `extractUser(c *fiber.Ctx) (string, error)`: Extracts user credentials from the Basic Auth header.

### `opensearch` Package

The `opensearch` package provides an adapter for storing and managing Casbin policies in OpenSearch.

#### `Adapter`

- **Fields:**
  - `client`: An OpenSearch client instance.
  - `index`: The OpenSearch index name for storing policies.

- **Methods:**
  - `NewAdapter(addresses []string, index string) (*Adapter, error)`: Initializes the adapter and creates the index if it does not exist.
  - `AddPolicy(sec string, ptype string, rule []string) error`: Adds a policy to the OpenSearch index.
  - `LoadPolicy(model model.Model) error`: Loads policies from OpenSearch into a Casbin model.
  - `RemovePolicy(sec string, ptype string, rule []string) error`: Removes a policy from OpenSearch.
  - `SavePolicy(model model.Model) error`: Saves a Casbin model to OpenSearch.
  - `clearPolicies() error`: Clears all policies from OpenSearch.
  - `RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error`: Removes policies matching specific criteria from OpenSearch.

### `config` Package

The `config` package provides functions for loading configuration values.

#### `Config`

- **Fields:**
  - `OpenSearchAddresses`: List of OpenSearch addresses.
  - `Index`: OpenSearch index name.
  - `ModelPath`: Path to the Casbin model file.

- **Functions:**
  - `LoadConfig() *Config`: Loads configuration from environment variables or default values.
  - `getEnv(key, fallback string) string`: Gets environment variable value or fallback.

## Setup

1. **Install Dependencies**:

   ```bash
   go mod tidy

2. **Set Up Environment Variables**:

    ```bash
    export OPENSEARCH_ADDRESSES="http://localhost:9200"
    export OPENSEARCH_INDEX="casbin_policies"
    export MODEL_PATH="rbac_model.conf"

3. **Run Your Application**:
    Import and use the packages in your application as needed.

