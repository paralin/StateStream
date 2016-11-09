import { StreamEntry, StreamEntryType } from './entry';

// Generic interface to stream storage
export interface IStorageBackend {
  // Retrieve the first snapshot before timestamp. Return nil for no data.
  getSnapshotBefore(timestamp: Date): StreamEntry | Promise<StreamEntry>;

  // Get the next entry after the timestamp. Return nil for no data.
  // Filter by the filter type, or don't filter if StreamEntryAny
  getEntryAfter(timestamp: Date, filterType: StreamEntryType): StreamEntry | Promise<StreamEntry>;

  // Store a stream entry.
  saveEntry(entry: StreamEntry): void;

  // Amend an old entry
  amendEntry(entry: StreamEntry, oldTimestamp: Date): void;
}
