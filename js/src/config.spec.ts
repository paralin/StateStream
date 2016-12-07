import {
  Config,
  RateConfig,
  DefaultStreamConfig,
} from './config';

describe('config', () => {
  it('should create a Config properly', () => {
    let conf: any = Config.create({
      recordRate: {
        keyframeFrequency: 10,
      },
    });
    expect(conf.recordRate.keyframeFrequency + '').toBe('10');
  });
  it('should create a RateConfig properly', () => {
    let conf: any = RateConfig.create({
      keyframeFrequency: 10,
    });
    expect(conf.keyframeFrequency + '').toBe('10');
  });
  it('should create a valid default config', () => {
    let conf: any = DefaultStreamConfig();
    expect(conf.recordRate).not.toBe(null);
  });
});
