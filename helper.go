package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-multierror"
)

func waitForIstioProxy() error {
	httpClient := &http.Client{
		Timeout: time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	var errs error
	for {
		select {
		case <-ctx.Done():
			errs = multierror.Append(errs, ctx.Err())
			return errs
		default:
			if err := checkIstioProxy(ctx, httpClient); err != nil {
				errs = multierror.Append(errs, err)
				switch err {
				case context.Canceled, context.DeadlineExceeded:
					return errs
				default:
					time.Sleep(250 * time.Millisecond)
					continue
				}
			}
			return nil
		}
	}
}

func checkIstioProxy(ctx context.Context, httpClient *http.Client) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:15021/healthz/ready", nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("istio proxy return non 2xx: %d", resp.StatusCode)
	}

	return nil
}
