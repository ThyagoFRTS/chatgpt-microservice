package main

import (
	"database/sql"
	"fmt"

	"github.com/ThyagoFRTS/chatgpt-microservice/chatservice/configs"
	"github.com/ThyagoFRTS/chatgpt-microservice/chatservice/internal/infra/grpc/server"
	"github.com/ThyagoFRTS/chatgpt-microservice/chatservice/internal/infra/repository"
	"github.com/ThyagoFRTS/chatgpt-microservice/chatservice/internal/infra/web"
	"github.com/ThyagoFRTS/chatgpt-microservice/chatservice/internal/infra/web/webserver"
	"github.com/ThyagoFRTS/chatgpt-microservice/chatservice/internal/usecase/chatcompletion"
	_ "github.com/go-sql-driver/mysql"

	"github.com/ThyagoFRTS/chatgpt-microservice/chatservice/internal/usecase/chatcompletionstream"
	"github.com/sashabaranov/go-openai"
)

func main() {
	configs, err := configs.LoadConfig(".")
	if err != nil {
		panic(err)
	}

	conn, err := sql.Open(configs.DBDriver, fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		configs.DBUser, configs.DBPassword, configs.DBHost, configs.DBPort, configs.DBName))
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	repo := repository.NewChatRepositoryMySQL(conn)
	client := openai.NewClient(configs.OpenAIApiKey)

	chatConfig := chatcompletion.ChatCompletionConfigInputDTO{
		Model:                configs.Model,
		ModelMaxTokens:       configs.ModelMaxTokens,
		Temperature:          float32(configs.Temperature),
		TopP:                 float32(configs.TopP),
		N:                    configs.N,
		Stop:                 configs.Stop,
		MaxTokens:            configs.MaxTokens,
		InitialSystemMessage: configs.InitialChatMessage,
	}

	chatConfigStream := chatcompletionstream.ChatCompletionConfigInputDTO{
		Model:                configs.Model,
		ModelMaxTokens:       configs.ModelMaxTokens,
		Temperature:          float32(configs.Temperature),
		TopP:                 float32(configs.TopP),
		N:                    configs.N,
		Stop:                 configs.Stop,
		MaxTokens:            configs.MaxTokens,
		InitialSystemMessage: configs.InitialChatMessage,
	}

	usecase := chatcompletion.NewChatCompletionUseCase(repo, client)

	streamChannel := make(chan chatcompletionstream.ChatCompletionOutputDTO)
	usecaseStream := chatcompletionstream.NewChatCompletionUseCase(repo, client, streamChannel)

	grpcServer := server.NewGRPCServer(*usecaseStream, chatConfigStream, configs.GRPCServerPort, configs.AuthToken, streamChannel)

	fmt.Println("Starting GRPC server on port " + configs.GRPCServerPort)
	go grpcServer.Start()

	webserver := webserver.NewWebServer(":" + configs.WebServerPort)

	webserverChatHandler := web.NewWebChatGPTHandler(*usecase, chatConfig, configs.AuthToken)

	webserver.AddHandler("/chat", webserverChatHandler.Handle)

	fmt.Println("Starting server on port " + configs.WebServerPort)
	webserver.Start()

}
