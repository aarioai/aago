package driver

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/aa/alog"
	"github.com/rabbitmq/amqp091-go"
	rabbit "github.com/wagslane/go-rabbitmq"
)

func CheckMongodbHealth(section string) func(app *aa.App, errChan chan<- error) {
	return func(app *aa.App, errChan chan<- error) {
		alog.Console("check health -> mongodb [" + section + "]")
		ctx := context.Background()
		client, _, e := NewMongodb(app, section)
		if e != nil {
			errChan <- fmt.Errorf("mongodb (%s) connect error: %s", section, e.Msg)
			return
		}
		defer client.Disconnect(ctx)

		if err := client.Ping(ctx, nil); err != nil {
			errChan <- fmt.Errorf("mongodb (%s) ping error: %s", section, err.Error())
			return
		}
		alog.Console("mongodb [" + section + "] is healthy")
		errChan <- nil
	}
}

func CheckMySQLHealth(section string) func(app *aa.App, errChan chan<- error) {
	return func(app *aa.App, errChan chan<- error) {
		alog.Console("check health -> mysql [" + section + "]")
		_, db, e := NewMysql(app, section)
		if e != nil {
			errChan <- fmt.Errorf("mysql (%s) connect error: %s", section, e.Msg)
			return
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			errChan <- fmt.Errorf("mysql (%s) ping error: %s", section, err.Error())
			return
		}
		alog.Console("mysql [" + section + "] is healthy")
		errChan <- nil
	}
}

func CheckPostgreHealth(section string) func(app *aa.App, errChan chan<- error) {
	return func(app *aa.App, errChan chan<- error) {
		alog.Console("check health -> postgres [" + section + "]")
		_, pool, e := NewPostgres(app, section)
		if e != nil {
			errChan <- fmt.Errorf("postgres (%s) connect error: %s", section, e.Msg)
			return
		}
		defer pool.Close()
		if err := pool.Ping(context.Background()); err != nil {
			errChan <- fmt.Errorf("postgres (%s) ping error: %s", section, err.Error())
			return
		}
		alog.Console("postgres [" + section + "] is healthy")
		errChan <- nil
	}
}

func CheckRabbitmqHealth(section string, tlsConfig *tls.Config, sasl []amqp091.Authentication, opts []func(*rabbit.ConnectionOptions)) func(app *aa.App, errChan chan<- error) {
	return func(app *aa.App, errChan chan<- error) {
		alog.Console("check health -> rabbitmq [" + section + "]")
		conn, e := NewRabbitmqPool(app, section, tlsConfig, sasl, opts)
		if e != nil {
			errChan <- fmt.Errorf("rabbitmq (%s) connect error: %s", section, e.Msg)
			return
		}
		consumer, err := rabbit.NewConsumer(
			conn,
			"temp_health_check_queue",
			rabbit.WithConsumerOptionsQueueAutoDelete, // delete after all
			rabbit.WithConsumerOptionsLogging,
		)
		if err != nil {
			errChan <- fmt.Errorf("rabbitmq (%s) ping error: %s", section, err.Error())
			return
		}
		defer consumer.Close()
		alog.Console("rabbitmq [" + section + "] is healthy")
		errChan <- nil
	}
}

func CheckInfluxdbHealth(section string) func(app *aa.App, errChan chan<- error) {
	return func(app *aa.App, errChan chan<- error) {
		alog.Console("check health -> influxdb [" + section + "]")
		data, e := NewInfluxdbPool(app, section, nil)
		if e != nil {
			errChan <- fmt.Errorf("influxdb (%s) connect error: %s", section, e.Msg)
			return
		}
		if ok, err := data.Client.Ping(context.Background()); err != nil {
			errChan <- fmt.Errorf("influxdb (%s) ping error: %s", section, err.Error())
			return
		} else if !ok {
			errChan <- fmt.Errorf("influxdb (%s) is not healthy", section)
			return
		}
		alog.Console("influxdb [" + section + "] is healthy")
		errChan <- nil
	}
}

func CheckRedisHealth(section string) func(app *aa.App, errChan chan<- error) {
	return func(app *aa.App, errChan chan<- error) {
		alog.Console("check health -> redis [" + section + "]")
		client, e := NewRedis(app, section)
		if e != nil {
			errChan <- fmt.Errorf("redis (%s) connect error: %s", section, e.Msg)
			return
		}
		defer client.Close()
		_, err := client.Ping(context.Background()).Result()
		if err != nil {
			errChan <- fmt.Errorf("redis (%s) ping error: %s", section, err.Error())
			return
		}
		alog.Console("redis [" + section + "] is healthy")
		errChan <- nil
	}
}
