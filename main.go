package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
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

func main() {

        /////////// for locat testing ////////////
        // err := godotenv.Load(".env")
    
        // if err != nil {
        //    log.Fatal("Error loading .env file")
        //  }
        //////////////////////////////////////////

	MINT_ACCOUNT := os.Getenv("MINT_ACCOUNT")
        DATABASE_URL := os.Getenv("DB_URL")
        MQ_ENDPOINT := os.Getenv("MQ_ENDPOINT")

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
                "",     // consumer
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

        sqlStatementInsert := fmt.Sprintf(`INSERT INTO %s (owner, amount, slot) VALUES ($1, $2, $3) ON CONFLICT(owner) WHERE $3 >= slot DO UPDATE SET amount=$2, slot=$3 RETURNING owner`,MINT_ACCOUNT)
        log.Printf("connected") 
        forever := make(chan bool)

        go func() {
                for d := range msgs {

                        message := new(Msg)
                        err = borsh.Deserialize(message, d.Body)
                        failOnError(err, "deserialize msg failed")
                        
                        owner := ""
                        error := dbpool.QueryRow(context.Background(),sqlStatementInsert, message.Owner, message.Amount, message.Slot).Scan(&owner)       
                        failOnError(error, "Insert into DB Failed")
                       
                        // log.Printf("{")
                        // log.Printf(" [owner] %s", owner)
                        // log.Printf(" [amount] %s", message.Amount)
                        // log.Printf(" [slot] %d", message.Slot)
                        // log.Printf("}")
                        d.Ack(false)
        
                }
        }()
        <-forever

}