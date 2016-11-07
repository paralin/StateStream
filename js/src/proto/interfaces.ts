export interface IConfig {
  record_rate?: IRateConfig;
}
export interface IRateConfig {
  keyframe_frequency?: number;
  change_frequency?: number;
}
