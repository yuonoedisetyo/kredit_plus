package main

import (
	"bytes"
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

type dataTransaction struct {
	IdKonsumen    string
	NomorKontrak  string
	OTR           float64
	AdminFee      float64
	JumlahCicilan int
	JumlahBunga   float64
	NamaAsset     string
	ParamKey      string
}

func transaction(c *gin.Context) {

	var errorMessage string
	jTransaction := dataTransaction{}

	log.SetFlags(0)

	// ------ start log file ------
	startTime = time.Now()
	dateNow := startTime.Format("2006-01-02")

	logFILE := logfile + "transaction_" + dateNow + ".log"

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
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	is_Json := isJSON(bodyString)
	if is_Json == false {
		errorMessage = "Error, Body - invalid json data"
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	var contentType string

	if values, _ := c.Request.Header["Content-Type"]; len(values) > 0 {
		contentType = values[0]
	}
	if contentType != "application/json" {
		errorMessage = "Error, Header - Content-Type is not application/json or empty value "
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	err1 := c.BindJSON(&jTransaction)
	if err1 != nil {
		errorMessage = "Error, Bind Json Data"
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	var signature string
	if values, _ := c.Request.Header["Signature"]; len(values) > 0 {
		signature = values[0]
	}
	if signature == "" {
		errorMessage = "Error, Header - Signature can't empty value"
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	ParamKey := trimReplace(jTransaction.ParamKey)
	if ParamKey == "" {
		errorMessage = errorMessage + "; " + "ParamKey - can't null value"
	}

	IdKonsumen := trimReplace(jTransaction.IdKonsumen)
	if IdKonsumen == "" {
		errorMessage = errorMessage + "; " + "IdKonsumen - can't null value"
	}
	NomorKontrak := trimReplace(jTransaction.NomorKontrak)
	if NomorKontrak == "" {
		errorMessage = errorMessage + "; " + "NomorKontrak - can't null value"
	}
	OTR := (jTransaction.OTR)
	if OTR < 1 {
		errorMessage = errorMessage + "; " + "OTR - can't zero value"
	}
	// AdminFee := (jTransaction.AdminFee)
	// if AdminFee < 1 {
	// 	errorMessage = errorMessage + "; " + "AdminFee - can't zero value"
	// }

	AdminFee := 0.00
	AdminFee = (2.00 / 100) * OTR
	JumlahCicilan := (jTransaction.JumlahCicilan)
	if JumlahCicilan < 1 {
		errorMessage = errorMessage + "; " + "JumlahCicilan - can't null value"
	}
	JumlahBunga := 2 //persen
	// JumlahBunga := (jTransaction.JumlahBunga)
	// if JumlahBunga < 1 {
	// 	errorMessage = errorMessage + "; " + "JumlahBunga - can't null value"
	// }
	NamaAsset := trimReplace(jTransaction.NamaAsset)
	if NamaAsset == "" {
		errorMessage = errorMessage + "; " + "NamaAsset - can't null value"
	}

	if errorMessage != "" {
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	// ------ start Check Signature ------
	errorMessage = ""
	valid_Signature := validSignature(bodyString, ParamKey)

	if valid_Signature == "" {
		errorMessage = "Error, return Valid Signature is empty value"
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
	}

	if valid_Signature != signature && errorMessage == "" {
		errorMessage = "Error, Header - Incorrect Signature=" + valid_Signature + "=" + signature
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
	}

	if errorMessage != "" {
		return
	}
	errorMessage = ""
	// ------ end Check Signature ------

	var LimitKredit float64

	sJumlahCicilan := fmt.Sprintf("%d", JumlahCicilan)

	query0 := "SELECT LimitKredit FROM limit_kredit where IdKonsumen='" + IdKonsumen + "' and tenor='" + sJumlahCicilan + "'"

	if err0 := db.QueryRow(query0).Scan(&LimitKredit); err0 != nil {

		errorMessage = fmt.Sprintf("Error running %q: %+v", query0, err0)
		errorMessageReturn := "Error - mendapatkan Limit Kredit"
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessageReturn, c)
		return
	}
	sLimitKredit := fmt.Sprintf("%.2f", LimitKredit)
	if LimitKredit < OTR {

		errorMessage = "Limit kredit untuk tenor " + sJumlahCicilan + " bulan tidak mencukupi. Sisa limit kredit saat ini " + sLimitKredit + ""
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}

	LimitKredit = LimitKredit - OTR

	tx, err := db.Begin()

	query := fmt.Sprintf("update limit_kredit set LimitKredit='%.2f' where IdKonsumen='%s' and tenor='%d'", LimitKredit, IdKonsumen, JumlahCicilan)

	result, err := tx.Exec(query)
	if err != nil {
		errorMessage := fmt.Sprintf("Error running %q: %+v", query, err)
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessage, c)
		return
	}
	RowCNT, _ := result.RowsAffected()
	if RowCNT == 0 {
		errorMessage = "Error - Update data  "
		errorMessageReturn := "Error - Update LimitKredit - " + query
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errorMessage, errorMessageReturn, c)
		return
	}
	// ------ end Update mst_voucher_detail  ------

	tx.Commit()

	query1 := fmt.Sprintf("INSERT INTO transaksi (IdKonsumen, NomorKontrak, OTR, AdminFee, JumlahCicilan, JumlahBunga, NamaAsset,InputDate) values ('%s','%s','%2.f','%2.f','%d','%d','%s',now());", IdKonsumen, NomorKontrak, OTR, AdminFee, JumlahCicilan, JumlahBunga, NamaAsset)
	_, err = db.Exec(query1)
	if err != nil {
		errMSG := fmt.Sprintf("Error running %q: %+v", query1, err)
		dataLogTransaction(logData, ip, allHeader, bodyJson, "1", "1", errMSG, "Error running", c)
		return
	}

	dataLogTransaction(logData, ip, allHeader, bodyJson, "0", "0", "", "", c)
	return

}

func dataLogTransaction(logData string, ip string, allHeader string, bodyJson string, errorCode string, errorCodeReturn string, errorMsg string, errorMsgReturn string, c *gin.Context) {

	endTime = time.Now()
	endTimeStr := endTime.String()

	diff := endTime.Sub(startTime)
	diffStr := diff.String()

	query1 := fmt.Sprintf("insert into mst_transaction_log (Source_IP,Data_IN,Header,StartDateTime,FinishDateTime,Duration,ErrorCode,ErrorMessage) values ('%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s');", trimReplace(ip), trimReplace(bodyJson), trimReplace(allHeader), trimReplace(startTime.String()), trimReplace(endTimeStr), trimReplace(diffStr), trimReplace(errorCode), trimReplace(errorMsg))

	_, err1 := db.Exec(query1)
	if err1 != nil {

		errorMessage := fmt.Sprintf("Error running %q: %+v", query1, err1)

		returnDataJsonTransaction(logData, errorCode, errorMessage, c)
		return
	}

	rex := regexp.MustCompile(`\r?\n`)
	codeError := "200"

	if errorMsg != "" {
		codeError = "500"
	}

	logDataNew := rex.ReplaceAllString(logData+codeError+"~"+endTime.String()+"~"+diff.String()+"~"+errorMsg, "")
	log.Println(logDataNew)

	returnDataJsonTransaction(logData, errorCodeReturn, errorMsg, c)
	return

}

func returnDataJsonTransaction(logData string, ErrorCode string, ErrorMessage string, c *gin.Context) {

	if strings.Contains(ErrorMessage, "Error running") == true {
		ErrorMessage = "Error Execute data"
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
