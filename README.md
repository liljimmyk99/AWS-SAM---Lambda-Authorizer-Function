# APIGATEWAYOAUTHINTERGRATION
The following lambda function is a generalized abstraction of a Lambda Authorizer function. A Lambda Authorizer Function is a function executed due to an API Gateway fall to check if OAuth Tokens or Header combinations should grant the user access to a resource/methodARN of the API.  These functions are customizable depending upon your organizations needs.

## Motivation

The Lambda function here solves the issue of securing AWS APIGateway endpoints using OAuth Tokens.  Not only can this function provide authentication (OAuth), it can also provides authorization.  This makes this lambda function valuable and easy to replicate.  Keep in mind this is an abstract version of a project I used in an internship.  Keep in mind that specific calls to your OAuth Server and Permissions/Authenication Server will vary based on your organization.

## Tech/framework used
- S3
- Logrus
- AWS Serverless Application Model (SAM)

## Features

The code here is already set to be used by others, the only thing that needs to be changed is the pointer to a JSON file in an S3 bucket.  Currently, there is a bug with Lambdas reading files outside of code in the SAM deployment  This project also generates IAM Policies for the user as well.

## How Authorizor Functions Work in AWS

There is a huge note, by default once a token is checked it is cached.  If the token is checked once; in future calls to the API the token is **NOT** going to be checked with the authorizor function again.  This applies with tokens both given access and denied access!

Authorizer functions are initated before the Lambda/AWS Resource being protected is invoked.  If the user is unauthorized, the lambda will not be invoked.  In future calls if the token is cached and the user has access to a resource, it invokes the resource with a minial delay.

Lastly, it is sad to report that unlike regular lambda functions which can be tested locally; Lambda Authorizer functions can **NOT** be tested locally.  When you need to test your API Locally with `sam local start-api` the authorizer function is skipped.  The only way to test it is to put it to AWS and check it when trying to hit your endpoints.  It is helpful to create a log group for your authorizer function to see what is happening.

## How This Lambda Works

This function is a Lambda function and as such it has a handle function, but unlike a regular lambda that passes in an `events.APIGatewayProxyRequest`, this function is given an `events.APIGatewayCustomAuthorizerRequest`.  It is given a Token, the Requested Resource(MethodARN), and Type(Token).  As soon as the lambda starts, it makes a call to the S3 bucket defined in the environment to read the json file into the array of structs.  If there is no failure it makes a call to the OAuth server to ensure the token is valid and to get the name of the bearer.  Next, the OAuth return along With the list of permissions we are checking(S3 JSON File) a loop is started.

In each iteration of the loop, the `isAuthorized` function is activated to check to see if the user has the permission being checked in the current iteration of the loop.  `isAuthorized` will be defined by the programmer to make a call to the authorization server.   If there are no permissions are possed by the user, an `Unauthorized` error is returned to the user.  If they do have one of the permissions; an IAM policy is generated and returned to allow the user access to the resources defined in the JSON file.

## Installation

To user this project, you must copy the `authorizer` directory to your project.  In your SAM Template with an API Follow this example:

### Globals Section of SAM Template
 ````
  Environment:
    Variables:
      JSON_S3_LOCATION_BUCKET: (InsertNameOfS3BucketHere)
      JSON_S3_LOCATION_KEY: example_permissions.json
      S3_REGION: (InsertRegionHere)
 ```` 

 ### APIGatway Definition IN SAM Template
 Refer to the **Auth** portion of this definition.  Change MyAuthFunction with the name of the authorizer function in your SAM Template
 ``````
	ApiGatewayApi:
		Type: AWS::Serverless::Api
		Properties:
		StageName: develop
		Auth:
			DefaultAuthorizer: MyOAuthTokenAuthorizer
			Authorizers: 
				MyOAuthTokenAuthorizer:
					FunctionArn: !GetAtt MyAuthFunction.Arn
 ``````
