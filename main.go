package main

import (
	//"strconv"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/nanmu42/gzip"
)

type LogFormatterParams struct {
	Request *http.Request

	// TimeStamp shows the time after the server returns a response.
	TimeStamp time.Time
	// StatusCode is HTTP response code.
	StatusCode int
	// Latency is how much time the server cost to process a certain request.
	Latency time.Duration
	// ClientIP equals Context's ClientIP method.
	ClientIP string
	// Method is the HTTP method given to the request.
	Method string
	// Path is a path the client requests.
	Path string
	// ErrorMessage is set if error has occurred in processing the request.
	ErrorMessage string

	// BodySize is the size of the Response Body
	BodySize int
	// Keys are the keys set on the request's context.
	Keys map[string]interface{}
	// contains filtered or unexported fields
}

func init() {

	err := godotenv.Load(".env")

	if err != nil {
		log.Println("Error loading .env file")
	}
}

var (
	//ctx context.Context
	db                                                                         *sql.DB
	connectionString, portrun, rootpath, logfile, tempfile, SecretKey, BaseURL string
	startTime                                                                  time.Time
	endTime                                                                    time.Time
	UserID                                                                     string
	totalPage                                                                  float64
	totalRecords                                                               int
)

func main() {
	logfile = os.Getenv("LOGFILE")
	tempfile = os.Getenv("TEMPFILE")
	connectionString = os.Getenv("STRINGCONNECTION")
	portrun = os.Getenv("PORTRUN")
	SecretKey = os.Getenv("SecretKey")
	BaseURL = os.Getenv("BASEURL")

	if logfile == "" || tempfile == "" || connectionString == "" || portrun == "" || SecretKey == "" {
		fmt.Println("Check Your .env file")
		return
	}

	db = connect()

	logFILE := logfile + "Access.log"

	f, _ := os.Create(logFILE)
	gin.DefaultWriter = io.MultiWriter(f)

	r := gin.New()
	r.ForwardedByClientIP = true
	r.Use(CORS())

	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {

		// your custom format
		return fmt.Sprintf("%s - [%s] \"%s\" \"%s\" \"%s\" \"%d\" \"%s\" \"%d\" \"%s\" \"%s\" \n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123Z),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.BodySize,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))

	r.Use(gin.Recovery())

	//r.Use(MaxAllowed(100))
	r.Use(gzip.DefaultHandler().Gin)

	r.LoadHTMLFiles("index.html")

	v1 := r.Group("api/v1")
	{
		v1.POST("/transaction", transaction) //internal use

	}

	r.Run(portrun)

}

func MaxAllowed(n int) gin.HandlerFunc {
	sem := make(chan struct{}, n)
	acquire := func() { sem <- struct{}{} }
	release := func() { <-sem }
	return func(c *gin.Context) {
		acquire()       // before request
		defer release() // after request
		c.Next()

	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Signature, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
		c.Writer.Header().Set("Content-Type", "application/json")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "SAMEORIGIN")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
