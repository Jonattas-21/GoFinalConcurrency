package main

import (
	"database/sql"
	"encoding/gob"
	"final-project/data"
	fmt "fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alexedwards/scs/redisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gomodule/redigo/redis"
	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const webPort = 80

func main() {
	//conect to DB
	db := initDB()

	//create sessions
	session := initSession()

	//create logs
	infolog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorlog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	//create a wait group
	wg := sync.WaitGroup{}

	//set up the app config
	app := Config{
		Session:       session,
		DB:            db,
		Wait:          &wg,
		Infolog:       infolog,
		Errorlog:      errorlog,
		Models:        data.New(db),
		ErrorChan:     make(chan error),
		ErrorChanDone: make(chan bool),
	}

	//set up email
	app.Mailer = app.createEmail()
	go app.ListenForMail()

	//listen for signals
	go app.ListenForShoutdown()

	//listen for errors
	go app.ListenForErrors()

	//listem for web connectios
	app.server()

	fmt.Println("Hello World")
}

func (app *Config) ListenForErrors() {

	for {
		select {
		case err := <-app.ErrorChan:
			app.Errorlog.Println(err)
		case <-app.ErrorChanDone:
			return
		}
	}

}

func (app *Config) server() {
	//start http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: app.routes(),
	}

	app.Infolog.Println("Starting server..")
	err := srv.ListenAndServe()

	if err != nil {
		log.Panic(err)
	}
}

func initDB() *sql.DB {
	fmt.Println("connecting to the database")
	conn := connectToDB()

	if conn == nil {
		panic("failed to connect to the database")
	}
	return conn
}

func connectToDB() *sql.DB {
	var count int = 0

	dns := os.Getenv("DSN")
	if dns == "" {
		dns = "host=localhost port=5432 user=admin password=admin dbname=FinalConcurrency sslmode=disable"
	}

	log.Println("connection is: ", dns)

	for {

		connection, err := openDB(dns)

		if err != nil {
			log.Println("PG is not ready...")
		} else {
			log.Print("PG is ready!")
			return connection
		}
		if count > 10 {
			log.Fatal("DB connection failed")
			return nil
		}

		time.Sleep(1 * time.Second)
		count++
		continue
	}
}

func openDB(dns string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dns)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func initSession() *scs.SessionManager {
	//setup session
	session := scs.New()

	gob.Register(data.User{})

	session.Store = redisstore.New(initRedis())
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = true
	return session
}

func initRedis() *redis.Pool {
	pool := &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "localhost:6379")
		},
	}
	return pool
}

func (app *Config) ListenForShoutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	app.Shoutdown()
	os.Exit(0)
}

func (app *Config) Shoutdown() {
	app.Infolog.Println("Run cleanup tasks...")

	//block until wg is runing
	app.Wait.Wait()

	app.Mailer.DoneChan <- true
	app.ErrorChanDone <- true

	app.Infolog.Println("Server is shut down")
	close(app.Mailer.MailertChan)
	close(app.Mailer.ErrorChan)
	close(app.Mailer.DoneChan)
	close(app.ErrorChan)
	close(app.ErrorChanDone)
}

func (app *Config) ListenForMail() {
	for {
		select {
		case msg := <-app.Mailer.MailertChan:
			go app.Mailer.SendEMail(msg, app.Mailer.ErrorChan)
		case err := <-app.Mailer.ErrorChan:
			app.Errorlog.Println("Error sending email: ", err)
		case <-app.Mailer.DoneChan:
			return
		}
	}
}

func (app *Config) createEmail() Mail {
	errorChan := make(chan error)
	mailerChan := make(chan Message, 1000)
	mailerDoneChan := make(chan bool)

	m := Mail{
		Domain:      "localhost",
		Host:        "localhost",
		Port:        1025,
		Encryption:  "none",
		FromName:    "info",
		FromAdress:  "Info@mycompany.com",
		Wait:        app.Wait,
		ErrorChan:   errorChan,
		MailertChan: mailerChan,
		DoneChan:    mailerDoneChan,
	}

	return m
}
