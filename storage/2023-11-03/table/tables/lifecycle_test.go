package tables

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/hashicorp/go-azure-sdk/resource-manager/storage/2023-01-01/storageaccounts"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/jackofallops/giovanni/storage/internal/testhelpers"
)

var _ StorageTable = Client{}

func TestTablesLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()

	client, err := testhelpers.Build(ctx, t)
	if err != nil {
		t.Fatal(err)
	}

	resourceGroup := fmt.Sprintf("acctestrg-%d", testhelpers.RandomInt())
	accountName := fmt.Sprintf("acctestsa%s", testhelpers.RandomString())
	tableName := fmt.Sprintf("table%d", testhelpers.RandomInt())

	testData, err := client.BuildTestResources(ctx, resourceGroup, accountName, storageaccounts.KindStorage)
	if err != nil {
		t.Fatal(err)
	}
	defer client.DestroyTestResources(ctx, resourceGroup, accountName)

	domainSuffix, ok := client.Environment.Storage.DomainSuffix()
	if !ok {
		t.Fatalf("storage didn't return a domain suffix for this environment")
	}
	tablesClient, err := NewWithBaseUri(fmt.Sprintf("https://%s.%s.%s", accountName, "table", *domainSuffix))
	if err != nil {
		t.Fatalf("building client for environment: %+v", err)
	}

	if err := client.PrepareWithSharedKeyAuth(tablesClient.Client, testData, auth.SharedKeyTable); err != nil {
		t.Fatalf("adding authorizer to client: %+v", err)
	}

	t.Logf("[DEBUG] Creating Table..")
	if _, err := tablesClient.Create(ctx, tableName); err != nil {
		t.Fatalf("Error creating Table %q: %s", tableName, err)
	}

	// first look it up directly and confirm it's there
	t.Logf("[DEBUG] Checking if Table exists..")
	if _, err := tablesClient.Exists(ctx, tableName); err != nil {
		t.Fatalf("Error checking if Table %q exists: %s", tableName, err)
	}

	// then confirm it exists in the Query too
	t.Logf("[DEBUG] Querying for Tables..")
	result, err := tablesClient.Query(ctx, QueryInput{MetaDataLevel: NoMetaData})
	if err != nil {
		t.Fatalf("Error retrieving Tables: %s", err)
	}
	found := false
	for _, v := range result.Tables {
		log.Printf("[DEBUG] Table: %q", v.TableName)

		if v.TableName == tableName {
			found = true
		}
	}
	if !found {
		t.Fatalf("%q was not found in the Query response!", tableName)
	}

	t.Logf("[DEBUG] Setting ACL's for Table %q..", tableName)
	acls := []SignedIdentifier{
		{
			Id: "MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=",
			AccessPolicy: AccessPolicy{
				Permission: "raud",
				Start:      "2020-11-26T08:49:37.0000000Z",
				Expiry:     "2020-11-27T08:49:37.0000000Z",
			},
		},
	}
	if _, err := tablesClient.SetACL(ctx, tableName, acls); err != nil {
		t.Fatalf("Error setting ACLs: %s", err)
	}

	t.Logf("[DEBUG] Retrieving ACL's for Table %q..", tableName)
	retrievedACLs, err := tablesClient.GetACL(ctx, tableName)
	if err != nil {
		t.Fatalf("Error retrieving ACLs: %s", err)
	}

	if len(retrievedACLs.SignedIdentifiers) != len(acls) {
		t.Fatalf("Expected %d but got %q ACLs", len(acls), len(retrievedACLs.SignedIdentifiers))
	}

	for i, retrievedAcl := range retrievedACLs.SignedIdentifiers {
		expectedAcl := acls[i]

		if retrievedAcl.Id != expectedAcl.Id {
			t.Fatalf("Expected ID to be %q but got %q", expectedAcl.Id, retrievedAcl.Id)
		}

		if retrievedAcl.AccessPolicy.Start != expectedAcl.AccessPolicy.Start {
			t.Fatalf("Expected Start to be %q but got %q", expectedAcl.AccessPolicy.Start, retrievedAcl.AccessPolicy.Start)
		}

		if retrievedAcl.AccessPolicy.Expiry != expectedAcl.AccessPolicy.Expiry {
			t.Fatalf("Expected Expiry to be %q but got %q", expectedAcl.AccessPolicy.Expiry, retrievedAcl.AccessPolicy.Expiry)
		}

		if retrievedAcl.AccessPolicy.Permission != expectedAcl.AccessPolicy.Permission {
			t.Fatalf("Expected Permission to be %q but got %q", expectedAcl.AccessPolicy.Permission, retrievedAcl.AccessPolicy.Permission)
		}
	}

	// set some properties
	props := StorageServiceProperties{
		Logging: &LoggingConfig{
			Version: "1.0",
			Delete:  true,
			Read:    true,
			Write:   true,
			RetentionPolicy: RetentionPolicy{
				Enabled: true,
				Days:    7,
			},
		},
		Cors: &Cors{
			CorsRule: []CorsRule{
				{
					AllowedMethods:  "GET,PUT",
					AllowedOrigins:  "http://www.example.com",
					ExposedHeaders:  "x-tempo-*",
					AllowedHeaders:  "x-tempo-*",
					MaxAgeInSeconds: 500,
				},
				{
					AllowedMethods:  "POST",
					AllowedOrigins:  "http://www.test.com",
					ExposedHeaders:  "*",
					AllowedHeaders:  "x-method-*",
					MaxAgeInSeconds: 200,
				},
			},
		},
		HourMetrics: &MetricsConfig{
			Version: "1.0",
			Enabled: false,
			RetentionPolicy: RetentionPolicy{
				Enabled: true,
				Days:    7,
			},
		},
		MinuteMetrics: &MetricsConfig{
			Version: "1.0",
			Enabled: false,
			RetentionPolicy: RetentionPolicy{
				Enabled: true,
				Days:    7,
			},
		},
	}

	_, err = tablesClient.SetServiceProperties(ctx, SetStorageServicePropertiesInput{Properties: props})
	if err != nil {
		t.Fatalf("SetServiceProperties failed: %s", err)
	}

	// tables take a while to apply properties, so unfortunately...
	time.Sleep(15 * time.Second)

	properties, err := tablesClient.GetServiceProperties(ctx)
	if err != nil {
		t.Fatalf("GetServiceProperties failed: %s", err)
	}

	if len(properties.Cors.CorsRule) > 1 {
		if properties.Cors.CorsRule[0].AllowedMethods != "GET,PUT" {
			t.Fatalf("CORS Methods weren't set!")
		}
		if properties.Cors.CorsRule[1].AllowedMethods != "POST" {
			t.Fatalf("CORS Methods weren't set!")
		}
	} else {
		t.Fatalf("CORS Methods weren't set!")
	}

	if properties.HourMetrics.Enabled {
		t.Fatalf("HourMetrics were enabled when they shouldn't be!")
	}

	if properties.MinuteMetrics.Enabled {
		t.Fatalf("MinuteMetrics were enabled when they shouldn't be!")
	}

	if !properties.Logging.Write {
		t.Fatalf("Logging Write's was not enabled when they should be!")
	}

	includeAPIS := true
	// set some properties
	props2 := StorageServiceProperties{
		Logging: &LoggingConfig{
			Version: "1.0",
			Delete:  true,
			Read:    true,
			Write:   true,
			RetentionPolicy: RetentionPolicy{
				Enabled: true,
				Days:    7,
			},
		},
		Cors: &Cors{
			CorsRule: []CorsRule{
				{
					AllowedMethods:  "PUT",
					AllowedOrigins:  "http://www.example.com",
					ExposedHeaders:  "x-tempo-*",
					AllowedHeaders:  "x-tempo-*",
					MaxAgeInSeconds: 500,
				},
			},
		},
		HourMetrics: &MetricsConfig{
			Version: "1.0",
			Enabled: true,
			RetentionPolicy: RetentionPolicy{
				Enabled: true,
				Days:    7,
			},
			IncludeAPIs: &includeAPIS,
		},
		MinuteMetrics: &MetricsConfig{
			Version: "1.0",
			Enabled: false,
			RetentionPolicy: RetentionPolicy{
				Enabled: true,
				Days:    7,
			},
		},
	}

	_, err = tablesClient.SetServiceProperties(ctx, SetStorageServicePropertiesInput{Properties: props2})
	if err != nil {
		t.Fatalf("SetServiceProperties failed: %s", err)
	}

	// tables take a while to apply properties, so unfortunately...
	time.Sleep(15 * time.Second)

	properties, err = tablesClient.GetServiceProperties(ctx)
	if err != nil {
		t.Fatalf("GetServiceProperties failed: %s", err)
	}

	if len(properties.Cors.CorsRule) == 1 {
		if properties.Cors.CorsRule[0].AllowedMethods != "PUT" {
			t.Fatalf("CORS Methods weren't set!")
		}
	} else {
		t.Fatalf("CORS Methods weren't set!")
	}

	if !properties.HourMetrics.Enabled {
		t.Fatalf("HourMetrics were enabled when they shouldn't be!")
	}

	if properties.MinuteMetrics.Enabled {
		t.Fatalf("MinuteMetrics were enabled when they shouldn't be!")
	}

	if !properties.Logging.Write {
		t.Fatalf("Logging Write's was not enabled when they should be!")
	}

	log.Printf("[DEBUG] Deleting..")

	t.Logf("[DEBUG] Deleting Table %q..", tableName)
	if _, err := tablesClient.Delete(ctx, tableName); err != nil {
		t.Fatalf("Error deleting %q: %s", tableName, err)
	}
}
