# aws-search
Simple AWS cross-account search tool, leveraging Credulous

## Installation

* Install [credulous](https://github.com/realestate-com-au/credulous) (or the Windows [variant](https://github.com/mefellows/credulous))
* Download [AWS Search](/mefellows/aws-search/releases) and put it on your `PATH`

## Usage

```
aws-search --action ami --id ami-12345678
aws-search --action instance-id --id i-12345678
```

NOTE: `AWS_REGION` environment variable will be used if no `-r` option is present.
