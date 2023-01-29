package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/joho/godotenv"
	"github.com/michimani/gotwi"
	"github.com/michimani/gotwi/tweet/managetweet"
	"github.com/michimani/gotwi/tweet/managetweet/types"
)

// ArticleInfo は、記事情報を定義した構造体。
type ArticleInfo struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

func main() {
	t := flag.Bool("local", false, "ローカル実行か否か")
	slug := flag.String("slug", "", "ローカル実行用の記事Slug")
	title := flag.String("title", "", "ローカル実行用の記事Title")
	flag.Parse()

	isLocal, err := isLocal(t, slug, title)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if isLocal {
		fmt.Println("local")
		ai := ArticleInfo{
			Slug:  *slug,
			Title: *title,
		}
		localController(ai)
		return
	}

	fmt.Println("production")
	lambda.Start(controller)
}

// isLocal は、ローカル環境の実行であるかを判定する。
func isLocal(t *bool, slug *string, title *string) (bool, error) {
	if !*t {
		return false, nil
	}

	if *slug == "" {
		fmt.Println("no exec")
		return false, fmt.Errorf("ローカル実行だがSlug指定が無いので処理不可能")
	}

	if *title == "" {
		fmt.Println("no exec")
		return false, fmt.Errorf("ローカル実行だがTitle指定が無いので処理不可能")
	}

	return true, nil
}

// localController は、ローカル環境での実行処理を行う。
func localController(ai ArticleInfo) {
	if err := godotenv.Load(); err != nil {
		fmt.Println(err)
		return
	}

	if err := useCase(ai); err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println("tweetしました。")
}

// controller は、API Gateway / AWS Step Functions / AWS Lambda 上での実行処理を行う。
func controller(input ArticleInfo) error {
	if input.Slug == "" {
		return fmt.Errorf("slug is empty")
	}
	if input.Title == "" {
		return fmt.Errorf("title is empty")
	}

	if err := useCase(input); err != nil {
		return err
	}

	return nil
}

// useCase は、アプリケーションのIFに依存しないメインの処理を行う。
func useCase(ai ArticleInfo) error {
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
