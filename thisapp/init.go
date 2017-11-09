package thisapp

import (
	"log"
	"fmt"
	"context"
	"expvar"
	"net/http"
	"time"
	"encoding/json"

	"gopkg.in/gcfg.v1"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/garyburd/redigo/redis"

	"github.com/alvintzz/docker-demo/nsq/publisher"
	"github.com/alvintzz/docker-demo/nsq/consumer"
)

//======================================================[Config Section]

type Configs struct {
	Settings       ConfigSetting
	Databases      ConfigDB
	Redis          ConfigRedis
	NSQs           ConfigNSQ
}
type ConfigSetting struct {
	SelfURL     string
	SelfPort    string
	PublicDir   string
	TemplateDir string
}
type ConfigDB struct {
	Conn string
	Type string
}
type ConfigRedis struct {
	Connection  string
  	IdleTimeout int64
  	MaxIdle     int
  	MaxActive   int
}
type ConfigNSQ struct {
	NSQD     string
  	Lookupds string
}

func readConfig(env string) (Configs, error) {
	fileName := fmt.Sprintf("simple_db.%s.ini", env)
	filePath := "/etc/config/"
	if env == "development" {
		filePath = "files/etc/config/"
	}

	var config Configs
	configLocation := fmt.Sprintf("%s%s", filePath, fileName)

	err := gcfg.ReadFileInto(&config, configLocation)
	if err != nil {
		err = fmt.Errorf("Failed to read config in %s. Error: %s", configLocation, err.Error())
	}
	return config, err
}

//=====================================================================


//===================================================[Product Database]
type PaymentDB interface {
	GetAll(ctx context.Context, orderBy, orderFrom string) ([]Payment, error)
}

type PostgresPaymentDB struct {
	connection *sqlx.DB
}

func NewPaymentDB(config *Configs) (*PostgresPaymentDB, error) {
	return newPaymentPostgresDB(config)
}

func newPaymentPostgresDB(config *Configs) (*PostgresPaymentDB, error) {
	db, err := sqlx.Connect("postgres", config.Databases.Conn)
	if err != nil {
		return nil, fmt.Errorf("DB open connection error. Error: %s", err.Error())
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("DB ping connection error. Error: %s", err.Error())
	}

	database := &PostgresPaymentDB{
		connection: db,
	}

	return database, nil
}

type Payment struct {
	PaymentID    int    `db:"id"`
	CustomerName string `db:"customer_name"`
}

var getAllQuery = "SELECT id, customer_name FROM payments ORDER BY %s %s LIMIT 100"
func (db *PostgresPaymentDB) GetAll(ctx context.Context, orderBy, orderFrom string) ([]Payment, error) {
	payments := []Payment{}

	query := fmt.Sprintf(getAllQuery, orderBy, orderFrom)
	err := db.connection.Select(&payments, query)
	if err != nil {
		err = fmt.Errorf("Failed to get Payment because %s", err.Error())
	}

	return payments, err
}

//====================================================================


//===================================================[Visitor Redis]
type VisitorDB interface {
	Get(ctx context.Context) (int, error)
	Set(ctx context.Context) error
}

type RedisVisitorDB struct {
	connection RedisConnection
}

func NewVisitorDB(config *Configs) (*RedisVisitorDB, error) {
	return newVisitorRedisDB(config)
}

func newVisitorRedisDB(config *Configs) (*RedisVisitorDB, error) {
	conn := NewRedisConnection(config.Redis.Connection, config.Redis.MaxIdle, config.Redis.MaxActive, config.Redis.IdleTimeout)
	red := &RedisVisitorDB{
		connection: conn,
	}

	return red, nil
}

func (red *RedisVisitorDB) Get(ctx context.Context) (int, error) {
	conn := red.connection.GetConn()
	defer conn.Close()

	count, err := redis.Int(conn.Do("GET", "visitor_count_today"))
	if err != nil {
		err = fmt.Errorf("Failed to get Visitor Count because %s", err.Error())
	}

	return count, err
}

func (red *RedisVisitorDB) Set(ctx context.Context) error {
	conn := red.connection.GetConn()
	defer conn.Close()

	_, err := conn.Do("INCR", "visitor_count_today")
	if err != nil {
		err = fmt.Errorf("Failed to increment Visitor Count because %s", err.Error())
	}

	return err
}

//====================================================================


//===================================================[Delay Insert NSQ]



//====================================================================


//=========================================================[App Module]

