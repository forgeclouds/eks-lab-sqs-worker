# eks-lab-sqs-worker

Ultra basic Go SQS consumer used by the KEDA lab (`labs/07-keda-sqs`).

```bash
# consume (QUEUE_URL is required; PROCESS_TIME defaults to 2s per message)
QUEUE_URL=https://sqs.us-east-1.amazonaws.com/<acct>/<queue> go run .

# produce 100 test messages
QUEUE_URL=... go run . -produce 100
```

- Receives one message at a time, so throughput grows with the number of pods. KEDA adds and removes pods, down to zero.
- On SIGTERM it stops receiving, finishes and deletes the current message, then exits.
- In the cluster, credentials come from EKS Pod Identity. Locally they come from your AWS profile.

## CI

Same pipeline as `eks-lab-hello-api`: OIDC → multi-arch build → ECR → bump `newTag` in
`eks-lab-gitops/workloads/sqs-worker/kustomization.yaml`.
