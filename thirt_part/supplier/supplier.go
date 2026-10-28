package supplier

import "catalog/thirt_part/supplier/dto"

type Supplier interface {
	FetchBatchManifest(updateID string) ([]string, error)

	FetchBatch(batchID string) ([]dto.Batch, error)
}
