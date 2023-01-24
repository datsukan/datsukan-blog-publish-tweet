package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/joho/godotenv"
	"github.com/michimani/gotwi"
	"github.com/michimani/gotwi/tweet/managetweet"
	"github.com/michimani/gotwi/tweet/managetweet/types"
)

// ErrorResponse は異常系のレスポンスを定義した構造体
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ArticleInfo は、記事情報を定義した構造体。
type ArticleInfo struct {
	slug  string
	title string
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
			slug:  *slug,
			title: *title,
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
}

// controller は、API Gateway / AWS Lambda 上での実行処理を行う。
func controller(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	slug := request.PathParameters["slug"]
	if slug == "" {
		err := fmt.Errorf("slug is empty")
		return responseBadRequestError(err)
	}

	title := request.PathParameters["title"]
	if slug == "" {
		err := fmt.Errorf("title is empty")
		return responseBadRequestError(err)
	}

	ai := ArticleInfo{
		slug:  slug,
		title: title,
	}

	if err := useCase(ai); err != nil {
		return responseInternalServerError(err)
	}

	return responseSuccess()
}

// useCase は、アプリケーションのIFに依存しないメインの処理を行う。
func useCase(ai ArticleInfo) error {
	if err := godotenv.Load(); err != nil {
		return err
	}

	in := &gotwi.NewClientInput{
		AuthenticationMethod: gotwi.AuthenMethodOAuth1UserContext,
		OAuthToken:           os.Getenv("GOTWI_ACCESS_TOKEN"),
		OAuthTokenSecret:     os.Getenv("GOTWI_ACCESS_TOKEN_SECRET"),
	}

	c, err := gotwi.NewClient(in)
	if err != nil {
		return err
	}

	t := fmt.Sprintf("新しいブログ記事を投稿しました🐣\n\n「%s」\n%s%s", ai.title, os.Getenv("BLOG_URL"), ai.slug)
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

// responseBadRequestError は、リクエスト不正のレスポンスを生成する。
func responseBadRequestError(rerr error) (events.APIGatewayProxyResponse, error) {
	b := ErrorResponse{
		Error:   "bad request",
		Message: rerr.Error(),
	}
	jb, err := json.Marshal(b)
	if err != nil {
		r := events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       err.Error(),
		}
		return r, nil
	}
	body := string(jb)

	r := events.APIGatewayProxyResponse{
		StatusCode: 400,
		Body:       body,
	}
	return r, nil
}

// responseInternalServerError は、システムエラーのレスポンスを生成する。
func responseInternalServerError(rerr error) (events.APIGatewayProxyResponse, error) {
	b := ErrorResponse{
		Error:   "internal server error",
		Message: rerr.Error(),
	}
	jb, err := json.Marshal(b)
	if err != nil {
		r := events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       err.Error(),
		}
		return r, nil
	}
	body := string(jb)

	r := events.APIGatewayProxyResponse{
		StatusCode: 500,
		Body:       body,
	}
	return r, nil
}

// responseSuccess は、処理成功時のレスポンスを生成する。
func responseSuccess() (events.APIGatewayProxyResponse, error) {
	r := events.APIGatewayProxyResponse{
		StatusCode: 200,
	}
	return r, nil
}
