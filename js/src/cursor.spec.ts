import { Cursor, CursorType } from './cursor';
import { Clone } from './entry';
import { MemoryBackend } from './memory_backend';
import { mockTime, cloneArr, sampleData } from './common.spec';

describe('Cursor', () => {
  let backend: MemoryBackend;

  beforeEach(() => {
    backend = new MemoryBackend(cloneArr(sampleData));
  });

  it('should compute state at a time', async () => {
    let cursor = new Cursor(backend, CursorType.ReadBidirectionalCursor);
    await cursor.init(mockTime(0));
    expect(cursor.isReady).toBe(true);

    cursor.setTimestamp(mockTime(0));
    expect(cursor.isReady).toBe(true);
  });

  it('should not do anything for setTimestamp on a write cursor', async () => {
    let cursor = new Cursor(backend, CursorType.WriteCursor);
    await cursor.init();
    expect(cursor.isReady).toBe(true);
    let state = Clone(cursor.state);
    cursor.setTimestamp(mockTime(0));
    expect(cursor.isReady).toBe(true);
    expect(cursor.state).toEqual(state);
  });

  it('should fast forward and rewind properly', async () => {
    let cursor = new Cursor(backend, CursorType.ReadBidirectionalCursor);
    await cursor.init(mockTime(0));
    expect(cursor.isReady).toBe(true);
    expect(cursor.state).toEqual({
      hello: true,
      goodbye: 4,
    });
    cursor.setTimestamp(mockTime(-9));
    expect(cursor.isReady).toBe(false);
    await cursor.computeState();
    expect(cursor.isReady).toBe(true);
    expect(cursor.state).toEqual({
      hello: {
        there: 1,
      },
    });
  });

  it('should emit stream entries properly', (done) => {
    (async () => {
      let cursor = new Cursor(backend, CursorType.ReadForwardCursor);
      let entryCount = 4;
      await cursor.init(mockTime(-9));
      cursor.streamEntries.subscribe((entry) => {
        if (!--entryCount) {
          done();
        }
      });
      cursor.setTimestamp(mockTime(0));
      cursor.computeState();
    })();
  });

  it('should not allow calling init twice', (done) => {
    (async () => {
      let cursor = new Cursor(backend, CursorType.ReadForwardCursor);
      await cursor.init(mockTime(-9));
      await cursor.init(mockTime(-8));
    })().then(() => {
      throw new Error('Expected to throw an exception.');
    }, () => {
      done();
    });
  });

  it('should not allow initing a write cursor with a snapshot', (done) => {
    ((async () => {
      let cursor = new Cursor(backend, CursorType.WriteCursor);
      await cursor.initWithSnapshot(null);
    })().then(() => {
      throw new Error('Should have thrown an error.');
    }, () => {
      done();
    }));
  });

  it('should init with snapshot correctly', async () => {
    let cursor = new Cursor(backend, CursorType.ReadForwardCursor);
    await cursor.initWithSnapshot(sampleData[0]);
    expect(cursor.isReady).toBe(true);
    expect(cursor.computedTimestamp).toEqual(sampleData[0].timestamp);
  });

  it('should use a rate config optimization', async () => {
    let cursor = new Cursor(backend, CursorType.ReadForwardCursor);
    cursor.setRateConfig({
      keyframe_frequency: 2000,
      change_frequency: 500,
    });
    await cursor.init(mockTime(-5));
    cursor.setTimestamp(mockTime(0));
    await cursor.computeState();
    expect(cursor.isReady).toBe(true);
  });
  it('should fast forward properly', async () => {
    let cursor = new Cursor(backend, CursorType.ReadForwardCursor);
    await cursor.init(mockTime(-9));
    cursor.setTimestamp(mockTime(0));
    await cursor.computeState();
    expect(cursor.isReady).toBe(true);
    expect(cursor.state).toEqual({
      hello: true,
      goodbye: 4,
    });
  });

  it('should rewind properly', async () => {
    let cursor = new Cursor(backend, CursorType.ReadBidirectionalCursor);
    await cursor.init(mockTime(-6.5));
    cursor.setTimestamp(mockTime(-9.5));
    await cursor.computeState();
    expect(cursor.isReady).toBe(true);
    expect(cursor.state).toEqual({
      hello: 'world',
    });
  });
});