type AppModule struct {
	config      *Configs
	stats       *expvar.Int
	log         *log.Logger
	httpClient  http.Client
	PaymentDB   PaymentDB
	VisitorDB   VisitorDB
	ProducerNSQ publisher.ProducerQueue
}

func NewAppModule(env string) (*AppModule) {
	config, err := readConfig(env)
	if err != nil {
		log.Fatalf("Failed to read config because %s", err.Error())
	}

	paymentDB, err := NewPaymentDB(&config)
	if err != nil {
		log.Fatalf("Failed to create product DB because %s", err.Error())
	}

	visitorDB, err := NewVisitorDB(&config)
	if err != nil {
		log.Fatalf("Failed to create visitor DB because %s", err.Error())
	}

	pub, err := publisher.AddPublisher(config.NSQs.NSQD)
	if err != nil {
		log.Fatalf("Failed to create NSQ Publisher because %s", err.Error())
	}

	consManager := consumer.NewConsumerManager(config.NSQs.NSQD)
	consManager.Register("ini_testing_docker", "ini_channel_docker", consumer.SimpleProcess)
	err = consManager.Run()
	if err != nil {
		log.Fatalf("Failed to create NSQ Consumer because %s", err.Error())
	}

	return &AppModule{
		config:      &config,
		PaymentDB:   paymentDB,
		VisitorDB:   visitorDB,
		ProducerNSQ: pub,
	}
}

//====================================================================


//===========================================================[Handlers]

type Response struct {
	ServerProcessTime string      `json:"server_process_time"`
	ErrorMessage      []string    `json:"message_error,omitempty"`
	StatusMessage     []string    `json:"message_status,omitempty"`
	Data interface{} `json:"data"`
}

type ResponseJSON func(rw http.ResponseWriter, r *http.Request) (interface{}, error)

func (fn ResponseJSON) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//JSON Universal Response Format
	response := Response{}

	//Init Context
	ctx := r.Context()

	//Set a 10 second timeout on responses. TODO: make this come from config
	ctx, cancelFn := context.WithTimeout(ctx, 10 * time.Second)
	defer cancelFn()

	//Add context into HTTP Request
	r = r.WithContext(ctx)

	//Start Timer
	start := time.Now()

	//Do the Function
	data, err := fn(w, r)

	//Record Elapsed Time
	response.ServerProcessTime = time.Since(start).String()

	w.Header().Set("Content-Type", "application/json")

	if data != nil {
		response.Data = data
		if buf, err := json.Marshal(response); err == nil {
			w.Write(buf)
			return
		}
	}

	if err != nil {
		response.ErrorMessage = []string{
			err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
	}

	buf, _ := json.Marshal(response)
	w.Write(buf)
	return
}

func (am *AppModule) DatabaseAPIHandler(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	ctx := r.Context()

	orderBy := r.FormValue("order_by")
	orderFrom := r.FormValue("order_from")

	products, err := am.PaymentDB.GetAll(ctx, orderBy, orderFrom)
	if err != nil {
		log.Println(err.Error())
		return nil, fmt.Errorf("Gagal get data dari database")
	}

	return products, nil
}

func (am *AppModule) RedisGetHandler(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	ctx := r.Context()

	count, err := am.VisitorDB.Get(ctx)
	if err != nil {
		log.Println(err.Error())
		return nil, fmt.Errorf("Gagal get data dari redis")
	}

	visitor := map[string]int{
		"Visitor": count,
	}

	return visitor, nil
}

func (am *AppModule) RedisSetHandler(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	ctx := r.Context()

	err := am.VisitorDB.Set(ctx)
	if err != nil {
		log.Println(err.Error())
		return nil, fmt.Errorf("Gagal increment data redis")
	}

	return map[string]string{}, nil
}

func (am *AppModule) PublishNSQHandler(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	ctx := r.Context()

	data := map[string]string{
		"message": r.FormValue("message"),
		"id":      r.FormValue("id"),
	}

	err := am.ProducerNSQ.Publish(ctx, "ini_testing_docker", data)
	if err != nil {
		log.Println(err.Error())
		return nil, fmt.Errorf("Gagal publish data to nsq")
	}

	return map[string]string{}, nil
}

func (am *AppModule) InitHandlers(mux *http.ServeMux) {
	mux.Handle("/api/db", ResponseJSON(am.DatabaseAPIHandler))
	mux.Handle("/api/redis/get", ResponseJSON(am.RedisGetHandler))
	mux.Handle("/api/redis/set", ResponseJSON(am.RedisSetHandler))
	mux.Handle("/api/nsq/set", ResponseJSON(am.PublishNSQHandler))
}

//====================================================================
