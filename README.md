# AWS Search
Simple AWS cross-account search tool, leveraging Credulous.

## Installation

* Install [Credulous](https://github.com/realestate-com-au/credulous) (or the Windows [variant](https://github.com/mefellows/credulous)), and source some creds.
* Download [AWS Search](/mefellows/aws-search/releases) and put it on your `PATH`

## Usage

```
aws-search --id ami-5678bbb --action ami          --region ap-southeast-2
aws-search --id i-1234aaaa  --action instance-id  --region ap-southeast-2
aws-search --id 54.34.3.1   --action public-ip    --region ap-southeast-2.
aws-search --id myapp       --action eb           --region ap-southeast-2
aws-search --id prd-a-123ab --action eb-env       --region ap-southeast-2
aws-search --id prd-b-123ab --action eb-resources --region ap-southeast-2
```

For verbose logging, simply add the `--verbose` flag.

## Unix philosophy

AWS Search is desgined to be combined with other tools, for example, it goes realli well 
with [jq](https://github.com/stedolan/jq) too:

```
./aws-search --id production-a-123aaa  --action eb-resources --region ap-southeast-2 | jq .EnvironmentResources.Instances[0].Id
```

NOTE: `AWS_REGION` environment variable will be used if no `-r` option is present.
