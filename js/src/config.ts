import { PROTO_DEFINITIONS } from './proto/definitions';
export { IConfig, IRateConfig } from './proto/interfaces';
import { IConfig, IRateConfig } from './proto/interfaces';

import * as pbjs from 'protobufjs';

const builder = pbjs.Root.fromJSON(PROTO_DEFINITIONS);

// tslint:disable-next-line
export const Config: pbjs.Type = <any>builder.lookup('stream.Config');
// tslint:disable-next-line
export const RateConfig: pbjs.Type = <any>builder.lookup('stream.RateConfig');

export function DefaultStreamConfig(): IConfig {
  return {
    recordRate: {
      keyframeFrequency: 60000,
      changeFrequency: 1000,
    },
  };
}
