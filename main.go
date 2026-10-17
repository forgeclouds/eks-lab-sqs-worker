// sqs-worker: the smallest queue consumer that still behaves well on Kubernetes.
//
//	worker            long-poll QUEUE_URL, "process" each message (sleep), delete it
//	worker -produce N send N test messages and exit (load for the KEDA lab)
//
// One message per receive keeps the demo honest: throughput scales with pod
// count, which is exactly what KEDA adds. AWS creds come from the default
// chain, i.e. EKS Pod Identity in the cluster, your profile locally.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

var version = "dev" // set at build time: -ldflags "-X main.version=<sha>"

func main() {
	produce := flag.Int("produce", 0, "send N test messages and exit")
	flag.Parse()

	queueURL := os.Getenv("QUEUE_URL")
	if queueURL == "" {
		log.Fatal("QUEUE_URL is required")
	}
	work := 2 * time.Second
	if v, err := time.ParseDuration(os.Getenv("PROCESS_TIME")); err == nil {
		work = v
	}

	// SIGTERM cancels ctx: we stop receiving but finish the current message.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := sqs.NewFromConfig(cfg)

	if *produce > 0 {
		for i := 0; i < *produce; i++ {
			body := fmt.Sprintf("job-%d-%d", time.Now().Unix(), i)
			if _, err := client.SendMessage(ctx, &sqs.SendMessageInput{QueueUrl: &queueURL, MessageBody: &body}); err != nil {
				log.Fatal(err)
			}
		}
		log.Printf("sent %d messages", *produce)
		return
	}

	host, _ := os.Hostname()
	log.Printf("worker %s version=%s polling %s", host, version, queueURL)
	for ctx.Err() == nil {
		out, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &queueURL,
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     20, // long polling: fewer empty receives, lower cost
		})
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("receive: %v", err)
				time.Sleep(5 * time.Second)
			}
			continue
		}
		for _, m := range out.Messages {
			log.Printf("processing %s", aws.ToString(m.Body))
			time.Sleep(work)
			// Delete with a fresh context so a SIGTERM mid-message doesn't
			// leave it to reappear after the visibility timeout.
			if _, err := client.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
				QueueUrl: &queueURL, ReceiptHandle: m.ReceiptHandle,
			}); err != nil {
				log.Printf("delete: %v", err)
			}
		}
	}
	log.Print("SIGTERM: stopped cleanly")
}
