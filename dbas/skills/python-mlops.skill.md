---
name: python-mlops
version: 1.0.0
description: Python MLOps pipeline development and best practices skill
---

## Python MLOps Instructions

When building MLOps pipelines, follow these guidelines:

1. Use Poetry or uv for dependency management, not raw pip
2. Separate data pipeline, training, and serving into distinct stages
3. Log all hyperparameters and metrics with MLflow or W&B
4. Version datasets and models with DVC or similar tools
5. Write unit tests for data validation transforms
6. Use type hints for all function signatures
7. Implement feature stores for training/serving consistency
8. Add drift detection monitors for production models
9. Use Pydantic for configuration and data validation schemas
10. Containerize with distroless Docker images for deployment
