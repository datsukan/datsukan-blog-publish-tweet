package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	attribute "github.com/datsukan/datsukan-blog-article-attribute"
	"github.com/joho/godotenv"
	"github.com/michimani/gotwi"
	"github.com/michimani/gotwi/tweet/managetweet"
	"github.com/michimani/gotwi/tweet/managetweet/types"
)

// Input は、リクエスト情報の構造体。
type Input struct {
	Token string `json:"token"`
	ID    string `json:"id"`
}

// ArticleInfo は、記事情報を定義した構造体。
type ArticleInfo struct {
	Slug  string
	Title string
}

func main() {
	t := flag.Bool("local", false, "ローカル実行か否か")
	id := flag.String("id", "", "ローカル実行用の記事ID")
	flag.Parse()

	isLocal, err := isLocal(t, id)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if isLocal {
		fmt.Println("local")
		localController(*id)
		return
	}

	fmt.Println("production")
	lambda.Start(controller)
}

// isLocal は、ローカル環境の実行であるかを判定する。
func isLocal(t *bool, id *string) (bool, error) {
	if !*t {
		return false, nil
	}

	if *id == "" {
		fmt.Println("no exec")
		return false, fmt.Errorf("ローカル実行だがID指定が無いので処理不可能")
	}

	return true, nil
}

// localController は、ローカル環境での実行処理を行う。
func localController(id string) {
	if err := godotenv.Load(); err != nil {
		fmt.Println(err)
		return
	}

	accessToken, spaceID, err := loadContentfulEnv()
	if err != nil {
		fmt.Println(err)
		return
	}

	aa, err := attribute.New(id, accessToken, spaceID)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := aa.Get(); err != nil {
		fmt.Println(err)
		return
	}

	ai := &ArticleInfo{
		Slug:  aa.Slug,
		Title: aa.Title,
	}

	if err := ai.useCase(); err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("tweetしました。")
}

// controller は、API Gateway / AWS Step Functions / AWS Lambda 上での実行処理を行う。
func controller(input Input) error {
	if input.Token != os.Getenv("API_TOKEN") {
		return fmt.Errorf("unauthorized access")
	}

	if input.ID == "" {
		return fmt.Errorf("id is empty")
	}

	accessToken, spaceID, err := loadContentfulEnv()
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("internal server error")
	}

	aa, err := attribute.New(input.ID, accessToken, spaceID)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("internal server error")
	}

	if err := aa.Get(); err != nil {
		fmt.Println(err)
		return fmt.Errorf("internal server error")
	}

	ai := &ArticleInfo{
		Slug:  aa.Slug,
		Title: aa.Title,
	}

	if err := ai.useCase(); err != nil {
		return err
	}

	return nil
}

// useCase は、アプリケーションのIFに依存しないメインの処理を行う。
func (ai *ArticleInfo) useCase() error {

	in := &gotwi.NewClientInput{
		AuthenticationMethod: gotwi.AuthenMethodOAuth1UserContext,
		OAuthToken:           os.Getenv("GOTWI_ACCESS_TOKEN"),
		OAuthTokenSecret:     os.Getenv("GOTWI_ACCESS_TOKEN_SECRET"),
	}

	c, err := gotwi.NewClient(in)
	if err != nil {
		return err
	}

	t := fmt.Sprintf("新しいブログ記事を投稿しました🐣\n\n「%s」\n%s%s", ai.Title, os.Getenv("BLOG_URL"), ai.Slug)
	if _, err := tweet(c, t); err != nil {
		return err
	}

	return nil
}

// tweet は、指定されたテキストをツイートする。
func tweet(c *gotwi.Client, text string) (string, error) {
	p := &types.CreateInput{
		Text: gotwi.String(text),
	}

	res, err := managetweet.Create(context.Background(), c, p)
	if err != nil {
		return "", err
	}

	return gotwi.StringValue(res.Data.ID), nil
}

// loadContentfulEnv は、 Contentful SDK の接続情報を環境変数から読み込む。
func loadContentfulEnv() (string, string, error) {
	accessToken := os.Getenv("CONTENTFUL_ACCESS_TOKEN")
	spaceID := os.Getenv("CONTENTFUL_SPACE_ID")

	if accessToken == "" || spaceID == "" {
		m := fmt.Sprintf("environment variable not set [ accessToken: %v, spaceID: %v ]", accessToken, spaceID)
		return "", "", errors.New(m)
	}

	return accessToken, spaceID, nil
}