### Authorizer Function Definition IN SAM Template
`````
	MyAuthFunction:
		Type: AWS::Serverless::Function
		Properties:
		CodeUri: authorizer/
		Handler: main
		Runtime: go1.x
		Role: !Sub arn:aws:iam::${AWS::AccountId}:role/lambda-with-full-access-artisinal
`````

### Permissions JSON File Format
Abstract Name refers to the that of the permision you would like to check if the user has.  The resources section refer to the resources the user has access to if they have the permission.  Refer to the **Making events.IAMPolicyStatement objects** section of this document for an indepth description of the format of these.
```
[
    {
        "abstractName": "worklion.project.arl.modify",
        "resources": ["*"]
    },
    {
        "abstractName": "directory.person.search",
        "resources": ["arn:aws:execute-api:us-east-1:390896819235:pku9j0czgc/Prod/GET/Something/Useful/Here", "arn:aws:execute-api:us-east-1:390896819235:pku9j0czgc/Prod/PUT/Something/Useful/Here", "arn:aws:execute-api:us-east-1:390896819235:y088me8t2h/develop/GET/", arn:aws:execute-api:us-east-1:390896819235:y088me8t2h/develop/PUT/"]
    }
] 
```

### Define isAuthorized Function

## Making events.IAMPolicyStatement objects
For this example we just want to give the user access to specific endpoints of this api and nothing else.  In order to do so, we are looking for the methodARN for each endpoint we want the user to have access to.  You can find out the methodARN of the exact for a particular function by running this authorizer function and printing out events.MethodARN in your lambda handler function.

If you want to create one by hand without running your server follow the example below
[]events.IAMPolicyStatement{{
			Action:   []string{"execute-api:Invoke"},
			Effect:   effect,
			Resource: []string{"arn:aws:execute-api:us-east-1:123456789012:abc1d2/Prod/PUT/", "arn:aws:execute-api:us-east-1:123456789012:abc1d2/Prod/GET/"},
		}},
The above example gives the user access to the base / endpoint using the PUT and GET HTTP methods

[]events.IAMPolicyStatement{{
			Action:   []string{"execute-api:Invoke"},
			Effect:   effect,
			Resource: []string{"arn:aws:execute-api:us-east-1:123456789012:abc1d2/Prod/GET/Something/Useful/Here", "arn:aws:execute-api:us-east-1:123456789012:abc1d2/Prod/PUT/Something/Useful/Here"},
		}},

The above example gives the user access to the /Something/Useful/Here endpoint with the GET and PUT HTTP methods

[]events.IAMPolicyStatement{{
			Action:   []string{"execute-api:Invoke"},
			Effect:   effect,
			Resource: []string{"arn:aws:execute-api:us-east-1:123456789012:abc1d2/Prod/GET/Something/Useful/Here", "arn:aws:execute-api:us-east-1:123456789012:abc1d2/Prod/PUT/Something/Useful/Here"},
		}},

In general the format loges like this: "arn:aws:execute-api:<region>:<awsAccount>:<apiid>/<Stage>/<HTTP Method>/<Path starting after "/">

Please use [this](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_elements.html) website to find out what values are acceptable for Action, Effect, and Resource


## Resources
Please visit [this](https://github.com/awslabs/aws-apigateway-lambda-authorizer-blueprints/blob/master/blueprints/go/main.go) GitHub repo to see original sample code used for /Authorizer-function/main.go

Please visit [this](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-use-lambda-authorizer.html) website about AWS Lambda Authorizers

Please visit [this](https://aws.amazon.com/blogs/security/use-aws-lambda-authorizers-with-a-third-party-identity-provider-to-secure-amazon-api-gateway-rest-apis/) website about implemention of 3rd party identity provider into an ApiGateway

Please visit [here](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-controlling-access-to-apis.html) for AWS SAM Spec for using a Lambda Authorizer Function

Please visit [here](https://www.youtube.com/watch?v=bK1-kQSxCR0) for a view tutorial of how to do this in Node JS

https://www.alexdebrie.com/posts/lambda-custom-authorizers/