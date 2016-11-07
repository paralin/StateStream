import { Cursor, CursorType } from './cursor';
import { Clone } from './entry';
import { MemoryBackend } from './memory_backend';
import { mockTime, cloneArr, sampleData } from './common.spec';

fdescribe('Cursor', () => {
  let backend: MemoryBackend;

  beforeEach(() => {
    backend = new MemoryBackend(cloneArr(sampleData));
  });

  it('should compute state at a time', () => {
    let cursor = new Cursor(backend, CursorType.ReadBidirectionalCursor);
    cursor.init(mockTime(0));
    expect(cursor.isReady).toBe(true);

    cursor.setTimestamp(mockTime(0));
    expect(cursor.isReady).toBe(true);
  });

  it('should not do anything for setTimestamp on a write cursor', () => {
    let cursor = new Cursor(backend, CursorType.WriteCursor);
    cursor.init();
    expect(cursor.isReady).toBe(true);
    let state = Clone(cursor.state);
    cursor.setTimestamp(mockTime(0));
    expect(cursor.isReady).toBe(true);
    expect(cursor.state).toEqual(state);
  });

  it('should fast forward and rewind properly', () => {
    let cursor = new Cursor(backend, CursorType.ReadBidirectionalCursor);
    cursor.init(mockTime(0));
    expect(cursor.isReady).toBe(true);
    expect(cursor.state).toEqual({
      hello: true,
      goodbye: 4,
    });
    cursor.setTimestamp(mockTime(-9));
    expect(cursor.isReady).toBe(false);
    cursor.computeState();
    expect(cursor.isReady).toBe(true);
    expect(cursor.state).toEqual({
      hello: {
        there: 1,
      },
    });
  });

  it('should emit stream entries properly', (done) => {
    let cursor = new Cursor(backend, CursorType.ReadForwardCursor);
    let entryCount = 4;
    cursor.init(mockTime(-9));
    cursor.streamEntries.subscribe((entry) => {
      if (!--entryCount) {
        done();
      }
    });
    cursor.setTimestamp(mockTime(0));
    cursor.computeState();
  });

  it('should not allow calling init twice', () => {
    expect(() => {
      let cursor = new Cursor(backend, CursorType.ReadForwardCursor);
      cursor.init(mockTime(-9));
      cursor.init(mockTime(-8));
    }).toThrow(new Error('Do not call init twice.'));
  });

  it('should not allow initing a write cursor with a snapshot', () => {
    expect(() => {
      let cursor = new Cursor(backend, CursorType.WriteCursor);
      cursor.initWithSnapshot(null);
    }).toThrow(new Error('Cannot initialize write cursor with snapshot.'));
  });

  it('should init with snapshot correctly', () => {
    let cursor = new Cursor(backend, CursorType.ReadForwardCursor);
    cursor.initWithSnapshot(sampleData[0]);
    expect(cursor.isReady).toBe(true);
    expect(cursor.computedTimestamp).toEqual(sampleData[0].timestamp);
  });

  it('should use a rate config optimization', () => {
    let cursor = new Cursor(backend, CursorType.ReadForwardCursor);
    cursor.setRateConfig({
      keyframe_frequency: 2000,
      change_frequency: 500,
    });
    cursor.init(mockTime(-5));
    cursor.setTimestamp(mockTime(0));
    cursor.computeState();
    expect(cursor.isReady).toBe(true);
  });
  it('should fast forward properly', () => {
    let cursor = new Cursor(backend, CursorType.ReadForwardCursor);
    cursor.init(mockTime(-9));
    cursor.setTimestamp(mockTime(0));
    cursor.computeState();
    expect(cursor.isReady).toBe(true);
    expect(cursor.state).toEqual({
      hello: true,
      goodbye: 4,
    });
  });

  it('should rewind properly', () => {
    let cursor = new Cursor(backend, CursorType.ReadBidirectionalCursor);
    cursor.init(mockTime(-6.5));
    cursor.setTimestamp(mockTime(-9.5));
    cursor.computeState();
    expect(cursor.isReady).toBe(true);
    expect(cursor.state).toEqual({
      hello: 'world',
    });
  });
});
