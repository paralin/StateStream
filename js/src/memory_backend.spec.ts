import { MemoryBackend } from './memory_backend';
import { StreamEntry, StreamEntryType } from './entry';
import { mockTime, cloneArr, sampleData } from './common.spec';

describe('MemoryBackend', () => {
  let backend: MemoryBackend;
  beforeEach(() => {
    backend = new MemoryBackend(cloneArr(sampleData));
  });

  it('should get snapshots correctly', () => {
    let timestamp = mockTime(-6);
    expect(backend.getSnapshotBefore(timestamp)).toEqual(sampleData[0]);
  });

  it('should get a later snapshot correctly', () => {
    let timestamp = mockTime(0);
    expect(backend.getSnapshotBefore(timestamp)).toEqual(sampleData[sampleData.length - 2]);
  });

  it('should return null for no snapshots', () => {
    let timestamp = mockTime(-11);
    expect(backend.getSnapshotBefore(timestamp)).toBe(null);
  });

  it('should get a snapshot after a time', () => {
    let timestamp = mockTime(-7);
    expect(backend.getEntryAfter(timestamp, StreamEntryType.StreamEntrySnapshot)).toEqual(sampleData[sampleData.length - 2]);
  });

  it('should save a new entry', () => {
    let entry: StreamEntry = {
      data: {testing: 'yes'},
      timestamp: mockTime(-1),
      type: StreamEntryType.StreamEntrySnapshot,
    };
    backend.saveEntry(entry);
    expect(backend.getEntryAfter(mockTime(-2), StreamEntryType.StreamEntrySnapshot)).toEqual(entry);
  });

  it('should amend an old entry', () => {
    let timestamp = mockTime(-8);
    let entry: StreamEntry = {
      data: {testing: 'yesagain'},
      timestamp: timestamp,
      type: StreamEntryType.StreamEntrySnapshot,
    };
    backend.amendEntry(entry, timestamp);
    expect(backend.getEntryAfter(mockTime(-9), StreamEntryType.StreamEntrySnapshot)).toEqual(entry);
  });
});
