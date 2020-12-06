package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type PermissionAndResourseCombos struct {
	//AbstractName is the name of the Permission the user should possess.
	AbstractName string `json: "abstractName"`

	//Resources is an array of APIGateway AWS ARN which relate to endpoints for the APIGateway
	//For example rn:aws:execute-api:<Region of API>:<AWS Account of API>:<API ID>/<STAGE>/<HTTP Method>/<Path to API from base "/">"
	Resources []string `json:"resources"`
}

var (
	S3_REGION               string
	JSON_S3_LOCATION_KEY    string
	JSON_S3_LOCATION_BUCKET string
)

func main() {
	log.SetFormatter(&logrus.TextFormatter{
		DisableColors:          false,
		ForceColors:            true,
		FullTimestamp:          true,
		DisableLevelTruncation: true,
	})
	S3_REGION = os.Getenv("S3_REGION")
	JSON_S3_LOCATION_BUCKET = os.Getenv("JSON_S3_LOCATION_BUCKET")
	JSON_S3_LOCATION_KEY = os.Getenv("JSON_S3_LOCATION_KEY")
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context, event events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	log.WithFields(logrus.Fields{"Type": event.Type, "MethodARN": event.MethodArn, "Token": event.AuthorizationToken}).Info("handleRequest function activated")
	token := event.AuthorizationToken

	//Getting Permissions from File
	perms, err := readPermissionJSON()
	if err != nil {
		return events.APIGatewayCustomAuthorizerResponse{}, errors.New("Internal Server Error")
	}

	//Contact OAuth server to ensure active token
	activeToken, oauthReturn, err := validateToken(token)
	if err != nil {
		return events.APIGatewayCustomAuthorizerResponse{}, errors.New("Unauthorized")
	}

	//Checking to see if Token is Not active
	if !activeToken {
		return events.APIGatewayCustomAuthorizerResponse{}, errors.New("UnAuthorized, token is expired")
	}
	//Checking for Permission; This is the permission with the most access rights
	policyDocument, err := permissionValidation(oauthReturn, perms)

	if err != nil {
		return events.APIGatewayCustomAuthorizerResponse{}, err
	}
	log.Info("Verified Access")
	return generateAuthResponse(oauthReturn, policyDocument), nil

}

func readPermissionJSON() ([]PermissionAndResourseCombos, error) {
	log.Info("readPermissionJSON function activated")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(S3_REGION)},
	)
	if err != nil {
		panic(err)
	}
	s3Client := s3.New(sess)

	requestInput := &s3.GetObjectInput{
		Bucket: aws.String(JSON_S3_LOCATION_BUCKET),
		Key:    aws.String(JSON_S3_LOCATION_KEY),
	}

	result, err := s3Client.GetObject(requestInput)
	if err != nil {
		fmt.Println(err)
	}
	defer result.Body.Close()
	body1, err := ioutil.ReadAll(result.Body)
	if err != nil {
		fmt.Println(err)
	}
	bodyString1 := fmt.Sprintf("%s", body1)

	var s3data []PermissionAndResourseCombos
	decoder := json.NewDecoder(strings.NewReader(bodyString1))
	err = decoder.Decode(&s3data)
	if err != nil {
		fmt.Println("twas an error")
	}

	log.WithField("Permissions", s3data).Info("Read in Permissions")
	return s3data, nil
}

func validateToken(userToken string) (bool, string, error) {
	log.WithField("Token", userToken).Info("validateToken function activated")

	//Call OAuth Server to Ensure Token is Valid

	//If Valid
	return true, "PrincipalID", nil

	//If invalid, expired, or error

	return false, "", errors.New("Unauthorized")

}

func permissionValidation(oauthResp string, perms []PermissionAndResourseCombos) (events.APIGatewayCustomAuthorizerPolicy, error) {
	log.WithField("OAuthResponse", oauthResp).Info("permissionValidation function activated")

	var indexValidPermission int
	//Loop Through Each permissionAndResource Combo
	for index, element := range perms {

		//For each iteration, checking if user has permission
		log.WithField("Abstract Name", element.AbstractName).Info("Running IsAuthorized for Permission")
		log.WithField("Resources Available", element.Resources).Info("Resource Possible")
		authorized, _ := isAuthorized(context.Background(), "princicpalID", "Permission_Name")
		log.WithField("authorized", authorized).Info("Return from IsAuthorized")

		if authorized == true {
			indexValidPermission = index
			break
		}

		if index == len(perms) {
			//If at the last permission in the array.  Meaning the person does not have access to this resource
			return events.APIGatewayCustomAuthorizerPolicy{}, errors.New("UnAuthorized")
		}
	}

	confirmedPermission := perms[indexValidPermission]
	return generatePolicyDocument("Allow", confirmedPermission.Resources), nil
}

func isAuthorized(ctx context.Context, principalID, permissionToCheck string) (bool, error) {
	log.WithFields(logrus.Fields{"context": ctx, "principalID": principalID, "permissionToCheck": permissionToCheck}).Info("isAuthorized function activated")

	//Make call to Authorization Server to check if principalID has permissionToCheck

	//If true
	return true, nil

	//else
	return false, errors.New("An Error Occured")
}

func generateAuthResponse(principalID string, policyDocument events.APIGatewayCustomAuthorizerPolicy) events.APIGatewayCustomAuthorizerResponse {
	log.WithFields(logrus.Fields{"principalID": principalID, "PolicyDocument": policyDocument}).Info("generateAuthResponse function activated")

	return events.APIGatewayCustomAuthorizerResponse{
		PrincipalID:    principalID,
		PolicyDocument: policyDocument,
	}
}

func generatePolicyDocument(effect string, resources []string) events.APIGatewayCustomAuthorizerPolicy {
	log.WithFields(logrus.Fields{"Effect": effect, "Resources": resources}).Info("generatePolicyDocument function activated")
	policyDocument := events.APIGatewayCustomAuthorizerPolicy{
		Version: "2012-10-17",
		Statement: []events.IAMPolicyStatement{{
			Action:   []string{"execute-api:Invoke"},
			Effect:   effect,
			Resource: resources,
		}},
	}
	log.WithField("Policy Document", policyDocument).Info("Policy Document created")
	return policyDocument
}
