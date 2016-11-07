/* tslint:disable:trailing-comma */
/* tslint:disable:quotemark */
/* tslint:disable:max-line-length */
export const PROTO_DEFINITIONS = {
    "package": "stream",
    "syntax": "proto3",
    "messages": [
        {
            "name": "Config",
            "fields": [
                {
                    "rule": "optional",
                    "type": "RateConfig",
                    "name": "record_rate",
                    "id": 1
                }
            ],
            "syntax": "proto3"
        },
        {
            "name": "RateConfig",
            "fields": [
                {
                    "rule": "optional",
                    "type": "int64",
                    "name": "keyframe_frequency",
                    "id": 1
                },
                {
                    "rule": "optional",
                    "type": "int64",
                    "name": "change_frequency",
                    "id": 2
                }
            ],
            "syntax": "proto3"
        }
    ],
    "isNamespace": true
};
