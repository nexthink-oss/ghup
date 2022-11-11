package auth

import (
	"context"
	"fmt"

	"github.com/apex/log"
	"github.com/google/go-github/v48/github"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

func NewTokenClient(ctx context.Context) (*github.Client, error) {
	if token := viper.GetString("token"); token != "" {
		switch token[0:4] {
		case "ghp_":
			log.Debug("detected Personal Access Token; commits will not be signed")
		case "ghs_":
			log.Debug("detected GitHub App-derived Token")
		default:
			return nil, fmt.Errorf("invalid/unknown GitHub Token specified")
		}
		src := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient := oauth2.NewClient(ctx, src)
		return github.NewClient(httpClient), nil
	}
	return nil, fmt.Errorf("no GitHub Token found")
}
