package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/kataras/iris/v12"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

var background = context.Background()

const QueueName = "hello"

func healthCheckRedis(ctx iris.Context) {
	redisAddr := getEnv("REDIS", "localhost:6379")
	log.Info().Msgf("Access Redis under %s", redisAddr)

	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	defer func() {
		_ = client.Close()
	}()

	pong, err := client.Ping(background).Result()
	if err != nil {
		log.Fatal().Msgf("Ping to Redis failed with: %s", err)
	}
	log.Info().Msgf("Redis returned %s", pong)

	if _, err := ctx.Writef("Redis answer is: %s", pong); err != nil {
		log.Fatal().Msgf("Write Iris Context failed with: %s", err)
	}
}

func helloView(ctx iris.Context) {
	ctx.ViewData("message", "Hello World")
	_ = ctx.View("hi.html")
}

func initRabbitMQ(queueName string) (*amqp.Channel, amqp.Queue) {
	host := getEnv("RABBITMQ", "localhost:5672")

	var conn *amqp.Connection = nil
	for i := 0; i < 5; i++ {
		var err error
		log.Debug().Msgf("Connect to RabbitMQ host %s", host)
		conn, err = amqp.Dial(fmt.Sprintf("amqp://guest:guest@%s", host))
		if err != nil {
			log.Warn().Msgf("Dial to RabbitMQ failed with %w", err)
		} else {
			break
		}
		time.Sleep(10 * time.Second)
	}
	if conn == nil {
		log.Fatal().Msgf("Dial to RabbitMQ failed after several retries")
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal().Msgf("Open Channel to RabbitMQ failed with %w", err)
	}

	q, err := ch.QueueDeclare(queueName, false, false, false, false, nil)
	if err != nil {
		log.Fatal().Msgf("Declare Queue to RabbitMQ failed with %w", err)
	}

	return ch, q
}

func broadcastMsg(ctx iris.Context) {
	msg := ctx.Params().Get("msg")
	if msg != "" {
		ch, q := initRabbitMQ(QueueName)
		defer func() { _ = ch.Close() }()
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := ch.PublishWithContext(
			timeoutCtx,
			"",
			q.Name,
			false,
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "text/plain",
				Body:         []byte(msg),
			})
		if err != nil {
			log.Fatal().Msgf("Publish message %s to %s failed with %w", msg, q.Name, err)
		}
		log.Info().Msgf("Published %s to %s", msg, q.Name)
	}
	ctx.StatusCode(http.StatusOK)
}

// Setup receiver (this may the same process as the sender)
func consumeBroadcast() {
	ch, q := initRabbitMQ(QueueName)
	defer func() { _ = ch.Close() }()
	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatal().Msgf("Consume from %s failed with %s", q.Name, err)
	}
	log.Info().Msgf("Consume forever at %s", q.Name)
	forever := make(chan bool)
	go func() {
		for d := range msgs {
			_ = d.Ack(true)
			log.Printf("Received: %s", d.Body)
			if string(d.Body) == "quit" {
				forever <- true
			}
		}
	}()
	<-forever
}

func broadcastEcho(ctx iris.Context) {
	msg := ctx.Params().Get("msg")

	ch, q := initRabbitMQ("echo")
	defer func() { _ = ch.Close() }()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ch.PublishWithContext(timeoutCtx,
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(msg),
		})
	if err != nil {
		log.Fatal().Msgf("Publish message %s to %s failed with %w", msg, q.Name, err)
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatal().Msgf("Consume from %s failed with %s", q.Name, err)
	}

	var d = <-msgs
	response := string(d.Body)
	log.Info().Msgf("Received %s from %s", response, q.Name)
	_, err = ctx.WriteString(response)
	if err != nil {
		log.Fatal().Msgf("Write to Iris context failed with %w", err)
	}

}

func createApp(logger io.Writer) *iris.Application {
	app := iris.New()
	app.Logger().SetOutput(logger)

	// Specify and load templates
	tmpl := iris.HTML(getEnv("TEMPLATES", "./views"), ".html")
	tmpl.Reload(true)

	// Add a greeting function
	tmpl.AddFunc("greet", func(s string) string {
		return "Greeting " + s + "!"
	})

	// Register the view engine
	app.RegisterView(tmpl)

	// Define the Hello API
	helloAPI := app.Party("/hello")
	{
		helloAPI.Get("/", helloView)
	}

	// Define the Health APIs
	healthAPI := app.Party("/health")
	{
		healthAPI.Get("/ping", func(ctx iris.Context) {
			if _, err := ctx.WriteString("pong"); err != nil {
				log.Fatal().Msgf("Write Iris Context failed: %s", err)
			}
		})
		healthAPI.Get("/redis", healthCheckRedis)
	}

	// REST API
	serviceAPI := app.Party("/api")
	{
		serviceAPI.Get("/broadcast/{msg}", broadcastMsg)
		serviceAPI.Get("/echo/{msg}", broadcastEcho)
		serviceAPI.Get("/greeting/{name}", func(ctx iris.Context) {
			name := ctx.Params().Get("name")
			if _, err := ctx.Writef("Hello %s", name); err != nil {
				log.Fatal().Msgf("Write Iris Context failed: %s", err)
			}
		})
		serviceAPI.Get("/add", func(ctx iris.Context) {
			a, _ := strconv.Atoi(ctx.URLParamDefault("a", "0"))
			b, _ := strconv.Atoi(ctx.URLParamDefault("b", "0"))
			sum := a + b
			log.Debug().Msgf("/add %d + %d = %d", a, b, sum)
			response := struct {
				Sum int `json:"sum"`
			}{Sum: sum}
			if _, err := ctx.JSON(response); err != nil {
				log.Fatal().Msgf("Write JSON failed: %s", err)
			}
		})
	}

	log.Info().Msg("Iris Framework is ready")
	return app
}

func initLogger() *os.File {
	writer := os.Stdout
	env := os.Getenv("LOGS")
	if env != "" {
		logFile := fmt.Sprintf("%s/hello-app.log", env)
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY, 0755)
		if err == nil {
			fmt.Println("Write logs into ", logFile)
			writer = file
		} else {
			fmt.Printf("Open log file failed %s\n", err)
		}
	}
	log.Logger = zerolog.New(writer).With().Timestamp().Logger()
	return writer
}

func main() {
	logger := initLogger()
	defer func() { _ = logger.Close() }()
	go consumeBroadcast()
	app := createApp(logger)
	url := fmt.Sprintf("%s:%s", getEnv("HOST", "localhost"), getEnv("PORT", "8000"))
	log.Info().Msgf("App is listing at %s", url)
	if err := app.Listen(url); err != nil {
		log.Fatal().Msgf("Listen to %s failed with %s", url, err)
	}
}

func getEnv(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		val = def
	}
	return val
}
