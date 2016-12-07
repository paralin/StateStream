/* tslint:disable:trailing-comma */
/* tslint:disable:quotemark */
/* tslint:disable:max-line-length */
export const PROTO_DEFINITIONS = {
  "nested": {
    "stream": {
      "nested": {
        "Config": {
          "fields": {
            "recordRate": {
              "type": "RateConfig",
              "id": 1
            }
          }
        },
        "RateConfig": {
          "fields": {
            "keyframeFrequency": {
              "type": "int64",
              "id": 1
            },
            "changeFrequency": {
              "type": "int64",
              "id": 2
            }
          }
        }
      }
    }
  }
};
