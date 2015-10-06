package main

// Simple AMI query tool: uses basic loops

import (
	"flag"
	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mefellows/credulous/credulous"
	"log"
	"os"
	"strings"
	"sync"
)

func main() {

	type config struct {
		region string
		id     string
		action string
	}

	// Get arguments
	c := &config{}

	flag.StringVar(&c.region, "region", os.Getenv("AWS_REGION"), "Region")
	flag.StringVar(&c.region, "r", os.Getenv("AWS_REGION"), "Region")
	flag.StringVar(&c.id, "id", "", "Resource ID to find")
	flag.StringVar(&c.action, "action", "instance", "AWS resource (one of [instance, ip, ami])")
	flag.Parse()

	if c.region == "" || c.id == "" || c.action == "" {
		flag.Usage()
		os.Exit(1)
	}

	accts := credulous.GetAccounts()
	creds := make(map[string]string)

	for _, acct := range accts {
		key, secret, err := credulous.GetCredentials(acct.Username, acct.Account)
		checkError(err)
		creds[key] = secret
	}

	var done sync.WaitGroup
	done.Add(len(accts))

	// Loop through all of the accounts, search for instance in parallel
	for key, value := range creds {
		go func(key string, value string) {
			svc := ec2.New(&aws.Config{
				Region:      aws.String(c.region),
				Credentials: credentials.NewStaticCredentials(key, value, ""),
			})

			switch strings.ToLower(c.action) {
			case "instance-id":
				queryInstance(svc, c.id)
			case "ami":
				queryAmi(svc, c.id)
			case "ip":
			default:
				log.Fatalf("Action '%s' is not a valid action", c.action)

			}
			done.Done()
		}(key, value)
	}
	done.Wait()
}

// Return true if AMI exists
func queryAmi(service *ec2.EC2, ami string) bool {
	input := ec2.DescribeImagesInput{
		ImageIds: []*string{&ami},
	}
	output, err := service.DescribeImages(&input)
	if len(output.Images) > 0 {
		checkError(err)
		image := output.Images[0]
		log.Printf("Found image in account (%s): %s, with name: %s\n", *image.OwnerId, *image.Name)
		log.Printf("Tags: %v", image.Tags)
		return true
	}
	return false
}
func queryInstance(service *ec2.EC2, id string) bool {
	params := &ec2.DescribeInstancesInput{
		MaxResults: aws.Int64(1),
		InstanceIds: []*string{
			aws.String(id),
		},
	}
	resp, err := service.DescribeInstances(params)
	checkError(err)
	log.Printf("Response: %v\n", resp)

	return true
}

func checkError(err error) {
	if err != nil {
		log.Printf("Error: ", err)
	}
}
