package main

import (
	"fmt"
	"os"

	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/resources/subscriptions"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

// myTenantClient allows us to add functionality to a basic TenantClient.
type myTenantClient struct {
	subscriptions.TenantsClient
}

// myListPreparer
func (client myTenantClient) myListPreparer() (request *http.Request, err error) {
	request, err = client.ListPreparer()
	if request.Header == nil {
		request.Header = make(map[string][]string)
	}
	request.Header.Add("Accept-Language", "en-US")
	request.Header.Add("Accept-Language", "en")
	request.Header.Add("Accept-Language", "*")
	return
}

// myList gets the tenants for your account, but with an additional Header sent.
func (client myTenantClient) myList() (result subscriptions.TenantListResult, err error) {
	req, err := client.myListPreparer()
	if err != nil {
		err = autorest.NewErrorWithError(err, "subscriptions.TenantsClient", "List", nil, "Failure preparing request")
		return
	}

	fmt.Println("Request Headers:")
	for k, v := range req.Header {
		fmt.Println("\t", k, v)
	}

	resp, err := client.ListSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "subscriptions.TenantsClient", "List", resp, "Failure sending request")
		return
	}

	result, err = client.ListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "subscriptions.TenantsClient", "List", resp, "Failure responding to request")
	}

	return
}

func main() {
	// Basic Program Setup
	var err error
	exitStatus := 1
	defer func() {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(exitStatus)
	}()

	// Begin Authentication Against Azure
	var config *adal.OAuthConfig
	config, err = adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, "common")
	if err != nil {
		return
	}

	const azureCLIClientID = "04b07795-8ddb-461a-bbee-02f9e1bf7b46" // We'll use this because of it's first party privledge and public well-known status.

	authClient := autorest.NewClientWithUserAgent("Go github.com/marstr/custom-header")

	var deviceCode *adal.DeviceCode
	deviceCode, err = adal.InitiateDeviceAuth(authClient, *config, azureCLIClientID, azure.PublicCloud.ServiceManagementEndpoint)
	if err != nil {
		return
	}
	_, err = fmt.Println(*deviceCode.Message)
	if err != nil {
		return
	}

	var tenantAgnosticAuthorizer autorest.Authorizer
	var tenantAgnosticToken *adal.Token
	tenantAgnosticToken, err = adal.WaitForUserCompletion(authClient, deviceCode)
	if err != nil {
		return
	}
	tenantAgnosticAuthorizer = autorest.NewBearerAuthorizer(tenantAgnosticToken)

	// List all tenants associated with the user who just logged in.
	clientTenant := myTenantClient{subscriptions.NewTenantsClient()}
	clientTenant.Authorizer = tenantAgnosticAuthorizer

	var tenantListChunk subscriptions.TenantListResult
	tenantListChunk, err = clientTenant.myList()
	if err != nil {
		return
	}

	if tenantListChunk.NextLink == nil {
		fmt.Printf("You're associated with %d tenants.\n", len(*tenantListChunk.Value))
	} else {
		fmt.Printf("You're associated with at least %d tenants.\n", len(*tenantListChunk.Value))
	}

	exitStatus = 0
}
