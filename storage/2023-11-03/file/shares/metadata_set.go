package shares

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/odata"
	"github.com/jackofallops/giovanni/storage/internal/metadata"
)

type SetMetaDataResponse struct {
	HttpResponse *http.Response
}

type SetMetaDataInput struct {
	MetaData map[string]string
}

// SetMetaData sets the MetaData on the specified Storage Share
func (c Client) SetMetaData(ctx context.Context, shareName string, input SetMetaDataInput) (result SetMetaDataResponse, err error) {

	if shareName == "" {
		err = fmt.Errorf("`shareName` cannot be an empty string")
		return
	}

	if strings.ToLower(shareName) != shareName {
		err = fmt.Errorf("`shareName` must be a lower-cased string")
		return
	}

	if err = metadata.Validate(input.MetaData); err != nil {
		err = fmt.Errorf("`metadata` is not valid: %+v", err)
		return
	}

	opts := client.RequestOptions{
		ContentType: "application/xml; charset=utf-8",
		ExpectedStatusCodes: []int{
			http.StatusOK,
		},
		HttpMethod: http.MethodPut,
		OptionsObject: SetMetaDataOptions{
			metaData: input.MetaData,
		},
		Path: fmt.Sprintf("/%s", shareName),
	}

	req, err := c.Client.NewRequest(ctx, opts)
	if err != nil {
		err = fmt.Errorf("building request: %+v", err)
		return
	}

	var resp *client.Response
	resp, err = req.Execute(ctx)
	if resp != nil && resp.Response != nil {
		result.HttpResponse = resp.Response
	}
	if err != nil {
		err = fmt.Errorf("executing request: %+v", err)
		return
	}

	return
}

type SetMetaDataOptions struct {
	metaData map[string]string
}

func (s SetMetaDataOptions) ToHeaders() *client.Headers {
	headers := metadata.SetMetaDataHeaders(s.metaData)
	return &headers
}

func (s SetMetaDataOptions) ToOData() *odata.Query {
	return nil
}

func (s SetMetaDataOptions) ToQuery() *client.QueryParams {
	out := &client.QueryParams{}
	out.Append("restype", "share")
	out.Append("comp", "metadata")
	return out
}
