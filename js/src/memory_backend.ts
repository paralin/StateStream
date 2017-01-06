import { IStorageBackend } from './backend';
import { StreamEntry, StreamEntryType } from './entry';
import binarySearch = require('binary-search');

// An in-memory state stream backend.
export class MemoryBackend implements IStorageBackend {
  constructor(public entries: StreamEntry[] = []) {
  }

  public getSnapshotBefore(time: Date): StreamEntry {
    // Binary search for the closest index to the snapshot.
    let idx = this.findClosest(time);
    if (idx >= this.entries.length) {
      idx = this.entries.length - 1;
    }
    if (idx < 0) {
      return null;
    }
    let timeNum = time.getTime();
    // Rewind until we are before the target & a snapshot.
    while (this.entries[idx].timestamp.getTime() >= timeNum ||
           this.entries[idx].type !== StreamEntryType.StreamEntrySnapshot) {
      idx--;
      if (idx < 0) {
        return null;
      }
    }
    return this.entries[idx];
  }

  public getEntryAfter(time: Date, filterType: StreamEntryType) {
    // Binary search for the closest index to the snapshot.
    let idx = this.findClosest(time);
    if (idx >= this.entries.length) {
      idx = this.entries.length - 1;
    }
    if (idx < 0) {
      return null;
    } else if (idx > 0) {
      idx--;
    }
    let timeNum = time.getTime();
    // Fast forward until we are after the target & matching.
    while (this.entries[idx].timestamp.getTime() <= timeNum ||
           (filterType !== StreamEntryType.StreamEntryAny &&
            this.entries[idx].type !== filterType)) {
      idx++;
      if (idx >= this.entries.length) {
        return null;
      }
    }
    return this.entries[idx];
  }

  public saveEntry(entry: StreamEntry) {
    this.entries.push(entry);
  }

  public amendEntry(entry: StreamEntry, oldTimestamp: Date) {
    let idx = this.findClosest(oldTimestamp);
    let ent = this.entries[idx];
    if (ent.timestamp.getTime() !== oldTimestamp.getTime()) {
      throw new Error('Cannot find that entry.');
    }
    this.entries[idx] = entry;
  }

  public findClosestEntry(timestamp: Date) {
    let idx = this.findClosest(timestamp);
    if (idx < 0) {
      return null;
    }
    return this.entries[idx];
  }

  // Time: closest to this time.
  // Exact matching time is always returned if found.
  public findClosest(time: Date): number {
    let timeNum: number = time.getTime();
    let idx: number =
      binarySearch(this.entries,
                   null,
                   (element: StreamEntry) => {
                     return element.timestamp.getTime() - timeNum;
                   });
    return idx >= 0 ? idx : ~idx;
  }
}
