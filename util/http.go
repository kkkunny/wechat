package util

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strings"
	"time"
)

// TimeOut 全局请求超时设置,默认1分钟
var TimeOut time.Duration = 60 * time.Second

// Proxy 代理
var Proxy func(*http.Request) (*url.URL, error)

// SetTimeOut 设置全局请求超时
func SetTimeOut(d time.Duration) {
	TimeOut = d
}

// SetProxy 设置全局代理
func SetProxy(p func(*http.Request) (*url.URL, error)) {
	Proxy = p
}

// httpClient() 带超时的http.Client
func httpClient() *http.Client {
	cli := &http.Client{Timeout: TimeOut}
	if Proxy != nil {
		cli.Transport = &http.Transport{Proxy: Proxy}
	}
	return cli
}

// GetJson 发送GET请求解析json
func GetJson(uri string, v interface{}) error {

	r, err := httpClient().Get(uri)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// GetXml 发送GET请求并解析xml
func GetXml(uri string, v interface{}) error {
	r, err := httpClient().Get(uri)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return xml.NewDecoder(r.Body).Decode(v)
}

// GetBody 发送GET请求，返回body字节
func GetBody(uri string) ([]byte, error) {
	resp, err := httpClient().Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http get err: uri=%v , statusCode=%v", uri, resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

// GetRawBody 发送GET请求，返回body字节
// func GetRawBody(uri string) (io.ReadCloser, error) {
// 	resp, err := httpClient().Get(uri)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("http get err: uri=%v , statusCode=%v", uri, resp.StatusCode)
// 	}
// 	return resp.Body, nil
// }

// PostJson 发送Json格式的POST请求
func PostJson(uri string, obj interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(obj)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient().Post(uri, "application/json;charset=utf-8", buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http post error : uri=%v , statusCode=%v", uri, resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

// PostJsonPtr 发送Json格式的POST请求并解析结果到result指针
func PostJsonPtr(uri string, obj interface{}, result interface{}, contentType ...string) (err error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	//	enc.SetEscapeHTML(false)
	err = enc.Encode(obj)
	if err != nil {
		return
	}
	ct := "application/json;charset=utf-8"
	if len(contentType) > 0 {
		ct = strings.Join(contentType, ";")
	}
	// fmt.Println("post buf:", buf.String()) // Debug
	resp, err := httpClient().Post(uri, ct, buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http post error : uri=%v , statusCode=%v", uri, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

// PostXmlPtr 发送Xml格式的POST请求并解析结果到result指针
func PostXmlPtr(uri string, obj interface{}, result interface{}) (err error) {
	buf := new(bytes.Buffer)
	enc := xml.NewEncoder(buf)
	//	enc.SetEscapeHTML(false)
	err = enc.Encode(obj)
	if err != nil {
		return
	}

	resp, err := httpClient().Post(uri, "application/xml;charset=utf-8", buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http post error : uri=%v , statusCode=%v", uri, resp.StatusCode)
	}
	return xml.NewDecoder(resp.Body).Decode(result)
}

// PostFileBytes 上传文件
func PostFileBytes(fieldname string, filename string, contentType string, data []byte, uri string) ([]byte, error) {
	fields := []MultipartFormField{
		{
			Fieldname:   fieldname,
			Value:       data,
			ContentType: contentType,
			Filename:    filename,
		},
	}
	return PostMultipartForm(fields, uri)
}

// GetFile 下载文件
func GetFile(filename, uri string) error {
	resp, err := httpClient().Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	return err
}

// MultipartFormField 文件或其他表单数据
type MultipartFormField struct {
	Fieldname   string
	Value       []byte
	ContentType string
	Filename    string
}

// PostMultipartForm 上传文件或其他表单数据
func PostMultipartForm(fields []MultipartFormField, uri string) (respBody []byte, err error) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	for _, field := range fields {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"; filelength=%d`, field.Fieldname, field.Filename, len(field.Value)))
		h.Set("Content-Type", field.ContentType)
		partWriter, e := bodyWriter.CreatePart(h)
		if e != nil {
			err = e
			return
		}
		valueReader := bytes.NewReader(field.Value)
		if _, err = io.Copy(partWriter, valueReader); err != nil {
			return
		}
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, e := httpClient().Post(uri, contentType, bodyBuf)
	if e != nil {
		err = e
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}
	return ioutil.ReadAll(resp.Body)
}
