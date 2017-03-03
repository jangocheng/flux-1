package consumer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

	"time"

	"github.com/golang/glog"
	. "github.com/yehohanan7/flux/cqrs"
	. "github.com/yehohanan7/flux/feed"
	"github.com/yehohanan7/flux/utils"
)

//Consumes events from the command component
type JsonEventConsumer struct {
	url          string
	handlerClass interface{}
	handlers     Handlers
}

//Send event to the consumer
func (consumer *JsonEventConsumer) send(event Event) {
	payload := event.Payload
	if handler, ok := consumer.handlers[reflect.TypeOf(payload)]; ok {
		handler(consumer.handlerClass, payload)
	}
}

func fetchJsonInto(url string, data interface{}) error {
	var body []byte
	res, err := http.Get(url)
	if err != nil {
		glog.Error("Error while getting ", err)
		return err
	}

	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		glog.Error("Error while reading the data ", err)
		return err
	}

	err = json.Unmarshal(body, data)

	fmt.Println("body: ", string(body))

	if err != nil {
		glog.Error("Error while decoding data ", err)
		return err
	}

	return nil
}

func (consumer *JsonEventConsumer) getEventFeed() (JsonEventFeed, error) {
	var feed = new(JsonEventFeed)
	err := fetchJsonInto(consumer.url, feed)
	if err != nil {
		return *feed, err
	}
	return *feed, nil
}

func (consumer *JsonEventConsumer) getEvent(entry EventEntry) (interface{}, error) {
	event := new(Event)
	for eventType, _ := range consumer.handlers {
		if eventType.String() == entry.EventType {
			event.Payload = reflect.New(eventType).Interface()
			err := fetchJsonInto(entry.Url, event)
			if err == nil {
				return reflect.ValueOf(event.Payload).Elem().Interface(), err
			}
		}
	}
	return nil, nil
}

func (consumer *JsonEventConsumer) Start() error {

	go utils.Every(3*time.Second, func() {
		glog.Info("Fetching events...")
		feed, err := consumer.getEventFeed()
		if err != nil {
			return
		}

		for _, entry := range feed.Events {
			event, err := consumer.getEvent(entry)
			glog.Info("event fetched", event)
			if err != nil {
				glog.Error(err)
			}

			fmt.Println("event type: ", reflect.TypeOf(event))
			fmt.Println(consumer.handlers)
			if handler, ok := consumer.handlers[reflect.TypeOf(event)]; ok {
				handler(consumer.handlerClass, event)
			}
		}
	})

	return nil
}

func (consumer *JsonEventConsumer) Stop() error {
	return nil
}

//New json event consumer
func NewEventConsumer(url string, handlerClass interface{}, store OffsetStore) EventConsumer {
	return &JsonEventConsumer{url, handlerClass, NewHandlers(handlerClass)}
}
