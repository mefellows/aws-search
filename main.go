package main

// Simple AMI query tool: uses basic loops

import (
	"encoding/json"
	"flag"
	"fmt"
	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/ec2"
	eb "github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/mefellows/credulous/credulous"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
)

func main() {

	type config struct {
		region  string
		id      string
		action  string
		verbose bool
	}

	// Get arguments
	c := &config{}

	flag.StringVar(&c.region, "region", os.Getenv("AWS_REGION"), "Region")
	flag.StringVar(&c.region, "r", os.Getenv("AWS_REGION"), "Region")
	flag.StringVar(&c.id, "id", "", "Resource ID to find")
	flag.StringVar(&c.action, "action", "instance", "AWS resource (one of [instance, ip, ami])")
	flag.BoolVar(&c.verbose, "verbose", false, "Verbose output. Warning: This may disrupt output/pipe processing")
	flag.Parse()

	if c.region == "" || c.id == "" || c.action == "" {
		flag.Usage()
		os.Exit(1)
	}

	if !c.verbose {
		log.SetOutput(ioutil.Discard)
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
			config := &aws.Config{
				Region:      aws.String(c.region),
				Credentials: credentials.NewStaticCredentials(key, value, ""),
			}
			svc := ec2.New(config)
			ebSvc := eb.New(config)

			var r interface{}
			switch strings.ToLower(c.action) {
			case "instance-id":
				r = queryInstance(svc, "instance-id", c.id)
			case "ami":
				r = queryAmi(svc, c.id)
			case "ip":
				r = queryInstance(svc, "private-ip-address", c.id)
			case "public-ip":
				r = queryInstance(svc, "ip-address", c.id)
			case "eb":
				r = queryBeanstalk(ebSvc, c.id)
			case "eb-resources":
				r = queryBeanstalkResources(ebSvc, c.id)
			case "eb-env":
				r = queryBeanstalkEnv(ebSvc, c.id)
			default:
				log.Fatalf("Action '%s' is not a valid action", c.action)
			}

			if r != nil {
				v, err := json.Marshal(r)
				checkError(err)
				fmt.Printf("%s", v)
			}
			done.Done()
		}(key, value)
	}
	done.Wait()
}

// Return true if AMI exists
func queryAmi(service *ec2.EC2, ami string) interface{} {
	input := ec2.DescribeImagesInput{
		ImageIds: []*string{&ami},
	}
	output, err := service.DescribeImages(&input)
	if len(output.Images) > 0 {
		checkError(err)
		image := output.Images[0]
		log.Printf("Found image in account (%s): %s, with name: %s\n", *image.OwnerId, *image.Name)
		log.Printf("Tags: %v", image.Tags)
		return image
	}
	return nil
}

func queryBeanstalk(svc *eb.ElasticBeanstalk, filterVal string) interface{} {
	params := &eb.DescribeApplicationsInput{
		ApplicationNames: []*string{
			aws.String(filterVal),
		},
	}
	resp, err := svc.DescribeApplications(params)
	checkError(err)
	if len(resp.Applications) > 0 {
		return resp
	}

	return nil
}

func queryBeanstalkResources(svc *eb.ElasticBeanstalk, filterVal string) interface{} {
	params := &eb.DescribeEnvironmentResourcesInput{
		EnvironmentName: aws.String(filterVal),
	}
	resp, err := svc.DescribeEnvironmentResources(params)
	checkError(err)

	if resp.EnvironmentResources != nil {
		return resp
	}

	return nil
}

func queryBeanstalkEnv(svc *eb.ElasticBeanstalk, filterVal string) interface{} {
	params := &eb.DescribeEnvironmentsInput{
		EnvironmentNames: []*string{aws.String(filterVal)},
	}
	resp, err := svc.DescribeEnvironments(params)
	checkError(err)
	if len(resp.Environments) > 0 {
		return resp
	}

	return nil
}

func queryInstance(service *ec2.EC2, filter string, filterVal string) interface{} {
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String(filter),
				Values: []*string{
					aws.String(filterVal),
				},
			},
		},
	}
	resp, err := service.DescribeInstances(params)
	if len(resp.Reservations) > 0 {
		checkError(err)
		return resp
	}
	return nil
}

func checkError(err error) {
	if err != nil {
		log.Printf("Error: ", err)
	}
}
