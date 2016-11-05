import { PROTO_DEFINITIONS } from './proto/definitions';
export { IConfig, IRateConfig } from './proto/interfaces';

import * as pbjs from 'protobufjs';

const builder = pbjs.loadJson(JSON.stringify(PROTO_DEFINITIONS));

// tslint:disable-next-line
export const Config = builder.build('stream.Config');
// tslint:disable-next-line
export const RateConfig = builder.build('stream.RateConfig');
