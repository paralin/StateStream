import {
  Config,
  RateConfig,
  DefaultStreamConfig,
} from './config';

describe('config', () => {
  it('should create a Config properly', () => {
    let conf: any = new Config({
      record_rate: {
        keyframe_frequency: 10,
      },
    });
    expect(conf.record_rate.keyframe_frequency + '').toBe('10');
  });
  it('should create a RateConfig properly', () => {
    let conf: any = new RateConfig({
      keyframe_frequency: 10,
    });
    expect(conf.keyframe_frequency + '').toBe('10');
  });
  it('should create a valid default config', () => {
    let conf: any = DefaultStreamConfig();
    expect(conf.record_rate).not.toBe(null);
  });
});
