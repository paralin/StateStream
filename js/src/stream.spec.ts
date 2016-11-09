import { MemoryBackend } from './memory_backend';
import { Stream } from './stream';
import { mockTime, cloneArr, sampleData } from './common.spec';

describe('Stream', () => {
  let backend: MemoryBackend;
  let stream: Stream;

  beforeEach(() => {
    backend = new MemoryBackend(cloneArr(sampleData));
    stream = new Stream(backend, null);
  });

  it('should write correctly', async () => {
    await stream.writeState(mockTime(0), {test: 5});
  });
});
