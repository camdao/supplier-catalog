# Risks and Limitations

## Duplicate Update Delivery

The third-party supplier may send the same update (e.g., `U001`) multiple times, potentially causing redundant processing or duplicate change records.

## Concurrent Batch Processing

The current system processes batches sequentially. Parallel processing in the future could introduce race conditions when multiple workers read and update the same product concurrently. Multiple application instances may also cause conflicts even with single-threaded execution per instance.