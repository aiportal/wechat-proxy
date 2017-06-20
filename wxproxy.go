package main

import (
    "net/http"
    "log"
    "fmt"
    "strings"
    "io/ioutil"
    "net/url"
    "time"
    "os"
)

// Get wechat query parameters
func wechat_query(form *url.Values) string {
    signature, timestamp, nonce := form.Get("signature"), form.Get("timestamp"), form.Get("nonce")
    echostr := form.Get("echostr")
    if signature == "" {
        signature = form.Get("msg_signature")
    }
    query := fmt.Sprintf("signature=%s&timestamp=%s&nonce=%s", signature, timestamp, nonce)
    if echostr != "" {
        query += fmt.Sprintf("&echostr=%s", echostr)
    }
    return query
}

// Get absolute url to dispatch
func normalize_url(url string, query string) string {
    if ! strings.HasPrefix(url, "http") {
        url = "http://" + url
    }
    if ! strings.Contains(url, "?") {
        url += "?"
    } else {
        url += "&"
    }
    return url + query
}

// process request
func do_request(request *http.Request, ch chan []byte) {
    client := &http.Client{
        Timeout: 5 * time.Second,
    }
    response, err := client.Do(request)
    body := []byte("")
    if err == nil {
        if response.StatusCode == 200 {
            body, _ = ioutil.ReadAll(response.Body)
        }
        fmt.Printf("%q [%d] \n", request.URL, len(body))
    } else {
        fmt.Printf("%q [error] \n", request.URL)
        fmt.Println(err)
    }
    ch <- body
}

// process wechat message
func svc_process(w http.ResponseWriter, r *http.Request) {
    fmt.Println(r.RequestURI)

    r.ParseForm()
    call_urls := r.Form["call"]
    if len(call_urls) > 0 {
        query := wechat_query(&r.Form)
        fmt.Printf("Query %q \n", query)
        chs := make([]chan []byte, len(call_urls))

        // request all urls
        for i, _url := range call_urls {
            _url = normalize_url(_url, query)
            request, _ := http.NewRequest(r.Method, _url, r.Body)

            chs[i] = make(chan []byte)
            go do_request(request, chs[i])
        }

        // return first none empty result
        for _, ch := range chs {
            result := <-ch
            close(ch)
            if len(result) > 0 {
                w.Write(result)
                return
            }
        }
    }
    w.Write([]byte(""))
}


func main() {
    http.HandleFunc("/", svc_process)

    // custom port or address by command line argument
    address := ":8080"
    if len(os.Args) > 1 {
        address = os.Args[1]
        if ! strings.Contains(address, ":") {
            address = ":" + address
        }
    }
    fmt.Printf("wechat proxy starting at %q ...\n", address)
    log.Fatal(http.ListenAndServe(address, nil))
}

