import { PROTO_DEFINITIONS } from './definitions';

import * as ProtoBuf from 'protobufjs';

describe('proto', () => {
  it('should build the proto correctly', () => {
    const builder = ProtoBuf.loadJson(JSON.stringify(PROTO_DEFINITIONS));
    expect((<any>builder.lookup('stream')).className).toBe('Namespace');
    expect((<any>builder.lookup('stream.Config')).className).toBe('Message');
  });
});
