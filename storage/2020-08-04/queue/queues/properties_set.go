package queues

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/odata"
)

type SetStorageServicePropertiesResponse struct {
	HttpResponse *client.Response
}

type SetStorageServicePropertiesInput struct {
	properties StorageServiceProperties
}

// SetServiceProperties sets the properties for this queue
func (c Client) SetServiceProperties(ctx context.Context, input SetStorageServicePropertiesInput) (resp SetStorageServicePropertiesResponse, err error) {

	opts := client.RequestOptions{
		ContentType: "application/xml; charset=utf-8",
		ExpectedStatusCodes: []int{
			http.StatusAccepted,
		},
		HttpMethod:    http.MethodPut,
		OptionsObject: setStorageServicePropertiesOptions{},
		Path:          "/",
	}

	req, err := c.Client.NewRequest(ctx, opts)
	if err != nil {
		err = fmt.Errorf("building request: %+v", err)
		return
	}

	err = req.Marshal(&input.properties)
	if err != nil {
		return resp, fmt.Errorf("marshalling request: %v", err)
	}

	resp.HttpResponse, err = req.Execute(ctx)
	if err != nil {
		err = fmt.Errorf("executing request: %+v", err)
		return
	}

	return
}

type setStorageServicePropertiesOptions struct{}

func (s setStorageServicePropertiesOptions) ToHeaders() *client.Headers {
	return nil
}

func (s setStorageServicePropertiesOptions) ToOData() *odata.Query {
	return nil
}

func (s setStorageServicePropertiesOptions) ToQuery() *client.QueryParams {
	out := &client.QueryParams{}
	out.Append("restype", "service")
	out.Append("comp", "properties")
	return out
}
