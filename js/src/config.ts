import { PROTO_DEFINITIONS } from './proto/definitions';
export { IConfig, IRateConfig } from './proto/interfaces';
import { IConfig, IRateConfig } from './proto/interfaces';

import * as pbjs from 'protobufjs';

const builder = pbjs.loadJson(JSON.stringify(PROTO_DEFINITIONS));

export interface IProto<T> {
  new(...args: any[]): T;
  decode(data: any): T;
}

// tslint:disable-next-line
export const Config: IProto<IConfig> = builder.build('stream.Config');
// tslint:disable-next-line
export const RateConfig: IProto<IRateConfig> = builder.build('stream.RateConfig');

export function DefaultStreamConfig(): IConfig {
  return new Config({
    record_rate: {
      keyframe_frequency: 60000,
      change_frequency: 1000,
    },
  });
}
