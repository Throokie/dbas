---
name: go-migration
version: 1.0.0
description: Database migration skill for DBAS Go projects
---

## Database Migration Instructions

When performing database migrations:

1. Always create both Up() and Down() migration functions
2. Test that Down() properly reverts Up() changes
3. Never modify existing migrations -- create new ones
4. Use database-specific types for columns
5. Add indexes for foreign keys and frequently queried columns
