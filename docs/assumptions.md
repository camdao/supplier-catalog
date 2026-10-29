# Assumptions

## Supplier Data

- Each product has a stable product ID.
- Each product update includes a monotonically increasing version.
- A higher product version represents newer information.
- Product withdrawal is represented as a versioned event.

## Batch Processing

- Each batch has a stable batch ID.
- The supplier data can be retrieved again after an interruption.
- Reprocessing a batch must not duplicate its effects.

## Update Lifecycle

- Each catalog update has a unique update ID.
- An update is complete only after all required batches have been processed successfully.
- Failed batches can be retried.