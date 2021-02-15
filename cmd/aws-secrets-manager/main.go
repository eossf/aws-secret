package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

func main() {
	info("Enter into main()")
	secretArn := os.Getenv("SECRET_ARN")
	fmt.Println("secretArn:", secretArn)
	var AWSRegion string
	if arn.IsARN(secretArn) {
		arnobj, _ := arn.Parse(secretArn)
		AWSRegion = arnobj.Region
	} else {
		info("Not a valid ARN")
		os.Exit(1)
	}

	sess, err := session.NewSession()
	svc := secretsmanager.New(sess, &aws.Config{
		Region: aws.String(AWSRegion),
	})

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretArn),
		VersionStage: aws.String("AWSCURRENT"),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case secretsmanager.ErrCodeResourceNotFoundException:
				fmt.Println(secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
			case secretsmanager.ErrCodeInvalidParameterException:
				fmt.Println(secretsmanager.ErrCodeInvalidParameterException, aerr.Error())
			case secretsmanager.ErrCodeInvalidRequestException:
				fmt.Println(secretsmanager.ErrCodeInvalidRequestException, aerr.Error())
			case secretsmanager.ErrCodeDecryptionFailure:
				fmt.Println(secretsmanager.ErrCodeDecryptionFailure, aerr.Error())
			case secretsmanager.ErrCodeInternalServiceError:
				fmt.Println(secretsmanager.ErrCodeInternalServiceError, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString, decodedBinarySecret string
	if result.SecretString != nil {
		secretString = *result.SecretString
		writeOutput(secretString)
	} else {
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			info("Base64 Decode Error")
			fmt.Println(err)
			return
		}
		decodedBinarySecret = string(decodedBinarySecretBytes[:len])
		writeOutput(decodedBinarySecret)
	}
}
func writeOutput(output string) {
	info("output string ")
	info(output)
	f, err := os.Create("/tmp/secret")
	if err != nil {
		return
	}
	defer f.Close()

	f.WriteString(output)
}

func info(txt string) {
	currentTime := time.Now()
	fmt.Println(currentTime.Format("2006-01-02 15:04:05"), " INFO ", txt)
}
