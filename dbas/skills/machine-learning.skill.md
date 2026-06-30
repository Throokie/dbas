---
name: machine-learning
version: 1.0.0
description: Machine learning pipeline, MLOps, model serving, and experiment tracking best practices
---

## Machine Learning / MLOps Instructions

When building ML systems, follow these guidelines:

1. **Experiment Tracking**: Log every experiment with parameters, metrics, and artifacts (MLflow, W&B, or DVC). Reproduce any result from the logged configuration.
2. **Data Versioning**: Version datasets alongside code using DVC or LakeFS. Never store large datasets in git — use remote storage with hash-based references.
3. **Feature Store**: Centralize feature definitions in a feature store (Feast, Tecton). Serve features consistently for training and inference.
4. **Training Pipeline**: Use DAG-based orchestration for training. Cache intermediate results. Run distributed training with Horovod or PyTorch DDP for large models.
5. **Model Registry**: Store all trained models in a model registry with version, metrics, lineage, and approval status. Promote only validated models to production.
6. **Serving**: Use model servers (Triton, TorchServe, BentoML) with batching and autoscaling. Separate inference from training infrastructure.
7. **A/B Testing**: Route traffic between model versions. Log predictions and outcomes for offline evaluation. Use statistical significance tests before full rollout.
8. **Monitoring**: Track prediction drift, feature drift, and model performance degradation. Alert on data quality shifts (missing features, distribution changes).
9. **CI/CD for ML**: Test data quality, feature computation, and model metrics in CI. Gate deployment on performance benchmarks against the current production model.
10. **Reproducibility**: Pin all dependencies (conda/pip freeze + Docker). Seed random number generators. Record the exact training environment.
11. **Bias & Fairness**: Audit training data for representation bias. Evaluate model performance across demographic subgroups. Document known limitations.
12. **Privacy**: Implement differential privacy for sensitive training data. Use on-device inference when possible. Never log raw user data in training pipelines.
