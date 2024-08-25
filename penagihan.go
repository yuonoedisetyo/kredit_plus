package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"strings"
)

type dataPenagihan struct {
	IdKonsumen     string
	NomorKontrak   string
	TotalPenagihan float64
	Periode        string
	ParamKey       string
}

func penagihan(c *gin.Context) {

	var errorMessage string
	jPenagihan := dataPenagihan{}

	log.SetFlags(0)

	// ------ start log file ------
	startTime = time.Now()
	dateNow := startTime.Format("2006-01-02")

	logFILE := logfile + "penagihan_" + dateNow + ".log"

	file, err := os.OpenFile(logFILE, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	log.SetOutput(file)
	// ------ end log file ------

	startTimeStr := startTime.String()

	// ------ start Read Header ------
	allHeader := readAllHeader(c)
	// ------ end Read Header ------

	method := c.Request.Method
	path := c.Request.URL.EscapedPath()

	XRealIp := ""
	if values, _ := c.Request.Header["X-Real-Ip"]; len(values) > 0 {
		XRealIp = values[0]
	}
	if XRealIp == "" {
	}

	var ip string

	ip = c.ClientIP()
	logData := startTimeStr + "~" + ip + "~" + method + "~" + path + "~" + allHeader + "~"

	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
	}
	// Restore the io.ReadCloser to its original state
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	bodyString := string(bodyBytes)

	bodyJson := trimReplace(string(bodyString))

	rex := regexp.MustCompile(`\r?\n`)
	logData = logData + rex.ReplaceAllString(bodyJson, "") + "~"

	if string(bodyString) == "" {
		errorMessage = "Error, Body is empty"
		dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	is_Json := isJSON(bodyString)
	if is_Json == false {
		errorMessage = "Error, Body - invalid json data"
		dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	var contentType string

	if values, _ := c.Request.Header["Content-Type"]; len(values) > 0 {
		contentType = values[0]
	}
	if contentType != "application/json" {
		errorMessage = "Error, Header - Content-Type is not application/json or empty value "
		dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	err1 := c.BindJSON(&jPenagihan)
	if err1 != nil {
		errorMessage = "Error, Bind Json Data"
		dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	var signature string
	if values, _ := c.Request.Header["Signature"]; len(values) > 0 {
		signature = values[0]
	}
	if signature == "" {
		errorMessage = "Error, Header - Signature can't empty value"
		dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	ParamKey := trimReplace(jPenagihan.ParamKey)
	if ParamKey == "" {
		errorMessage = errorMessage + "; " + "ParamKey - can't null value"
	}

	if errorMessage != "" {
		dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	// ------ start Check Signature ------
	errorMessage = ""
	valid_Signature := validSignature(bodyString, ParamKey)

	if valid_Signature == "" {
		errorMessage = "Error, return Valid Signature is empty value"
		dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
	}

	if valid_Signature != signature && errorMessage == "" {
		errorMessage = "Error, Header - Incorrect Signature=" + valid_Signature + "=" + signature
		dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
	}

	if errorMessage != "" {
		return
	}
	errorMessage = ""
	// ------ end Check Signature ------

	jPenagihans := []dataPenagihan{}

	query := `SELECT 
	IdKonsumen,
	NomorKontrak,
	sum(OTR) as JumlahTagihan FROM transaksi where StatusPenagihan='PENDING' and InputDate >= DATE_SUB(CURDATE(), INTERVAL 1 MONTH) group by IdKonsumen`

	rows, err := db.Query(query)
	defer rows.Close()
	if err != nil {
		errorMessage := fmt.Sprintf("Error running %q: %+v", query, err)

		dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}
	for rows.Next() {
		err = rows.Scan(&jPenagihan.IdKonsumen, &jPenagihan.NomorKontrak, &jPenagihan.TotalPenagihan)
		jPenagihans = append(jPenagihans, jPenagihan)
		if err != nil {

			errorMessage := fmt.Sprintf("Error running %q: %+v", query, err)

			dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
			return
		}
	}

	sliceLength := len(jPenagihans)

	for i := 0; i < sliceLength; i++ {
		go func(i int) {

			IdKonsumen := jPenagihans[i].IdKonsumen
			TotalPenagihan := jPenagihans[i].TotalPenagihan

			IdPenagihan := generateRandomID()
			currentTime := time.Now()
			Periode := currentTime.Format("January")

			query1 := fmt.Sprintf("INSERT INTO penagihan (IdPenagihan, IdKonsumen, TotalTagihan, Periode, InputDate) values ('%s','%s','%2.f','%s',now());", IdPenagihan, IdKonsumen, TotalPenagihan, Periode)
			_, err = db.Exec(query1)
			if err != nil {
				errorMessage := fmt.Sprintf("Error running %q: %+v", query1, err)
				dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
				return
			}

			tx, err := db.Begin()

			query := fmt.Sprintf("update transaksi set StatusPenagihan='TERTAGIH' where IdKonsumen='%s'", IdKonsumen)

			result, err := tx.Exec(query)
			if err != nil {
				errorMessage := fmt.Sprintf("Error running %q: %+v", query, err)
				dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
				return
			}
			RowCNT, _ := result.RowsAffected()
			if RowCNT == 0 {
				errorMessage = "Error - Update data  "
				errorMessageReturn := "Error - Update StatusPenagihan - " + query
				dataLogPenagihan(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessageReturn, c)
				return
			}
			// ------ end Update mst_voucher_detail  ------

			tx.Commit()

		}(i)
	}

	dataLogPenagihan(logData, ip, allHeader, bodyJson, "0", "0", "", "", c)
	return

}

func generateRandomID() string {
	// Get the current date and time
	currentTime := time.Now().Format("20060102150405") // Format as YYYYMMDDHHMMSS

	// Generate a random 8-byte slice
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	// Convert the random bytes to a hexadecimal string
	randomHex := hex.EncodeToString(randomBytes)

	// Combine the datetime string with the random hex string
	randomID := fmt.Sprintf("%s%s", currentTime, randomHex)

	return randomID
}

func dataLogPenagihan(logData string, ip string, allHeader string, bodyJson string, errorCode string, errorCodeReturn string, errorMsg string, errorMsgReturn string, c *gin.Context) {

	endTime = time.Now()
	endTimeStr := endTime.String()

	diff := endTime.Sub(startTime)
	diffStr := diff.String()

	query1 := fmt.Sprintf("insert into mst_transaction_log (Source_IP,Data_IN,Header,StartDateTime,FinishDateTime,Duration,ErrorCode,ErrorMessage) values ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s');", trimReplace(ip), trimReplace(bodyJson), trimReplace(allHeader), trimReplace(startTime.String()), trimReplace(endTimeStr), trimReplace(diffStr), trimReplace(errorCode), trimReplace(errorMsg))

	_, err1 := db.Exec(query1)
	if err1 != nil {

		errorMessage := fmt.Sprintf("Error running %q: %+v", query1, err1)

		returnDataJsonPenagihan(logData, errorCode, errorMessage, c)
		return
	}

	rex := regexp.MustCompile(`\r?\n`)
	codeError := "200"

	if errorMsg != "" {
		codeError = "500"
	}

	logDataNew := rex.ReplaceAllString(logData+codeError+"~"+endTime.String()+"~"+diff.String()+"~"+errorMsg, "")
	log.Println(logDataNew)

	returnDataJsonPenagihan(logData, errorCodeReturn, errorMsg, c)
	return

}

func returnDataJsonPenagihan(logData string, ErrorCode string, ErrorMessage string, c *gin.Context) {

	if strings.Contains(ErrorMessage, "Error running") == true {
		// ErrorMessage = "Error Execute data"
	}

	if ErrorCode == "504" {
		c.String(http.StatusUnauthorized, "")
	} else {

		c.PureJSON(http.StatusOK, gin.H{
			"ErrorCode":    ErrorCode,
			"ErrorMessage": ErrorMessage,
		})
	}

	runtime.GC()

	return
}
