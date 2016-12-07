export interface IConfig {
  recordRate?: IRateConfig;
}

export interface IRateConfig {
  keyframeFrequency?: number;
  changeFrequency?: number;
}
