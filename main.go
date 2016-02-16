package main

// Simple AMI query tool: uses basic loops

import (
	"encoding/json"
	"flag"
	"fmt"
	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	eb "github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/mefellows/credulous/credulous"
	"github.com/vaughan0/go-ini"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {

	type config struct {
		region    string
		id        string
		action    string
		verbose   bool
		credulous bool
		timeout   time.Duration
	}

	// Get arguments
	c := &config{}

	flag.StringVar(&c.region, "region", os.Getenv("AWS_REGION"), "Region")
	flag.StringVar(&c.region, "r", os.Getenv("AWS_REGION"), "Region")
	flag.StringVar(&c.id, "id", "", "Resource ID to find")
	flag.StringVar(&c.action, "action", "instance", "AWS resource (one of [instance, ami, ip, public-ip, eb, eb-resources, eb-env]")
	flag.BoolVar(&c.credulous, "credulous", false, "Use credulous accounts instead of the stored aws profiles (default)")
	flag.BoolVar(&c.verbose, "verbose", false, "Verbose output. Warning: This may disrupt output/pipe processing")
	flag.DurationVar(&c.timeout, "timeout", 5*time.Second, "Timeout for the search. Defaults to 5s")
	flag.Parse()

	if c.region == "" || c.id == "" || c.action == "" {
		flag.Usage()
		os.Exit(1)
	}

	if !c.verbose {
		log.SetOutput(ioutil.Discard)
	}

	var configs []*aws.Config
	if c.credulous {
		configs = listCredulous()
	} else {
		configs = listProfiles()
	}

	doneChan := make(chan bool, 1)
	go func() {
		var done sync.WaitGroup
		done.Add(len(configs))

		// Loop through all of the accounts, search for instance in parallel
		for _, config := range configs {
			go func(config *aws.Config) {
				sess := session.New(config)
				svc := ec2.New(sess)
				ebSvc := eb.New(sess)

				var r interface{}
				switch strings.ToLower(c.action) {
				case "instance":
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
					doneChan <- true
				}
				done.Done()
			}(config)
		}
		done.Wait()
	}()

	// Wait up to timeout, or when first result comes back
	select {
	case <-time.After(c.timeout):
		log.Fatalf("Timeout waiting for all accounts to return")
	case <-doneChan:
		os.Exit(0)
	}
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

func listCredulous() []*aws.Config {
	accts := credulous.GetAccounts()
	configs := make([]*aws.Config, len(accts))

	for i, acct := range accts {
		key, secret, err := credulous.GetCredentials(acct.Username, acct.Account)
		checkError(err)
		config := &aws.Config{
			Credentials: credentials.NewStaticCredentials(key, secret, ""),
		}
		configs[i] = config
	}
	return configs
}

// Lists all profiles in the default ~/.aws/credentials directory
func listProfiles() []*aws.Config {
	// Make sure the config file exists
	config := os.Getenv("HOME") + "/.aws/credentials"

	if _, err := os.Stat(config); os.IsNotExist(err) {
		fmt.Println("No credentials file found at: %s", config)
		os.Exit(1)
	}

	file, _ := ini.LoadFile(config)
	configs := make([]*aws.Config, 0)

	for key, _ := range file {
		config := &aws.Config{
			Credentials: credentials.NewSharedCredentials("", key),
		}
		configs = append(configs, config)
	}

	return configs
}

func checkError(err error) {
	if err != nil {
		log.Printf("Error: ", err)
	}
}
