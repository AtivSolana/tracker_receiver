package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	"github.com/near/borsh-go"
	"github.com/streadway/amqp"
)

type Msg struct {
	Owner string
	Amount string
	Slot uint32  
}

func failOnError(err error, msg string) {
        if err != nil {
                log.Fatalf("%s: %s", msg, err)
        }
}

func goDotEnvVariable(key string) string {
        
        err := godotenv.Load()
        failOnError(err, "enviornment variables loading failed")

        return os.Getenv(key)
}



func main() {

        

	MINT_ACCOUNT := goDotEnvVariable("MINT_ACCOUNT")
        DATABASE_URL := goDotEnvVariable("DB_URL")
        MQ_ENDPOINT := goDotEnvVariable("MQ_ENDPOINT")

	conn, err := amqp.Dial(MQ_ENDPOINT)
        
        failOnError(err, "Failed to connect to RabbitMQ")
        defer conn.Close()

        ch, err := conn.Channel()
        failOnError(err, "Failed to open a channel")
        defer ch.Close()

        err = ch.ExchangeDeclare(
                "Account_Changed", // name
                "direct",      // type
                true,          // durable
                false,         // auto-deleted
                false,         // internal
                false,         // no-wait
                nil,           // arguments
        )
        failOnError(err, "Failed to declare an exchange")

        q, err := ch.QueueDeclare(
                MINT_ACCOUNT,    // name
                true, // durable
                false, // delete when unused
                false,  // exclusive
                false, // no-wait
                nil,   // arguments
        )
        failOnError(err, "Failed to declare a queue")

        err = ch.QueueBind(
                q.Name,        // queue name
                MINT_ACCOUNT,    // routing key
                "Account_Changed", // exchange
                false,
                nil)
        failOnError(err, "Failed to bind a queue")

        msgs, err := ch.Consume(
                q.Name, // queue
                "b",     // consumer
                false,   // auto ack
                false,  // exclusive
                false,  // no local
                false,  // no wait
                nil,    // args
        )
        failOnError(err, "Failed to register a consumer")

        dbpool, err := pgxpool.Connect(context.Background(), DATABASE_URL)
        failOnError(err, "Failed to connect Database")
        defer dbpool.Close()

        sqlStatement := fmt.Sprintf(`INSERT INTO %s (owner, amount, slot) VALUES ($1, $2, $3) ON CONFLICT(owner) DO UPDATE SET amount=$2, slot=$3 RETURNING owner`,MINT_ACCOUNT)
        
        forever := make(chan bool)

        go func() {
                for d := range msgs {

                        message := new(Msg)
                        err = borsh.Deserialize(message, d.Body)
                        failOnError(err, "deserialize msg failed")

                        if err != nil {
                                log.Printf(" [x] %s", err)
                        }
                        owner := ""
                        err := dbpool.QueryRow(context.Background(),sqlStatement, message.Owner, message.Amount, message.Slot).Scan(&owner)
                        failOnError(err, "insert into db failed")
                        log.Printf("{")
                        log.Printf(" [owner] %s", owner)
                        log.Printf(" [amount] %s", message.Amount)
                        log.Printf(" [slot] %d", message.Slot)
                        log.Printf("}")
                        d.Ack(false)
                }
        }()
        <-forever

}