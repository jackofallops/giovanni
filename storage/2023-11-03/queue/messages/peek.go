package messages

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/odata"
)

type PeekInput struct {
	// NumberOfMessages specifies the (maximum) number of messages that should be peak'd from the front of the queue.
	// This can be a maximum of 32.
	NumberOfMessages int
}

// Peek retrieves one or more messages from the front of the queue, but doesn't alter the visibility of the messages
func (c Client) Peek(ctx context.Context, queueName string, input PeekInput) (result QueueMessagesListResponse, err error) {

	if queueName == "" {
		return result, fmt.Errorf("`queueName` cannot be an empty string")
	}

	if strings.ToLower(queueName) != queueName {
		return result, fmt.Errorf("`queueName` must be a lower-cased string")
	}

	if input.NumberOfMessages < 1 || input.NumberOfMessages > 32 {
		return result, fmt.Errorf("`input.NumberOfMessages` must be between 1 and 32")
	}

	opts := client.RequestOptions{
		ContentType: "application/xml; charset=utf-8",
		ExpectedStatusCodes: []int{
			http.StatusOK,
		},
		HttpMethod: http.MethodGet,
		OptionsObject: peekOptions{
			numberOfMessages: input.NumberOfMessages,
		},
		Path: fmt.Sprintf("/%s/messages", queueName),
	}

	req, err := c.Client.NewRequest(ctx, opts)
	if err != nil {
		err = fmt.Errorf("building request: %+v", err)
		return
	}

	var resp *client.Response
	resp, err = req.Execute(ctx)
	if resp != nil {
		result.HttpResponse = resp.Response

		err = resp.Unmarshal(&result)
		if err != nil {
			err = fmt.Errorf("unmarshalling response: %+v", err)
			return
		}
	}
	if err != nil {
		err = fmt.Errorf("executing request: %+v", err)
		return
	}

	return
}

type peekOptions struct {
	numberOfMessages int
}

func (p peekOptions) ToHeaders() *client.Headers {
	return nil
}

func (p peekOptions) ToOData() *odata.Query {
	return nil
}

func (p peekOptions) ToQuery() *client.QueryParams {
	out := &client.QueryParams{}
	out.Append("numofmessages", strconv.Itoa(p.numberOfMessages))
	out.Append("peekonly", "true")
	return out
}
