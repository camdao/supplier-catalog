package supplier

import dto_fake "catalog/thirt_part/supplier/dto"

type Supplier interface {
	FetchBatchManifest(updateID string) ([]string, error)
	FetchBatch(batchID string) ([]dto_fake.BatchItem, error)
}
