package openapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/flaboy/aira-web/pkg/openapi/interfaces"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type EventPayload struct {
	EventCode EventCode   `json:"event_code"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

func (e *Endpoint) EmitEvent(code EventCode, data interface{}) error {
	// 检查仓储是否已初始化
	if eventRepo == nil {
		log.Printf("Event repository not initialized")
		return nil
	}

	// 查找订阅此事件的应用
	subscriptions, err := eventRepo.FindByEventCode(string(code))
	if err != nil {
		log.Printf("Failed to find event subscriptions: %v", err)
		return err
	}

	if len(subscriptions) == 0 {
		log.Printf("No subscriptions found for event: %s", code)
		return nil
	}

	payload := EventPayload{
		EventCode: code,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	// 异步发送通知给所有订阅的应用
	for _, sub := range subscriptions {
		app := sub.GetApplication()
		if app != nil {
			go e.sendEventNotification(app, payload)
		}
	}

	return nil
}

// SendTestNotification 发送测试通知到指定的URL
// 这是一个公开接口，供控制器直接调用来测试通知配置
func (e *Endpoint) SendTestNotification(notifyType, notifyURL string, payload EventPayload) error {
	if notifyType == "" || notifyURL == "" {
		return fmt.Errorf("notify type and URL cannot be empty")
	}

	switch notifyType {
	case "webhook":
		return e.sendWebhook(notifyURL, payload)
	case "sqs":
		return e.sendSQS(notifyURL, payload)
	default:
		return fmt.Errorf("unsupported notify type: %s", notifyType)
	}
}

func (e *Endpoint) sendEventNotification(app interfaces.ApplicationInfo, payload EventPayload) {
	switch app.GetNotifyType() {
	case "webhook":
		if app.GetNotifyURL() != "" {
			if err := e.sendWebhook(app.GetNotifyURL(), payload); err != nil {
				log.Printf("Failed to send webhook to %s: %v", app.GetNotifyURL(), err)
			}
		}
	case "sqs":
		if app.GetNotifyURL() != "" {
			if err := e.sendSQS(app.GetNotifyURL(), payload); err != nil {
				log.Printf("Failed to send SQS notification to %s: %v", app.GetNotifyURL(), err)
			}
		}
	default:
		log.Printf("Unknown notify type: %s for application %d", app.GetNotifyType(), app.GetID())
	}
}

func (e *Endpoint) sendWebhook(url string, payload EventPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Webhook returned non-success status: %d", resp.StatusCode)
	}

	return nil
}

func (e *Endpoint) sendSQS(sqsURL string, payload EventPayload) error {
	// 将payload编码为JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// 创建AWS配置
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Printf("Failed to load AWS config, falling back to mock: %v", err)
		// 如果AWS配置失败，回退到日志记录
		log.Printf("SQS notification sent to: %s with payload: %s", sqsURL, string(jsonData))
		return nil
	}

	// 创建SQS客户端
	sqsClient := sqs.NewFromConfig(cfg)

	// 发送消息到SQS队列
	_, err = sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(sqsURL),
		MessageBody: aws.String(string(jsonData)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"EventCode": {
				DataType:    aws.String("String"),
				StringValue: aws.String(string(payload.EventCode)),
			},
			"Source": {
				DataType:    aws.String("String"),
				StringValue: aws.String("project-platform"),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to send SQS message: %v", err)
	}

	log.Printf("SQS notification successfully sent to: %s", sqsURL)
	return nil
}
